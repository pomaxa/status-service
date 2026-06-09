package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// The tests in this file drive the wrapped error branches of every repository
// by closing (or mutating) the database before the call so the underlying
// query/exec fails. They follow the existing closed-DB technique used across
// the package and assert on the real returned error.

// ---------- SystemRepo ----------

func TestSystemRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemRepo(db)
	db.Close()

	sys, _ := domain.NewSystem("X", "", "", "")
	if err := repo.Create(context.Background(), sys); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestSystemRepo_GetByID_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemRepo(db)
	db.Close()

	if _, err := repo.GetByID(context.Background(), 1); err == nil {
		t.Fatal("expected GetByID error on closed DB")
	}
}

func TestSystemRepo_GetAll_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemRepo(db)
	db.Close()

	if _, err := repo.GetAll(context.Background()); err == nil {
		t.Fatal("expected GetAll error on closed DB")
	}
}

func TestSystemRepo_Update_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemRepo(db)
	db.Close()

	sys, _ := domain.NewSystem("X", "", "", "")
	sys.ID = 1
	if err := repo.Update(context.Background(), sys); err == nil {
		t.Fatal("expected Update error on closed DB")
	}
}

func TestSystemRepo_Delete_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemRepo(db)
	db.Close()

	if err := repo.Delete(context.Background(), 1); err == nil {
		t.Fatal("expected Delete error on closed DB")
	}
}

// ---------- DependencyRepo ----------

func TestDependencyRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDependencyRepo(db)
	db.Close()

	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.HeartbeatMethod = "GET"
	if err := repo.Create(context.Background(), dep); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestDependencyRepo_GetByID_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDependencyRepo(db)
	db.Close()

	if _, err := repo.GetByID(context.Background(), 1); err == nil {
		t.Fatal("expected GetByID error on closed DB")
	}
}

func TestDependencyRepo_GetBySystemID_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDependencyRepo(db)
	db.Close()

	if _, err := repo.GetBySystemID(context.Background(), 1); err == nil {
		t.Fatal("expected GetBySystemID error on closed DB")
	}
}

func TestDependencyRepo_GetAllWithHeartbeat_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDependencyRepo(db)
	db.Close()

	if _, err := repo.GetAllWithHeartbeat(context.Background()); err == nil {
		t.Fatal("expected GetAllWithHeartbeat error on closed DB")
	}
}

func TestDependencyRepo_Update_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDependencyRepo(db)
	db.Close()

	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	dep.HeartbeatMethod = "GET"
	if err := repo.Update(context.Background(), dep); err == nil {
		t.Fatal("expected Update error on closed DB")
	}
}

func TestDependencyRepo_Delete_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDependencyRepo(db)
	db.Close()

	if err := repo.Delete(context.Background(), 1); err == nil {
		t.Fatal("expected Delete error on closed DB")
	}
}

// TestDecodeHeaders_MalformedJSON covers the json.Unmarshal error branch of
// decodeHeaders by storing invalid JSON in heartbeat_headers and reading it back.
func TestDecodeHeaders_MalformedJSON(t *testing.T) {
	if got := decodeHeaders("{not valid json"); got != nil {
		t.Errorf("expected nil for malformed headers JSON, got %v", got)
	}
	if got := decodeHeaders(""); got != nil {
		t.Errorf("expected nil for empty headers JSON, got %v", got)
	}
}

