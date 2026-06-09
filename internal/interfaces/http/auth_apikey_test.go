package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"status-incident/internal/domain"
)

// ============= Auth middleware (enabled) =============

func basicAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestAuth_APIRequiresCredentials(t *testing.T) {
	f := newFullServer(t, true)

	// No credentials -> 401 JSON
	w := f.do("GET", "/api/systems", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Authentication required") {
		t.Errorf("expected auth-required error, got %s", w.Body.String())
	}
}

func TestAuth_APIBasicAuthValid(t *testing.T) {
	f := newFullServer(t, true)

	r := httptest.NewRequest("GET", "/api/systems", nil)
	r.Header.Set("Authorization", basicAuthHeader("admin", "secret"))
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid basic auth, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_APIBasicAuthInvalid(t *testing.T) {
	f := newFullServer(t, true)

	r := httptest.NewRequest("GET", "/api/systems", nil)
	r.Header.Set("Authorization", basicAuthHeader("admin", "wrong"))
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with bad password, got %d", w.Code)
	}
}

func TestAuth_APIKeyValidAndInvalid(t *testing.T) {
	f := newFullServer(t, true)
	ctx := context.Background()

	// Create a valid API key directly via repo
	keyValue, err := domain.GenerateAPIKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	apiKey := &domain.APIKey{
		Name:    "ci",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"admin"},
		Enabled: true,
	}
	if err := f.apiKeyRepo.Create(ctx, apiKey); err != nil {
		t.Fatalf("create key: %v", err)
	}

	// Valid X-API-Key header
	r := httptest.NewRequest("GET", "/api/systems", nil)
	r.Header.Set("X-API-Key", keyValue)
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid api key, got %d: %s", w.Code, w.Body.String())
	}

	// Invalid X-API-Key header
	r = httptest.NewRequest("GET", "/api/systems", nil)
	r.Header.Set("X-API-Key", "definitely-not-valid")
	w = httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with bad api key, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Invalid API key") {
		t.Errorf("expected invalid api key message, got %s", w.Body.String())
	}

	// Valid Bearer token
	r = httptest.NewRequest("GET", "/api/systems", nil)
	r.Header.Set("Authorization", "Bearer "+keyValue)
	w = httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with bearer token, got %d", w.Code)
	}
}

func TestAuth_WebRequiresAuth(t *testing.T) {
	f := newFullServer(t, true)

	// No credentials on protected web route -> 401 + WWW-Authenticate
	w := f.do("GET", "/", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Errorf("expected WWW-Authenticate header")
	}
}

func TestAuth_WebBasicAuthValid(t *testing.T) {
	f := newFullServer(t, true)
	f.seedSystem(t, "AuthSys")

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", basicAuthHeader("admin", "secret"))
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid basic auth, got %d", w.Code)
	}
}

func TestAuth_WebInvalidAPIKey(t *testing.T) {
	f := newFullServer(t, true)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-API-Key", "bad-key")
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid web api key, got %d", w.Code)
	}
}

func TestAuth_WebSessionCookie(t *testing.T) {
	f := newFullServer(t, true)
	f.seedSystem(t, "CookieSys")

	// Create a session via the store, then send the cookie
	token, err := f.authMW.sessionStore.Create("admin", 24*60*60*1e9)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	r := httptest.NewRequest("GET", "/admin", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: token})
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid session, got %d", w.Code)
	}
}

