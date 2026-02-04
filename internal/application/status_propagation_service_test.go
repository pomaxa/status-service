package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
)

func TestNewStatusPropagationService(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()

	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestStatusPropagationService_SetNotificationService(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	webhookRepo := NewMockWebhookRepository()
	notifService := NewNotificationService(webhookRepo, systemRepo, depRepo)
	service.SetNotificationService(notifService)
	// No panic means success
}

func TestStatusPropagationService_PropagateStatusToSystem_NoDeps(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	changed, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if changed {
		t.Error("expected no change when system has no dependencies")
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_AllGreen(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	dep1, _ := domain.NewDependency(1, "Redis", "Cache")
	dep1.ID = 1
	dep1.Status = domain.StatusGreen
	dep2, _ := domain.NewDependency(1, "PostgreSQL", "Database")
	dep2.ID = 2
	dep2.Status = domain.StatusGreen
	depRepo.Dependencies[1] = dep1
	depRepo.Dependencies[2] = dep2

	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	changed, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if changed {
		t.Error("expected no change when all dependencies are green and system is green")
	}
	if system.Status != domain.StatusGreen {
		t.Errorf("expected system status to remain green, got %q", system.Status)
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_OneYellow(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	dep1, _ := domain.NewDependency(1, "Redis", "Cache")
	dep1.ID = 1
	dep1.Status = domain.StatusGreen
	dep2, _ := domain.NewDependency(1, "PostgreSQL", "Database")
	dep2.ID = 2
	dep2.Status = domain.StatusYellow
	depRepo.Dependencies[1] = dep1
	depRepo.Dependencies[2] = dep2

	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	changed, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !changed {
		t.Error("expected status change when dependency is yellow")
	}
	if system.Status != domain.StatusYellow {
		t.Errorf("expected system status to be yellow, got %q", system.Status)
	}
	if len(logRepo.Logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logRepo.Logs))
	}
	if logRepo.Logs[0].Source != domain.SourcePropagation {
		t.Errorf("expected source to be propagation, got %q", logRepo.Logs[0].Source)
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_OneRed(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	dep1, _ := domain.NewDependency(1, "Redis", "Cache")
	dep1.ID = 1
	dep1.Status = domain.StatusGreen
	dep2, _ := domain.NewDependency(1, "PostgreSQL", "Database")
	dep2.ID = 2
	dep2.Status = domain.StatusRed
	depRepo.Dependencies[1] = dep1
	depRepo.Dependencies[2] = dep2

	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	changed, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !changed {
		t.Error("expected status change when dependency is red")
	}
	if system.Status != domain.StatusRed {
		t.Errorf("expected system status to be red, got %q", system.Status)
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_RedOverridesYellow(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	dep1, _ := domain.NewDependency(1, "Redis", "Cache")
	dep1.ID = 1
	dep1.Status = domain.StatusYellow
	dep2, _ := domain.NewDependency(1, "PostgreSQL", "Database")
	dep2.ID = 2
	dep2.Status = domain.StatusRed
	dep3, _ := domain.NewDependency(1, "MongoDB", "NoSQL")
	dep3.ID = 3
	dep3.Status = domain.StatusGreen
	depRepo.Dependencies[1] = dep1
	depRepo.Dependencies[2] = dep2
	depRepo.Dependencies[3] = dep3

	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	changed, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !changed {
		t.Error("expected status change")
	}
	if system.Status != domain.StatusRed {
		t.Errorf("expected system status to be red (worst case), got %q", system.Status)
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_Recovery(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusRed // System was red
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	dep1, _ := domain.NewDependency(1, "Redis", "Cache")
	dep1.ID = 1
	dep1.Status = domain.StatusGreen // All deps now green
	dep2, _ := domain.NewDependency(1, "PostgreSQL", "Database")
	dep2.ID = 2
	dep2.Status = domain.StatusGreen
	depRepo.Dependencies[1] = dep1
	depRepo.Dependencies[2] = dep2

	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	changed, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !changed {
		t.Error("expected status change for recovery")
	}
	if system.Status != domain.StatusGreen {
		t.Errorf("expected system status to recover to green, got %q", system.Status)
	}
	if len(logRepo.Logs) != 1 {
		t.Errorf("expected 1 log entry for recovery, got %d", len(logRepo.Logs))
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_SystemNotFound(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	_, err := service.PropagateStatusToSystem(context.Background(), 999)
	if err == nil {
		t.Error("expected error for non-existent system")
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_SystemRepoError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("database error")
	}
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	_, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err == nil {
		t.Error("expected error from repository")
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_DepRepoError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	depRepo.GetBySystemIDFunc = func(ctx context.Context, systemID int64) ([]*domain.Dependency, error) {
		return nil, errors.New("database error")
	}
	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	_, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err == nil {
		t.Error("expected error from dependency repository")
	}
}

func TestStatusPropagationService_PropagateStatusToSystem_NoChangeWhenSameStatus(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusYellow // System already yellow
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	dep1, _ := domain.NewDependency(1, "Redis", "Cache")
	dep1.ID = 1
	dep1.Status = domain.StatusYellow // Dependency also yellow
	depRepo.Dependencies[1] = dep1

	logRepo := NewMockStatusLogRepository()
	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	changed, err := service.PropagateStatusToSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if changed {
		t.Error("expected no change when aggregate status matches system status")
	}
	if len(logRepo.Logs) != 0 {
		t.Errorf("expected no log entries when no change, got %d", len(logRepo.Logs))
	}
}