// TestDependencyRepo_GetByID_MalformedHeaders persists a row with invalid header
// JSON and verifies GetByID still scans the row (decodeHeaders returns nil).
func TestDependencyRepo_GetByID_MalformedHeaders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dependencies (id, system_id, name, heartbeat_method, heartbeat_headers)
		VALUES (1, 1, 'Dep', 'GET', '{bad json')
	`); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}

	repo := NewDependencyRepo(db)
	dep, err := repo.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if dep == nil {
		t.Fatal("expected dependency, got nil")
	}
	if dep.HeartbeatHeaders != nil {
		t.Errorf("expected nil headers for malformed JSON, got %v", dep.HeartbeatHeaders)
	}
}

// ---------- LogRepo ----------

func TestLogRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLogRepo(db)
	db.Close()

	id := int64(1)
	logEntry := &domain.StatusLog{
		SystemID:  &id,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Source:    domain.SourceManual,
		CreatedAt: time.Now(),
	}
	if err := repo.Create(context.Background(), logEntry); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestLogRepo_QueryErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLogRepo(db)
	db.Close()

	ctx := context.Background()
	now := time.Now()

	if _, err := repo.GetBySystemID(ctx, 1, 10); err == nil {
		t.Error("expected GetBySystemID error on closed DB")
	}
	if _, err := repo.GetByDependencyID(ctx, 1, 10); err == nil {
		t.Error("expected GetByDependencyID error on closed DB")
	}
	if _, err := repo.GetAll(ctx, 10); err == nil {
		t.Error("expected GetAll error on closed DB")
	}
	if _, err := repo.GetByTimeRange(ctx, now.Add(-time.Hour), now); err == nil {
		t.Error("expected GetByTimeRange error on closed DB")
	}
	if _, err := repo.GetSystemLogsByTimeRange(ctx, 1, now.Add(-time.Hour), now); err == nil {
		t.Error("expected GetSystemLogsByTimeRange error on closed DB")
	}
	if _, err := repo.GetDependencyLogsByTimeRange(ctx, 1, now.Add(-time.Hour), now); err == nil {
		t.Error("expected GetDependencyLogsByTimeRange error on closed DB")
	}
}

// ---------- IncidentRepo ----------

func TestIncidentRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewIncidentRepo(db)
	db.Close()

	inc, _ := domain.NewIncident("X", "msg", domain.SeverityMinor)
	if err := repo.Create(context.Background(), inc); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestIncidentRepo_QueryErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewIncidentRepo(db)
	db.Close()
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Error("expected GetByID error on closed DB")
	}
	if _, err := repo.GetAll(ctx, 10); err == nil {
		t.Error("expected GetAll(limit) error on closed DB")
	}
	if _, err := repo.GetAll(ctx, 0); err == nil {
		t.Error("expected GetAll(no limit) error on closed DB")
	}
	if _, err := repo.GetActive(ctx); err == nil {
		t.Error("expected GetActive error on closed DB")
	}
	if _, err := repo.GetRecent(ctx, 7); err == nil {
		t.Error("expected GetRecent error on closed DB")
	}
	if err := repo.Update(ctx, &domain.Incident{ID: 1}); err == nil {
		t.Error("expected Update error on closed DB")
	}
	if err := repo.Delete(ctx, 1); err == nil {
		t.Error("expected Delete error on closed DB")
	}
	if _, err := repo.GetUpdates(ctx, 1); err == nil {
		t.Error("expected GetUpdates error on closed DB")
	}
}

func TestIncidentRepo_CreateUpdate_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewIncidentRepo(db)
	db.Close()

	u, _ := domain.NewIncidentUpdate(1, domain.IncidentIdentified, "msg", "admin")
	if err := repo.CreateUpdate(context.Background(), u); err == nil {
		t.Fatal("expected CreateUpdate error on closed DB")
	}
}

// ---------- MaintenanceRepo ----------

func TestMaintenanceRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMaintenanceRepo(db)
	db.Close()

	start := time.Now()
	m, _ := domain.NewMaintenance("X", "", start, start.Add(time.Hour))
	if err := repo.Create(context.Background(), m); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestMaintenanceRepo_QueryErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMaintenanceRepo(db)
	db.Close()
	ctx := context.Background()
	now := time.Now()

	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Error("expected GetByID error on closed DB")
	}
	if _, err := repo.GetAll(ctx); err == nil {
		t.Error("expected GetAll error on closed DB")
	}
	if _, err := repo.GetActive(ctx); err == nil {
		t.Error("expected GetActive error on closed DB")
	}
	if _, err := repo.GetUpcoming(ctx); err == nil {
		t.Error("expected GetUpcoming error on closed DB")
	}
	if _, err := repo.GetByTimeRange(ctx, now.Add(-time.Hour), now); err == nil {
		t.Error("expected GetByTimeRange error on closed DB")
	}
	m, _ := domain.NewMaintenance("X", "", now, now.Add(time.Hour))
	m.ID = 1
	if err := repo.Update(ctx, m); err == nil {
		t.Error("expected Update error on closed DB")
	}
	if err := repo.Delete(ctx, 1); err == nil {
		t.Error("expected Delete error on closed DB")
	}
}

// ---------- WebhookRepo ----------

func TestWebhookRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)
	db.Close()

	w, _ := domain.NewWebhook("X", "https://example.com/hook", domain.WebhookTypeGeneric)
	if err := repo.Create(context.Background(), w); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestWebhookRepo_QueryErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)
	db.Close()
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Error("expected GetByID error on closed DB")
	}
	if _, err := repo.GetAll(ctx); err == nil {
		t.Error("expected GetAll error on closed DB")
	}
	if _, err := repo.GetEnabled(ctx); err == nil {
		t.Error("expected GetEnabled error on closed DB")
	}
	w, _ := domain.NewWebhook("X", "https://example.com/hook", domain.WebhookTypeGeneric)
	w.ID = 1
	if err := repo.Update(ctx, w); err == nil {
		t.Error("expected Update error on closed DB")
	}
	if err := repo.Delete(ctx, 1); err == nil {
		t.Error("expected Delete error on closed DB")
	}
}

// ---------- APIKeyRepo ----------

func TestAPIKeyRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepo(db)
	db.Close()

	key := &domain.APIKey{
		Name:    "K",
		Key:     "sk_err",
		KeyHash: domain.HashAPIKey("sk_err"),
		Scopes:  []string{"read"},
		Enabled: true,
	}
	if err := repo.Create(context.Background(), key); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestAPIKeyRepo_QueryErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepo(db)
	db.Close()
	ctx := context.Background()

	if _, err := repo.GetByKey(ctx, "sk_missing"); err == nil {
		t.Error("expected GetByKey error on closed DB")
	}
	if _, err := repo.GetAll(ctx); err == nil {
		t.Error("expected GetAll error on closed DB")
	}
	key := &domain.APIKey{ID: 1, Name: "K", Scopes: []string{"read"}}
	if err := repo.Update(ctx, key); err == nil {
		t.Error("expected Update error on closed DB")
	}
	if err := repo.Delete(ctx, 1); err == nil {
		t.Error("expected Delete error on closed DB")
	}
	if err := repo.UpdateLastUsed(ctx, 1); err == nil {
		t.Error("expected UpdateLastUsed error on closed DB")
	}
}

// TestAPIKeyRepo_GetByKey_LegacyHashMigration covers the legacy fallback path in
// GetByKey: a record whose stored key_hash differs from HashAPIKey(keyValue) is
// found via key_value and then opportunistically migrated to the SHA-256 hash.
func TestAPIKeyRepo_GetByKey_LegacyHashMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	keyValue := "sk_legacy_value"
	// Store a deliberately mismatched hash so getByKeyHash misses and the
	// getByKeyValue fallback + migration branch runs.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO api_keys (name, key_value, key_hash, scopes, enabled)
		VALUES ('legacy', ?, 'legacy-random-hash', '["read"]', 1)
	`, keyValue); err != nil {
		t.Fatalf("insert legacy key: %v", err)
	}

	repo := NewAPIKeyRepo(db)
	key, err := repo.GetByKey(ctx, keyValue)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if key == nil {
		t.Fatal("expected to find legacy key by value")
	}
	if key.KeyHash != domain.HashAPIKey(keyValue) {
		t.Errorf("expected key_hash to be migrated to SHA-256, got %q", key.KeyHash)
	}

	// Subsequent lookup should now hit the hash path and return the same key.
	again, err := repo.GetByKey(ctx, keyValue)
	if err != nil {
		t.Fatalf("second GetByKey() error = %v", err)
	}
	if again == nil {
		t.Fatal("expected to find migrated key by hash")
	}
}

