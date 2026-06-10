package http

import (
	"net/http"
	"strings"
	"testing"
)

// TestInternalError_BodyIsGenericNoLeak guards #30: on a 5xx, the response body
// must be a generic message and must NOT echo internal/driver detail (e.g.
// "sql: database is closed") to the client.
func TestInternalError_BodyIsGenericNoLeak(t *testing.T) {
	f := newFullServer(t, false)
	if err := f.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	w := f.do("GET", "/api/systems", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after DB close, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "internal server error") {
		t.Errorf("500 body should be generic, got %q", body)
	}
	for _, leak := range []string{"sql", "database", "sqlite", "failed to query"} {
		if strings.Contains(strings.ToLower(body), leak) {
			t.Errorf("500 body leaks internal detail %q: %s", leak, body)
		}
	}
}

// TestMutation_NotFoundVsValidation guards #22: a mutation on a missing resource
// returns 404, while a genuine validation failure still returns 400 with its
// (user-actionable) message.
func TestMutation_NotFoundVsValidation(t *testing.T) {
	f := newFullServer(t, false)

	// Missing resource -> 404.
	w := f.do("PUT", "/api/systems/9999", []byte(`{"name":"x"}`))
	if w.Code != http.StatusNotFound {
		t.Errorf("update missing system: expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// Create a system, then drive an invalid status update -> 400 (validation).
	cw := f.do("POST", "/api/systems", []byte(`{"name":"S"}`))
	if cw.Code != http.StatusCreated && cw.Code != http.StatusOK {
		t.Fatalf("seed system: unexpected %d: %s", cw.Code, cw.Body.String())
	}
	w = f.do("POST", "/api/systems/1/status", []byte(`{"status":"purple"}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid status: expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(strings.ToLower(w.Body.String()), "status") {
		t.Errorf("validation 400 should surface an actionable message, got %q", w.Body.String())
	}
}
