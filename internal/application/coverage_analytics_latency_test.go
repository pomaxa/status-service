package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
	"time"
)

// ============= Analytics Service error-path coverage =============

func TestAnalyticsService_GetSystemAnalytics_RepoError(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return nil, errors.New("db error")
	}
	service := NewAnalyticsService(analyticsRepo, NewMockStatusLogRepository())
	if _, err := service.GetSystemAnalytics(context.Background(), 1, "24h"); err == nil {
		t.Error("expected error")
	}
}

func TestAnalyticsService_GetDependencyAnalytics_RepoError(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetUptimeByDependencyIDFunc = func(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.Analytics, error) {
		return nil, errors.New("db error")
	}
	service := NewAnalyticsService(analyticsRepo, NewMockStatusLogRepository())
	if _, err := service.GetDependencyAnalytics(context.Background(), 1, "24h"); err == nil {
		t.Error("expected error")
	}
}

func TestAnalyticsService_GetOverallAnalytics_RepoError(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetOverallAnalyticsFunc = func(ctx context.Context, start, end time.Time) (*domain.Analytics, error) {
		return nil, errors.New("db error")
	}
	service := NewAnalyticsService(analyticsRepo, NewMockStatusLogRepository())
	if _, err := service.GetOverallAnalytics(context.Background(), "24h"); err == nil {
		t.Error("expected error")
	}
}

func TestAnalyticsService_GetSystemIncidents_RepoError(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetIncidentsBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
		return nil, errors.New("db error")
	}
	service := NewAnalyticsService(analyticsRepo, NewMockStatusLogRepository())
	if _, err := service.GetSystemIncidents(context.Background(), 1, "24h"); err == nil {
		t.Error("expected error")
	}
}

func TestAnalyticsService_GetDependencyIncidents_RepoError(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetIncidentsByDependencyIDFunc = func(ctx context.Context, dependencyID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
		return nil, errors.New("db error")
	}
	service := NewAnalyticsService(analyticsRepo, NewMockStatusLogRepository())
	if _, err := service.GetDependencyIncidents(context.Background(), 1, "24h"); err == nil {
		t.Error("expected error")
	}
}

func TestAnalyticsService_GetAllLogs_RepoError(t *testing.T) {
	logRepo := NewMockStatusLogRepository()
	logRepo.GetAllFunc = func(ctx context.Context, limit int) ([]*domain.StatusLog, error) {
		return nil, errors.New("db error")
	}
	service := NewAnalyticsService(NewMockAnalyticsRepository(), logRepo)
	if _, err := service.GetAllLogs(context.Background(), 10); err == nil {
		t.Error("expected error")
	}
}

func TestAnalyticsService_CreateLog_RepoError(t *testing.T) {
	logRepo := NewMockStatusLogRepository()
	logRepo.CreateFunc = func(ctx context.Context, log *domain.StatusLog) error {
		return errors.New("db error")
	}
	service := NewAnalyticsService(NewMockAnalyticsRepository(), logRepo)

	sysID := int64(1)
	log := domain.NewStatusLog(&sysID, nil, domain.StatusGreen, domain.StatusRed, "msg", domain.SourceManual)
	if err := service.CreateLog(context.Background(), log); err == nil {
		t.Error("expected create error")
	}
}

// ============= Latency Service error-path coverage =============

func TestLatencyService_GetDependencyLatencyStats_GetByIDError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewLatencyService(NewMockLatencyRepository(), depRepo)
	if _, err := service.GetDependencyLatencyStats(context.Background(), 1, "24h"); err == nil {
		t.Error("expected get dependency error")
	}
}

func TestLatencyService_GetDependencyLatencyStats_StatsError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Dep", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	latencyRepo := NewMockLatencyRepository()
	latencyRepo.GetStatsFunc = func(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.LatencyStats, error) {
		return nil, errors.New("stats error")
	}
	service := NewLatencyService(latencyRepo, depRepo)
	if _, err := service.GetDependencyLatencyStats(context.Background(), 1, "24h"); err == nil {
		t.Error("expected stats error")
	}
}
