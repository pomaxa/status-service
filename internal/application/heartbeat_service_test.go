package application

import (
	"context"
	"status-incident/internal/domain"
	"testing"
)

func TestNewHeartbeatService(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	checker := NewMockHealthChecker()

	service := NewHeartbeatService(depRepo, logRepo, checker)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestHeartbeatService_SetLatencyRepo(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	checker := NewMockHealthChecker()
	service := NewHeartbeatService(depRepo, logRepo, checker)

	latencyRepo := NewMockLatencyRepository()
	service.SetLatencyRepo(latencyRepo)
	// No panic means success
}

func TestHeartbeatService_SetNotificationService(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	checker := NewMockHealthChecker()
	service := NewHeartbeatService(depRepo, logRepo, checker)

	webhookRepo := NewMockWebhookRepository()
	systemRepo := NewMockSystemRepository()
	notifService := NewNotificationService(webhookRepo, systemRepo, depRepo)
	service.SetNotificationService(notifService)
	// No panic means success
}

func TestHeartbeatService_CheckAllDependencies_NoDeps(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	checker := NewMockHealthChecker()
	service := NewHeartbeatService(depRepo, logRepo, checker)

	err := service.CheckAllDependencies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHeartbeatService_CheckAllDependencies_WithDeps(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{
			Healthy:    true,
			LatencyMs:  25,
			StatusCode: 200,
		}
	}

	service := NewHeartbeatService(depRepo, logRepo, checker)

	err := service.CheckAllDependencies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHeartbeatService_CheckAllDependencies_WithLatencyRepo(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	latencyRepo := NewMockLatencyRepository()

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{
			Healthy:    true,
			LatencyMs:  30,
			StatusCode: 200,
		}
	}

	service := NewHeartbeatService(depRepo, logRepo, checker)
	service.SetLatencyRepo(latencyRepo)

	err := service.CheckAllDependencies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that latency was recorded
	if len(latencyRepo.Records) != 1 {
		t.Errorf("expected 1 latency record, got %d", len(latencyRepo.Records))
	}
}

func TestHeartbeatService_CheckAllDependencies_FailingCheck(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{
			Healthy:    false,
			LatencyMs:  100,
			StatusCode: 500,
		}
	}

	service := NewHeartbeatService(depRepo, logRepo, checker)

	err := service.CheckAllDependencies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dependency should have recorded failure
	if dep.ConsecutiveFailures != 1 {
		t.Errorf("expected 1 consecutive failure, got %d", dep.ConsecutiveFailures)
	}
}

func TestHeartbeatService_ForceCheck(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{
			Healthy:    true,
			LatencyMs:  15,
			StatusCode: 200,
		}
	}

	service := NewHeartbeatService(depRepo, logRepo, checker)

	result, err := service.ForceCheck(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil dependency")
	}
	if result.LastLatency != 15 {
		t.Errorf("expected last latency 15, got %d", result.LastLatency)
	}
}

func TestHeartbeatService_ForceCheck_NotFound(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	checker := NewMockHealthChecker()
	service := NewHeartbeatService(depRepo, logRepo, checker)

	_, err := service.ForceCheck(context.Background(), 999)
	if err == nil {
		t.Error("expected error for non-existent dependency")
	}
}

func TestHeartbeatService_ForceCheck_NoHeartbeat(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	// No heartbeat configured
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	checker := NewMockHealthChecker()
	service := NewHeartbeatService(depRepo, logRepo, checker)

	_, err := service.ForceCheck(context.Background(), 1)
	if err == nil {
		t.Error("expected error for dependency without heartbeat")
	}
}

func TestHeartbeatService_CheckWithStatusChange(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.Status = domain.StatusGreen
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	// Simulate 2 consecutive failures already
	dep.ConsecutiveFailures = 2
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{
			Healthy:    false,
			LatencyMs:  0,
			StatusCode: 503,
		}
	}

	service := NewHeartbeatService(depRepo, logRepo, checker)

	err := service.CheckAllDependencies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After 3 failures, status should change to red and log should be created
	if dep.Status != domain.StatusRed {
		t.Errorf("expected status red after 3 failures, got %q", dep.Status)
	}
	if len(logRepo.Logs) != 1 {
		t.Errorf("expected 1 log for status change, got %d", len(logRepo.Logs))
	}
}
