package http

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status-incident/internal/domain"
)

func TestSessionStore_GetDeleteExpired(t *testing.T) {
	store := NewSessionStore()

	// Create + Get valid
	tok, err := store.Create("alice", time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if s := store.Get(tok); s == nil || s.Username != "alice" {
		t.Fatalf("expected valid session for alice")
	}

	// Get unknown token -> nil
	if s := store.Get("nope"); s != nil {
		t.Errorf("expected nil for unknown token")
	}

	// Expired session is removed on Get
	expTok, _ := store.Create("bob", -time.Hour)
	if s := store.Get(expTok); s != nil {
		t.Errorf("expected nil for expired session")
	}

	// Delete
	store.Delete(tok)
	if s := store.Get(tok); s != nil {
		t.Errorf("expected nil after delete")
	}
}

func TestValidateBasicAuth(t *testing.T) {
	m := NewAuthMiddleware(true, "admin", "secret", nil)

	// valid
	if u := m.validateBasicAuth(basicAuthHeader("admin", "secret")); u == nil || u.Username != "admin" {
		t.Errorf("expected valid user for correct creds")
	}
	// wrong password
	if u := m.validateBasicAuth(basicAuthHeader("admin", "wrong")); u != nil {
		t.Errorf("expected nil for wrong password")
	}
	// not valid base64
	if u := m.validateBasicAuth("Basic !!!notbase64!!!"); u != nil {
		t.Errorf("expected nil for invalid base64")
	}
	// no colon separator
	if u := m.validateBasicAuth("Basic " + b64("nocolon")); u != nil {
		t.Errorf("expected nil for malformed credentials")
	}
}

func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func TestValidateAPIKey_NilRepoAndDisabled(t *testing.T) {
	// nil repo -> nil user
	m := NewAuthMiddleware(true, "admin", "secret", nil)
	r := httptest.NewRequest("GET", "/", nil)
	if u := m.validateAPIKey(r, "anykey"); u != nil {
		t.Errorf("expected nil with nil repo")
	}

	// disabled key -> nil user
	repo := &stubAPIKeyRepo{}
	keyVal, _ := domain.GenerateAPIKey()
	repo.key = &domain.APIKey{ID: 1, Name: "k", Key: keyVal, KeyHash: domain.HashAPIKey(keyVal), Enabled: false, Scopes: []string{"read"}}
	m2 := NewAuthMiddleware(true, "admin", "secret", repo)
	if u := m2.validateAPIKey(r, keyVal); u != nil {
		t.Errorf("expected nil for disabled key")
	}

	// repo error -> nil
	repoErr := &stubAPIKeyRepo{err: true}
	m3 := NewAuthMiddleware(true, "admin", "secret", repoErr)
	if u := m3.validateAPIKey(r, keyVal); u != nil {
		t.Errorf("expected nil on repo error")
	}

	// valid enabled key -> user, with last-used fired
	repoOK := &stubAPIKeyRepo{}
	repoOK.key = &domain.APIKey{ID: 2, Name: "ok", Key: keyVal, KeyHash: domain.HashAPIKey(keyVal), Enabled: true, Scopes: []string{"admin"}}
	m4 := NewAuthMiddleware(true, "admin", "secret", repoOK)
	if u := m4.validateAPIKey(r, keyVal); u == nil || !u.IsAPIKey {
		t.Errorf("expected api key user")
	}
}

func TestValidateSession_Nil(t *testing.T) {
	m := NewAuthMiddleware(true, "admin", "secret", nil)
	if u := m.validateSession("no-such-token"); u != nil {
		t.Errorf("expected nil for unknown session token")
	}
}

func TestLoginHandler_MethodNotAllowed(t *testing.T) {
	m := NewAuthMiddleware(true, "admin", "secret", nil)
	r := httptest.NewRequest("PUT", "/login", nil)
	w := httptest.NewRecorder()
	m.LoginHandler(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestLogoutHandler_NoCookie(t *testing.T) {
	m := NewAuthMiddleware(true, "admin", "secret", nil)
	r := httptest.NewRequest("GET", "/logout", nil)
	w := httptest.NewRecorder()
	m.LogoutHandler(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
}

func TestRequireAuth_Disabled(t *testing.T) {
	m := NewAuthMiddleware(false, "", "", nil)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	h := m.RequireAuth(next)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !called {
		t.Errorf("expected next called when auth disabled")
	}
}

func TestRequireAPIAuth_Disabled(t *testing.T) {
	m := NewAuthMiddleware(false, "", "", nil)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	h := m.RequireAPIAuth(next)
	r := httptest.NewRequest("GET", "/api/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !called {
		t.Errorf("expected next called when api auth disabled")
	}
}

func TestRequireAuth_BearerAndBasicAndSession(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	keyVal, _ := domain.GenerateAPIKey()
	repo.key = &domain.APIKey{ID: 1, Name: "k", Key: keyVal, KeyHash: domain.HashAPIKey(keyVal), Enabled: true, Scopes: []string{"admin"}}
	m := NewAuthMiddleware(true, "admin", "secret", repo)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := m.RequireAuth(next)

	// Bearer token (valid)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+keyVal)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("bearer: expected 200, got %d", w.Code)
	}

	// Bearer token (invalid) -> falls through to 401
	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer bad-token")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad bearer: expected 401, got %d", w.Code)
	}

	// Basic auth (valid)
	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", basicAuthHeader("admin", "secret"))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("basic: expected 200, got %d", w.Code)
	}

	// Session cookie (valid)
	tok, _ := m.sessionStore.Create("admin", time.Hour)
	r = httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: tok})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("session: expected 200, got %d", w.Code)
	}
}

func TestRequireAPIAuth_BearerBasic(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	keyVal, _ := domain.GenerateAPIKey()
	repo.key = &domain.APIKey{ID: 1, Name: "k", Key: keyVal, KeyHash: domain.HashAPIKey(keyVal), Enabled: true, Scopes: []string{"admin"}}
	m := NewAuthMiddleware(true, "admin", "secret", repo)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := m.RequireAPIAuth(next)

	// Bearer valid
	r := httptest.NewRequest("GET", "/api/x", nil)
	r.Header.Set("Authorization", "Bearer "+keyVal)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("api bearer: expected 200, got %d", w.Code)
	}

	// Basic valid
	r = httptest.NewRequest("GET", "/api/x", nil)
	r.Header.Set("Authorization", basicAuthHeader("admin", "secret"))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("api basic: expected 200, got %d", w.Code)
	}
}

// stubAPIKeyRepo is a minimal APIKeyRepository for auth unit tests.
type stubAPIKeyRepo struct {
	key *domain.APIKey
	err bool
}

func (s *stubAPIKeyRepo) Create(ctx context.Context, k *domain.APIKey) error { return nil }
func (s *stubAPIKeyRepo) GetByKey(ctx context.Context, key string) (*domain.APIKey, error) {
	if s.err {
		return nil, context.DeadlineExceeded
	}
	if s.key != nil && s.key.Key == key {
		return s.key, nil
	}
	return nil, nil
}
func (s *stubAPIKeyRepo) GetAll(ctx context.Context) ([]*domain.APIKey, error) { return nil, nil }
func (s *stubAPIKeyRepo) Update(ctx context.Context, k *domain.APIKey) error   { return nil }
func (s *stubAPIKeyRepo) Delete(ctx context.Context, id int64) error           { return nil }
func (s *stubAPIKeyRepo) UpdateLastUsed(ctx context.Context, id int64) error   { return nil }
