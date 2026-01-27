package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

func TestLogRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	log := domain.NewStatusLog(&system.ID, nil, domain.StatusGreen, domain.StatusYellow, "System degraded", domain.SourceManual)

	err := repo.Create(ctx, log)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if log.ID == 0 {
		t.Error("expected log ID to be set after Create()")
	}
}

func TestLogRepo_Create_ForDependency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	depRepo := NewDependencyRepo(db)
	dep, _ := domain.NewDependency(system.ID, "Test Dep", "")
	dep.HeartbeatMethod = "GET" // Required by schema
	if err := depRepo.Create(context.Background(), dep); err != nil {
		t.Fatalf("failed to create dependency: %v", err)
	}

	repo := NewLogRepo(db)
	ctx := context.Background()

	log := domain.NewStatusLog(nil, &dep.ID, domain.StatusGreen, domain.StatusRed, "Dependency failed", domain.SourceHeartbeat)

	err := repo.Create(ctx, log)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if log.ID == 0 {
		t.Error("expected log ID to be set after Create()")
	}
}

func TestLogRepo_GetBySystemID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	// Create multiple logs for the system
	for i := 0; i < 5; i++ {
		log := domain.NewStatusLog(&system.ID, nil, domain.StatusGreen, domain.StatusYellow, "Log message", domain.SourceManual)
		if err := repo.Create(ctx, log); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(time.Millisecond)
	}

	// Get logs
	logs, err := repo.GetBySystemID(ctx, system.ID, 10)
	if err != nil {
		t.Fatalf("GetBySystemID() error = %v", err)
	}

	if len(logs) != 5 {
		t.Errorf("GetBySystemID() returned %d logs, want 5", len(logs))
	}

	// Verify system ID
	for _, log := range logs {
		if log.SystemID == nil || *log.SystemID != system.ID {
			t.Error("log SystemID mismatch")
		}
	}
}

func TestLogRepo_GetBySystemID_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	// Create multiple logs
	for i := 0; i < 10; i++ {
		log := domain.NewStatusLog(&system.ID, nil, domain.StatusGreen, domain.StatusYellow, "Log message", domain.SourceManual)
		repo.Create(ctx, log)
		time.Sleep(time.Millisecond)
	}

	// Get with limit
	logs, err := repo.GetBySystemID(ctx, system.ID, 5)
	if err != nil {
		t.Fatalf("GetBySystemID() error = %v", err)
	}

	if len(logs) != 5 {
		t.Errorf("GetBySystemID(5) returned %d logs, want 5", len(logs))
	}
}

func TestLogRepo_GetByDependencyID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	depRepo := NewDependencyRepo(db)
	dep, _ := domain.NewDependency(system.ID, "Test Dep", "")
	dep.HeartbeatMethod = "GET" // Required by schema
	depRepo.Create(context.Background(), dep)

	repo := NewLogRepo(db)
	ctx := context.Background()

	// Create logs for the dependency
	for i := 0; i < 3; i++ {
		log := domain.NewStatusLog(nil, &dep.ID, domain.StatusGreen, domain.StatusRed, "Dep log", domain.SourceHeartbeat)
		repo.Create(ctx, log)
		time.Sleep(time.Millisecond)
	}

	logs, err := repo.GetByDependencyID(ctx, dep.ID, 10)
	if err != nil {
		t.Fatalf("GetByDependencyID() error = %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("GetByDependencyID() returned %d logs, want 3", len(logs))
	}

	for _, log := range logs {
		if log.DependencyID == nil || *log.DependencyID != dep.ID {
			t.Error("log DependencyID mismatch")
		}
	}
}

func TestLogRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	// Create some logs
	for i := 0; i < 7; i++ {
		log := domain.NewStatusLog(&system.ID, nil, domain.StatusGreen, domain.StatusYellow, "Log", domain.SourceManual)
		repo.Create(ctx, log)
		time.Sleep(time.Millisecond)
	}

	logs, err := repo.GetAll(ctx, 100)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(logs) != 7 {
		t.Errorf("GetAll() returned %d logs, want 7", len(logs))
	}
}

