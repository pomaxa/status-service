package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"strings"
	"testing"
	"time"
)

// ============= StatusPropagation save/log/notify branches =============

func TestStatusPropagationService_SaveError(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system
	systemRepo.UpdateFunc = func(ctx context.Context, s *domain.System) error {
		return errors.New("save error")
	}

	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "DB", "")
	dep.ID = 1
	dep.Status = domain.StatusRed
	depRepo.Dependencies[1] = dep

	service := NewStatusPropagationService(systemRepo, depRepo, NewMockStatusLogRepository())

	if _, err := service.PropagateStatusToSystem(ctx, 1); err == nil {
		t.Error("expected save error")
	}
}

func TestStatusPropagationService_LogErrorAndNotify(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "", "", "")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system

	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "DB", "")
	dep.ID = 1
	dep.Status = domain.StatusRed
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	logRepo.CreateFunc = func(ctx context.Context, log *domain.StatusLog) error {
		return errors.New("log error") // non-fatal
	}

	service := NewStatusPropagationService(systemRepo, depRepo, logRepo)

	// Wire notification service to cover the notify branch.
	webhookRepo := NewMockWebhookRepository()
	notifService := NewNotificationService(webhookRepo, systemRepo, depRepo)
	service.SetNotificationService(notifService)

	changed, err := service.PropagateStatusToSystem(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected status to change")
	}
	time.Sleep(10 * time.Millisecond)
}

// ============= Dependency propagation-error branches =============

func TestDependencyService_UpdateDependencyStatus_PropagationError(t *testing.T) {
	ctx := context.Background()
	// Propagation service whose system lookup fails -> PropagateStatusToSystem errors.
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("system lookup error")
	}

	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(7, "DB", "")
	dep.ID = 1
	dep.Status = domain.StatusGreen
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	propService := NewStatusPropagationService(systemRepo, depRepo, logRepo)
	service.SetPropagationService(propService)

	// Status changes green->red, triggering propagation which errors (non-fatal).
	if _, err := service.UpdateDependencyStatus(ctx, 1, "red", "down"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDependencyService_DeleteDependency_PropagationError(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("system lookup error")
	}

	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(7, "DB", "")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	propService := NewStatusPropagationService(systemRepo, depRepo, logRepo)
	service.SetPropagationService(propService)

	// Delete triggers propagation (systemID > 0 from the dep) which errors (non-fatal).
	if err := service.DeleteDependency(ctx, 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ============= Heartbeat recovery + propagation-error branches =============

func TestHeartbeatService_CheckDependency_RecoveryMessage(t *testing.T) {
	ctx := context.Background()
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	dep.Status = domain.StatusRed // start unhealthy so a success is a recovery
	depRepo.Dependencies[1] = dep

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Healthy: true, LatencyMs: 12, StatusCode: 200}
	}
	service := NewHeartbeatService(depRepo, NewMockStatusLogRepository(), checker)

	if _, err := service.ForceCheck(ctx, 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dep.Status != domain.StatusGreen {
		t.Errorf("expected recovery to green, got %v", dep.Status)
	}
}

func TestHeartbeatService_CheckDependency_PropagationError(t *testing.T) {
	ctx := context.Background()
	systemRepo := NewMockSystemRepository()
	systemRepo.GetByIDFunc = func(ctx context.Context, id int64) (*domain.System, error) {
		return nil, errors.New("system lookup error")
	}

	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(7, "Redis", "")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://redis.example.com/health", Interval: 60})
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()

	checker := NewMockHealthChecker()
	checker.CheckWithConfigFunc = func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
		return domain.HealthCheckResult{Healthy: false, LatencyMs: 80, StatusCode: 500}
	}
	service := NewHeartbeatService(depRepo, logRepo, checker)

	propService := NewStatusPropagationService(systemRepo, depRepo, logRepo)
	service.SetPropagationService(propService)

	// Status changes green->yellow, propagation errors (non-fatal).
	if _, err := service.ForceCheck(ctx, 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ============= Telegram/Discord both system+dependency name branch =============

func TestNotificationService_formatTelegramPayload_SystemAndDependency(t *testing.T) {
	s := &NotificationService{}
	payload := &domain.NotificationPayload{
		Event:      domain.EventStatusChange,
		Timestamp:  time.Now(),
		System:     &domain.SystemInfo{ID: 1, Name: "API"},
		Dependency: &domain.DepInfo{ID: 2, Name: "DB"},
		NewStatus:  domain.StatusRed,
		Source:     "heartbeat",
	}
	body, err := s.formatTelegramPayload("https://api.telegram.org/bot1/sendMessage", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), "API / DB") {
		t.Errorf("expected combined entity name, got %s", string(body))
	}
}

func TestNotificationService_formatDiscordPayload_SystemAndDependency(t *testing.T) {
	s := &NotificationService{}
	payload := &domain.NotificationPayload{
		Event:      domain.EventStatusChange,
		Timestamp:  time.Now(),
		System:     &domain.SystemInfo{ID: 1, Name: "API"},
		Dependency: &domain.DepInfo{ID: 2, Name: "DB"},
		NewStatus:  domain.StatusRed,
		Source:     "heartbeat",
	}
	body, err := s.formatDiscordPayload(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), "API / DB") {
		t.Errorf("expected combined entity name, got %s", string(body))
	}
}
