package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
	"time"
)

// ============= SLA Service error-path & dependency-report coverage =============

func TestSLAService_GenerateCustomReport_GetSystemsError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetAllFunc = func(ctx context.Context) ([]*domain.System, error) {
		return nil, errors.New("db error")
	}
	service := NewSLAService(systemRepo, nil, nil, NewMockSLAReportRepository(), nil, nil, nil)

	start := time.Now().AddDate(0, -1, 0)
	if _, err := service.GenerateCustomReport(context.Background(), "t", "monthly", start, time.Now(), "admin"); err == nil {
		t.Error("expected get systems error")
	}
}

// failingSLAReportRepo makes Create fail to cover the save-error branch.
type failingSLAReportRepo struct{}

func (f *failingSLAReportRepo) Create(ctx context.Context, r *domain.SLAReport) error {
	return errors.New("save error")
}
func (f *failingSLAReportRepo) GetByID(ctx context.Context, id int64) (*domain.SLAReport, error) {
	return nil, nil
}
func (f *failingSLAReportRepo) GetAll(ctx context.Context, limit int) ([]*domain.SLAReport, error) {
	return nil, nil
}
func (f *failingSLAReportRepo) GetByPeriod(ctx context.Context, start, end time.Time) ([]*domain.SLAReport, error) {
	return nil, nil
}
func (f *failingSLAReportRepo) Delete(ctx context.Context, id int64) error { return nil }

func TestSLAService_GenerateCustomReport_SaveError(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	systemRepo.Create(ctx, system)

	service := NewSLAService(systemRepo, NewMockDependencyRepository(), NewMockAnalyticsRepository(), &failingSLAReportRepo{}, nil, NewMockLatencyRepository(), nil)

	start := time.Now().AddDate(0, -1, 0)
	if _, err := service.GenerateCustomReport(ctx, "t", "monthly", start, time.Now(), "admin"); err == nil {
		t.Error("expected save error")
	}
}

// generateSystemReport: analytics error causes the system to be skipped (continue).
func TestSLAService_GenerateCustomReport_SystemAnalyticsErrorSkipped(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	systemRepo.Create(ctx, system)

	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return nil, errors.New("analytics error")
	}

	service := NewSLAService(systemRepo, NewMockDependencyRepository(), analyticsRepo, NewMockSLAReportRepository(), nil, NewMockLatencyRepository(), nil)

	start := time.Now().AddDate(0, -1, 0)
	report, err := service.GenerateCustomReport(ctx, "t", "monthly", start, time.Now(), "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.SystemReports) != 0 {
		t.Errorf("expected system to be skipped, got %d reports", len(report.SystemReports))
	}
}

// Covers generateSystemReport's dependency loop AND generateDependencyReport (was 0%).
func TestSLAService_GenerateCustomReport_WithDependencies(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	systemRepo.Create(ctx, system)

	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(system.ID, "Postgres", "primary db")
	depRepo.Create(ctx, dep)
	// A second dependency whose analytics fail, to exercise the continue branch.
	depBad, _ := domain.NewDependency(system.ID, "Cache", "")
	depRepo.Create(ctx, depBad)

	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetUptimeByDependencyIDFunc = func(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.Analytics, error) {
		if dependencyID == depBad.ID {
			return nil, errors.New("dep analytics error")
		}
		return &domain.Analytics{UptimePercent: 99.5, AvailabilityPercent: 99.7}, nil
	}

	latencyRepo := NewMockLatencyRepository()
	service := NewSLAService(systemRepo, depRepo, analyticsRepo, NewMockSLAReportRepository(), nil, latencyRepo, nil)

	start := time.Now().AddDate(0, -1, 0)
	report, err := service.GenerateCustomReport(ctx, "t", "monthly", start, time.Now(), "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.SystemReports) != 1 {
		t.Fatalf("expected 1 system report, got %d", len(report.SystemReports))
	}
	depReports := report.SystemReports[0].DependencyReports
	if len(depReports) != 1 {
		t.Fatalf("expected 1 dependency report (bad one skipped), got %d", len(depReports))
	}
	if depReports[0].DependencyName != "Postgres" {
		t.Errorf("expected Postgres dependency report, got %q", depReports[0].DependencyName)
	}
	// Latency stats from the mock should be populated.
	if depReports[0].TotalChecks == 0 {
		t.Error("expected latency stats to be populated on dependency report")
	}
}

