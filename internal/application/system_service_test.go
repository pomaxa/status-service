package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
)

func TestNewSystemService(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	logRepo := NewMockStatusLogRepository()

	service := NewSystemService(systemRepo, logRepo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestSystemService_CreateSystem(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	system, err := service.CreateSystem(context.Background(), "API", "Main API", "https://api.example.com", "Team")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if system == nil {
		t.Fatal("expected non-nil system")
	}
	if system.Name != "API" {
		t.Errorf("expected name 'API', got %q", system.Name)
	}
	if system.ID == 0 {
		t.Error("expected system ID to be set")
	}
}

func TestSystemService_CreateSystem_EmptyName(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	_, err := service.CreateSystem(context.Background(), "", "Description", "", "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestSystemService_CreateSystem_RepoError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.CreateFunc = func(ctx context.Context, s *domain.System) error {
		return errors.New("database error")
	}
	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	_, err := service.CreateSystem(context.Background(), "API", "Description", "", "")
	if err == nil {
		t.Error("expected error from repository")
	}
}

func TestSystemService_GetSystem(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "Description", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system

	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	result, err := service.GetSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil system")
	}
	if result.Name != "API" {
		t.Errorf("expected name 'API', got %q", result.Name)
	}
}

func TestSystemService_GetSystem_NotFound(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	result, err := service.GetSystem(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// GetSystem returns nil without error for non-existent system
	if result != nil {
		t.Error("expected nil result for non-existent system")
	}
}

func TestSystemService_GetAllSystems(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system1, _ := domain.NewSystem("API", "", "", "")
	system1.ID = 1
	system2, _ := domain.NewSystem("Database", "", "", "")
	system2.ID = 2
	systemRepo.Systems[1] = system1
	systemRepo.Systems[2] = system2

	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	systems, err := service.GetAllSystems(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(systems) != 2 {
		t.Errorf("expected 2 systems, got %d", len(systems))
	}
}

func TestSystemService_UpdateSystem(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "Old Description", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system

	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	result, err := service.UpdateSystem(context.Background(), 1, "New API", "New Description", "https://new.example.com", "New Team")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "New API" {
		t.Errorf("expected name 'New API', got %q", result.Name)
	}
	if result.Description != "New Description" {
		t.Errorf("expected description 'New Description', got %q", result.Description)
	}
}

func TestSystemService_UpdateSystem_NotFound(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	_, err := service.UpdateSystem(context.Background(), 999, "Name", "Desc", "", "")
	if err == nil {
		t.Error("expected error for non-existent system")
	}
}

func TestSystemService_UpdateSystemStatus(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system

	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	result, err := service.UpdateSystemStatus(context.Background(), 1, "yellow", "High latency detected")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != domain.StatusYellow {
		t.Errorf("expected status yellow, got %q", result.Status)
	}

	// Check that log was created
	if len(logRepo.Logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logRepo.Logs))
	}
}

func TestSystemService_UpdateSystemStatus_SameStatus(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system

	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	webhookRepo := NewMockWebhookRepository()
	notifService := NewNotificationService(webhookRepo, systemRepo, NewMockDependencyRepository())
	service.SetNotificationService(notifService)

	_, err := service.UpdateSystemStatus(context.Background(), 1, "green", "No change")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Log is always created, but notification is only sent if status changed
	if len(logRepo.Logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logRepo.Logs))
	}
}

func TestSystemService_UpdateSystemStatus_InvalidStatus(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system

	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	_, err := service.UpdateSystemStatus(context.Background(), 1, "invalid", "Bad status")
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestSystemService_DeleteSystem(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system

	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	err := service.DeleteSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := systemRepo.Systems[1]; exists {
		t.Error("expected system to be deleted")
	}
}

func TestSystemService_GetSystemLogs(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	logRepo := NewMockStatusLogRepository()

	systemID := int64(1)
	logRepo.Logs = append(logRepo.Logs, &domain.StatusLog{
		ID:        1,
		SystemID:  &systemID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
	})

	service := NewSystemService(systemRepo, logRepo)

	logs, err := service.GetSystemLogs(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestSystemService_SetNotificationService(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewSystemService(systemRepo, logRepo)

	webhookRepo := NewMockWebhookRepository()
	notifService := NewNotificationService(webhookRepo, systemRepo, NewMockDependencyRepository())

	service.SetNotificationService(notifService)

	// Test that status change triggers notification (no panic)
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system

	_, err := service.UpdateSystemStatus(context.Background(), 1, "red", "Test notification")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
