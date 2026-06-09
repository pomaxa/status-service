package sqlite

import (
	"context"
	"status-incident/internal/domain"
	"testing"
	"time"
)

// insertSystemLog is a small helper to persist a status log for a system.
func insertSystemLog(t *testing.T, db *DB, systemID int64, old, next domain.Status, at time.Time) {
	t.Helper()
	logRepo := NewLogRepo(db)
	id := systemID
	if err := logRepo.Create(context.Background(), &domain.StatusLog{
		SystemID:  &id,
		OldStatus: old,
		NewStatus: next,
		Message:   "log",
		Source:    domain.SourceManual,
		CreatedAt: at,
	}); err != nil {
		t.Fatalf("failed to insert system log: %v", err)
	}
}

// insertDependencyLog persists a status log for a dependency.
func insertDependencyLog(t *testing.T, db *DB, depID int64, old, next domain.Status, at time.Time) {
	t.Helper()
	logRepo := NewLogRepo(db)
	id := depID
	if err := logRepo.Create(context.Background(), &domain.StatusLog{
		DependencyID: &id,
		OldStatus:    old,
		NewStatus:    next,
		Message:      "log",
		Source:       domain.SourceManual,
		CreatedAt:    at,
	}); err != nil {
		t.Fatalf("failed to insert dependency log: %v", err)
	}
}

func TestAnalyticsRepo_GetIncidentsBySystemID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now

	insertSystemLog(t, db, 1, domain.StatusGreen, domain.StatusRed, start.Add(2*time.Hour))
	insertSystemLog(t, db, 1, domain.StatusRed, domain.StatusGreen, start.Add(4*time.Hour))

	repo := NewAnalyticsRepo(db)
	incidents, err := repo.GetIncidentsBySystemID(ctx, 1, start, end)
	if err != nil {
		t.Fatalf("GetIncidentsBySystemID() error = %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidents))
	}
	if incidents[0].SystemID == nil || *incidents[0].SystemID != 1 {
		t.Errorf("expected incident system id 1")
	}
}

func TestAnalyticsRepo_GetIncidentsBySystemID_QueryError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepo(db)
	db.Close() // force query failure inside logRepo

	if _, err := repo.GetIncidentsBySystemID(context.Background(), 1, time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected error from GetIncidentsBySystemID on closed DB")
	}
}

func TestAnalyticsRepo_GetIncidentsByDependencyID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO dependencies (id, system_id, name) VALUES (1, 1, 'Dep')"); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now

	insertDependencyLog(t, db, 1, domain.StatusGreen, domain.StatusRed, start.Add(3*time.Hour))
	insertDependencyLog(t, db, 1, domain.StatusRed, domain.StatusGreen, start.Add(6*time.Hour))

	repo := NewAnalyticsRepo(db)
	incidents, err := repo.GetIncidentsByDependencyID(ctx, 1, start, end)
	if err != nil {
		t.Fatalf("GetIncidentsByDependencyID() error = %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidents))
	}
	if incidents[0].DependencyID == nil || *incidents[0].DependencyID != 1 {
		t.Errorf("expected incident dependency id 1")
	}
}

func TestAnalyticsRepo_GetIncidentsByDependencyID_QueryError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepo(db)
	db.Close()

	if _, err := repo.GetIncidentsByDependencyID(context.Background(), 1, time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected error from GetIncidentsByDependencyID on closed DB")
	}
}

func TestAnalyticsRepo_GetUptimeByDependencyID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO dependencies (id, system_id, name) VALUES (1, 1, 'Dep')"); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now

	insertDependencyLog(t, db, 1, domain.StatusGreen, domain.StatusRed, start.Add(2*time.Hour))
	insertDependencyLog(t, db, 1, domain.StatusRed, domain.StatusGreen, start.Add(8*time.Hour))

	repo := NewAnalyticsRepo(db)
	analytics, err := repo.GetUptimeByDependencyID(ctx, 1, start, end)
	if err != nil {
		t.Fatalf("GetUptimeByDependencyID() error = %v", err)
	}
	if analytics.EntityType != "dependency" {
		t.Errorf("expected entity type dependency, got %s", analytics.EntityType)
	}
	if analytics.EntityName != "Dep" {
		t.Errorf("expected entity name Dep, got %s", analytics.EntityName)
	}
	if analytics.TotalIncidents != 1 {
		t.Errorf("expected 1 incident, got %d", analytics.TotalIncidents)
	}
}

func TestAnalyticsRepo_GetUptimeByDependencyID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAnalyticsRepo(db)

	if _, err := repo.GetUptimeByDependencyID(context.Background(), 999, time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected error for unknown dependency id")
	}
}

func TestAnalyticsRepo_GetUptimeByDependencyID_LogsQueryError(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO dependencies (id, system_id, name) VALUES (1, 1, 'Dep')"); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}
	repo := NewAnalyticsRepo(db)

	// Drop the status_log table so the name lookup succeeds but the log query fails.
	if _, err := db.ExecContext(ctx, "DROP TABLE status_log"); err != nil {
		t.Fatalf("drop status_log: %v", err)
	}

	if _, err := repo.GetUptimeByDependencyID(ctx, 1, time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected error when status_log query fails")
	}
}

