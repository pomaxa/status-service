package application

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
)

// DependencyService handles dependency-related use cases
type DependencyService struct {
	depRepo             domain.DependencyRepository
	logRepo             domain.StatusLogRepository
	notificationService *NotificationService
}

// NewDependencyService creates a new DependencyService
func NewDependencyService(depRepo domain.DependencyRepository, logRepo domain.StatusLogRepository) *DependencyService {
	return &DependencyService{
		depRepo: depRepo,
		logRepo: logRepo,
	}
}

// SetNotificationService sets the notification service for sending webhooks
func (s *DependencyService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

// CreateDependency creates a new dependency for a system
func (s *DependencyService) CreateDependency(ctx context.Context, systemID int64, name, description string) (*domain.Dependency, error) {
	dep, err := domain.NewDependency(systemID, name, description)
	if err != nil {
		return nil, fmt.Errorf("invalid dependency data: %w", err)
	}

	if err := s.depRepo.Create(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to create dependency: %w", err)
	}

	return dep, nil
}

// GetDependency retrieves a dependency by ID
func (s *DependencyService) GetDependency(ctx context.Context, id int64) (*domain.Dependency, error) {
	dep, err := s.depRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}
	return dep, nil
}

// GetDependenciesBySystem retrieves all dependencies for a system
func (s *DependencyService) GetDependenciesBySystem(ctx context.Context, systemID int64) ([]*domain.Dependency, error) {
	deps, err := s.depRepo.GetBySystemID(ctx, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}
	return deps, nil
}

// UpdateDependency updates dependency name and description
func (s *DependencyService) UpdateDependency(ctx context.Context, id int64, name, description string) (*domain.Dependency, error) {
	dep, err := s.depRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}
	if dep == nil {
		return nil, fmt.Errorf("dependency not found: %d", id)
	}

	if err := dep.Update(name, description); err != nil {
		return nil, fmt.Errorf("invalid update data: %w", err)
	}

	if err := s.depRepo.Update(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to update dependency: %w", err)
	}

	return dep, nil
}

// SetHeartbeat configures heartbeat checking for a dependency
func (s *DependencyService) SetHeartbeat(ctx context.Context, id int64, url string, interval int) (*domain.Dependency, error) {
	dep, err := s.depRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}
	if dep == nil {
		return nil, fmt.Errorf("dependency not found: %d", id)
	}

	if err := dep.SetHeartbeat(url, interval); err != nil {
		return nil, fmt.Errorf("invalid heartbeat config: %w", err)
	}

	if err := s.depRepo.Update(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to update dependency: %w", err)
	}

	return dep, nil
}

// ClearHeartbeat removes heartbeat checking for a dependency
func (s *DependencyService) ClearHeartbeat(ctx context.Context, id int64) (*domain.Dependency, error) {
	dep, err := s.depRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}
	if dep == nil {
		return nil, fmt.Errorf("dependency not found: %d", id)
	}

	dep.ClearHeartbeat()

	if err := s.depRepo.Update(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to update dependency: %w", err)
	}

	return dep, nil
}

// UpdateDependencyStatus changes dependency status with logging
func (s *DependencyService) UpdateDependencyStatus(ctx context.Context, id int64, statusStr, message string) (*domain.Dependency, error) {
	dep, err := s.depRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}
	if dep == nil {
		return nil, fmt.Errorf("dependency not found: %d", id)
	}

	newStatus, err := domain.NewStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	oldStatus := dep.Status

	if err := dep.UpdateStatus(newStatus); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	if err := s.depRepo.Update(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to save dependency: %w", err)
	}

	// Log the status change
	log := domain.NewStatusLog(nil, &id, oldStatus, newStatus, message, domain.SourceManual)
	if err := s.logRepo.Create(ctx, log); err != nil {
		fmt.Printf("failed to log status change: %v\n", err)
	}

	// Send notifications
	if s.notificationService != nil && oldStatus != newStatus {
		go s.notificationService.NotifyStatusChange(ctx, log)
	}

	return dep, nil
}

// DeleteDependency removes a dependency
func (s *DependencyService) DeleteDependency(ctx context.Context, id int64) error {
	if err := s.depRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete dependency: %w", err)
	}
	return nil
}

// GetDependencyLogs retrieves status logs for a dependency
func (s *DependencyService) GetDependencyLogs(ctx context.Context, id int64, limit int) ([]*domain.StatusLog, error) {
	logs, err := s.logRepo.GetByDependencyID(ctx, id, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	return logs, nil
}
