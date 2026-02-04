package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
)

func TestNewDependencyService(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()

	service := NewDependencyService(depRepo, logRepo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestDependencyService_CreateDependency(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	dep, err := service.CreateDependency(context.Background(), 1, "Redis", "Cache layer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dep == nil {
		t.Fatal("expected non-nil dependency")
	}
	if dep.Name != "Redis" {
		t.Errorf("expected name 'Redis', got %q", dep.Name)
	}
	if dep.ID == 0 {
		t.Error("expected dependency ID to be set")
	}
}

func TestDependencyService_CreateDependency_EmptyName(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	_, err := service.CreateDependency(context.Background(), 1, "", "Description")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestDependencyService_CreateDependency_RepoError(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	depRepo.CreateFunc = func(ctx context.Context, d *domain.Dependency) error {
		return errors.New("database error")
	}
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	_, err := service.CreateDependency(context.Background(), 1, "Redis", "Description")
	if err == nil {
		t.Error("expected error from repository")
	}
}

func TestDependencyService_GetDependency(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	result, err := service.GetDependency(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil dependency")
	}
	if result.Name != "Redis" {
		t.Errorf("expected name 'Redis', got %q", result.Name)
	}
}

func TestDependencyService_GetDependenciesBySystem(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep1, _ := domain.NewDependency(1, "Redis", "Cache")
	dep1.ID = 1
	dep2, _ := domain.NewDependency(1, "PostgreSQL", "Database")
	dep2.ID = 2
	dep3, _ := domain.NewDependency(2, "MongoDB", "Other system")
	dep3.ID = 3
	depRepo.Dependencies[1] = dep1
	depRepo.Dependencies[2] = dep2
	depRepo.Dependencies[3] = dep3

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	deps, err := service.GetDependenciesBySystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestDependencyService_UpdateDependency(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Old description")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	result, err := service.UpdateDependency(context.Background(), 1, "Redis Cache", "New description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "Redis Cache" {
		t.Errorf("expected name 'Redis Cache', got %q", result.Name)
	}
	if result.Description != "New description" {
		t.Errorf("expected description 'New description', got %q", result.Description)
	}
}

func TestDependencyService_UpdateDependency_NotFound(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	_, err := service.UpdateDependency(context.Background(), 999, "Name", "Desc")
	if err == nil {
		t.Error("expected error for non-existent dependency")
	}
}

func TestDependencyService_SetHeartbeat(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	result, err := service.SetHeartbeat(context.Background(), 1, "https://redis.example.com/health", 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasHeartbeat() {
		t.Error("expected heartbeat to be configured")
	}
}

func TestDependencyService_SetHeartbeatConfig(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "API", "External API")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	config := domain.HeartbeatConfig{
		URL:          "https://api.example.com/health",
		Interval:     30,
		Method:       "POST",
		ExpectStatus: "201",
		ExpectBody:   "OK",
		Headers:      map[string]string{"Authorization": "Bearer token"},
	}

	result, err := service.SetHeartbeatConfig(context.Background(), 1, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasHeartbeat() {
		t.Error("expected heartbeat to be configured")
	}
	cfg := result.GetHeartbeatConfig()
	if cfg.Method != "POST" {
		t.Errorf("expected method 'POST', got %q", cfg.Method)
	}
}

func TestDependencyService_SetHeartbeatConfig_NotFound(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	_, err := service.SetHeartbeatConfig(context.Background(), 999, domain.HeartbeatConfig{URL: "http://test.com", Interval: 60})
	if err == nil {
		t.Error("expected error for non-existent dependency")
	}
}

func TestDependencyService_ClearHeartbeat(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.SetHeartbeatConfig(domain.HeartbeatConfig{URL: "https://test.com", Interval: 60})
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	result, err := service.ClearHeartbeat(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasHeartbeat() {
		t.Error("expected heartbeat to be cleared")
	}
}

func TestDependencyService_ClearHeartbeat_NotFound(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	_, err := service.ClearHeartbeat(context.Background(), 999)
	if err == nil {
		t.Error("expected error for non-existent dependency")
	}
}

func TestDependencyService_UpdateDependencyStatus(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	result, err := service.UpdateDependencyStatus(context.Background(), 1, "yellow", "High latency")
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

func TestDependencyService_UpdateDependencyStatus_InvalidStatus(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	_, err := service.UpdateDependencyStatus(context.Background(), 1, "invalid", "Bad status")
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestDependencyService_UpdateDependencyStatus_NotFound(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	_, err := service.UpdateDependencyStatus(context.Background(), 999, "green", "Test")
	if err == nil {
		t.Error("expected error for non-existent dependency")
	}
}

func TestDependencyService_DeleteDependency(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	err := service.DeleteDependency(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := depRepo.Dependencies[1]; exists {
		t.Error("expected dependency to be deleted")
	}
}

func TestDependencyService_GetDependencyLogs(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()

	depID := int64(1)
	logRepo.Logs = append(logRepo.Logs, &domain.StatusLog{
		ID:           1,
		DependencyID: &depID,
		OldStatus:    domain.StatusGreen,
		NewStatus:    domain.StatusRed,
	})

	service := NewDependencyService(depRepo, logRepo)

	logs, err := service.GetDependencyLogs(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestDependencyService_SetNotificationService(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	webhookRepo := NewMockWebhookRepository()
	systemRepo := NewMockSystemRepository()
	notifService := NewNotificationService(webhookRepo, systemRepo, depRepo)

	service.SetNotificationService(notifService)

	// Test that status change triggers notification (no panic)
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	_, err := service.UpdateDependencyStatus(context.Background(), 1, "red", "Test notification")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDependencyService_SetPropagationService(t *testing.T) {
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	systemRepo := NewMockSystemRepository()
	propagationService := NewStatusPropagationService(systemRepo, depRepo, logRepo)
	service.SetPropagationService(propagationService)
	// No panic means success
}

func TestDependencyService_UpdateDependencyStatus_PropagatesStatus(t *testing.T) {
	// Set up system
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system

	// Set up dependencies
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.Status = domain.StatusGreen
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	// Set up propagation service
	propagationService := NewStatusPropagationService(systemRepo, depRepo, logRepo)
	service.SetPropagationService(propagationService)

	// Update dependency status to red
	_, err := service.UpdateDependencyStatus(context.Background(), 1, "red", "Manual update")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dependency should be red
	if dep.Status != domain.StatusRed {
		t.Errorf("expected dependency status red, got %q", dep.Status)
	}

	// System should be red (propagated)
	if system.Status != domain.StatusRed {
		t.Errorf("expected system status to be propagated to red, got %q", system.Status)
	}

	// Should have 2 logs: one for dependency, one for system propagation
	if len(logRepo.Logs) != 2 {
		t.Errorf("expected 2 logs (dep + propagation), got %d", len(logRepo.Logs))
	}
}

func TestDependencyService_DeleteDependency_PropagatesStatus(t *testing.T) {
	// Set up system with red status
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusRed
	systemRepo.Systems[1] = system

	// Set up dependencies - one red, one green
	depRepo := NewMockDependencyRepository()
	depRed, _ := domain.NewDependency(1, "Redis", "Cache")
	depRed.ID = 1
	depRed.Status = domain.StatusRed
	depGreen, _ := domain.NewDependency(1, "PostgreSQL", "Database")
	depGreen.ID = 2
	depGreen.Status = domain.StatusGreen
	depRepo.Dependencies[1] = depRed
	depRepo.Dependencies[2] = depGreen

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	// Set up propagation service
	propagationService := NewStatusPropagationService(systemRepo, depRepo, logRepo)
	service.SetPropagationService(propagationService)

	// Delete the red dependency
	err := service.DeleteDependency(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// System should recover to green (only green dependency left)
	if system.Status != domain.StatusGreen {
		t.Errorf("expected system status to recover to green after deleting red dependency, got %q", system.Status)
	}

	// Should have 1 log for system recovery
	if len(logRepo.Logs) != 1 {
		t.Errorf("expected 1 log for system recovery, got %d", len(logRepo.Logs))
	}
}

func TestDependencyService_UpdateDependencyStatus_NoChangeNoPropagation(t *testing.T) {
	// Set up system
	systemRepo := NewMockSystemRepository()
	system, _ := domain.NewSystem("API", "API Service", "https://api.example.com", "team@example.com")
	system.ID = 1
	system.Status = domain.StatusGreen
	systemRepo.Systems[1] = system

	// Set up dependencies
	depRepo := NewMockDependencyRepository()
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	dep.Status = domain.StatusGreen // Already green
	depRepo.Dependencies[1] = dep

	logRepo := NewMockStatusLogRepository()
	service := NewDependencyService(depRepo, logRepo)

	// Set up propagation service
	propagationService := NewStatusPropagationService(systemRepo, depRepo, logRepo)
	service.SetPropagationService(propagationService)

	// Update dependency status to green (same status)
	_, err := service.UpdateDependencyStatus(context.Background(), 1, "green", "No change")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 1 log (the status change log is always created even if status is same)
	// But no propagation should happen
	if len(logRepo.Logs) != 1 {
		t.Errorf("expected 1 log (no propagation when status unchanged), got %d", len(logRepo.Logs))
	}

	// System should remain green
	if system.Status != domain.StatusGreen {
		t.Errorf("expected system status to remain green, got %q", system.Status)
	}
}
