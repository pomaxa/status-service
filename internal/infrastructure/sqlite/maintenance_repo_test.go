package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

func TestMaintenanceRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	maintenance, err := domain.NewMaintenance("Scheduled Maintenance", "Database upgrade", start, end)
	if err != nil {
		t.Fatalf("failed to create maintenance: %v", err)
	}

	err = repo.Create(ctx, maintenance)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if maintenance.ID == 0 {
		t.Error("expected maintenance ID to be set after Create()")
	}
}

func TestMaintenanceRepo_Create_WithSystemIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create some systems
	sysRepo := NewSystemRepo(db)
	sys1, _ := domain.NewSystem("System 1", "", "", "")
	sys2, _ := domain.NewSystem("System 2", "", "", "")
	sysRepo.Create(context.Background(), sys1)
	sysRepo.Create(context.Background(), sys2)

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	maintenance, _ := domain.NewMaintenance("Targeted Maintenance", "Affecting specific systems", start, end)
	maintenance.SetSystemIDs([]int64{sys1.ID, sys2.ID})

	if err := repo.Create(ctx, maintenance); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, maintenance.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(retrieved.SystemIDs) != 2 {
		t.Errorf("SystemIDs count = %d, want 2", len(retrieved.SystemIDs))
	}
}

func TestMaintenanceRepo_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	maintenance, _ := domain.NewMaintenance("Test Maintenance", "Test description", start, end)
	if err := repo.Create(ctx, maintenance); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, maintenance.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByID() returned nil")
	}

	if retrieved.ID != maintenance.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, maintenance.ID)
	}
	if retrieved.Title != maintenance.Title {
		t.Errorf("Title = %s, want %s", retrieved.Title, maintenance.Title)
	}
	if retrieved.Description != maintenance.Description {
		t.Errorf("Description = %s, want %s", retrieved.Description, maintenance.Description)
	}
}

func TestMaintenanceRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestMaintenanceRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	// Create multiple maintenances
	for i := 0; i < 3; i++ {
		start := time.Now().Add(time.Duration(i+1) * time.Hour)
		end := start.Add(time.Hour)
		m, _ := domain.NewMaintenance("Maintenance "+string(rune('A'+i)), "Description", start, end)
		repo.Create(ctx, m)
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(all) != 3 {
		t.Errorf("GetAll() returned %d maintenances, want 3", len(all))
	}
}

func TestMaintenanceRepo_GetActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	// Create active maintenance (happening now)
	activeStart := time.Now().Add(-30 * time.Minute)
	activeEnd := time.Now().Add(30 * time.Minute)
	active, _ := domain.NewMaintenance("Active Maintenance", "Currently active", activeStart, activeEnd)

	// Create scheduled maintenance (in future)
	scheduledStart := time.Now().Add(2 * time.Hour)
	scheduledEnd := scheduledStart.Add(time.Hour)
	scheduled, _ := domain.NewMaintenance("Scheduled Maintenance", "Future", scheduledStart, scheduledEnd)

	// Create completed maintenance (in past)
	completedStart := time.Now().Add(-3 * time.Hour)
	completedEnd := time.Now().Add(-2 * time.Hour)
	completed, _ := domain.NewMaintenance("Completed Maintenance", "Past", completedStart, completedEnd)

	for _, m := range []*domain.Maintenance{active, scheduled, completed} {
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get active
	actives, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}

	if len(actives) != 1 {
		t.Errorf("GetActive() returned %d maintenances, want 1", len(actives))
	}

	if actives[0].Title != "Active Maintenance" {
		t.Errorf("active maintenance title = %s, want Active Maintenance", actives[0].Title)
	}
}

func TestMaintenanceRepo_GetUpcoming(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	// Create scheduled maintenance (in future)
	start1 := time.Now().Add(1 * time.Hour)
	end1 := start1.Add(time.Hour)
	scheduled1, _ := domain.NewMaintenance("Upcoming 1", "First upcoming", start1, end1)

	start2 := time.Now().Add(3 * time.Hour)
	end2 := start2.Add(time.Hour)
	scheduled2, _ := domain.NewMaintenance("Upcoming 2", "Second upcoming", start2, end2)

	// Create active maintenance (happening now)
	activeStart := time.Now().Add(-30 * time.Minute)
	activeEnd := time.Now().Add(30 * time.Minute)
	active, _ := domain.NewMaintenance("Active", "Currently active", activeStart, activeEnd)

	for _, m := range []*domain.Maintenance{scheduled1, scheduled2, active} {
		repo.Create(ctx, m)
	}

	// Get upcoming
	upcoming, err := repo.GetUpcoming(ctx)
	if err != nil {
		t.Fatalf("GetUpcoming() error = %v", err)
	}

	if len(upcoming) != 2 {
		t.Errorf("GetUpcoming() returned %d maintenances, want 2", len(upcoming))
	}

	// Should be ordered by start_time ASC
	if upcoming[0].Title != "Upcoming 1" {
		t.Errorf("first upcoming title = %s, want Upcoming 1", upcoming[0].Title)
	}
}

