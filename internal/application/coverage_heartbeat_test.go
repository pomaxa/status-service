package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
	"time"
)

// ============= Heartbeat Service error-path coverage =============

func TestHeartbeatService_CheckAllDependencies_GetError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetWithHeartbeatFunc = func(ctx context.Context) ([]*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewHeartbeatService(depRepo, NewMockStatusLogRepository(), NewMockHealthChecker())

	if err := service.CheckAllDependencies(context.Background()); err == nil {
		t.Error("expected error from GetAllWithHeartbeat")
	}
}

func TestHeartbeatService_CheckDependency_CheckError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Error: errors.New("connection refused")}
	}
	service := NewHeartbeatService(depRepo, NewMockStatusLogRepository(), checker)

	// CheckAllDependencies swallows per-dependency check errors
	if err := service.CheckAllDependencies(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHeartbeatService_ForceCheck_CheckError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Error: errors.New("connection refused")}
	}
	service := NewHeartbeatService(depRepo, NewMockStatusLogRepository(), checker)

	if _, err := service.ForceCheck(context.Background(), 1); err == nil {
		t.Error("expected error propagated from checkDependency")
	}
}

func TestHeartbeatService_ForceCheck_GetError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.Dependency, error) {
		return nil, errors.New("db error")
	}
	service := NewHeartbeatService(depRepo, NewMockStatusLogRepository(), NewMockHealthChecker())

	if _, err := service.ForceCheck(context.Background(), 1); err == nil {
		t.Error("expected get error")
	}
}

func TestHeartbeatService_CheckDependency_UpdateError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep
	depRepo.UpdateFunc = func(ctx context.Context, d *domain.Dependency) error {
		return errors.New("update error")
	}

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Healthy: true, LatencyMs: 10, StatusCode: 200}
	}
	service := NewHeartbeatService(depRepo, NewMockStatusLogRepository(), checker)

	// ForceCheck surfaces the dependency update error
	if _, err := service.ForceCheck(context.Background(), 1); err == nil {
		t.Error("expected update error from checkDependency")
	}
}

func TestHeartbeatService_CheckDependency_LatencyRecordError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	latencyRepo := NewMockLatencyRepository()
	latencyRepo.RecordFunc = func(ctx context.Context, record *domain.LatencyRecord) error {
		return errors.New("record error")
	}

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Healthy: true, LatencyMs: 10, StatusCode: 200}
	}
	service := NewHeartbeatService(depRepo, NewMockStatusLogRepository(), checker)
	service.SetLatencyRepo(latencyRepo)

	// Latency record error is non-fatal
	if _, err := service.ForceCheck(context.Background(), 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHeartbeatService_CheckDependency_StatusChangeWithLogError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{
		URL:      "https://redis.example.com/health",
		Interval: 60,
	})
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	logRepo.CreateFunc = func(ctx context.Context, log *domain.StatusLog) error {
		return errors.New("log error")
	}

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Healthy: false, LatencyMs: 50, StatusCode: 503}
	}
	service := NewHeartbeatService(depRepo, logRepo, checker)

	// Status change triggers log creation, which fails (non-fatal)
	if _, err := service.ForceCheck(context.Background(), 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHeartbeatService_CheckDependency_StatusChangePropagatesAndNotifies(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{
		URL:      "https://redis.example.com/health",
		Interval: 60,
	})
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Healthy: false, LatencyMs: 50, StatusCode: 503}
	}

	service := NewHeartbeatService(depRepo, logRepo, checker)

	// Wire up notification + propagation services to cover those branches.
	webhookRepo := NewMockWebhookRepository()
	notifService := NewNotificationService(webhookRepo, systemRepo, depRepo)
	service.SetNotificationService(notifService)

	propService := NewStatusPropagationService(systemRepo, depRepo, logRepo)
	service.SetPropagationService(propService)

	if _, err := service.ForceCheck(ctx, 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Give the notification goroutine time to start.
	time.Sleep(10 * time.Millisecond)
}

// TestHeartbeatService_CheckDependency_UnverifiableRecordsFailure guards against
// the fail-open regression: when the checker cannot complete a check (URL/SSRF
// validation block, malformed request -> result.Error != nil), the dependency
// must NOT be left reporting its prior (green) status with a zero LastCheck —
// which both lies about health and re-probes every cycle. It should record a
// failure and advance LastCheck.
func TestHeartbeatService_CheckDependency_UnverifiableRecordsFailure(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	if dep.Status != domain.StatusGreen || !dep.LastCheck.IsZero() {
		t.Fatalf("precondition: expected fresh green dependency, got status=%s lastCheck=%v", dep.Status, dep.LastCheck)
	}

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Error: errors.New("blocked: SSRF validation failed")}
	}
	service := NewHeartbeatService(depRepo, NewMockStatusLogRepository(), checker)

	// CheckAllDependencies swallows the per-dependency error; the dependency
	// pointer is mutated in place.
	_ = service.CheckAllDependencies(context.Background())

	if dep.LastCheck.IsZero() {
		t.Error("LastCheck must advance on an unverifiable check (else NeedsCheck re-probes every cycle)")
	}
	if dep.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1", dep.ConsecutiveFailures)
	}
	if dep.Status == domain.StatusGreen {
		t.Error("status must leave green when the check could not be verified (fail-open bug)")
	}
}