// TestAPIKeyRepo_GetAll_MalformedScopes covers the json.Unmarshal failure branch
// in GetAll (falls back to default scopes).
func TestAPIKeyRepo_GetAll_MalformedScopes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO api_keys (name, key_value, key_hash, scopes, enabled)
		VALUES ('bad', 'sk_bad', 'h', 'not-json', 1)
	`); err != nil {
		t.Fatalf("insert key: %v", err)
	}

	repo := NewAPIKeyRepo(db)
	keys, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if len(keys[0].Scopes) != 1 || keys[0].Scopes[0] != "read" {
		t.Errorf("expected fallback scopes [read], got %v", keys[0].Scopes)
	}
}

// ---------- SLAReportRepo ----------

func TestSLAReportRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSLAReportRepo(db)
	db.Close()

	report := domain.NewSLAReport("R", "monthly", time.Now().Add(-time.Hour), time.Now(), "admin")
	if err := repo.Create(context.Background(), report); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestSLAReportRepo_QueryErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSLAReportRepo(db)
	db.Close()
	ctx := context.Background()
	now := time.Now()

	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Error("expected GetByID error on closed DB")
	}
	if _, err := repo.GetAll(ctx, 10); err == nil {
		t.Error("expected GetAll error on closed DB")
	}
	if _, err := repo.GetByPeriod(ctx, now.Add(-time.Hour), now); err == nil {
		t.Error("expected GetByPeriod error on closed DB")
	}
	if err := repo.Delete(ctx, 1); err == nil {
		t.Error("expected Delete error on closed DB")
	}
}

// TestSLAReportRepo_GetByID_MalformedData covers the json.Unmarshal error branch
// in GetByID when report_data is invalid JSON.
func TestSLAReportRepo_GetByID_MalformedData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO sla_reports (id, title, period, period_start, period_end, report_data)
		VALUES (1, 'R', 'monthly', ?, ?, 'not-json')
	`, time.Now().Add(-time.Hour), time.Now()); err != nil {
		t.Fatalf("insert report: %v", err)
	}

	repo := NewSLAReportRepo(db)
	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Fatal("expected error unmarshalling malformed report_data")
	}
}

