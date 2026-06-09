package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// TestSLAReportRepo_GetAll_DefaultLimit covers the `limit <= 0 => limit = 100`
// branch in SLAReportRepo.GetAll.
func TestSLAReportRepo_GetAll_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	report := domain.NewSLAReport("R", "monthly", time.Now().Add(-time.Hour), time.Now(), "admin")
	repo := NewSLAReportRepo(db)
	if err := repo.Create(ctx, report); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	reports, err := repo.GetAll(ctx, 0) // limit <= 0 triggers default
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
}

// TestSLABreachRepo_GetAll_DefaultLimit covers the default-limit branch in
// SLABreachRepo.GetAll.
func TestSLABreachRepo_GetAll_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	repo := NewSLABreachRepo(db)
	if err := repo.Create(ctx, &domain.SLABreachEvent{
		SystemID:    1,
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.0,
		Period:      "monthly",
		PeriodStart: time.Now().Add(-time.Hour),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	breaches, err := repo.GetAll(ctx, 0) // default limit branch
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(breaches) != 1 {
		t.Fatalf("expected 1 breach, got %d", len(breaches))
	}
}

// TestSLABreachRepo_GetBySystemID_DefaultLimit covers the default-limit branch
// in SLABreachRepo.GetBySystemID.
func TestSLABreachRepo_GetBySystemID_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	repo := NewSLABreachRepo(db)
	if err := repo.Create(ctx, &domain.SLABreachEvent{
		SystemID:    1,
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.0,
		Period:      "monthly",
		PeriodStart: time.Now().Add(-time.Hour),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	breaches, err := repo.GetBySystemID(ctx, 1, 0) // default limit branch
	if err != nil {
		t.Fatalf("GetBySystemID() error = %v", err)
	}
	if len(breaches) != 1 {
		t.Fatalf("expected 1 breach, got %d", len(breaches))
	}
}

// ---------- scan-error branches ----------
//
// SQLite stores values with loose typing, so a text value written into a
// numeric-affinity column is accepted on INSERT but fails to scan into the Go
// destination type, exercising the "failed to scan" error branches in the row
// loops and single-row scanners.

func TestSystemRepo_GetAll_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO systems (id, name, status, sla_target, created_at, updated_at)
		VALUES (1, 'n', 'green', 'not-a-float', ?, ?)
	`, time.Now(), time.Now()); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	repo := NewSystemRepo(db)
	if _, err := repo.GetAll(ctx); err == nil {
		t.Fatal("expected scan error from non-numeric sla_target")
	}
}

func TestDependencyRepo_ScanErrors(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dependencies (id, system_id, name, status, heartbeat_method, last_latency)
		VALUES (1, 1, 'd', 'green', 'GET', 'not-an-int')
	`); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}

	repo := NewDependencyRepo(db)
	if _, err := repo.GetBySystemID(ctx, 1); err == nil {
		t.Error("expected scan error in scanDependencies from non-numeric last_latency")
	}
	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Error("expected scan error in scanDependency from non-numeric last_latency")
	}
}

func TestSLABreachRepo_ScanErrors(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO sla_breaches (id, system_id, breach_type, sla_target, actual_value, period, period_start, period_end, detected_at)
		VALUES (1, 1, 'uptime', 'not-a-float', 98.0, 'monthly', ?, ?, ?)
	`, time.Now(), time.Now(), time.Now()); err != nil {
		t.Fatalf("insert breach: %v", err)
	}

	repo := NewSLABreachRepo(db)
	if _, err := repo.GetAll(ctx, 10); err == nil {
		t.Error("expected scan error in scanBreaches from non-numeric sla_target")
	}
	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Error("expected scan error in GetByID from non-numeric sla_target")
	}
}

func TestSLAReportRepo_ScanErrors(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO sla_reports (id, title, period, period_start, period_end, overall_uptime, report_data)
		VALUES (1, 't', 'monthly', ?, ?, 'not-a-float', '{}')
	`, time.Now(), time.Now()); err != nil {
		t.Fatalf("insert report: %v", err)
	}

	repo := NewSLAReportRepo(db)
	if _, err := repo.GetAll(ctx, 10); err == nil {
		t.Error("expected scan error in GetAll from non-numeric overall_uptime")
	}
	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Error("expected scan error in GetByID from non-numeric overall_uptime")
	}
	if _, err := repo.GetByPeriod(ctx, time.Now().Add(-48*time.Hour), time.Now().Add(48*time.Hour)); err == nil {
		t.Error("expected scan error in GetByPeriod from non-numeric overall_uptime")
	}
}

