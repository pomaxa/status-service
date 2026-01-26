package sqlite

import (
	"context"
	"database/sql"
	"status-incident/internal/domain"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with all tables
func setupTestDB(t *testing.T) *DB {
	sqlDB, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	db := &DB{DB: sqlDB}

	// Apply all migrations
	for _, m := range migrations {
		if _, err := db.Exec(m.SQL); err != nil {
			t.Fatalf("failed to apply migration %d: %v", m.Version, err)
		}
	}

	return db
}

// TestOverallVsPerSystemCorrelation verifies that overall analytics
// correlates with per-system analytics (average of all systems).
// This test demonstrates a bug where overall uptime calculation
// differs from the average of individual system uptimes.
func TestOverallVsPerSystemCorrelation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create two systems
	_, err := db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (1, 'System A', 'Test A', 'http://a.test', 'green')")
	if err != nil {
		t.Fatalf("failed to create system A: %v", err)
	}
	_, err = db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (2, 'System B', 'Test B', 'http://b.test', 'green')")
	if err != nil {
		t.Fatalf("failed to create system B: %v", err)
	}

	// Time setup: 24-hour period
	now := time.Now()
	periodStart := now.Add(-24 * time.Hour)
	periodEnd := now

	// Create status logs that show incidents at different times
	// System A: incident from hour 2 to hour 4 (2 hours down out of 24 = 91.67% uptime)
	// System B: incident from hour 10 to hour 14 (4 hours down out of 24 = 83.33% uptime)
	// Expected overall: average = (91.67 + 83.33) / 2 = 87.5% uptime
	// Or: total down time = 6 hours out of 48 system-hours = 87.5%

	logRepo := NewLogRepo(db)

	// System A logs
	sysAID := int64(1)
	// Start of incident at hour 2
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysAID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "System A down",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(2 * time.Hour),
	})
	// End of incident at hour 4
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysAID,
		OldStatus: domain.StatusRed,
		NewStatus: domain.StatusGreen,
		Message:   "System A recovered",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(4 * time.Hour),
	})

	// System B logs
	sysBID := int64(2)
	// Start of incident at hour 10
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysBID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "System B down",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(10 * time.Hour),
	})
	// End of incident at hour 14
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysBID,
		OldStatus: domain.StatusRed,
		NewStatus: domain.StatusGreen,
		Message:   "System B recovered",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(14 * time.Hour),
	})

	analyticsRepo := NewAnalyticsRepo(db)

	// Get per-system analytics
	analyticsA, err := analyticsRepo.GetUptimeBySystemID(ctx, 1, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("failed to get system A analytics: %v", err)
	}
	analyticsB, err := analyticsRepo.GetUptimeBySystemID(ctx, 2, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("failed to get system B analytics: %v", err)
	}

	// Get overall analytics
	overall, err := analyticsRepo.GetOverallAnalytics(ctx, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("failed to get overall analytics: %v", err)
	}

	// Calculate expected average
	avgUptime := (analyticsA.UptimePercent + analyticsB.UptimePercent) / 2
	avgAvailability := (analyticsA.AvailabilityPercent + analyticsB.AvailabilityPercent) / 2

	t.Logf("System A - Uptime: %.2f%%, Availability: %.2f%%, Incidents: %d",
		analyticsA.UptimePercent, analyticsA.AvailabilityPercent, analyticsA.TotalIncidents)
	t.Logf("System B - Uptime: %.2f%%, Availability: %.2f%%, Incidents: %d",
		analyticsB.UptimePercent, analyticsB.AvailabilityPercent, analyticsB.TotalIncidents)
	t.Logf("Expected Average - Uptime: %.2f%%, Availability: %.2f%%", avgUptime, avgAvailability)
	t.Logf("Overall (actual) - Uptime: %.2f%%, Availability: %.2f%%, Incidents: %d",
		overall.UptimePercent, overall.AvailabilityPercent, overall.TotalIncidents)

	// Check correlation - overall should be close to average of per-system
	const tolerance = 1.0 // 1% tolerance

	uptimeDiff := abs(overall.UptimePercent - avgUptime)
	if uptimeDiff > tolerance {
		t.Errorf("Overall uptime (%.2f%%) does not correlate with average per-system uptime (%.2f%%), diff: %.2f%%",
			overall.UptimePercent, avgUptime, uptimeDiff)
	}

	availabilityDiff := abs(overall.AvailabilityPercent - avgAvailability)
	if availabilityDiff > tolerance {
		t.Errorf("Overall availability (%.2f%%) does not correlate with average per-system availability (%.2f%%), diff: %.2f%%",
			overall.AvailabilityPercent, avgAvailability, availabilityDiff)
	}

	// Total incidents should be sum of per-system incidents
	expectedTotalIncidents := analyticsA.TotalIncidents + analyticsB.TotalIncidents
	if overall.TotalIncidents != expectedTotalIncidents {
		t.Errorf("Overall incidents (%d) should equal sum of per-system incidents (%d)",
			overall.TotalIncidents, expectedTotalIncidents)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestOverallCorrelation_OverlappingIncidents verifies correlation when
// multiple systems have overlapping incident periods.
func TestOverallCorrelation_OverlappingIncidents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create two systems
	db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (1, 'System A', 'Test A', 'http://a.test', 'green')")
	db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (2, 'System B', 'Test B', 'http://b.test', 'green')")

	now := time.Now()
	periodStart := now.Add(-24 * time.Hour)
	periodEnd := now

	logRepo := NewLogRepo(db)

	// System A: incident from hour 2 to hour 8 (6 hours down = 75% uptime)
	sysAID := int64(1)
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysAID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "System A down",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(2 * time.Hour),
	})
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysAID,
		OldStatus: domain.StatusRed,
		NewStatus: domain.StatusGreen,
		Message:   "System A recovered",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(8 * time.Hour),
	})

	// System B: incident from hour 4 to hour 10 (6 hours down = 75% uptime)
	// Overlaps with System A from hour 4 to hour 8
	sysBID := int64(2)
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysBID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "System B down",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(4 * time.Hour),
	})
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysBID,
		OldStatus: domain.StatusRed,
		NewStatus: domain.StatusGreen,
		Message:   "System B recovered",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(10 * time.Hour),
	})

	analyticsRepo := NewAnalyticsRepo(db)

	analyticsA, _ := analyticsRepo.GetUptimeBySystemID(ctx, 1, periodStart, periodEnd)
	analyticsB, _ := analyticsRepo.GetUptimeBySystemID(ctx, 2, periodStart, periodEnd)
	overall, _ := analyticsRepo.GetOverallAnalytics(ctx, periodStart, periodEnd)

	// Both systems have 75% uptime, so overall should also be 75%
	expectedAvg := (analyticsA.UptimePercent + analyticsB.UptimePercent) / 2

	t.Logf("System A: %.2f%%, System B: %.2f%%, Expected: %.2f%%, Overall: %.2f%%",
		analyticsA.UptimePercent, analyticsB.UptimePercent, expectedAvg, overall.UptimePercent)

	if abs(overall.UptimePercent-expectedAvg) > 1.0 {
		t.Errorf("Overall uptime (%.2f%%) should equal average (%.2f%%) for overlapping incidents",
			overall.UptimePercent, expectedAvg)
	}
}

