package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

func createTestSystem(t *testing.T, db *DB) *domain.System {
	t.Helper()
	repo := NewSystemRepo(db)
	ctx := context.Background()

	system, err := domain.NewSystem("Test System", "For dependency tests", "", "")
	if err != nil {
		t.Fatalf("failed to create test system: %v", err)
	}

	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("failed to persist test system: %v", err)
	}

	return system
}

func TestDependencyRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	dep, err := domain.NewDependency(system.ID, "Test Dependency", "A test dependency")
	if err != nil {
		t.Fatalf("failed to create dependency: %v", err)
	}
	dep.HeartbeatMethod = "GET" // Required by schema

	err = repo.Create(ctx, dep)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if dep.ID == 0 {
		t.Error("expected dependency ID to be set after Create()")
	}
}

func TestDependencyRepo_Create_WithHeartbeat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	dep, _ := domain.NewDependency(system.ID, "API Service", "External API")
	err := dep.SetHeartbeatConfig(domain.HeartbeatConfig{
		URL:          "https://api.example.com/health",
		Interval:     60,
		Method:       "GET",
		Headers:      map[string]string{"Authorization": "Bearer token123"},
		ExpectStatus: "200",
		ExpectBody:   "ok",
	})
	if err != nil {
		t.Fatalf("SetHeartbeatConfig() error = %v", err)
	}

	if err := repo.Create(ctx, dep); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, dep.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.HeartbeatURL != "https://api.example.com/health" {
		t.Errorf("HeartbeatURL = %s, want https://api.example.com/health", retrieved.HeartbeatURL)
	}
	if retrieved.HeartbeatInterval != 60 {
		t.Errorf("HeartbeatInterval = %d, want 60", retrieved.HeartbeatInterval)
	}
	if retrieved.HeartbeatMethod != "GET" {
		t.Errorf("HeartbeatMethod = %s, want GET", retrieved.HeartbeatMethod)
	}
	if retrieved.HeartbeatHeaders["Authorization"] != "Bearer token123" {
		t.Errorf("HeartbeatHeaders[Authorization] = %s, want Bearer token123", retrieved.HeartbeatHeaders["Authorization"])
	}
	if retrieved.HeartbeatExpectStatus != "200" {
		t.Errorf("HeartbeatExpectStatus = %s, want 200", retrieved.HeartbeatExpectStatus)
	}
	if retrieved.HeartbeatExpectBody != "ok" {
		t.Errorf("HeartbeatExpectBody = %s, want ok", retrieved.HeartbeatExpectBody)
	}
}

func TestDependencyRepo_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	dep, _ := domain.NewDependency(system.ID, "Test Dependency", "Description")
	dep.HeartbeatMethod = "GET" // Required by schema
	if err := repo.Create(ctx, dep); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, dep.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByID() returned nil")
	}

	if retrieved.ID != dep.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, dep.ID)
	}
	if retrieved.SystemID != system.ID {
		t.Errorf("SystemID = %d, want %d", retrieved.SystemID, system.ID)
	}
	if retrieved.Name != dep.Name {
		t.Errorf("Name = %s, want %s", retrieved.Name, dep.Name)
	}
	if retrieved.Description != dep.Description {
		t.Errorf("Description = %s, want %s", retrieved.Description, dep.Description)
	}
}

func TestDependencyRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewDependencyRepo(db)
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestDependencyRepo_GetBySystemID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system1 := createTestSystem(t, db)

	// Create second system
	sysRepo := NewSystemRepo(db)
	system2, _ := domain.NewSystem("Second System", "", "", "")
	if err := sysRepo.Create(context.Background(), system2); err != nil {
		t.Fatalf("failed to create second system: %v", err)
	}

	repo := NewDependencyRepo(db)
	ctx := context.Background()

	// Create dependencies for system1
	dep1, _ := domain.NewDependency(system1.ID, "Dep Alpha", "")
	dep1.HeartbeatMethod = "GET"
	dep2, _ := domain.NewDependency(system1.ID, "Dep Beta", "")
	dep2.HeartbeatMethod = "GET"
	// Create dependency for system2
	dep3, _ := domain.NewDependency(system2.ID, "Dep Gamma", "")
	dep3.HeartbeatMethod = "GET"

	for _, dep := range []*domain.Dependency{dep1, dep2, dep3} {
		if err := repo.Create(ctx, dep); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get dependencies for system1
	deps, err := repo.GetBySystemID(ctx, system1.ID)
	if err != nil {
		t.Fatalf("GetBySystemID() error = %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("GetBySystemID() returned %d dependencies, want 2", len(deps))
	}

	// Should be ordered by name ASC
	if deps[0].Name != "Dep Alpha" {
		t.Errorf("first dependency name = %s, want Dep Alpha", deps[0].Name)
	}
	if deps[1].Name != "Dep Beta" {
		t.Errorf("second dependency name = %s, want Dep Beta", deps[1].Name)
	}
}

func TestDependencyRepo_GetBySystemID_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	deps, err := repo.GetBySystemID(ctx, system.ID)
	if err != nil {
		t.Fatalf("GetBySystemID() error = %v", err)
	}

	if deps != nil && len(deps) != 0 {
		t.Errorf("GetBySystemID() returned %d dependencies, want 0", len(deps))
	}
}

