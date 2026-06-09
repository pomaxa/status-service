package http

import (
	"net/http"
	"testing"
)

// TestErrorPaths_DBClosed closes the underlying DB so every repository query
// fails, exercising the 500 / error branches across handlers.
func TestErrorPaths_DBClosed(t *testing.T) {
	f := newFullServer(t, false)
	// seed before closing so IDs exist in routing but queries still fail
	f.seedSystem(t, "ErrSys")
	if err := f.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	// API endpoints that wrap repo errors as 500
	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{"GET", "/api/systems", nil},
		{"GET", "/api/systems/1", nil},
		{"POST", "/api/systems", []byte(`{"name":"x"}`)},
		{"PUT", "/api/systems/1", []byte(`{"name":"x"}`)},
		{"DELETE", "/api/systems/1", nil},
		{"POST", "/api/systems/1/status", []byte(`{"status":"green"}`)},
		{"GET", "/api/systems/1/logs", nil},
		{"GET", "/api/systems/1/analytics", nil},
		{"GET", "/api/systems/1/dependencies", nil},
		{"POST", "/api/systems/1/dependencies", []byte(`{"name":"d"}`)},
		{"GET", "/api/dependencies/1", nil},
		{"PUT", "/api/dependencies/1", []byte(`{"name":"d"}`)},
		{"DELETE", "/api/dependencies/1", nil},
		{"POST", "/api/dependencies/1/status", []byte(`{"status":"green"}`)},
		{"GET", "/api/dependencies/1/logs", nil},
		{"GET", "/api/dependencies/1/analytics", nil},
		{"GET", "/api/logs", nil},
		{"GET", "/api/analytics", nil},
		{"GET", "/api/export", nil},
		{"GET", "/api/export/logs", nil},
		{"GET", "/api/maintenances", nil},
		{"GET", "/api/maintenances/active", nil},
		{"GET", "/api/maintenances/upcoming", nil},
		{"GET", "/api/maintenances/1", nil},
		{"DELETE", "/api/maintenances/1", nil},
		{"GET", "/api/incidents", nil},
		{"GET", "/api/incidents/active", nil},
		{"GET", "/api/incidents/recent", nil},
		{"GET", "/api/incidents/1", nil},
		{"DELETE", "/api/incidents/1", nil},
		{"GET", "/api/incidents/1/updates", nil},
		{"GET", "/api/webhooks", nil},
		{"GET", "/api/webhooks/1", nil},
		{"GET", "/api/sla/reports", nil},
		{"GET", "/api/sla/reports/1", nil},
		{"DELETE", "/api/sla/reports/1", nil},
		{"GET", "/api/sla/breaches", nil},
		{"GET", "/api/sla/breaches?unacknowledged=true", nil},
		{"POST", "/api/sla/breaches/check", nil},
		{"GET", "/api/systems/1/sla/breaches", nil},
		{"GET", "/api/apikeys", nil},
	}

	for _, tc := range cases {
		w := f.do(tc.method, tc.path, tc.body)
		if w.Code < 400 {
			t.Errorf("%s %s: expected an error status (>=400) with closed DB, got %d", tc.method, tc.path, w.Code)
		}
	}
}

// TestErrorPaths_WebPagesDBClosed exercises the 500 branches in web handlers.
func TestErrorPaths_WebPagesDBClosed(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "WebErrSys")
	if err := f.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	for _, path := range []string{"/", "/admin", "/logs"} {
		w := f.do("GET", path, nil)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("GET %s with closed DB: expected 500, got %d", path, w.Code)
		}
	}
}

// TestSLAPage_Unavailable covers the handleSLAPage branch where slaService is nil.
func TestSLAPage_Unavailable(t *testing.T) {
	f := newFullServer(t, false)
	f.server.slaService = nil
	w := f.do("GET", "/sla", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when sla service nil, got %d", w.Code)
	}
}
