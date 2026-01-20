package application

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
)

// SystemService handles system-related use cases
type SystemService struct {
	systemRepo          domain.SystemRepository
	logRepo             domain.StatusLogRepository
	notificationService *NotificationService
}

// NewSystemService creates a new SystemService
func NewSystemService(systemRepo domain.SystemRepository, logRepo domain.StatusLogRepository) *SystemService {
	return &SystemService{
		systemRepo: systemRepo,
		logRepo:    logRepo,
	}
}

// SetNotificationService sets the notification service for sending webhooks
func (s *SystemService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

// CreateSystem creates a new system
func (s *SystemService) CreateSystem(ctx context.Context, name, description, url, owner string) (*domain.System, error) {
	system, err := domain.NewSystem(name, description, url, owner)
	if err != nil {
		return nil, fmt.Errorf("invalid system data: %w", err)
	}

	if err := s.systemRepo.Create(ctx, system); err != nil {
		return nil, fmt.Errorf("failed to create system: %w", err)
	}

	return system, nil
}

// GetSystem retrieves a system by ID
func (s *SystemService) GetSystem(ctx context.Context, id int64) (*domain.System, error) {
	system, err := s.systemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system: %w", err)
	}
	return system, nil
}

// GetAllSystems retrieves all systems
func (s *SystemService) GetAllSystems(ctx context.Context) ([]*domain.System, error) {
	systems, err := s.systemRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get systems: %w", err)
	}
	return systems, nil
}

// UpdateSystem updates system name, description, url and owner
func (s *SystemService) UpdateSystem(ctx context.Context, id int64, name, description, url, owner string) (*domain.System, error) {
	system, err := s.systemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system: %w", err)
	}
	if system == nil {
		return nil, fmt.Errorf("system not found: %d", id)
	}

	if err := system.Update(name, description, url, owner); err != nil {
		return nil, fmt.Errorf("invalid update data: %w", err)
	}

	if err := s.systemRepo.Update(ctx, system); err != nil {
		return nil, fmt.Errorf("failed to update system: %w", err)
	}

	return system, nil
}

// UpdateSystemStatus changes system status with logging
func (s *SystemService) UpdateSystemStatus(ctx context.Context, id int64, statusStr, message string) (*domain.System, error) {
	system, err := s.systemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system: %w", err)
	}
	if system == nil {
		return nil, fmt.Errorf("system not found: %d", id)
	}

	newStatus, err := domain.NewStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	oldStatus := system.Status

	if err := system.UpdateStatus(newStatus); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	if err := s.systemRepo.Update(ctx, system); err != nil {
		return nil, fmt.Errorf("failed to save system: %w", err)
	}

	// Log the status change
	statusLog := domain.NewStatusLog(&id, nil, oldStatus, newStatus, message, domain.SourceManual)
	if err := s.logRepo.Create(ctx, statusLog); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("failed to log status change: %v\n", err)
	}

	// Send notifications
	if s.notificationService != nil && oldStatus != newStatus {
		go s.notificationService.NotifyStatusChange(ctx, statusLog)
	}

	return system, nil
}

// DeleteSystem removes a system
func (s *SystemService) DeleteSystem(ctx context.Context, id int64) error {
	if err := s.systemRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete system: %w", err)
	}
	return nil
}

// GetSystemLogs retrieves status logs for a system
func (s *SystemService) GetSystemLogs(ctx context.Context, id int64, limit int) ([]*domain.StatusLog, error) {
	logs, err := s.logRepo.GetBySystemID(ctx, id, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	return logs, nil
}
