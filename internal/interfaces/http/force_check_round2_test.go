package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"status-incident/internal/application"
	"status-incident/internal/domain"
	"status-incident/internal/infrastructure/http_checker"
	"status-incident/internal/infrastructure/sqlite"
)

// TestAPIForceCheck_Success covers the success path of apiForceCheck
// (api_handlers.go: s.respondJSON(w, http.StatusOK, dep)), which is only
// reachable when ForceCheck returns no error. The fully-wired test server
// uses an SSRF-protected checker that blocks every reachable test host, so
// this test constructs a minimal Server backed by a checker that permits
// private addresses and points the dependency's heartbeat at a live
// httptest backend returning 200.
func TestAPIForceCheck_Success(t *testing.T) {
	// Live backend the checker can actually reach.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	// Real SQLite repos so GetByID/Update succeed.
	dbPath := t.TempDir() + "/forcecheck.db"
	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	systemRepo := sqlite.NewSystemRepo(db)
	depRepo := sqlite.NewDependencyRepo(db)
	logRepo := sqlite.NewLogRepo(db)

	ctx := context.Background()
	sys, err := domain.NewSystem("ForceCheckSys", "desc", "https://example.com", "owner")
	if err != nil {
		t.Fatalf("new system: %v", err)
	}
	if err := systemRepo.Create(ctx, sys); err != nil {
		t.Fatalf("create system: %v", err)
	}

	// Seed a heartbeat-enabled dependency directly via the repo, bypassing the
	// domain-level SSRF validation in SetHeartbeat so we can target 127.0.0.1.
	dep := &domain.Dependency{
		SystemID:              sys.ID,
		Name:                  "Backend",
		Status:                domain.StatusGreen,
		HeartbeatURL:          backend.URL,
		HeartbeatInterval:     60,
		HeartbeatMethod:       "GET",
		HeartbeatExpectStatus: "200",
	}
	if err := depRepo.Create(ctx, dep); err != nil {
		t.Fatalf("create dependency: %v", err)
	}

	// Checker that allows private/loopback addresses so validateURL passes.
	checker := http_checker.NewWithOptions(2*time.Second, true)
	hb := application.NewHeartbeatService(depRepo, logRepo, checker)

	s := &Server{heartbeatService: hb}

	r := reqWithID("POST", "/api/dependencies/1/check", "id", "1", nil)
	w := httptest.NewRecorder()
	s.apiForceCheck(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("force check success: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Backend") {
		t.Errorf("expected dependency JSON in response, got %s", w.Body.String())
	}
}
