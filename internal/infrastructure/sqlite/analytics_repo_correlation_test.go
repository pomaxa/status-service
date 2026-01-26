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
