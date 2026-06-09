package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"status-incident/internal/domain"
)

// ============= Maintenance routes =============

func TestMaintenanceRoutes(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "MaintSys")

	start := time.Now().Add(time.Hour).Format(time.RFC3339)
	end := time.Now().Add(2 * time.Hour).Format(time.RFC3339)

	// Create
	w := f.do("POST", "/api/maintenances", mustJSON(maintenanceRequest{
		Title: "Upgrade", Description: "db upgrade", StartTime: start, EndTime: end, SystemIDs: []int64{1},
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create maintenance: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Create invalid body
	w = f.do("POST", "/api/maintenances", []byte(`bad`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create bad body: expected 400, got %d", w.Code)
	}

	// Create invalid start time
	w = f.do("POST", "/api/maintenances", mustJSON(maintenanceRequest{
		Title: "x", StartTime: "not-a-time", EndTime: end,
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create bad start: expected 400, got %d", w.Code)
	}

	// Create invalid end time
	w = f.do("POST", "/api/maintenances", mustJSON(maintenanceRequest{
		Title: "x", StartTime: start, EndTime: "not-a-time",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create bad end: expected 400, got %d", w.Code)
	}

	// List
	w = f.do("GET", "/api/maintenances", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list maintenances: expected 200, got %d", w.Code)
	}

	// Active
	w = f.do("GET", "/api/maintenances/active", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("active: expected 200, got %d", w.Code)
	}

	// Upcoming
	w = f.do("GET", "/api/maintenances/upcoming", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("upcoming: expected 200, got %d", w.Code)
	}

	// Get single
	w = f.do("GET", "/api/maintenances/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get maintenance: expected 200, got %d", w.Code)
	}

	// Get invalid ID
	w = f.do("GET", "/api/maintenances/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("get invalid: expected 400, got %d", w.Code)
	}

	// Get not found
	w = f.do("GET", "/api/maintenances/9999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get not found: expected 404, got %d", w.Code)
	}

	// Update
	w = f.do("PUT", "/api/maintenances/1", mustJSON(maintenanceRequest{
		Title: "Upgrade v2", StartTime: start, EndTime: end, SystemIDs: []int64{1},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("update maintenance: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Update invalid ID
	w = f.do("PUT", "/api/maintenances/abc", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update invalid id: expected 400, got %d", w.Code)
	}

	// Update invalid body
	w = f.do("PUT", "/api/maintenances/1", []byte(`bad`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update bad body: expected 400, got %d", w.Code)
	}

	// Update invalid start
	w = f.do("PUT", "/api/maintenances/1", mustJSON(maintenanceRequest{Title: "x", StartTime: "bad", EndTime: end}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update bad start: expected 400, got %d", w.Code)
	}

	// Update invalid end
	w = f.do("PUT", "/api/maintenances/1", mustJSON(maintenanceRequest{Title: "x", StartTime: start, EndTime: "bad"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update bad end: expected 400, got %d", w.Code)
	}

	// Cancel
	w = f.do("POST", "/api/maintenances/1/cancel", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("cancel maintenance: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Cancel invalid ID
	w = f.do("POST", "/api/maintenances/abc/cancel", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("cancel invalid: expected 400, got %d", w.Code)
	}

	// Delete
	w = f.do("DELETE", "/api/maintenances/1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete maintenance: expected 204, got %d", w.Code)
	}

	// Delete invalid ID
	w = f.do("DELETE", "/api/maintenances/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete invalid: expected 400, got %d", w.Code)
	}
}

// ============= Dependency heartbeat / force-check / latency / uptime =============

func TestDependencyHeartbeatAndLatency(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "HBSys") // creates dependency id 1

	// Record a latency measurement so the stats query has rows to aggregate.
	if err := f.latencyRepo.Record(context.Background(), &domain.LatencyRecord{
		DependencyID: 1, LatencyMs: 120, Success: true, StatusCode: 200, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("record latency: %v", err)
	}

	// Set heartbeat
	w := f.do("POST", "/api/dependencies/1/heartbeat", mustJSON(setHeartbeatRequest{
		URL: "https://hb.example.com", Interval: 60, Method: "GET", ExpectStatus: "200",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("set heartbeat: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Set heartbeat invalid ID
	w = f.do("POST", "/api/dependencies/abc/heartbeat", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("set heartbeat invalid id: expected 400, got %d", w.Code)
	}

	// Set heartbeat invalid body
	w = f.do("POST", "/api/dependencies/1/heartbeat", []byte(`bad`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("set heartbeat bad body: expected 400, got %d", w.Code)
	}

	// Force check (heartbeat is configured; checker will fail to reach the host but the handler returns the dep)
	w = f.do("POST", "/api/dependencies/1/check", nil)
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Fatalf("force check: unexpected %d: %s", w.Code, w.Body.String())
	}

	// Force check invalid ID
	w = f.do("POST", "/api/dependencies/abc/check", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("force check invalid: expected 400, got %d", w.Code)
	}

	// Latency
	w = f.do("GET", "/api/dependencies/1/latency?period=24h", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("latency: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Latency invalid ID
	w = f.do("GET", "/api/dependencies/abc/latency", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("latency invalid: expected 400, got %d", w.Code)
	}

	// Uptime
	w = f.do("GET", "/api/dependencies/1/uptime?days=30", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("uptime: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Uptime invalid ID
	w = f.do("GET", "/api/dependencies/abc/uptime", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("uptime invalid: expected 400, got %d", w.Code)
	}

	// Clear heartbeat
	w = f.do("DELETE", "/api/dependencies/1/heartbeat", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("clear heartbeat: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Clear heartbeat invalid ID
	w = f.do("DELETE", "/api/dependencies/abc/heartbeat", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("clear heartbeat invalid: expected 400, got %d", w.Code)
	}
}

func TestLatencyService_Unavailable(t *testing.T) {
	// When latencyService is nil, the handlers return 503.
	server, _, _ := setupTestServer()
	server.latencyService = nil

	r := httptest.NewRequest("GET", "/api/dependencies/1/latency", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	r = r.WithContext(withChi(r, rctx))
	w := httptest.NewRecorder()
	server.apiGetDependencyLatency(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("latency nil service: expected 503, got %d", w.Code)
	}

	r = httptest.NewRequest("GET", "/api/dependencies/1/uptime", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	r = r.WithContext(withChi(r, rctx))
	w = httptest.NewRecorder()
	server.apiGetDependencyUptime(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("uptime nil service: expected 503, got %d", w.Code)
	}
}

// ============= Webhook routes via full server =============

func TestWebhookRoutes_FullServer(t *testing.T) {
	f := newFullServer(t, false)

	// Create webhook
	w := f.do("POST", "/api/webhooks", mustJSON(webhookRequest{
		Name: "Slack", URL: "https://hooks.slack.com/x", Type: "slack", Events: []string{"status_change"},
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create webhook: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Create with empty name -> 400 (NewWebhook validation)
	w = f.do("POST", "/api/webhooks", mustJSON(webhookRequest{Name: "", URL: "https://x", Type: "slack"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create empty name: expected 400, got %d", w.Code)
	}

	// Update with empty name -> 400 (Update validation)
	w = f.do("PUT", "/api/webhooks/1", mustJSON(webhookRequest{Name: "", URL: "https://x", Type: "slack"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update empty name: expected 400, got %d", w.Code)
	}

	// Update invalid body
	w = f.do("PUT", "/api/webhooks/1", []byte(`bad`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update bad body: expected 400, got %d", w.Code)
	}

	// Update not-found webhook
	w = f.do("PUT", "/api/webhooks/9999", mustJSON(webhookRequest{Name: "x", URL: "https://x", Type: "slack"}))
	if w.Code != http.StatusNotFound {
		t.Fatalf("update not found: expected 404, got %d", w.Code)
	}

	// Update existing with enabled toggle
	enabled := false
	w = f.do("PUT", "/api/webhooks/1", mustJSON(webhookRequest{
		Name: "Slack2", URL: "https://hooks.slack.com/y", Type: "discord",
		Events: []string{"status_change", "incident"}, SystemIDs: []int64{1}, Enabled: &enabled,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("update webhook: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete not-found
	w = f.do("DELETE", "/api/webhooks/9999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("delete not found: expected 404, got %d", w.Code)
	}

	// Delete invalid ID
	w = f.do("DELETE", "/api/webhooks/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete invalid: expected 400, got %d", w.Code)
	}

	// TestWebhook (notification service will try to send; webhook 1 exists)
	w = f.do("POST", "/api/webhooks/1/test", nil)
	// Sending to a non-resolvable/blocked host fails -> 404 per handler. Either is acceptable as long as it ran.
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Fatalf("test webhook: unexpected %d: %s", w.Code, w.Body.String())
	}

	// TestWebhook invalid ID
	w = f.do("POST", "/api/webhooks/abc/test", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("test webhook invalid: expected 400, got %d", w.Code)
	}
}