func TestAnalyticsRepo_GetUptimeBySystemID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAnalyticsRepo(db)

	if _, err := repo.GetUptimeBySystemID(context.Background(), 999, time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected error for unknown system id")
	}
}

func TestAnalyticsRepo_GetUptimeBySystemID_LogsQueryError(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	repo := NewAnalyticsRepo(db)

	if _, err := db.ExecContext(ctx, "DROP TABLE status_log"); err != nil {
		t.Fatalf("drop status_log: %v", err)
	}

	if _, err := repo.GetUptimeBySystemID(ctx, 1, time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected error when status_log query fails")
	}
}

func TestAnalyticsRepo_GetOverallAnalytics_QueryError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepo(db)
	db.Close()

	if _, err := repo.GetOverallAnalytics(context.Background(), time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected error querying systems on closed DB")
	}
}

// TestAnalyticsRepo_GetOverallAnalytics_PeriodBuckets exercises the period
// string bucketing branches (1h, 24h, 7d, 30d, and >30d default) in
// GetOverallAnalytics.
func TestAnalyticsRepo_GetOverallAnalytics_PeriodBuckets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	repo := NewAnalyticsRepo(db)
	now := time.Now()

	cases := []struct {
		name  string
		start time.Time
		want  string
	}{
		{"1h", now.Add(-30 * time.Minute), "1h"},
		{"24h", now.Add(-12 * time.Hour), "24h"},
		{"7d", now.Add(-3 * 24 * time.Hour), "7d"},
		{"30d", now.Add(-20 * 24 * time.Hour), "30d"},
		{"long", now.Add(-60 * 24 * time.Hour), "60d"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := repo.GetOverallAnalytics(ctx, tc.start, now)
			if err != nil {
				t.Fatalf("GetOverallAnalytics() error = %v", err)
			}
			if a.Period != tc.want {
				t.Errorf("period = %q, want %q", a.Period, tc.want)
			}
		})
	}
}

// TestAnalyticsRepo_GetOverallAnalytics_SkipsFailingSystem drops the systems
// table after collecting IDs is not possible here; instead we verify the
// per-system aggregation path runs across multiple systems with incidents.
func TestAnalyticsRepo_GetOverallAnalytics_MultipleSystems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (?, ?)", i, "Sys"); err != nil {
			t.Fatalf("insert system: %v", err)
		}
	}

	now := time.Now()
	start := now.Add(-24 * time.Hour)

	// System 1 has a resolved incident.
	insertSystemLog(t, db, 1, domain.StatusGreen, domain.StatusRed, start.Add(2*time.Hour))
	insertSystemLog(t, db, 1, domain.StatusRed, domain.StatusGreen, start.Add(5*time.Hour))

	repo := NewAnalyticsRepo(db)
	a, err := repo.GetOverallAnalytics(ctx, start, now)
	if err != nil {
		t.Fatalf("GetOverallAnalytics() error = %v", err)
	}
	if a.TotalIncidents != 1 {
		t.Errorf("expected 1 total incident, got %d", a.TotalIncidents)
	}
}

// TestAnalyticsRepo_BuildAnalytics_OngoingIncident covers the ongoing-incident
// path: an incident that starts but never ends within the period. This
// exercises the currentIncident-at-end branch in calculateIncidents and the
// ongoing duration capping in buildAnalytics.
func TestAnalyticsRepo_BuildAnalytics_OngoingIncident(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	now := time.Now()
	// Period ends in the past so the ongoing incident is capped at `end`.
	start := now.Add(-48 * time.Hour)
	end := now.Add(-24 * time.Hour)

	// Incident starts within the period and never recovers (ongoing).
	insertSystemLog(t, db, 1, domain.StatusGreen, domain.StatusRed, start.Add(6*time.Hour))
	// A later degraded log raises severity tracking but does not end it.
	insertSystemLog(t, db, 1, domain.StatusRed, domain.StatusYellow, start.Add(8*time.Hour))

	repo := NewAnalyticsRepo(db)
	a, err := repo.GetUptimeBySystemID(ctx, 1, start, end)
	if err != nil {
		t.Fatalf("GetUptimeBySystemID() error = %v", err)
	}
	if a.OngoingIncidents != 1 {
		t.Errorf("expected 1 ongoing incident, got %d", a.OngoingIncidents)
	}
	if a.TotalDowntime <= 0 {
		t.Errorf("expected positive downtime for ongoing incident, got %v", a.TotalDowntime)
	}
}