// TestOverallCorrelation_SingleSystem verifies that overall equals
// per-system when there's only one system.
func TestOverallCorrelation_SingleSystem(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create only one system
	db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (1, 'System A', 'Test A', 'http://a.test', 'green')")

	now := time.Now()
	periodStart := now.Add(-24 * time.Hour)
	periodEnd := now

	logRepo := NewLogRepo(db)

	// System A: incident from hour 6 to hour 12 (6 hours down = 75% uptime)
	sysAID := int64(1)
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysAID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "System A down",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(6 * time.Hour),
	})
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysAID,
		OldStatus: domain.StatusRed,
		NewStatus: domain.StatusGreen,
		Message:   "System A recovered",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(12 * time.Hour),
	})

	analyticsRepo := NewAnalyticsRepo(db)

	analyticsA, _ := analyticsRepo.GetUptimeBySystemID(ctx, 1, periodStart, periodEnd)
	overall, _ := analyticsRepo.GetOverallAnalytics(ctx, periodStart, periodEnd)

	t.Logf("System A: %.2f%%, Overall: %.2f%%", analyticsA.UptimePercent, overall.UptimePercent)

	if abs(overall.UptimePercent-analyticsA.UptimePercent) > 0.01 {
		t.Errorf("Overall (%.2f%%) should equal single system (%.2f%%)",
			overall.UptimePercent, analyticsA.UptimePercent)
	}

	if overall.TotalIncidents != analyticsA.TotalIncidents {
		t.Errorf("Overall incidents (%d) should equal single system incidents (%d)",
			overall.TotalIncidents, analyticsA.TotalIncidents)
	}
}

