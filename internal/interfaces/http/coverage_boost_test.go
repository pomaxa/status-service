package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// TestWebHandlers_TemplateLoadErrors points the server at a template directory
// that is missing the page templates, exercising the loadTemplate error (500)
// branches in every web handler.
func TestWebHandlers_TemplateLoadErrors(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "TmplErrSys")

	// Repoint the server at an empty template dir so ParseFiles fails.
	emptyDir := t.TempDir()
	f.server.templateDir = emptyDir

	for _, path := range []string{"/", "/systems/1", "/admin", "/logs", "/analytics", "/sla", "/status"} {
		w := f.do("GET", path, nil)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("GET %s with missing templates: expected 500, got %d", path, w.Code)
		}
	}

	// Also exercise loadStandaloneTemplate error explicitly (used by /status).
	if _, err := f.server.loadStandaloneTemplate("does-not-exist"); err == nil {
		t.Errorf("expected error loading missing standalone template")
	}
	// Repoint to a directory with a layout but missing page -> still errors.
	if err := os.WriteFile(filepath.Join(emptyDir, "layout.html"), []byte(`{{define "layout.html"}}x{{end}}`), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	if _, err := f.server.loadTemplate("missing-page"); err == nil {
		t.Errorf("expected error loading template with missing page")
	}
}

// TestHandleLogs_CacheHit seeds two logs that reference the same system and
// dependency so the name caches are hit on the second iteration.
func TestHandleLogs_CacheHit(t *testing.T) {
	f := newFullServer(t, false)
	sys, dep := f.seedSystem(t, "CacheSys") // creates one log already
	ctx := context.Background()

	sysID := sys.ID
	depID := dep.ID
	// add a second log referencing the same system + dependency
	log2 := domain.NewStatusLog(&sysID, &depID, domain.StatusYellow, domain.StatusGreen, "recovered", domain.SourceManual)
	if err := f.logRepo.Create(ctx, log2); err != nil {
		t.Fatalf("create log2: %v", err)
	}

	w := f.do("GET", "/logs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("logs: expected 200, got %d", w.Code)
	}
}

// TestMetrics_WithIncidents ensures the severity/status counting loop in
// handleMetrics runs against real incident data.
func TestMetrics_WithIncidents(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "MetricsIncSys")

	w := f.do("POST", "/api/incidents", mustJSON(incidentRequest{Title: "Down", Message: "x", Severity: "critical"}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create incident: %d %s", w.Code, w.Body.String())
	}

	w = f.do("GET", "/metrics", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("metrics: expected 200, got %d", w.Code)
	}
}

// TestRecentIncidents_WithData covers the toIncidentResponse loop in
// apiGetRecentIncidents.
func TestRecentIncidents_WithData(t *testing.T) {
	f := newFullServer(t, false)
	ctx := context.Background()

	inc, err := domain.NewIncident("Recent", "msg", domain.SeverityMinor)
	if err != nil {
		t.Fatalf("new incident: %v", err)
	}
	if err := f.incidentRepo.Create(ctx, inc); err != nil {
		t.Fatalf("create incident: %v", err)
	}
	// resolve it so it counts as "recent"
	resolvedAt := time.Now()
	inc.Status = domain.IncidentResolved
	inc.ResolvedAt = &resolvedAt
	if err := f.incidentRepo.Update(ctx, inc); err != nil {
		t.Fatalf("update incident: %v", err)
	}

	w := f.do("GET", "/api/incidents/recent?days=30", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("recent: expected 200, got %d", w.Code)
	}
}

// TestAuth_WebAPIKeyValid covers the RequireAuth X-API-Key success branch on a
// protected web route.
func TestAuth_WebAPIKeyValid(t *testing.T) {
	f := newFullServer(t, true)
	f.seedSystem(t, "WebKeySys")
	ctx := context.Background()

	keyVal, _ := domain.GenerateAPIKey()
	if err := f.apiKeyRepo.Create(ctx, &domain.APIKey{
		Name: "web", Key: keyVal, KeyHash: domain.HashAPIKey(keyVal), Enabled: true, Scopes: []string{"admin"},
	}); err != nil {
		t.Fatalf("create key: %v", err)
	}

	r := httptest.NewRequest("GET", "/admin", nil)
	r.Header.Set("X-API-Key", keyVal)
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("web with api key: expected 200, got %d", w.Code)
	}
}

// TestLogout_WithCookie covers the LogoutHandler branch that deletes an existing
// session.
func TestLogout_WithCookie(t *testing.T) {
	f := newFullServer(t, true)
	tok, _ := f.authMW.sessionStore.Create("admin", time.Hour)

	r := httptest.NewRequest("GET", "/logout", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: tok})
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("logout: expected 303, got %d", w.Code)
	}
	// the session should now be gone
	if f.authMW.sessionStore.Get(tok) != nil {
		t.Errorf("expected session deleted after logout")
	}
}

// TestSLA_AcknowledgeExistingBreach seeds a breach and acknowledges it with an
// empty body (default actor branch) and then the success path.
func TestSLA_AcknowledgeExistingBreach(t *testing.T) {
	f := newFullServer(t, false)
	sys, _ := f.seedSystem(t, "BreachAckSys")
	ctx := context.Background()

	breach := &domain.SLABreachEvent{
		SystemID:    sys.ID,
		SystemName:  sys.Name,
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 95.0,
		Period:      "monthly",
		PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}
	if err := f.breachRepo.Create(ctx, breach); err != nil {
		t.Fatalf("create breach: %v", err)
	}

	// empty body -> default actor "system", then success
	w := f.do("POST", "/api/sla/breaches/1/acknowledge", []byte(``))
	if w.Code != http.StatusOK {
		t.Fatalf("ack existing breach: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSLA_SystemSLAErrorsDBClosed covers the non-"not found" 500 branches in
// GetSystemSLA and UpdateSystemSLATarget, plus latency/uptime heatmap errors.
func TestSLA_SystemSLAErrorsDBClosed(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "SLAErrSys")
	if err := f.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{"GET", "/api/systems/1/sla", nil},
		{"PUT", "/api/systems/1/sla-target", []byte(`{"sla_target":99.9}`)},
		{"GET", "/api/dependencies/1/latency", nil},
		{"GET", "/api/dependencies/1/uptime", nil},
	}
	for _, tc := range cases {
		w := f.do(tc.method, tc.path, tc.body)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("%s %s with closed DB: expected 500, got %d", tc.method, tc.path, w.Code)
		}
	}
}