// TestAnalyticsRepo_GetOverallAnalytics_SkipsFailingSystems covers the
// continue branch: every per-system GetUptimeBySystemID call fails (status_log
// table dropped), so all systems are skipped while n > 0.
func TestAnalyticsRepo_GetOverallAnalytics_SkipsFailingSystems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	for i := 1; i <= 2; i++ {
		if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (?, 'Sys')", i); err != nil {
			t.Fatalf("insert system: %v", err)
		}
	}

	// Drop status_log so the per-system log query fails for every system,
	// forcing the loop's `continue` path.
	if _, err := db.ExecContext(ctx, "DROP TABLE status_log"); err != nil {
		t.Fatalf("drop status_log: %v", err)
	}

	repo := NewAnalyticsRepo(db)
	now := time.Now()
	a, err := repo.GetOverallAnalytics(ctx, now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatalf("GetOverallAnalytics() error = %v", err)
	}
	// All systems skipped => aggregates remain zero.
	if a.TotalIncidents != 0 {
		t.Errorf("expected 0 incidents when all systems fail, got %d", a.TotalIncidents)
	}
}

// TestAnalyticsRepo_BuildAnalytics_DirectCapping calls the unexported
// buildAnalytics directly with crafted incident periods. The query layer always
// filters logs to the [start,end] window, so the duration-capping branches
// (incident started before start, ended after end, ongoing extending past end)
// can only be exercised by constructing IncidentPeriods spanning the bounds.
func TestAnalyticsRepo_BuildAnalytics_DirectCapping(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAnalyticsRepo(db)

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now.Add(-1 * time.Hour) // in the past so time.Now().After(end) is true

	startedBefore := start.Add(-2 * time.Hour)
	endedAfter := end.Add(2 * time.Hour)

	resolvedOverflow := endedAfter
	incidents := []domain.IncidentPeriod{
		// Resolved incident that started before `start` and ended after `end`.
		{
			SystemID:    int64Ptr(1),
			StartedAt:   startedBefore,
			EndedAt:     &resolvedOverflow,
			Duration:    endedAfter.Sub(startedBefore),
			MaxSeverity: domain.StatusRed,
		},
		// Ongoing incident that started inside the window (caps at end).
		{
			SystemID:    int64Ptr(1),
			StartedAt:   start.Add(2 * time.Hour),
			EndedAt:     nil,
			MaxSeverity: domain.StatusYellow,
		},
		// Ongoing incident that started before the window (duration == full period).
		{
			SystemID:    int64Ptr(1),
			StartedAt:   startedBefore,
			EndedAt:     nil,
			MaxSeverity: domain.StatusRed,
		},
	}

	a := repo.buildAnalytics(1, "system", "Sys", start, end, nil, incidents)
	if a.TotalIncidents != 3 {
		t.Errorf("expected 3 incidents, got %d", a.TotalIncidents)
	}
	if a.ResolvedIncidents != 1 {
		t.Errorf("expected 1 resolved incident, got %d", a.ResolvedIncidents)
	}
	if a.OngoingIncidents != 2 {
		t.Errorf("expected 2 ongoing incidents, got %d", a.OngoingIncidents)
	}
	// Downtime is clamped: greenDuration cannot go negative => uptime stays >= 0.
	if a.UptimePercent < 0 {
		t.Errorf("uptime should be clamped to >= 0, got %.2f", a.UptimePercent)
	}
}

func int64Ptr(v int64) *int64 { return &v }

// TestAnalyticsRepo_CalculateIncidents_SeverityEscalation covers the branch in
// calculateIncidents that raises MaxSeverity when a later log within the same
// incident has a higher severity than the current maximum.
func TestAnalyticsRepo_CalculateIncidents_SeverityEscalation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	now := time.Now()
	start := now.Add(-24 * time.Hour)

	// Incident starts yellow, escalates to red, then recovers.
	insertSystemLog(t, db, 1, domain.StatusGreen, domain.StatusYellow, start.Add(2*time.Hour))
	insertSystemLog(t, db, 1, domain.StatusYellow, domain.StatusRed, start.Add(3*time.Hour))
	insertSystemLog(t, db, 1, domain.StatusRed, domain.StatusGreen, start.Add(5*time.Hour))

	repo := NewAnalyticsRepo(db)
	incidents, err := repo.GetIncidentsBySystemID(ctx, 1, start, now)
	if err != nil {
		t.Fatalf("GetIncidentsBySystemID() error = %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidents))
	}
	if incidents[0].MaxSeverity != domain.StatusRed {
		t.Errorf("expected MaxSeverity red after escalation, got %v", incidents[0].MaxSeverity)
	}
}

// TestAnalyticsRepo_BuildAnalytics_PeriodBuckets covers the period-string
// bucketing in buildAnalytics (24h, 7d, 30d, default >30d).
func TestAnalyticsRepo_BuildAnalytics_PeriodBuckets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	repo := NewAnalyticsRepo(db)
	now := time.Now()

	cases := []struct {
		name  string
		start time.Time
		want  string
	}{
		{"24h", now.Add(-12 * time.Hour), "24h"},
		{"7d", now.Add(-3 * 24 * time.Hour), "7d"},
		{"30d", now.Add(-20 * 24 * time.Hour), "30d"},
		{"long", now.Add(-90 * 24 * time.Hour), "90d"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := repo.GetUptimeBySystemID(ctx, 1, tc.start, now)
			if err != nil {
				t.Fatalf("GetUptimeBySystemID() error = %v", err)
			}
			if a.Period != tc.want {
				t.Errorf("period = %q, want %q", a.Period, tc.want)
			}
		})
	}
}
