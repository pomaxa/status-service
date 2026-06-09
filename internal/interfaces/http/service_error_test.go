package http

import (
	"net/http"
	"testing"
)

// TestServiceErrorBranches drives mutating handlers against valid-but-nonexistent
// IDs (and empty bodies) so the wrapped service-error branches (400) and the
// "default actor" branches are executed.
func TestServiceErrorBranches(t *testing.T) {
	f := newFullServer(t, false)

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
		want   int
	}{
		// Incident mutations on a non-existent incident (empty body -> default "by").
		{"ack_incident_empty_body", "POST", "/api/incidents/9999/acknowledge", []byte(``), http.StatusBadRequest},
		{"resolve_incident_empty_body", "POST", "/api/incidents/9999/resolve", []byte(``), http.StatusBadRequest},
		{"update_incident_status_default_by", "POST", "/api/incidents/9999/status", []byte(`{"status":"identified","message":"x"}`), http.StatusBadRequest},
		{"add_incident_update_default_by", "POST", "/api/incidents/9999/updates", []byte(`{"message":"x"}`), http.StatusBadRequest},
		// Maintenance cancel on non-existent.
		{"cancel_maintenance", "POST", "/api/maintenances/9999/cancel", nil, http.StatusBadRequest},
		// Dependency status / heartbeat on non-existent dependency.
		{"update_dep_status", "POST", "/api/dependencies/9999/status", []byte(`{"status":"green"}`), http.StatusBadRequest},
		{"set_heartbeat", "POST", "/api/dependencies/9999/heartbeat", []byte(`{"url":"https://x","interval":60}`), http.StatusBadRequest},
		{"clear_heartbeat", "DELETE", "/api/dependencies/9999/heartbeat", nil, http.StatusBadRequest},
		{"force_check", "POST", "/api/dependencies/9999/check", nil, http.StatusBadRequest},
		// System status / update on non-existent.
		{"update_system_status", "POST", "/api/systems/9999/status", []byte(`{"status":"green"}`), http.StatusBadRequest},
		{"update_system", "PUT", "/api/systems/9999", []byte(`{"name":"x"}`), http.StatusBadRequest},
		// SLA generate report should succeed; the error branch is covered via closed-DB test.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := f.do(tc.method, tc.path, tc.body)
			if w.Code != tc.want {
				t.Errorf("%s %s: expected %d, got %d: %s", tc.method, tc.path, tc.want, w.Code, w.Body.String())
			}
		})
	}
}
