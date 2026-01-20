package application

import (
	"context"
	"status-incident/internal/domain"
	"testing"
	"time"
)

func TestNewMaintenanceService(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()

	service := NewMaintenanceService(maintenanceRepo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestMaintenanceService_CreateMaintenance(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	service := NewMaintenanceService(maintenanceRepo)

	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)

	maint, err := service.CreateMaintenance(
		context.Background(),
		"Database Migration",
		"Upgrading to PostgreSQL 15",
		start,
		end,
		[]int64{1, 2},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if maint == nil {
		t.Fatal("expected non-nil maintenance")
	}
	if maint.Title != "Database Migration" {
		t.Errorf("expected title 'Database Migration', got %q", maint.Title)
	}
	if len(maint.SystemIDs) != 2 {
		t.Errorf("expected 2 system IDs, got %d", len(maint.SystemIDs))
	}
}

func TestMaintenanceService_CreateMaintenance_NoSystems(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	service := NewMaintenanceService(maintenanceRepo)

	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)

	maint, err := service.CreateMaintenance(
		context.Background(),
		"General Maintenance",
		"System updates",
		start,
		end,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if maint == nil {
		t.Fatal("expected non-nil maintenance")
	}
}

func TestMaintenanceService_CreateMaintenance_EmptyTitle(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	service := NewMaintenanceService(maintenanceRepo)

	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)

	_, err := service.CreateMaintenance(
		context.Background(),
		"",
		"Description",
		start,
		end,
		nil,
	)
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestMaintenanceService_GetMaintenance(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)
	maint, _ := domain.NewMaintenance("Test", "Desc", start, end)
	maint.ID = 1
	maintenanceRepo.Maintenances[1] = maint

	service := NewMaintenanceService(maintenanceRepo)

	result, err := service.GetMaintenance(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil maintenance")
	}
	if result.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", result.Title)
	}
}

func TestMaintenanceService_GetAllMaintenances(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)
	maint1, _ := domain.NewMaintenance("Maintenance 1", "Desc", start, end)
	maint1.ID = 1
	maint2, _ := domain.NewMaintenance("Maintenance 2", "Desc", start, end)
	maint2.ID = 2
	maintenanceRepo.Maintenances[1] = maint1
	maintenanceRepo.Maintenances[2] = maint2

	service := NewMaintenanceService(maintenanceRepo)

	maintenances, err := service.GetAllMaintenances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(maintenances) != 2 {
		t.Errorf("expected 2 maintenances, got %d", len(maintenances))
	}
}

func TestMaintenanceService_GetActiveMaintenances(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	// Active maintenance: started in the past, ends in the future
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(2 * time.Hour)
	maint, _ := domain.NewMaintenance("Active", "Desc", start, end)
	maint.ID = 1
	maintenanceRepo.Maintenances[1] = maint

	// Future maintenance: starts in the future
	start2 := time.Now().Add(5 * time.Hour)
	end2 := time.Now().Add(7 * time.Hour)
	maint2, _ := domain.NewMaintenance("Future", "Desc", start2, end2)
	maint2.ID = 2
	maintenanceRepo.Maintenances[2] = maint2

	service := NewMaintenanceService(maintenanceRepo)

	maintenances, err := service.GetActiveMaintenances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(maintenances) != 1 {
		t.Errorf("expected 1 active maintenance, got %d", len(maintenances))
	}
}

func TestMaintenanceService_GetUpcomingMaintenances(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	// Upcoming maintenance: starts in the future
	start := time.Now().Add(5 * time.Hour)
	end := time.Now().Add(7 * time.Hour)
	maint, _ := domain.NewMaintenance("Upcoming", "Desc", start, end)
	maint.ID = 1
	maintenanceRepo.Maintenances[1] = maint

	service := NewMaintenanceService(maintenanceRepo)

	maintenances, err := service.GetUpcomingMaintenances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(maintenances) != 1 {
		t.Errorf("expected 1 upcoming maintenance, got %d", len(maintenances))
	}
}

func TestMaintenanceService_UpdateMaintenance(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)
	maint, _ := domain.NewMaintenance("Old Title", "Old Desc", start, end)
	maint.ID = 1
	maintenanceRepo.Maintenances[1] = maint

	service := NewMaintenanceService(maintenanceRepo)

	newStart := time.Now().Add(2 * time.Hour)
	newEnd := time.Now().Add(5 * time.Hour)

	result, err := service.UpdateMaintenance(
		context.Background(),
		1,
		"New Title",
		"New Description",
		newStart,
		newEnd,
		[]int64{1, 2, 3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %q", result.Title)
	}
	if len(result.SystemIDs) != 3 {
		t.Errorf("expected 3 system IDs, got %d", len(result.SystemIDs))
	}
}

func TestMaintenanceService_UpdateMaintenance_NotFound(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	service := NewMaintenanceService(maintenanceRepo)

	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)

	_, err := service.UpdateMaintenance(
		context.Background(),
		999,
		"Title",
		"Desc",
		start,
		end,
		nil,
	)
	if err == nil {
		t.Error("expected error for non-existent maintenance")
	}
}

func TestMaintenanceService_CancelMaintenance(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)
	maint, _ := domain.NewMaintenance("Test", "Desc", start, end)
	maint.ID = 1
	maintenanceRepo.Maintenances[1] = maint

	service := NewMaintenanceService(maintenanceRepo)

	result, err := service.CancelMaintenance(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != domain.MaintenanceCancelled {
		t.Error("expected maintenance to be cancelled")
	}
}

func TestMaintenanceService_CancelMaintenance_NotFound(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	service := NewMaintenanceService(maintenanceRepo)

	_, err := service.CancelMaintenance(context.Background(), 999)
	if err == nil {
		t.Error("expected error for non-existent maintenance")
	}
}

func TestMaintenanceService_DeleteMaintenance(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	start := time.Now().Add(1 * time.Hour)
	end := time.Now().Add(3 * time.Hour)
	maint, _ := domain.NewMaintenance("Test", "Desc", start, end)
	maint.ID = 1
	maintenanceRepo.Maintenances[1] = maint

	service := NewMaintenanceService(maintenanceRepo)

	err := service.DeleteMaintenance(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := maintenanceRepo.Maintenances[1]; exists {
		t.Error("expected maintenance to be deleted")
	}
}

func TestMaintenanceService_IsSystemUnderMaintenance(t *testing.T) {
	maintenanceRepo := NewMockMaintenanceRepository()
	// Active maintenance for system 1
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(2 * time.Hour)
	maint, _ := domain.NewMaintenance("Active", "Desc", start, end)
	maint.ID = 1
	maint.SetSystemIDs([]int64{1, 2})
	maintenanceRepo.Maintenances[1] = maint

	service := NewMaintenanceService(maintenanceRepo)

	// System 1 should be under maintenance
	under, activeMaint, err := service.IsSystemUnderMaintenance(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !under {
		t.Error("expected system 1 to be under maintenance")
	}
	if activeMaint == nil {
		t.Error("expected active maintenance to be returned")
	}

	// System 3 should not be under maintenance
	under, activeMaint, err = service.IsSystemUnderMaintenance(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if under {
		t.Error("expected system 3 to not be under maintenance")
	}
	if activeMaint != nil {
		t.Error("expected nil maintenance for system not under maintenance")
	}
}