// generateDependencyReport with nil latencyRepo: latency stats block skipped.
func TestSLAService_GenerateCustomReport_WithDependencies_NoLatencyRepo(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	systemRepo.Create(ctx, system)

	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(system.ID, "Postgres", "")
	depRepo.Create(ctx, dep)

	service := NewSLAService(systemRepo, depRepo, NewMockAnalyticsRepository(), NewMockSLAReportRepository(), nil, nil, nil)

	start := time.Now().AddDate(0, -1, 0)
	report, err := service.GenerateCustomReport(ctx, "t", "monthly", start, time.Now(), "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.SystemReports) != 1 || len(report.SystemReports[0].DependencyReports) != 1 {
		t.Fatalf("expected 1 system report with 1 dependency report")
	}
}

// generateSystemReport: depRepo.GetBySystemID error -> deps set to nil, no panic.
func TestSLAService_GenerateCustomReport_DepRepoError(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	systemRepo.Create(ctx, system)

	depRepo := NewMockDependencyRepository()
	depRepo.GetBySystemIDFunc = func(ctx context.Context, systemID int64) ([]*domain.Dependency, error) {
		return nil, errors.New("dep repo error")
	}

	service := NewSLAService(systemRepo, depRepo, NewMockAnalyticsRepository(), NewMockSLAReportRepository(), nil, NewMockLatencyRepository(), nil)

	start := time.Now().AddDate(0, -1, 0)
	report, err := service.GenerateCustomReport(ctx, "t", "monthly", start, time.Now(), "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.SystemReports) != 1 {
		t.Fatalf("expected 1 system report, got %d", len(report.SystemReports))
	}
	if len(report.SystemReports[0].DependencyReports) != 0 {
		t.Error("expected no dependency reports when dep repo errors")
	}
}

func TestSLAService_CheckForBreaches_GetSystemsError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetAllFunc = func(ctx context.Context) ([]*domain.System, error) {
		return nil, errors.New("db error")
	}
	service := NewSLAService(systemRepo, nil, nil, nil, nil, nil, nil)

	if _, err := service.CheckForBreaches(context.Background(), "monthly"); err == nil {
		t.Error("expected get systems error")
	}
}

func TestSLAService_CheckForBreaches_AnalyticsErrorSkipped(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	systemRepo.Create(ctx, system)

	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return nil, errors.New("analytics error")
	}
	service := NewSLAService(systemRepo, nil, analyticsRepo, nil, NewMockSLABreachRepository(), nil, nil)

	breaches, err := service.CheckForBreaches(ctx, "monthly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(breaches) != 0 {
		t.Errorf("expected 0 breaches, got %d", len(breaches))
	}
}

func TestSLAService_CheckForBreaches_WithNotification(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.SetSLATarget(99.9)
	systemRepo.Create(ctx, system)

	analyticsRepo := NewMockAnalyticsRepository()
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return &domain.Analytics{UptimePercent: 98.0}, nil
	}

	webhookRepo := NewMockWebhookRepository()
	notifService := NewNotificationService(webhookRepo, systemRepo, NewMockDependencyRepository())

	service := NewSLAService(systemRepo, nil, analyticsRepo, nil, NewMockSLABreachRepository(), nil, notifService)

	breaches, err := service.CheckForBreaches(ctx, "monthly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(breaches) != 1 {
		t.Errorf("expected 1 breach, got %d", len(breaches))
	}
	time.Sleep(10 * time.Millisecond)
}

func TestSLAService_GetSystemSLAStatus_GetError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("db error")
	}
	service := NewSLAService(systemRepo, nil, nil, nil, nil, nil, nil)

	if _, err := service.GetSystemSLAStatus(context.Background(), 1, "monthly"); err == nil {
		t.Error("expected get system error")
	}
}

func TestSLAService_UpdateSystemSLATarget_GetError(t *testing.T) {
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("db error")
	}
	service := NewSLAService(systemRepo, nil, nil, nil, nil, nil, nil)

	if err := service.UpdateSystemSLATarget(context.Background(), 1, 99.9); err == nil {
		t.Error("expected get system error")
	}
}
