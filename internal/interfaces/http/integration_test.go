package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"status-incident/internal/application"
	"status-incident/internal/domain"
	"status-incident/internal/infrastructure/http_checker"
	"status-incident/internal/infrastructure/sqlite"
)

// ============= Full Server Integration Harness =============

// minimalTemplates writes a set of templates that exercise the template funcs
// and reference the data fields used by web handlers.
func writeMinimalTemplates(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"layout.html": `{{define "layout.html"}}<!DOCTYPE html><html><body>{{template "content" .}}</body></html>{{end}}`,
		"dashboard.html": `{{define "content"}}<h1>Dashboard</h1>
<div class="{{overallStatusClass .Systems}}">{{overallStatusText .Systems}}</div>
{{range .Systems}}<div class="{{statusClass .Status}} {{statusTextClass .Status}}">{{statusText .Status}} {{statusIcon .Status}}</div>{{end}}{{end}}`,
		"system.html": `{{define "content"}}<h1>{{.System.Name}}</h1>
{{range .Dependencies}}<span class="{{statusClass .Status}}">{{.Name}}</span>{{end}}
{{range .Logs}}<li>{{.Message}}</li>{{end}}{{end}}`,
		"admin.html":     `{{define "content"}}<h1>Admin</h1>{{range .Systems}}<div>{{.Name}}{{range .Dependencies}}<span>{{headersJSON .Headers}}</span>{{end}}</div>{{end}}{{end}}`,
		"logs.html":      `{{define "content"}}<h1>Logs</h1>{{range .Logs}}<li>{{.SystemName}}/{{.DependencyName}}: {{.Message}}</li>{{end}}{{end}}`,
		"analytics.html": `{{define "content"}}<h1>Analytics</h1>{{if .Overall}}<div>{{formatPercent .Overall.UptimePercent}}</div>{{end}}{{range .Systems}}<div>{{.Name}}</div>{{end}}{{end}}`,
		"sla.html":       `{{define "content"}}<h1>SLA</h1>{{range .Reports}}<div>{{.Title}}</div>{{end}}{{range .Breaches}}<div>{{.SystemName}}</div>{{end}}{{range .Systems}}<div>{{.Name}}</div>{{end}}{{end}}`,
		"public.html": `{{define "public.html"}}<!DOCTYPE html><html><body><h1>{{.Title}}</h1>
<div class="{{overallStatusClass .Systems}}">{{overallStatusText .Systems}}</div>
{{range .Systems}}<div class="{{statusClass .Status}}">{{statusText .Status}}</div>{{end}}
{{range .ActiveMaintenance}}<div>{{.Title}}</div>{{end}}
{{range .UpcomingMaintenance}}<div>{{.Title}}</div>{{end}}
{{range .ActiveIncidents}}<div>{{.Title}}</div>{{end}}
<footer>{{.UpdatedAt}}</footer></body></html>{{end}}`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write template %s: %v", name, err)
		}
	}
	return dir
}

// fullServerFixture holds a fully-wired server backed by a real in-memory DB.
type fullServerFixture struct {
	server       *Server
	db           *sqlite.DB
	systemRepo   domain.SystemRepository
	depRepo      domain.DependencyRepository
	logRepo      domain.StatusLogRepository
	webhookRepo  domain.WebhookRepository
	apiKeyRepo   domain.APIKeyRepository
	latencyRepo  domain.LatencyRepository
	breachRepo   domain.SLABreachRepository
	incidentRepo domain.IncidentRepository
	authMW       *AuthMiddleware
	templateDir  string
}