func TestWebhookRepo_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO webhooks (id, name, url, type, events, enabled, created_at, updated_at)
		VALUES (1, 'n', 'u', 'generic', '[]', 'not-a-bool', ?, ?)
	`, time.Now(), time.Now()); err != nil {
		t.Fatalf("insert webhook: %v", err)
	}

	repo := NewWebhookRepo(db)
	if _, err := repo.GetAll(ctx); err == nil {
		t.Fatal("expected scan error in queryWebhooks from non-bool enabled")
	}
	if _, err := repo.GetByID(ctx, 1); err == nil {
		t.Fatal("expected scan error in GetByID from non-bool enabled")
	}
}

func TestAPIKeyRepo_GetAll_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO api_keys (id, name, key_value, key_hash, scopes, enabled, created_at)
		VALUES (1, 'n', 'v', 'h', '["read"]', 'not-a-bool', ?)
	`, time.Now()); err != nil {
		t.Fatalf("insert api key: %v", err)
	}

	repo := NewAPIKeyRepo(db)
	if _, err := repo.GetAll(ctx); err == nil {
		t.Fatal("expected scan error in GetAll from non-bool enabled")
	}
}

// TestAPIKeyRepo_GetByKey_MalformedScopes covers the scanAPIKeyRow
// json.Unmarshal fallback branch (single-row path), where invalid scopes JSON
// falls back to the default ["read"] scope.
func TestAPIKeyRepo_GetByKey_MalformedScopes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	keyValue := "sk_bad_scopes"
	if _, err := db.ExecContext(ctx, `
		INSERT INTO api_keys (id, name, key_value, key_hash, scopes, enabled, created_at)
		VALUES (1, 'n', ?, ?, 'not-json', 1, ?)
	`, keyValue, domain.HashAPIKey(keyValue), time.Now()); err != nil {
		t.Fatalf("insert api key: %v", err)
	}

	repo := NewAPIKeyRepo(db)
	key, err := repo.GetByKey(ctx, keyValue)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if key == nil {
		t.Fatal("expected key, got nil")
	}
	if len(key.Scopes) != 1 || key.Scopes[0] != "read" {
		t.Errorf("expected fallback scopes [read], got %v", key.Scopes)
	}
}

// TestAPIKeyRepo_GetAll_WithTimestamps covers the expires_at / last_used
// `Valid` branches in the GetAll scan loop by storing both timestamps.
func TestAPIKeyRepo_GetAll_WithTimestamps(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	expires := time.Now().Add(24 * time.Hour)
	lastUsed := time.Now().Add(-time.Hour)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO api_keys (id, name, key_value, key_hash, scopes, enabled, expires_at, last_used, created_at)
		VALUES (1, 'n', 'v', 'h', '["read"]', 1, ?, ?, ?)
	`, expires, lastUsed, time.Now()); err != nil {
		t.Fatalf("insert api key: %v", err)
	}

	repo := NewAPIKeyRepo(db)
	keys, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
	if keys[0].LastUsed == nil {
		t.Error("expected LastUsed to be set")
	}
}

// TestSLABreachRepo_GetAll_WithAckedAt covers the ackedAt `Valid` branch in
// scanBreaches by storing an acknowledged breach with acked_at set.
func TestSLABreachRepo_GetAll_WithAckedAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO sla_breaches (id, system_id, breach_type, sla_target, actual_value, period, period_start, period_end, detected_at, acknowledged, acked_by, acked_at)
		VALUES (1, 1, 'uptime', 99.9, 98.0, 'monthly', ?, ?, ?, 1, 'admin', ?)
	`, time.Now(), time.Now(), time.Now(), time.Now()); err != nil {
		t.Fatalf("insert breach: %v", err)
	}

	repo := NewSLABreachRepo(db)
	breaches, err := repo.GetAll(ctx, 10)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(breaches) != 1 {
		t.Fatalf("expected 1 breach, got %d", len(breaches))
	}
	if breaches[0].AckedAt == nil {
		t.Error("expected AckedAt to be set in scanBreaches")
	}
}