func TestMaintenanceRepo_GetByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	// Create maintenances at different times
	now := time.Now()

	// Maintenance within range
	m1Start := now.Add(1 * time.Hour)
	m1End := now.Add(2 * time.Hour)
	m1, _ := domain.NewMaintenance("In Range", "Within time range", m1Start, m1End)

	// Maintenance outside range (too early)
	m2Start := now.Add(-5 * time.Hour)
	m2End := now.Add(-4 * time.Hour)
	m2, _ := domain.NewMaintenance("Before Range", "Before time range", m2Start, m2End)

	// Maintenance outside range (too late)
	m3Start := now.Add(10 * time.Hour)
	m3End := now.Add(11 * time.Hour)
	m3, _ := domain.NewMaintenance("After Range", "After time range", m3Start, m3End)

	for _, m := range []*domain.Maintenance{m1, m2, m3} {
		repo.Create(ctx, m)
	}

	// Query for overlapping maintenances
	queryStart := now
	queryEnd := now.Add(3 * time.Hour)

	results, err := repo.GetByTimeRange(ctx, queryStart, queryEnd)
	if err != nil {
		t.Fatalf("GetByTimeRange() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("GetByTimeRange() returned %d maintenances, want 1", len(results))
	}

	if len(results) > 0 && results[0].Title != "In Range" {
		t.Errorf("maintenance title = %s, want In Range", results[0].Title)
	}
}

func TestMaintenanceRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	start := time.Now().Add(time.Hour)
	end := start.Add(2 * time.Hour)

	maintenance, _ := domain.NewMaintenance("Original Title", "Original Description", start, end)
	if err := repo.Create(ctx, maintenance); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	newStart := start.Add(time.Hour)
	newEnd := newStart.Add(3 * time.Hour)
	maintenance.Update("Updated Title", "Updated Description", newStart, newEnd)

	if err := repo.Update(ctx, maintenance); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, err := repo.GetByID(ctx, maintenance.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Title = %s, want Updated Title", retrieved.Title)
	}
	if retrieved.Description != "Updated Description" {
		t.Errorf("Description = %s, want Updated Description", retrieved.Description)
	}
}

func TestMaintenanceRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)

	maintenance, _ := domain.NewMaintenance("To Delete", "Will be deleted", start, end)
	if err := repo.Create(ctx, maintenance); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, maintenance.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, maintenance.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil after deletion")
	}
}

func TestMaintenanceRepo_StatusRefresh(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	// Create active maintenance
	activeStart := time.Now().Add(-30 * time.Minute)
	activeEnd := time.Now().Add(30 * time.Minute)
	maintenance, _ := domain.NewMaintenance("Active Test", "Testing status refresh", activeStart, activeEnd)

	if err := repo.Create(ctx, maintenance); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// GetByID should refresh status
	retrieved, err := repo.GetByID(ctx, maintenance.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Status != domain.MaintenanceInProgress {
		t.Errorf("Status = %s, want in_progress", retrieved.Status)
	}
}

func TestMaintenanceRepo_CancelledNotInActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	// Create maintenance that would be active, but cancelled
	activeStart := time.Now().Add(-30 * time.Minute)
	activeEnd := time.Now().Add(30 * time.Minute)
	maintenance, _ := domain.NewMaintenance("Cancelled", "Was cancelled", activeStart, activeEnd)
	maintenance.Cancel()

	if err := repo.Create(ctx, maintenance); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// GetActive should not include cancelled
	actives, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}

	for _, a := range actives {
		if a.ID == maintenance.ID {
			t.Error("cancelled maintenance should not appear in GetActive()")
		}
	}
}

func TestMaintenanceRepo_StatusTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMaintenanceRepo(db)
	ctx := context.Background()

	testCases := []struct {
		name   string
		start  time.Time
		end    time.Time
		cancel bool
		expect domain.MaintenanceStatus
	}{
		{
			name:   "scheduled",
			start:  time.Now().Add(2 * time.Hour),
			end:    time.Now().Add(3 * time.Hour),
			cancel: false,
			expect: domain.MaintenanceScheduled,
		},
		{
			name:   "in_progress",
			start:  time.Now().Add(-30 * time.Minute),
			end:    time.Now().Add(30 * time.Minute),
			cancel: false,
			expect: domain.MaintenanceInProgress,
		},
		{
			name:   "completed",
			start:  time.Now().Add(-3 * time.Hour),
			end:    time.Now().Add(-2 * time.Hour),
			cancel: false,
			expect: domain.MaintenanceCompleted,
		},
		{
			name:   "cancelled",
			start:  time.Now().Add(time.Hour),
			end:    time.Now().Add(2 * time.Hour),
			cancel: true,
			expect: domain.MaintenanceCancelled,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := domain.NewMaintenance("Status Test "+tc.name, "", tc.start, tc.end)
			if tc.cancel {
				m.Cancel()
			}

			if err := repo.Create(ctx, m); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			retrieved, err := repo.GetByID(ctx, m.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if retrieved.Status != tc.expect {
				t.Errorf("Status = %s, want %s", retrieved.Status, tc.expect)
			}
		})
	}
}