// TestSLAReportRepo_GetAll_MalformedData covers the json.Unmarshal fallback in
// GetAll where report_data is invalid (SystemReports set to nil, no error).
func TestSLAReportRepo_GetAll_MalformedData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO sla_reports (id, title, period, period_start, period_end, report_data)
		VALUES (1, 'R', 'monthly', ?, ?, 'not-json')
	`, time.Now().Add(-time.Hour), time.Now()); err != nil {
		t.Fatalf("insert report: %v", err)
	}

	repo := NewSLAReportRepo(db)
	reports, err := repo.GetAll(ctx, 10)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].SystemReports != nil {
		t.Errorf("expected nil SystemReports for malformed data, got %v", reports[0].SystemReports)
	}
}

// TestSLAReportRepo_GetByPeriod_MalformedData covers the GetByPeriod malformed
// report_data fallback branch.
func TestSLAReportRepo_GetByPeriod_MalformedData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	start := time.Now().Add(-2 * time.Hour)
	end := time.Now()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO sla_reports (id, title, period, period_start, period_end, report_data)
		VALUES (1, 'R', 'monthly', ?, ?, 'not-json')
	`, start, end); err != nil {
		t.Fatalf("insert report: %v", err)
	}

	repo := NewSLAReportRepo(db)
	reports, err := repo.GetByPeriod(ctx, start.Add(-time.Hour), end.Add(time.Hour))
	if err != nil {
		t.Fatalf("GetByPeriod() error = %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].SystemReports != nil {
		t.Errorf("expected nil SystemReports for malformed data, got %v", reports[0].SystemReports)
	}
}

// ---------- SLABreachRepo ----------

func TestSLABreachRepo_Create_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSLABreachRepo(db)
	db.Close()

	breach := &domain.SLABreachEvent{
		SystemID:    1,
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.0,
		Period:      "monthly",
		PeriodStart: time.Now().Add(-time.Hour),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}
	if err := repo.Create(context.Background(), breach); err == nil {
		t.Fatal("expected Create error on closed DB")
	}
}

func TestSLABreachRepo_QueryErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSLABreachRepo(db)
	db.Close()
	ctx := context.Background()
	now := time.Now()

	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Error("expected GetByID error on closed DB")
	}
	if _, err := repo.GetAll(ctx, 10); err == nil {
		t.Error("expected GetAll error on closed DB")
	}
	if _, err := repo.GetUnacknowledged(ctx); err == nil {
		t.Error("expected GetUnacknowledged error on closed DB")
	}
	if _, err := repo.GetBySystemID(ctx, 1, 10); err == nil {
		t.Error("expected GetBySystemID error on closed DB")
	}
	if _, err := repo.GetByPeriod(ctx, now.Add(-time.Hour), now); err == nil {
		t.Error("expected GetByPeriod error on closed DB")
	}
	if err := repo.Acknowledge(ctx, 1, "admin"); err == nil {
		t.Error("expected Acknowledge error on closed DB")
	}
}

// ---------- LatencyRepo ----------

func TestLatencyRepo_Record_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLatencyRepo(db)
	db.Close()

	rec := &domain.LatencyRecord{DependencyID: 1, LatencyMs: 10, Success: true, StatusCode: 200}
	if err := repo.Record(context.Background(), rec); err == nil {
		t.Fatal("expected Record error on closed DB")
	}
}

func TestLatencyRepo_QueryErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLatencyRepo(db)
	db.Close()
	ctx := context.Background()
	now := time.Now()

	if _, err := repo.GetByDependency(ctx, 1, now.Add(-time.Hour), now, 100); err == nil {
		t.Error("expected GetByDependency error on closed DB")
	}
	if _, err := repo.GetAggregated(ctx, 1, now.Add(-time.Hour), now, 5); err == nil {
		t.Error("expected GetAggregated error on closed DB")
	}
	if _, err := repo.GetDailyUptime(ctx, 1, 7); err == nil {
		t.Error("expected GetDailyUptime error on closed DB")
	}
	if _, err := repo.GetStats(ctx, 1, now.Add(-time.Hour), now); err == nil {
		t.Error("expected GetStats error on closed DB")
	}
	if err := repo.Cleanup(ctx, now); err == nil {
		t.Error("expected Cleanup error on closed DB")
	}
}
