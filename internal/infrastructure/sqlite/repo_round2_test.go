package sqlite

import (
	"context"
	"testing"
	"time"
)

// This file squeezes the remaining reachable branches that the first pass left
// open in the plural scan* loops and a couple of getters.
//
// Scan-error technique: store a BLOB (x'..') in a DATETIME column. go-sqlite3
// stores it verbatim and the subsequent Scan into a time.Time destination fails
// with "unsupported Scan, storing driver.Value type []uint8 into type
// *time.Time", which exercises the rows.Scan error branch. This is more robust
// than non-datetime text (which go-sqlite3 silently accepts) and does not depend
// on a nullable column being present in the table.

// ---------- IncidentRepo: scanIncidents ----------

// TestIncidentRepo_GetAll_ScanError drives the rows.Scan error branch of
// scanIncidents (the plural loop used by GetAll/GetActive/GetRecent).
func TestIncidentRepo_GetAll_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO incidents (id, title, status, severity, system_ids, message, created_at, updated_at)
		VALUES (1, 't', 'investigating', 'minor', '[]', 'm', x'deadbeef', ?)
	`, time.Now()); err != nil {
		t.Fatalf("insert incident: %v", err)
	}

	repo := NewIncidentRepo(db)
	if _, err := repo.GetAll(ctx, 0); err == nil {
		t.Fatal("expected scan error in scanIncidents from blob created_at")
	}
}

// TestIncidentRepo_GetAll_WithSystemIDs covers the systemIDsJSON != "" unmarshal
// branch inside scanIncidents (the plural loop). The existing GetAll tests use
// repo.Create, which always marshals nil SystemIDs to the literal "null", so the
// non-empty unmarshal branch is never reached there.
func TestIncidentRepo_GetAll_WithSystemIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO incidents (id, title, status, severity, system_ids, message, created_at, updated_at)
		VALUES (1, 't', 'investigating', 'minor', '[7,8,9]', 'm', ?, ?)
	`, time.Now(), time.Now()); err != nil {
		t.Fatalf("insert incident: %v", err)
	}

	repo := NewIncidentRepo(db)
	all, err := repo.GetAll(ctx, 0)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(all))
	}
	if len(all[0].SystemIDs) != 3 {
		t.Errorf("expected 3 system IDs parsed in scanIncidents, got %v", all[0].SystemIDs)
	}
}

// ---------- IncidentRepo: GetUpdates ----------

// TestIncidentRepo_GetUpdates_ScanError drives the rows.Scan error branch in
// GetUpdates by storing an incident_update whose created_at is a BLOB.
func TestIncidentRepo_GetUpdates_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO incidents (id, title, status, severity, message, created_at, updated_at)
		VALUES (1, 't', 'investigating', 'minor', 'm', ?, ?)
	`, time.Now(), time.Now()); err != nil {
		t.Fatalf("insert incident: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO incident_updates (id, incident_id, status, message, created_at, created_by)
		VALUES (1, 1, 'identified', 'm', x'deadbeef', 'admin')
	`); err != nil {
		t.Fatalf("insert incident_update: %v", err)
	}

	repo := NewIncidentRepo(db)
	if _, err := repo.GetUpdates(ctx, 1); err == nil {
		t.Fatal("expected scan error in GetUpdates from blob created_at")
	}
}

// ---------- MaintenanceRepo: scanMaintenances ----------