func TestLogRepo_GetByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	start := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Create logs
	for i := 0; i < 3; i++ {
		log := domain.NewStatusLog(&system.ID, nil, domain.StatusGreen, domain.StatusYellow, "In range", domain.SourceManual)
		repo.Create(ctx, log)
		time.Sleep(time.Millisecond)
	}

	end := time.Now()

	logs, err := repo.GetByTimeRange(ctx, start, end)
	if err != nil {
		t.Fatalf("GetByTimeRange() error = %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("GetByTimeRange() returned %d logs, want 3", len(logs))
	}
}

func TestLogRepo_GetSystemLogsByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sysRepo := NewSystemRepo(db)
	system1, _ := domain.NewSystem("System 1", "", "", "")
	system2, _ := domain.NewSystem("System 2", "", "", "")
	sysRepo.Create(context.Background(), system1)
	sysRepo.Create(context.Background(), system2)

	repo := NewLogRepo(db)
	ctx := context.Background()

	start := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Create logs for both systems
	for i := 0; i < 3; i++ {
		log1 := domain.NewStatusLog(&system1.ID, nil, domain.StatusGreen, domain.StatusYellow, "System 1 log", domain.SourceManual)
		log2 := domain.NewStatusLog(&system2.ID, nil, domain.StatusGreen, domain.StatusRed, "System 2 log", domain.SourceManual)
		repo.Create(ctx, log1)
		repo.Create(ctx, log2)
		time.Sleep(time.Millisecond)
	}

	end := time.Now()

	// Get logs for system1 only
	logs, err := repo.GetSystemLogsByTimeRange(ctx, system1.ID, start, end)
	if err != nil {
		t.Fatalf("GetSystemLogsByTimeRange() error = %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("GetSystemLogsByTimeRange() returned %d logs, want 3", len(logs))
	}

	for _, log := range logs {
		if log.SystemID == nil || *log.SystemID != system1.ID {
			t.Error("log SystemID mismatch")
		}
	}
}

func TestLogRepo_GetDependencyLogsByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	depRepo := NewDependencyRepo(db)
	dep1, _ := domain.NewDependency(system.ID, "Dep 1", "")
	dep1.HeartbeatMethod = "GET" // Required by schema
	dep2, _ := domain.NewDependency(system.ID, "Dep 2", "")
	dep2.HeartbeatMethod = "GET" // Required by schema
	depRepo.Create(context.Background(), dep1)
	depRepo.Create(context.Background(), dep2)

	repo := NewLogRepo(db)
	ctx := context.Background()

	start := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Create logs for both dependencies
	for i := 0; i < 4; i++ {
		log1 := domain.NewStatusLog(nil, &dep1.ID, domain.StatusGreen, domain.StatusYellow, "Dep 1 log", domain.SourceHeartbeat)
		log2 := domain.NewStatusLog(nil, &dep2.ID, domain.StatusGreen, domain.StatusRed, "Dep 2 log", domain.SourceHeartbeat)
		repo.Create(ctx, log1)
		repo.Create(ctx, log2)
		time.Sleep(time.Millisecond)
	}

	end := time.Now()

	// Get logs for dep1 only
	logs, err := repo.GetDependencyLogsByTimeRange(ctx, dep1.ID, start, end)
	if err != nil {
		t.Fatalf("GetDependencyLogsByTimeRange() error = %v", err)
	}

	if len(logs) != 4 {
		t.Errorf("GetDependencyLogsByTimeRange() returned %d logs, want 4", len(logs))
	}

	for _, log := range logs {
		if log.DependencyID == nil || *log.DependencyID != dep1.ID {
			t.Error("log DependencyID mismatch")
		}
	}
}