// TestOverallCorrelation_NoIncidents verifies 100% uptime when no incidents.
func TestOverallCorrelation_NoIncidents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create systems without any incidents
	db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (1, 'System A', 'Test A', 'http://a.test', 'green')")
	db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (2, 'System B', 'Test B', 'http://b.test', 'green')")

	now := time.Now()
	periodStart := now.Add(-24 * time.Hour)
	periodEnd := now

	analyticsRepo := NewAnalyticsRepo(db)

	analyticsA, _ := analyticsRepo.GetUptimeBySystemID(ctx, 1, periodStart, periodEnd)
	analyticsB, _ := analyticsRepo.GetUptimeBySystemID(ctx, 2, periodStart, periodEnd)
	overall, _ := analyticsRepo.GetOverallAnalytics(ctx, periodStart, periodEnd)

	// All should be 100%
	if analyticsA.UptimePercent != 100 {
		t.Errorf("System A uptime should be 100%%, got %.2f%%", analyticsA.UptimePercent)
	}
	if analyticsB.UptimePercent != 100 {
		t.Errorf("System B uptime should be 100%%, got %.2f%%", analyticsB.UptimePercent)
	}
	if overall.UptimePercent != 100 {
		t.Errorf("Overall uptime should be 100%%, got %.2f%%", overall.UptimePercent)
	}
	if overall.TotalIncidents != 0 {
		t.Errorf("Overall incidents should be 0, got %d", overall.TotalIncidents)
	}
}

// TestOverallCorrelation_NoSystems verifies behavior with no systems.
func TestOverallCorrelation_NoSystems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	now := time.Now()
	periodStart := now.Add(-24 * time.Hour)
	periodEnd := now

	analyticsRepo := NewAnalyticsRepo(db)

	overall, err := analyticsRepo.GetOverallAnalytics(ctx, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return 100% when no systems exist
	if overall.UptimePercent != 100 {
		t.Errorf("Overall uptime should be 100%% with no systems, got %.2f%%", overall.UptimePercent)
	}
	if overall.AvailabilityPercent != 100 {
		t.Errorf("Overall availability should be 100%% with no systems, got %.2f%%", overall.AvailabilityPercent)
	}
}

// TestOverallCorrelation_YellowVsRed verifies different handling of
// yellow (degraded) vs red (down) status for uptime vs availability.
func TestOverallCorrelation_YellowVsRed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create two systems
	db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (1, 'System A', 'Test A', 'http://a.test', 'green')")
	db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (2, 'System B', 'Test B', 'http://b.test', 'green')")

	now := time.Now()
	periodStart := now.Add(-24 * time.Hour)
	periodEnd := now

	logRepo := NewLogRepo(db)

	// System A: yellow (degraded) for 6 hours
	// Uptime = 75% (not green), Availability = 100% (not red)
	sysAID := int64(1)
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysAID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusYellow,
		Message:   "System A degraded",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(6 * time.Hour),
	})
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysAID,
		OldStatus: domain.StatusYellow,
		NewStatus: domain.StatusGreen,
		Message:   "System A recovered",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(12 * time.Hour),
	})

	// System B: red (down) for 6 hours
	// Uptime = 75% (not green), Availability = 75% (not red)
	sysBID := int64(2)
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysBID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "System B down",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(6 * time.Hour),
	})
	logRepo.Create(ctx, &domain.StatusLog{
		SystemID:  &sysBID,
		OldStatus: domain.StatusRed,
		NewStatus: domain.StatusGreen,
		Message:   "System B recovered",
		Source:    domain.SourceManual,
		CreatedAt: periodStart.Add(12 * time.Hour),
	})

	analyticsRepo := NewAnalyticsRepo(db)

	analyticsA, _ := analyticsRepo.GetUptimeBySystemID(ctx, 1, periodStart, periodEnd)
	analyticsB, _ := analyticsRepo.GetUptimeBySystemID(ctx, 2, periodStart, periodEnd)
	overall, _ := analyticsRepo.GetOverallAnalytics(ctx, periodStart, periodEnd)

	t.Logf("System A (yellow) - Uptime: %.2f%%, Availability: %.2f%%",
		analyticsA.UptimePercent, analyticsA.AvailabilityPercent)
	t.Logf("System B (red) - Uptime: %.2f%%, Availability: %.2f%%",
		analyticsB.UptimePercent, analyticsB.AvailabilityPercent)
	t.Logf("Overall - Uptime: %.2f%%, Availability: %.2f%%",
		overall.UptimePercent, overall.AvailabilityPercent)

	// Both have same uptime (75%) because both were not green
	expectedUptime := (analyticsA.UptimePercent + analyticsB.UptimePercent) / 2
	if abs(overall.UptimePercent-expectedUptime) > 1.0 {
		t.Errorf("Overall uptime (%.2f%%) should be average (%.2f%%)",
			overall.UptimePercent, expectedUptime)
	}

	// Availability differs: A=100% (yellow is available), B=75% (red is unavailable)
	expectedAvailability := (analyticsA.AvailabilityPercent + analyticsB.AvailabilityPercent) / 2
	if abs(overall.AvailabilityPercent-expectedAvailability) > 1.0 {
		t.Errorf("Overall availability (%.2f%%) should be average (%.2f%%)",
			overall.AvailabilityPercent, expectedAvailability)
	}

	// System A should have higher availability than uptime
	if analyticsA.AvailabilityPercent <= analyticsA.UptimePercent {
		t.Errorf("System A availability (%.2f%%) should be > uptime (%.2f%%) for yellow status",
			analyticsA.AvailabilityPercent, analyticsA.UptimePercent)
	}
}

