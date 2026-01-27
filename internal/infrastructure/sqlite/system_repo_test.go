package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

func TestSystemRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	system, err := domain.NewSystem("Test System", "A test system", "https://example.com", "team@example.com")
	if err != nil {
		t.Fatalf("failed to create system: %v", err)
	}

	err = repo.Create(ctx, system)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if system.ID == 0 {
		t.Error("expected system ID to be set after Create()")
	}
}

func TestSystemRepo_Create_MultipleSystems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	system1, _ := domain.NewSystem("System 1", "First system", "", "")
	system2, _ := domain.NewSystem("System 2", "Second system", "", "")

	if err := repo.Create(ctx, system1); err != nil {
		t.Fatalf("Create() system1 error = %v", err)
	}
	if err := repo.Create(ctx, system2); err != nil {
		t.Fatalf("Create() system2 error = %v", err)
	}

	if system1.ID == system2.ID {
		t.Error("expected different IDs for different systems")
	}
}

func TestSystemRepo_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	// Create a system first
	system, _ := domain.NewSystem("Test System", "A test system", "https://example.com", "team@example.com")
	system.SLATarget = 99.5
	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Retrieve it
	retrieved, err := repo.GetByID(ctx, system.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByID() returned nil")
	}

	if retrieved.ID != system.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, system.ID)
	}
	if retrieved.Name != system.Name {
		t.Errorf("Name = %s, want %s", retrieved.Name, system.Name)
	}
	if retrieved.Description != system.Description {
		t.Errorf("Description = %s, want %s", retrieved.Description, system.Description)
	}
	if retrieved.URL != system.URL {
		t.Errorf("URL = %s, want %s", retrieved.URL, system.URL)
	}
	if retrieved.Owner != system.Owner {
		t.Errorf("Owner = %s, want %s", retrieved.Owner, system.Owner)
	}
	if retrieved.Status != system.Status {
		t.Errorf("Status = %s, want %s", retrieved.Status, system.Status)
	}
	if retrieved.SLATarget != system.SLATarget {
		t.Errorf("SLATarget = %f, want %f", retrieved.SLATarget, system.SLATarget)
	}
}

func TestSystemRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestSystemRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	// Create multiple systems
	systems := []struct {
		name string
		desc string
	}{
		{"Alpha System", "First"},
		{"Beta System", "Second"},
		{"Gamma System", "Third"},
	}

	for _, s := range systems {
		system, _ := domain.NewSystem(s.name, s.desc, "", "")
		if err := repo.Create(ctx, system); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get all
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(all) != len(systems) {
		t.Errorf("GetAll() returned %d systems, want %d", len(all), len(systems))
	}

	// Should be ordered by name ASC
	if all[0].Name != "Alpha System" {
		t.Errorf("first system name = %s, want Alpha System", all[0].Name)
	}
	if all[1].Name != "Beta System" {
		t.Errorf("second system name = %s, want Beta System", all[1].Name)
	}
	if all[2].Name != "Gamma System" {
		t.Errorf("third system name = %s, want Gamma System", all[2].Name)
	}
}

func TestSystemRepo_GetAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if all == nil {
		// nil is acceptable for empty results
		return
	}

	if len(all) != 0 {
		t.Errorf("GetAll() returned %d systems, want 0", len(all))
	}
}

func TestSystemRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	// Create a system
	system, _ := domain.NewSystem("Original Name", "Original Description", "https://old.com", "old@example.com")
	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update it
	system.Name = "Updated Name"
	system.Description = "Updated Description"
	system.URL = "https://new.com"
	system.Owner = "new@example.com"
	system.Status = domain.StatusYellow
	system.SLATarget = 99.0
	system.UpdatedAt = time.Now()

	if err := repo.Update(ctx, system); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(ctx, system.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %s, want Updated Name", retrieved.Name)
	}
	if retrieved.Description != "Updated Description" {
		t.Errorf("Description = %s, want Updated Description", retrieved.Description)
	}
	if retrieved.URL != "https://new.com" {
		t.Errorf("URL = %s, want https://new.com", retrieved.URL)
	}
	if retrieved.Owner != "new@example.com" {
		t.Errorf("Owner = %s, want new@example.com", retrieved.Owner)
	}
	if retrieved.Status != domain.StatusYellow {
		t.Errorf("Status = %s, want yellow", retrieved.Status)
	}
	if retrieved.SLATarget != 99.0 {
		t.Errorf("SLATarget = %f, want 99.0", retrieved.SLATarget)
	}
}

func TestSystemRepo_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	system := &domain.System{
		ID:        99999,
		Name:      "Non-existent",
		UpdatedAt: time.Now(),
	}

	err := repo.Update(ctx, system)
	if err == nil {
		t.Error("expected error when updating non-existent system")
	}
}

func TestSystemRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	// Create a system
	system, _ := domain.NewSystem("To Delete", "Will be deleted", "", "")
	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete it
	if err := repo.Delete(ctx, system.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	retrieved, err := repo.GetByID(ctx, system.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil after deletion")
	}
}

func TestSystemRepo_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error when deleting non-existent system")
	}
}

func TestSystemRepo_StatusPersistence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSystemRepo(db)
	ctx := context.Background()

	statuses := []domain.Status{domain.StatusGreen, domain.StatusYellow, domain.StatusRed}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			system, _ := domain.NewSystem("Status Test "+string(status), "", "", "")
			system.Status = status

			if err := repo.Create(ctx, system); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			retrieved, err := repo.GetByID(ctx, system.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if retrieved.Status != status {
				t.Errorf("Status = %s, want %s", retrieved.Status, status)
			}
		})
	}
}
