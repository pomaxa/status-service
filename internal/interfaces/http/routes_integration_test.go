package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// ============= Router / Web page routes =============

func TestServeHTTP_WebPages(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "Gateway")

	cases := []struct {
		name string
		path string
	}{
		{"dashboard", "/"},
		{"system_detail", "/systems/1"},
		{"admin", "/admin"},
		{"logs", "/logs"},
		{"analytics", "/analytics"},
		{"analytics_period", "/analytics?period=7d"},
		{"sla", "/sla"},
		{"public_status", "/status"},
		{"metrics", "/metrics"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := f.do("GET", tc.path, nil)
			if w.Code != http.StatusOK {
				t.Fatalf("GET %s: expected 200, got %d: %s", tc.path, w.Code, w.Body.String())
			}
			// securityHeaders middleware should be present on all responses
			if w.Header().Get("X-Frame-Options") != "DENY" {
				t.Errorf("expected security headers, got none")
			}
		})
	}
}

func TestServeHTTP_SystemDetail_NotFound(t *testing.T) {
	f := newFullServer(t, false)
	w := f.do("GET", "/systems/999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServeHTTP_SystemDetail_InvalidID(t *testing.T) {
	f := newFullServer(t, false)
	w := f.do("GET", "/systems/notanumber", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestMetricsContent(t *testing.T) {
	f := newFullServer(t, false)
	sys, dep := f.seedSystem(t, "MetricsSys")
	ctx := context.Background()

	// give the dependency latency + failures so those metric lines render
	dep.LastLatency = 42
	dep.ConsecutiveFailures = 3
	dep.Status = domain.StatusRed
	if err := f.depRepo.Update(ctx, dep); err != nil {
		t.Fatalf("update dep: %v", err)
	}
	sys.Status = domain.StatusYellow
	if err := f.systemRepo.Update(ctx, sys); err != nil {
		t.Fatalf("update sys: %v", err)
	}

	w := f.do("GET", "/metrics", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"status_incident_system_status",
		"status_incident_dependency_status",
		"status_incident_dependency_latency_ms",
		"status_incident_systems_total",
		"status_incident_incidents_active",
		"status_incident_maintenances_active",
		"status_incident_sla_breaches_unacknowledged",
		"MetricsSys",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics body missing %q", want)
		}
	}
}

func TestPublicStatus_WithMaintenanceAndIncidents(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "PublicSys")
	ctx := context.Background()

	// active maintenance (started in past, ends in future)
	w := f.do("POST", "/api/maintenances", mustJSON(maintenanceRequest{
		Title:       "Active Window",
		Description: "ongoing",
		StartTime:   time.Now().Add(-time.Hour).Format(time.RFC3339),
		EndTime:     time.Now().Add(time.Hour).Format(time.RFC3339),
		SystemIDs:   []int64{1},
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create maintenance: %d %s", w.Code, w.Body.String())
	}

	// upcoming maintenance
	w = f.do("POST", "/api/maintenances", mustJSON(maintenanceRequest{
		Title:     "Upcoming Window",
		StartTime: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Add(25 * time.Hour).Format(time.RFC3339),
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create upcoming: %d", w.Code)
	}

	// active incident
	w = f.do("POST", "/api/incidents", mustJSON(incidentRequest{
		Title:    "Outage",
		Message:  "investigating",
		Severity: "major",
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create incident: %d %s", w.Code, w.Body.String())
	}
	_ = ctx

	w = f.do("GET", "/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("public status: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Active Window") {
		t.Errorf("expected active maintenance in public page")
	}
	if !strings.Contains(body, "Outage") {
		t.Errorf("expected active incident in public page")
	}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
