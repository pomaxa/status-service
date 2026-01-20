package http

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"status-incident/internal/domain"
	"strings"
)

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	enabled    bool
	username   string
	password   string
	apiKeyRepo domain.APIKeyRepository
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(enabled bool, username, password string, apiKeyRepo domain.APIKeyRepository) *AuthMiddleware {
	return &AuthMiddleware{
		enabled:    enabled,
		username:   username,
		password:   password,
		apiKeyRepo: apiKeyRepo,
	}
}

// IsEnabled returns whether authentication is enabled
func (m *AuthMiddleware) IsEnabled() bool {
	return m.enabled
}

// RequireAuth middleware that requires authentication for protected routes
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Try API key first (for programmatic access)
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			if user := m.validateAPIKey(r, apiKey); user != nil {
				ctx := domain.ContextWithUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Try Authorization header (Bearer token or Basic auth)
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if user := m.validateAPIKey(r, token); user != nil {
					ctx := domain.ContextWithUser(r.Context(), user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			} else if strings.HasPrefix(authHeader, "Basic ") {
				if user := m.validateBasicAuth(authHeader); user != nil {
					ctx := domain.ContextWithUser(r.Context(), user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// Check session cookie for web UI
		if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
			if user := m.validateSession(cookie.Value); user != nil {
				ctx := domain.ContextWithUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Return 401 with WWW-Authenticate header for browsers
		w.Header().Set("WWW-Authenticate", `Basic realm="Status Incident Admin"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// RequireAPIAuth middleware for API routes (returns JSON errors)
func (m *AuthMiddleware) RequireAPIAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Try API key
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			if user := m.validateAPIKey(r, apiKey); user != nil {
				ctx := domain.ContextWithUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Invalid API key"}`))
			return
		}

		// Try Bearer token
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if user := m.validateAPIKey(r, token); user != nil {
				ctx := domain.ContextWithUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try Basic auth
		if strings.HasPrefix(authHeader, "Basic ") {
			if user := m.validateBasicAuth(authHeader); user != nil {
				ctx := domain.ContextWithUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Authentication required"}`))
	})
}

// validateBasicAuth validates basic auth credentials
func (m *AuthMiddleware) validateBasicAuth(authHeader string) *domain.User {
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil
	}

	username, password := parts[0], parts[1]

	// Constant time comparison to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(m.username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(m.password)) == 1

	if usernameMatch && passwordMatch {
		return &domain.User{
			Username: username,
			IsAPIKey: false,
			Scopes:   []string{"admin"},
		}
	}

	return nil
}

// validateAPIKey validates an API key
func (m *AuthMiddleware) validateAPIKey(r *http.Request, key string) *domain.User {
	if m.apiKeyRepo == nil {
		return nil
	}

	apiKey, err := m.apiKeyRepo.GetByKey(r.Context(), key)
	if err != nil || apiKey == nil {
		return nil
	}

	if !apiKey.IsValid() {
		return nil
	}

	// Update last used timestamp (fire and forget)
	go m.apiKeyRepo.UpdateLastUsed(r.Context(), apiKey.ID)

	return &domain.User{
		Username: apiKey.Name,
		IsAPIKey: true,
		APIKeyID: apiKey.ID,
		Scopes:   apiKey.Scopes,
	}
}

// validateSession validates a session token (simple implementation)
func (m *AuthMiddleware) validateSession(sessionToken string) *domain.User {
	// For simplicity, session token is just base64(username:password)
	// In production, use proper session management
	decoded, err := base64.StdEncoding.DecodeString(sessionToken)
	if err != nil {
		return nil
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil
	}

	username, password := parts[0], parts[1]
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(m.username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(m.password)) == 1

	if usernameMatch && passwordMatch {
		return &domain.User{
			Username: username,
			IsAPIKey: false,
			Scopes:   []string{"admin"},
		}
	}

	return nil
}

// LoginHandler handles login form submission
func (m *AuthMiddleware) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Show login form
		m.renderLoginPage(w, "")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(m.username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(m.password)) == 1

	if usernameMatch && passwordMatch {
		// Set session cookie
		sessionToken := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400 * 7, // 7 days
		})

		// Redirect to dashboard
		redirect := r.URL.Query().Get("redirect")
		if redirect == "" {
			redirect = "/"
		}
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}

	m.renderLoginPage(w, "Invalid username or password")
}

// LogoutHandler handles logout
func (m *AuthMiddleware) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (m *AuthMiddleware) renderLoginPage(w http.ResponseWriter, errorMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	errorHTML := ""
	if errorMsg != "" {
		errorHTML = `<div class="error">` + errorMsg + `</div>`
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - Status Incident</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            background: white;
            padding: 2.5rem;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            width: 100%;
            max-width: 400px;
        }
        h1 {
            text-align: center;
            color: #1a1a2e;
            margin-bottom: 0.5rem;
            font-size: 1.5rem;
        }
        .subtitle {
            text-align: center;
            color: #6b7280;
            margin-bottom: 2rem;
            font-size: 0.9rem;
        }
        .form-group {
            margin-bottom: 1.25rem;
        }
        label {
            display: block;
            margin-bottom: 0.5rem;
            color: #374151;
            font-weight: 500;
            font-size: 0.9rem;
        }
        input[type="text"],
        input[type="password"] {
            width: 100%;
            padding: 0.75rem 1rem;
            border: 1px solid #d1d5db;
            border-radius: 8px;
            font-size: 1rem;
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        input:focus {
            outline: none;
            border-color: #3b82f6;
            box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
        }
        button {
            width: 100%;
            padding: 0.875rem;
            background: #3b82f6;
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 1rem;
            font-weight: 500;
            cursor: pointer;
            transition: background 0.2s;
        }
        button:hover {
            background: #2563eb;
        }
        .error {
            background: #fef2f2;
            color: #dc2626;
            padding: 0.75rem 1rem;
            border-radius: 8px;
            margin-bottom: 1.25rem;
            font-size: 0.9rem;
            border: 1px solid #fecaca;
        }
        .public-link {
            text-align: center;
            margin-top: 1.5rem;
            padding-top: 1.5rem;
            border-top: 1px solid #e5e7eb;
        }
        .public-link a {
            color: #6b7280;
            text-decoration: none;
            font-size: 0.9rem;
        }
        .public-link a:hover {
            color: #3b82f6;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <h1>Status Incident</h1>
        <p class="subtitle">Sign in to admin panel</p>
        ` + errorHTML + `
        <form method="POST">
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autofocus>
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit">Sign In</button>
        </form>
        <div class="public-link">
            <a href="/status">View public status page &rarr;</a>
        </div>
    </div>
</body>
</html>`

	w.Write([]byte(html))
}