func TestDependencyRepo_GetAllWithHeartbeat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	// Create dependency with heartbeat
	depWithHB, _ := domain.NewDependency(system.ID, "With Heartbeat", "")
	depWithHB.SetHeartbeatConfig(domain.HeartbeatConfig{
		URL:      "https://api.example.com/health",
		Interval: 30,
	})

	// Create dependency without heartbeat
	depWithoutHB, _ := domain.NewDependency(system.ID, "Without Heartbeat", "")
	depWithoutHB.HeartbeatMethod = "GET" // Required by schema

	for _, dep := range []*domain.Dependency{depWithHB, depWithoutHB} {
		if err := repo.Create(ctx, dep); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get only with heartbeat
	deps, err := repo.GetAllWithHeartbeat(ctx)
	if err != nil {
		t.Fatalf("GetAllWithHeartbeat() error = %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("GetAllWithHeartbeat() returned %d dependencies, want 1", len(deps))
	}

	if deps[0].Name != "With Heartbeat" {
		t.Errorf("dependency name = %s, want With Heartbeat", deps[0].Name)
	}
}

func TestDependencyRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	dep, _ := domain.NewDependency(system.ID, "Original Name", "Original Description")
	dep.HeartbeatMethod = "GET" // Required by schema
	if err := repo.Create(ctx, dep); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	dep.Name = "Updated Name"
	dep.Description = "Updated Description"
	dep.Status = domain.StatusYellow
	dep.LastCheck = time.Now()
	dep.LastLatency = 150
	dep.LastStatusCode = 200
	dep.ConsecutiveFailures = 2
	dep.UpdatedAt = time.Now()

	if err := repo.Update(ctx, dep); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, err := repo.GetByID(ctx, dep.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %s, want Updated Name", retrieved.Name)
	}
	if retrieved.Description != "Updated Description" {
		t.Errorf("Description = %s, want Updated Description", retrieved.Description)
	}
	if retrieved.Status != domain.StatusYellow {
		t.Errorf("Status = %s, want yellow", retrieved.Status)
	}
	if retrieved.LastLatency != 150 {
		t.Errorf("LastLatency = %d, want 150", retrieved.LastLatency)
	}
	if retrieved.LastStatusCode != 200 {
		t.Errorf("LastStatusCode = %d, want 200", retrieved.LastStatusCode)
	}
	if retrieved.ConsecutiveFailures != 2 {
		t.Errorf("ConsecutiveFailures = %d, want 2", retrieved.ConsecutiveFailures)
	}
}

func TestDependencyRepo_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewDependencyRepo(db)
	ctx := context.Background()

	dep := &domain.Dependency{
		ID:        99999,
		Name:      "Non-existent",
		UpdatedAt: time.Now(),
	}

	err := repo.Update(ctx, dep)
	if err == nil {
		t.Error("expected error when updating non-existent dependency")
	}
}

func TestDependencyRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	dep, _ := domain.NewDependency(system.ID, "To Delete", "")
	dep.HeartbeatMethod = "GET" // Required by schema
	if err := repo.Create(ctx, dep); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, dep.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, dep.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil after deletion")
	}
}

func TestDependencyRepo_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewDependencyRepo(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error when deleting non-existent dependency")
	}
}

func TestDependencyRepo_HeadersSerialization(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	dep, _ := domain.NewDependency(system.ID, "API with Headers", "")
	headers := map[string]string{
		"Authorization": "Bearer secret-token",
		"X-Custom":      "custom-value",
		"Content-Type":  "application/json",
	}
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{
		URL:      "https://api.example.com/health",
		Interval: 30,
		Headers:  headers,
	})

	if err := repo.Create(ctx, dep); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, dep.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(retrieved.HeartbeatHeaders) != len(headers) {
		t.Errorf("HeartbeatHeaders count = %d, want %d", len(retrieved.HeartbeatHeaders), len(headers))
	}

	for key, value := range headers {
		if retrieved.HeartbeatHeaders[key] != value {
			t.Errorf("HeartbeatHeaders[%s] = %s, want %s", key, retrieved.HeartbeatHeaders[key], value)
		}
	}
}

func TestDependencyRepo_StatusPersistence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewDependencyRepo(db)
	ctx := context.Background()

	statuses := []domain.Status{domain.StatusGreen, domain.StatusYellow, domain.StatusRed}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			dep, _ := domain.NewDependency(system.ID, "Status Test "+string(status), "")
			dep.Status = status
			dep.HeartbeatMethod = "GET" // Required by schema

			if err := repo.Create(ctx, dep); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			retrieved, err := repo.GetByID(ctx, dep.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if retrieved.Status != status {
				t.Errorf("Status = %s, want %s", retrieved.Status, status)
			}
		})
	}
}