func TestAuth_LoginAndLogout(t *testing.T) {
	f := newFullServer(t, true)

	// GET login renders the form
	w := f.do("GET", "/login", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("login GET: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Sign in to admin panel") {
		t.Errorf("expected login form HTML")
	}

	// POST login with valid creds -> redirect 303 + Set-Cookie
	form := strings.NewReader("username=admin&password=secret")
	r := httptest.NewRequest("POST", "/login", form)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wr := httptest.NewRecorder()
	f.server.ServeHTTP(wr, r)
	if wr.Code != http.StatusSeeOther {
		t.Fatalf("login POST valid: expected 303, got %d", wr.Code)
	}
	if len(wr.Result().Cookies()) == 0 {
		t.Errorf("expected session cookie on successful login")
	}

	// POST login with invalid creds -> re-render with error
	form2 := strings.NewReader("username=admin&password=nope")
	r2 := httptest.NewRequest("POST", "/login", form2)
	r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wr2 := httptest.NewRecorder()
	f.server.ServeHTTP(wr2, r2)
	if wr2.Code != http.StatusOK {
		t.Fatalf("login POST invalid: expected 200, got %d", wr2.Code)
	}
	if !strings.Contains(wr2.Body.String(), "Invalid username or password") {
		t.Errorf("expected invalid creds error on page")
	}

	// Logout -> redirect to /login
	wl := f.do("GET", "/logout", nil)
	if wl.Code != http.StatusSeeOther {
		t.Fatalf("logout: expected 303, got %d", wl.Code)
	}
}

func TestAuth_LoginRedirectSanitize(t *testing.T) {
	f := newFullServer(t, true)

	cases := []struct {
		redirect string
		want     string
	}{
		{"/admin", "/admin"},
		{"", "/"},
		{"http://evil.com", "/"},
		{"//evil.com", "/"},
		{"/path\\back", "/"},
	}
	for _, tc := range cases {
		form := strings.NewReader("username=admin&password=secret")
		r := httptest.NewRequest("POST", "/login?redirect="+tc.redirect, form)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		f.server.ServeHTTP(w, r)
		if loc := w.Header().Get("Location"); loc != tc.want {
			t.Errorf("redirect %q: expected Location %q, got %q", tc.redirect, tc.want, loc)
		}
	}
}

// ============= API Key handler routes =============

func TestAPIKeyHandlers_CRUD(t *testing.T) {
	f := newFullServer(t, true)
	auth := basicAuthHeader("admin", "secret")

	authedDo := func(method, target, body string) *httptest.ResponseRecorder {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, target, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
		} else {
			r = httptest.NewRequest(method, target, nil)
		}
		r.Header.Set("Authorization", auth)
		w := httptest.NewRecorder()
		f.server.ServeHTTP(w, r)
		return w
	}

	// Create
	w := authedDo("POST", "/api/apikeys", `{"name":"deploy","scopes":["read","write"],"expires_in_days":30}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create apikey: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created apiKeyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Key == "" {
		t.Errorf("expected key value on creation")
	}

	// Create with empty name -> 400
	w = authedDo("POST", "/api/apikeys", `{"name":""}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create empty name: expected 400, got %d", w.Code)
	}

	// Create with invalid body -> 400
	w = authedDo("POST", "/api/apikeys", `not json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create invalid body: expected 400, got %d", w.Code)
	}

	// Create defaulting scopes (no scopes provided)
	w = authedDo("POST", "/api/apikeys", `{"name":"noscopes"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create default scopes: expected 201, got %d", w.Code)
	}

	// List
	w = authedDo("GET", "/api/apikeys", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list apikeys: expected 200, got %d", w.Code)
	}
	var list []apiKeyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("expected >=2 keys, got %d", len(list))
	}

	// Toggle
	w = authedDo("PUT", "/api/apikeys/1/toggle", `{"enabled":false}`)
	if w.Code != http.StatusOK {
		t.Fatalf("toggle: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Toggle invalid ID
	w = authedDo("PUT", "/api/apikeys/abc/toggle", `{"enabled":false}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("toggle invalid id: expected 400, got %d", w.Code)
	}

	// Toggle invalid body
	w = authedDo("PUT", "/api/apikeys/1/toggle", `nope`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("toggle invalid body: expected 400, got %d", w.Code)
	}

	// Toggle not-found ID
	w = authedDo("PUT", "/api/apikeys/9999/toggle", `{"enabled":true}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("toggle not found: expected 404, got %d", w.Code)
	}

	// Delete
	w = authedDo("DELETE", "/api/apikeys/1", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}

	// Delete invalid ID
	w = authedDo("DELETE", "/api/apikeys/abc", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete invalid id: expected 400, got %d", w.Code)
	}
}
