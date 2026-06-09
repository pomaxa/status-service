package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
)

// ============= System Service error-path coverage =============

func TestSystemService_GetSystem_RepoError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("db error")
	}
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	if _, err := service.GetSystem(context.Background(), 1); err == nil {
		t.Error("expected error from repository")
	}
}

func TestSystemService_GetAllSystems_RepoError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetAllFunc = func(ctx context.Context) ([]*domain.System, error) {
		return nil, errors.New("db error")
	}
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	if _, err := service.GetAllSystems(context.Background()); err == nil {
		t.Error("expected error from repository")
	}
}

func TestSystemService_UpdateSystem_GetError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("db error")
	}
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateSystem(context.Background(), 1, "name", "desc", "", ""); err == nil {
		t.Error("expected error from repository")
	}
}

func TestSystemService_UpdateSystem_InvalidUpdateData(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	// Empty name triggers system.Update validation error
	if _, err := service.UpdateSystem(context.Background(), 1, "", "desc", "", ""); err == nil {
		t.Error("expected invalid update data error")
	}
}

func TestSystemService_UpdateSystem_SaveError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system
	systemRepo.UpdateFunc = func(ctx context.Context, s *domain.System) error {
		return errors.New("save error")
	}
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateSystem(context.Background(), 1, "New", "desc", "", ""); err == nil {
		t.Error("expected save error")
	}
}

func TestSystemService_UpdateSystemStatus_GetError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("db error")
	}
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateSystemStatus(context.Background(), 1, "red", "msg"); err == nil {
		t.Error("expected error from repository")
	}
}

func TestSystemService_UpdateSystemStatus_NotFound(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateSystemStatus(context.Background(), 999, "red", "msg"); err == nil {
		t.Error("expected not found error")
	}
}

func TestSystemService_UpdateSystemStatus_SaveError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system
	systemRepo.UpdateFunc = func(ctx context.Context, s *domain.System) error {
		return errors.New("save error")
	}
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateSystemStatus(context.Background(), 1, "red", "msg"); err == nil {
		t.Error("expected save error")
	}
}

func TestSystemService_UpdateSystemStatus_LogError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system

	logRepo := NewMockStatusLogRepository()
	logRepo.CreateFunc = func(ctx context.Context, log *domain.StatusLog) error {
		return errors.New("log error")
	}
	service := NewSystemService(systemRepo, logRepo)

	// Log failure is non-fatal; operation still succeeds
	if _, err := service.UpdateSystemStatus(context.Background(), 1, "red", "msg"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSystemService_DeleteSystem_RepoError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.DeleteFunc = func(ctx context.Context, id int64) error {
		return errors.New("delete error")
	}
	service := NewSystemService(systemRepo, NewMockStatusLogRepository())

	if err := service.DeleteSystem(context.Background(), 1); err == nil {
		t.Error("expected delete error")
	}
}

func TestSystemService_GetSystemLogs_RepoError(t *testing.T) {
	logRepo := NewMockStatusLogRepository()
	logRepo.GetBySystemIDFunc = func(ctx context.Context, systemID int64, limit int) ([]*domain.StatusLog, error) {
		return nil, errors.New("db error")
	}
	service := NewSystemService(NewMockSystemRepository(), logRepo)

	if _, err := service.GetSystemLogs(context.Background(), 1, 10); err == nil {
		t.Error("expected error from repository")
	}
}