// TestOverallCorrelation_ManySystemsWithVariedDowntime tests correlation
// with multiple systems having different downtime durations.
func TestOverallCorrelation_ManySystemsWithVariedDowntime(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create 5 systems
	for i := 1; i <= 5; i++ {
		db.ExecContext(ctx, "INSERT INTO systems (id, name, description, url, status) VALUES (?, ?, '', '', 'green')",
			i, "System "+string(rune('A'+i-1)))
	}

	now := time.Now()
	periodStart := now.Add(-24 * time.Hour)
	periodEnd := now

	logRepo := NewLogRepo(db)

	// Create incidents with different durations:
	// System A: 1 hour down (95.83% uptime)
	// System B: 2 hours down (91.67% uptime)
	// System C: 3 hours down (87.50% uptime)
	// System D: 4 hours down (83.33% uptime)
	// System E: 5 hours down (79.17% uptime)
	downtimes := []int{1, 2, 3, 4, 5}
	for i, hours := range downtimes {
		sysID := int64(i + 1)
		logRepo.Create(ctx, &domain.StatusLog{
			SystemID:  &sysID,
			OldStatus: domain.StatusGreen,
			NewStatus: domain.StatusRed,
			Message:   "Down",
			Source:    domain.SourceManual,
			CreatedAt: periodStart.Add(time.Duration(i) * time.Hour),
		})
		logRepo.Create(ctx, &domain.StatusLog{
			SystemID:  &sysID,
			OldStatus: domain.StatusRed,
			NewStatus: domain.StatusGreen,
			Message:   "Recovered",
			Source:    domain.SourceManual,
			CreatedAt: periodStart.Add(time.Duration(i+hours) * time.Hour),
		})
	}

	analyticsRepo := NewAnalyticsRepo(db)

	// Get per-system analytics and calculate average
	var totalUptime float64
	for i := 1; i <= 5; i++ {
		analytics, _ := analyticsRepo.GetUptimeBySystemID(ctx, int64(i), periodStart, periodEnd)
		totalUptime += analytics.UptimePercent
		t.Logf("System %d: %.2f%% uptime", i, analytics.UptimePercent)
	}
	expectedAvg := totalUptime / 5

	overall, _ := analyticsRepo.GetOverallAnalytics(ctx, periodStart, periodEnd)

	t.Logf("Expected average: %.2f%%, Overall: %.2f%%", expectedAvg, overall.UptimePercent)

	if abs(overall.UptimePercent-expectedAvg) > 1.0 {
		t.Errorf("Overall uptime (%.2f%%) should equal average (%.2f%%)",
			overall.UptimePercent, expectedAvg)
	}

	if overall.TotalIncidents != 5 {
		t.Errorf("Overall should have 5 incidents, got %d", overall.TotalIncidents)
	}
}
