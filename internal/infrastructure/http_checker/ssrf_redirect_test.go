package http_checker

import (
	"net/http"
	"testing"
	"time"
)

// TestGuardedCheckerRejectsPrivateRedirect guards against the SSRF bypass where
// a heartbeat target that passes the initial public-host check returns a 3xx
// redirect to an internal/metadata host (e.g. 169.254.169.254) that was then
// followed without re-validation. The production checker (allowPrivate=false)
// must reject such a redirect hop; the allowPrivate checker must not.
func TestGuardedCheckerRejectsPrivateRedirect(t *testing.T) {
	guarded := NewWithOptions(5*time.Second, false)
	if guarded.client.Transport == nil {
		t.Fatal("production checker must install a guarded transport")
	}

	metadata, _ := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/", nil)
	if err := guarded.client.CheckRedirect(metadata, nil); err == nil {
		t.Error("guarded checker followed a redirect to a link-local/metadata host")
	}
	loopback, _ := http.NewRequest("GET", "http://127.0.0.1/admin", nil)
	if err := guarded.client.CheckRedirect(loopback, nil); err == nil {
		t.Error("guarded checker followed a redirect to loopback")
	}

	// allowPrivate mode (tests / explicit opt-in) keeps following private hops.
	open := NewWithOptions(5*time.Second, true)
	if err := open.client.CheckRedirect(loopback, nil); err != nil {
		t.Errorf("allowPrivate checker should not guard redirects, got %v", err)
	}
}