// TestMaintenanceRepo_GetAll_ScanError drives the rows.Scan error branch of
// scanMaintenances via a row with a BLOB start_time.
func TestMaintenanceRepo_GetAll_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO maintenances (id, title, description, start_time, end_time, system_ids, status, created_at, updated_at)
		VALUES (1, 't', '', x'deadbeef', ?, '[]', 'scheduled', ?, ?)
	`, time.Now(), time.Now(), time.Now()); err != nil {
		t.Fatalf("insert maintenance: %v", err)
	}

	repo := NewMaintenanceRepo(db)
	if _, err := repo.GetAll(ctx); err == nil {
		t.Fatal("expected scan error in scanMaintenances from blob start_time")
	}
}

// TestMaintenanceRepo_GetAll_WithSystemIDs covers the systemIDsJSON unmarshal
// branch inside scanMaintenances (the plural loop).
func TestMaintenanceRepo_GetAll_WithSystemIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO maintenances (id, title, description, start_time, end_time, system_ids, status, created_at, updated_at)
		VALUES (1, 't', '', ?, ?, '[4,5]', 'scheduled', ?, ?)
	`, time.Now(), time.Now().Add(time.Hour), time.Now(), time.Now()); err != nil {
		t.Fatalf("insert maintenance: %v", err)
	}

	repo := NewMaintenanceRepo(db)
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 maintenance, got %d", len(all))
	}
	if len(all[0].SystemIDs) != 2 {
		t.Errorf("expected 2 system IDs parsed in scanMaintenances, got %v", all[0].SystemIDs)
	}
}

// TestMaintenanceRepo_GetActive_ScanError drives the scanMaintenances error
// propagation path in GetActive: a row matching the active window but whose
// created_at is a BLOB makes scanMaintenances fail, so GetActive returns the
// error before the RefreshStatus loop. created_at is not part of the WHERE
// clause, so the row still matches the active-window filter.
func TestMaintenanceRepo_GetActive_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO maintenances (id, title, description, start_time, end_time, system_ids, status, created_at, updated_at)
		VALUES (1, 't', '', ?, ?, '[]', 'in_progress', x'deadbeef', ?)
	`, start, end, time.Now()); err != nil {
		t.Fatalf("insert maintenance: %v", err)
	}

	repo := NewMaintenanceRepo(db)
	if _, err := repo.GetActive(ctx); err == nil {
		t.Fatal("expected scan error propagated from scanMaintenances in GetActive")
	}
}

// ---------- LogRepo: scanLogs ----------

// TestLogRepo_GetAll_ScanError drives the rows.Scan error branch of scanLogs
// via a row whose created_at is a BLOB.
func TestLogRepo_GetAll_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO status_log (id, system_id, old_status, new_status, message, source, created_at)
		VALUES (1, NULL, 'green', 'red', 'm', 'manual', x'deadbeef')
	`); err != nil {
		t.Fatalf("insert status_log: %v", err)
	}

	repo := NewLogRepo(db)
	if _, err := repo.GetAll(ctx, 10); err == nil {
		t.Fatal("expected scan error in scanLogs from blob created_at")
	}
}

// ---------- LatencyRepo: GetDailyUptime per-row scan error ----------

// TestLatencyRepo_GetDailyUptime_ScanError drives the rows.Scan error branch in
// the GetDailyUptime aggregation loop. A BLOB created_at makes SQLite's
// date(created_at) return NULL for that group, and scanning NULL into the plain
// `string` day destination fails inside the loop.
func TestLatencyRepo_GetDailyUptime_ScanError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "INSERT INTO systems (id, name) VALUES (1, 'Sys')"); err != nil {
		t.Fatalf("insert system: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO dependencies (id, system_id, name, heartbeat_method) VALUES (1, 1, 'd', 'GET')"); err != nil {
		t.Fatalf("insert dependency: %v", err)
	}
	// created_at >= startDate filter: a BLOB compares greater than the datetime
	// startDate string in SQLite's type ordering, so the row is included and the
	// date() group key becomes NULL.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO latency_history (id, dependency_id, latency_ms, success, status_code, created_at)
		VALUES (1, 1, 10, 1, 200, x'deadbeef')
	`); err != nil {
		t.Fatalf("insert latency: %v", err)
	}

	repo := NewLatencyRepo(db)
	if _, err := repo.GetDailyUptime(ctx, 1, 7); err == nil {
		t.Fatal("expected scan error in GetDailyUptime from NULL date group")
	}
}
