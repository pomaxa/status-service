package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
)

// ============= Dependency Service error-path coverage =============

func TestDependencyService_GetDependency_RepoError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.GetDependency(context.Background(), 1); err == nil {
		t.Error("expected error from repository")
	}
}

func TestDependencyService_GetDependenciesBySystem_RepoError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetBySystemIDFunc = func(ctx context.Context, systemID int64) ([]*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.GetDependenciesBySystem(context.Background(), 1); err == nil {
		t.Error("expected error from repository")
	}
}

func TestDependencyService_UpdateDependency_GetError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateDependency(context.Background(), 1, "name", "desc"); err == nil {
		t.Error("expected error from repository")
	}
}

func TestDependencyService_UpdateDependency_InvalidData(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	// Empty name triggers dep.Update validation error
	if _, err := service.UpdateDependency(context.Background(), 1, "", "desc"); err == nil {
		t.Error("expected invalid update data error")
	}
}

func TestDependencyService_UpdateDependency_SaveError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep
	depRepo.UpdateFunc = func(ctx context.Context, d *domain.Dependency) error {
		return errors.New("save error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateDependency(context.Background(), 1, "New", "desc"); err == nil {
		t.Error("expected save error")
	}
}

func TestDependencyService_SetHeartbeatConfig_GetError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.SetHeartbeatConfig(context.Background(), 1, domain.HeartbeatConfig{URL: "https://x.com", Interval: 60}); err == nil {
		t.Error("expected error from repository")
	}
}

func TestDependencyService_SetHeartbeatConfig_InvalidConfig(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	// Empty URL is an invalid heartbeat config
	if _, err := service.SetHeartbeatConfig(context.Background(), 1, domain.HeartbeatConfig{URL: "", Interval: 60}); err == nil {
		t.Error("expected invalid heartbeat config error")
	}
}

func TestDependencyService_SetHeartbeatConfig_SaveError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep
	depRepo.UpdateFunc = func(ctx context.Context, d *domain.Dependency) error {
		return errors.New("save error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.SetHeartbeatConfig(context.Background(), 1, domain.HeartbeatConfig{URL: "https://x.com", Interval: 60}); err == nil {
		t.Error("expected save error")
	}
}

func TestDependencyService_ClearHeartbeat_GetError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.ClearHeartbeat(context.Background(), 1); err == nil {
		t.Error("expected error from repository")
	}
}

func TestDependencyService_ClearHeartbeat_SaveError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep
	depRepo.UpdateFunc = func(ctx context.Context, d *domain.Dependency) error {
		return errors.New("save error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.ClearHeartbeat(context.Background(), 1); err == nil {
		t.Error("expected save error")
	}
}

func TestDependencyService_UpdateDependencyStatus_GetError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateDependencyStatus(context.Background(), 1, "red", "msg"); err == nil {
		t.Error("expected error from repository")
	}
}

func TestDependencyService_UpdateDependencyStatus_SaveError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep
	depRepo.UpdateFunc = func(ctx context.Context, d *domain.Dependency) error {
		return errors.New("save error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if _, err := service.UpdateDependencyStatus(context.Background(), 1, "red", "msg"); err == nil {
		t.Error("expected save error")
	}
}

func TestDependencyService_UpdateDependencyStatus_LogError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	logRepo.CreateFunc = func(ctx context.Context, log *domain.StatusLog) error {
		return errors.New("log error")
	}
	service := NewDependencyService(depRepo, logRepo)

	// Log failure is non-fatal
	if _, err := service.UpdateDependencyStatus(context.Background(), 1, "red", "msg"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDependencyService_DeleteDependency_RepoError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.DeleteFunc = func(ctx context.Context, id int64) error {
		return errors.New("delete error")
	}
	service := NewDependencyService(depRepo, NewMockStatusLogRepository())

	if err := service.DeleteDependency(context.Background(), 1); err == nil {
		t.Error("expected delete error")
	}
}

func TestDependencyService_GetDependencyLogs_RepoError(t *testing.T) {
	logRepo := NewMockStatusLogRepository()
	logRepo.GetByDependencyIDFunc = func(ctx context.Context, dependencyID int64, limit int) ([]*domain.StatusLog, error) {
		return nil, errors.New("db error")
	}
	service := NewDependencyService(NewMockDependencyRepository(), logRepo)

	if _, err := service.GetDependencyLogs(context.Background(), 1, 10); err == nil {
		t.Error("expected error from repository")
	}
}