// newFullServer builds a Server with real SQLite repositories wired through NewServer.
// authEnabled toggles the auth middleware.
func newFullServer(t *testing.T, authEnabled bool) *fullServerFixture {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	systemRepo := sqlite.NewSystemRepo(db)
	depRepo := sqlite.NewDependencyRepo(db)
	logRepo := sqlite.NewLogRepo(db)
	analyticsRepo := sqlite.NewAnalyticsRepo(db)
	webhookRepo := sqlite.NewWebhookRepo(db)
	maintenanceRepo := sqlite.NewMaintenanceRepo(db)
	incidentRepo := sqlite.NewIncidentRepo(db)
	latencyRepo := sqlite.NewLatencyRepo(db)
	reportRepo := sqlite.NewSLAReportRepo(db)
	breachRepo := sqlite.NewSLABreachRepo(db)
	apiKeyRepo := sqlite.NewAPIKeyRepo(db)

	systemService := application.NewSystemService(systemRepo, logRepo)
	depService := application.NewDependencyService(depRepo, logRepo)
	analyticsService := application.NewAnalyticsService(analyticsRepo, logRepo)
	maintenanceService := application.NewMaintenanceService(maintenanceRepo)
	incidentService := application.NewIncidentService(incidentRepo)
	latencyService := application.NewLatencyService(latencyRepo, depRepo)
	notificationService := application.NewNotificationService(webhookRepo, systemRepo, depRepo)
	checker := http_checker.New(2 * time.Second)
	heartbeatService := application.NewHeartbeatService(depRepo, logRepo, checker)
	slaService := application.NewSLAService(systemRepo, depRepo, analyticsRepo, reportRepo, breachRepo, latencyRepo, notificationService)

	webhookHandlers := NewWebhookHandlers(webhookRepo, notificationService)
	slaHandlers := NewSLAHandlers(slaService)
	apiKeyHandlers := NewAPIKeyHandlers(apiKeyRepo)

	var authMW *AuthMiddleware
	if authEnabled {
		authMW = NewAuthMiddleware(true, "admin", "secret", apiKeyRepo)
	} else {
		authMW = NewAuthMiddleware(false, "", "", apiKeyRepo)
	}

	tmplDir := writeMinimalTemplates(t)

	server := NewServer(
		systemService,
		depService,
		heartbeatService,
		analyticsService,
		maintenanceService,
		incidentService,
		latencyService,
		slaService,
		webhookHandlers,
		slaHandlers,
		apiKeyHandlers,
		authMW,
		tmplDir,
	)

	return &fullServerFixture{
		server:       server,
		db:           db,
		systemRepo:   systemRepo,
		depRepo:      depRepo,
		logRepo:      logRepo,
		webhookRepo:  webhookRepo,
		apiKeyRepo:   apiKeyRepo,
		latencyRepo:  latencyRepo,
		breachRepo:   breachRepo,
		incidentRepo: incidentRepo,
		authMW:       authMW,
		templateDir:  tmplDir,
	}
}

func unmarshalBody(w *httptest.ResponseRecorder, v interface{}) error {
	return json.Unmarshal(w.Body.Bytes(), v)
}

// withChi returns a context carrying the supplied chi route context.
func withChi(r *http.Request, rctx *chi.Context) context.Context {
	return context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
}

func (f *fullServerFixture) do(method, target string, body []byte) *httptest.ResponseRecorder {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	return w
}

// seedSystem creates a system with one dependency and a log entry directly via repos.
func (f *fullServerFixture) seedSystem(t *testing.T, name string) (*domain.System, *domain.Dependency) {
	t.Helper()
	ctx := context.Background()
	sys, err := domain.NewSystem(name, "desc", "https://example.com", "owner")
	if err != nil {
		t.Fatalf("new system: %v", err)
	}
	if err := f.systemRepo.Create(ctx, sys); err != nil {
		t.Fatalf("create system: %v", err)
	}

	dep := &domain.Dependency{SystemID: sys.ID, Name: "Database", Description: "primary", Status: domain.StatusGreen}
	if err := f.depRepo.Create(ctx, dep); err != nil {
		t.Fatalf("create dependency: %v", err)
	}

	sysID := sys.ID
	depID := dep.ID
	log := domain.NewStatusLog(&sysID, &depID, domain.StatusGreen, domain.StatusYellow, "degraded", domain.SourceManual)
	if err := f.logRepo.Create(ctx, log); err != nil {
		t.Fatalf("create log: %v", err)
	}
	return sys, dep
}