func TestLogRepo_StatusPersistence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	testCases := []struct {
		oldStatus domain.Status
		newStatus domain.Status
	}{
		{domain.StatusGreen, domain.StatusYellow},
		{domain.StatusYellow, domain.StatusRed},
		{domain.StatusRed, domain.StatusGreen},
	}

	for _, tc := range testCases {
		t.Run(string(tc.oldStatus)+"_to_"+string(tc.newStatus), func(t *testing.T) {
			log := domain.NewStatusLog(&system.ID, nil, tc.oldStatus, tc.newStatus, "Status change", domain.SourceManual)
			if err := repo.Create(ctx, log); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			logs, err := repo.GetBySystemID(ctx, system.ID, 100)
			if err != nil {
				t.Fatalf("GetBySystemID() error = %v", err)
			}

			// Find our log
			var found *domain.StatusLog
			for _, l := range logs {
				if l.ID == log.ID {
					found = l
					break
				}
			}

			if found == nil {
				t.Fatal("log not found")
			}

			if found.OldStatus != tc.oldStatus {
				t.Errorf("OldStatus = %s, want %s", found.OldStatus, tc.oldStatus)
			}
			if found.NewStatus != tc.newStatus {
				t.Errorf("NewStatus = %s, want %s", found.NewStatus, tc.newStatus)
			}
		})
	}
}

func TestLogRepo_SourcePersistence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	sources := []domain.ChangeSource{domain.SourceManual, domain.SourceHeartbeat}

	for _, source := range sources {
		t.Run(string(source), func(t *testing.T) {
			log := domain.NewStatusLog(&system.ID, nil, domain.StatusGreen, domain.StatusYellow, "Test", source)
			if err := repo.Create(ctx, log); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			logs, err := repo.GetBySystemID(ctx, system.ID, 100)
			if err != nil {
				t.Fatalf("GetBySystemID() error = %v", err)
			}

			var found *domain.StatusLog
			for _, l := range logs {
				if l.ID == log.ID {
					found = l
					break
				}
			}

			if found == nil {
				t.Fatal("log not found")
			}

			if found.Source != source {
				t.Errorf("Source = %s, want %s", found.Source, source)
			}
		})
	}
}

func TestLogRepo_MessagePersistence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	longMessage := "This is a very long message that contains special characters like: !@#$%^&*()_+-=[]{}|;':\",./<>? and also unicode: 日本語 emoji: 🚀"

	log := domain.NewStatusLog(&system.ID, nil, domain.StatusGreen, domain.StatusRed, longMessage, domain.SourceManual)
	if err := repo.Create(ctx, log); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	logs, err := repo.GetBySystemID(ctx, system.ID, 1)
	if err != nil {
		t.Fatalf("GetBySystemID() error = %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("no logs returned")
	}

	if logs[0].Message != longMessage {
		t.Errorf("Message = %s, want %s", logs[0].Message, longMessage)
	}
}

func TestLogRepo_OrderByCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	system := createTestSystem(t, db)
	repo := NewLogRepo(db)
	ctx := context.Background()

	// Create logs with distinct timestamps
	messages := []string{"First", "Second", "Third"}
	for _, msg := range messages {
		log := domain.NewStatusLog(&system.ID, nil, domain.StatusGreen, domain.StatusYellow, msg, domain.SourceManual)
		repo.Create(ctx, log)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// GetBySystemID orders DESC
	logs, _ := repo.GetBySystemID(ctx, system.ID, 10)
	if logs[0].Message != "Third" {
		t.Errorf("expected most recent log first, got %s", logs[0].Message)
	}

	// GetByTimeRange orders ASC
	start := time.Now().Add(-time.Minute)
	end := time.Now().Add(time.Minute)
	logsAsc, _ := repo.GetByTimeRange(ctx, start, end)
	if logsAsc[0].Message != "First" {
		t.Errorf("expected oldest log first in time range query, got %s", logsAsc[0].Message)
	}
}