// TestWebhookRepo_GetAll_WithSystemIDs covers the systemIDsJSON.Valid branch in
// queryWebhooks by storing a webhook scoped to specific system IDs.
func TestWebhookRepo_GetAll_WithSystemIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO webhooks (id, name, url, type, events, system_ids, enabled, created_at, updated_at)
		VALUES (1, 'n', 'https://e.com/h', 'generic', '["status_change"]', '[1,2,3]', 1, ?, ?)
	`, time.Now(), time.Now()); err != nil {
		t.Fatalf("insert webhook: %v", err)
	}

	repo := NewWebhookRepo(db)
	hooks, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(hooks))
	}
	if len(hooks[0].SystemIDs) == 0 {
		t.Error("expected SystemIDs to be parsed in queryWebhooks")
	}
}

// TestDependencyRepo_GetBySystemID_AllFields covers the optional-field Valid
// branches in scanDependencies (heartbeat_method, heartbeat_headers, last_check)
// by storing a dependency with all of them set.
func TestDependencyRepo_GetBySystemID_AllFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dependencies (id, system_id, name, status, heartbeat_url, heartbeat_method, heartbeat_headers, last_check)
		VALUES (1, 1, 'd', 'green', 'https://e.com/hb', 'POST', '{"Authorization":"Bearer x"}', ?)
	`, time.Now()); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}

	repo := NewDependencyRepo(db)
	deps, err := repo.GetBySystemID(ctx, 1)
	if err != nil {
		t.Fatalf("GetBySystemID() error = %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}
	d := deps[0]
	if d.HeartbeatMethod != "POST" {
		t.Errorf("expected heartbeat method POST, got %q", d.HeartbeatMethod)
	}
	if len(d.HeartbeatHeaders) == 0 {
		t.Error("expected heartbeat headers to be decoded")
	}
	if d.LastCheck.IsZero() {
		t.Error("expected last_check to be set")
	}
}

// TestLatencyRepo_ScanErrors covers the row-scan error branches of
// GetByDependency, GetAggregated and the first GetStats scan, by storing a
// non-numeric latency_ms value.
func TestLatencyRepo_ScanErrors(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO dependencies (id, system_id, name, heartbeat_method) VALUES (1, 1, 'd', 'GET')"); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO latency_history (dependency_id, latency_ms, success, status_code, created_at)
		VALUES (1, 'abc', 1, 200, ?)
	`, time.Now()); err != nil {
		t.Fatalf("insert latency: %v", err)
	}

	repo := NewLatencyRepo(db)
	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)

	if _, err := repo.GetByDependency(ctx, 1, start, end, 100); err == nil {
		t.Error("expected scan error in GetByDependency from non-numeric latency_ms")
	}
	if _, err := repo.GetAggregated(ctx, 1, start, end, 5); err == nil {
		t.Error("expected scan error in GetAggregated from non-numeric latency_ms")
	}
	if _, err := repo.GetStats(ctx, 1, start, end); err == nil {
		t.Error("expected scan error in GetStats from non-numeric latency_ms")
	}
}

// TestLatencyRepo_GetStats_Over100Samples inserts more than 100 successful
// records so GetStats computes P99 via the len(latencies) > 100 branch (the
// existing percentile test uses exactly 100, hitting the else branch).
func TestLatencyRepo_GetStats_Over100Samples(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO dependencies (id, system_id, name, heartbeat_method) VALUES (1, 1, 'd', 'GET')"); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}

	repo := NewLatencyRepo(db)
	now := time.Now()
	for i := 0; i < 150; i++ {
		if err := repo.Record(ctx, &domain.LatencyRecord{
			DependencyID: 1,
			LatencyMs:    int64(i + 1),
			Success:      true,
			StatusCode:   200,
		}); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	stats, err := repo.GetStats(ctx, 1, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	if stats.TotalChecks != 150 {
		t.Errorf("expected 150 checks, got %d", stats.TotalChecks)
	}
	if stats.P99LatencyMs == 0 {
		t.Error("expected non-zero P99 latency for >100 samples")
	}
	if stats.UptimePercent != 100 {
		t.Errorf("expected 100%% uptime with all successes, got %.2f", stats.UptimePercent)
	}
}

// TestAPIKeyRepo_GetByKey_ScanError covers the scanAPIKeyRow non-ErrNoRows error
// branch via getByKeyHash: a non-bool enabled value makes Scan fail.
func TestAPIKeyRepo_GetByKey_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	keyValue := "sk_scan_err"
	if _, err := db.ExecContext(ctx, `
		INSERT INTO api_keys (id, name, key_value, key_hash, scopes, enabled, created_at)
		VALUES (1, 'n', ?, ?, '["read"]', 'not-a-bool', ?)
	`, keyValue, domain.HashAPIKey(keyValue), time.Now()); err != nil {
		t.Fatalf("insert api key: %v", err)
	}

	repo := NewAPIKeyRepo(db)
	if _, err := repo.GetByKey(ctx, keyValue); err == nil {
		t.Fatal("expected scan error from non-bool enabled in GetByKey")
	}
}
