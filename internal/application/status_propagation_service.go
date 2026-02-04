package application

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
)

// StatusPropagationService handles propagating dependency status changes to parent systems
type StatusPropagationService struct {
	systemRepo          domain.SystemRepository
	depRepo             domain.DependencyRepository
	logRepo             domain.StatusLogRepository
	notificationService *NotificationService
}

// NewStatusPropagationService creates a new StatusPropagationService
func NewStatusPropagationService(
	systemRepo domain.SystemRepository,
	depRepo domain.DependencyRepository,
	logRepo domain.StatusLogRepository,
) *StatusPropagationService {
	return &StatusPropagationService{
		systemRepo: systemRepo,
		depRepo:    depRepo,
		logRepo:    logRepo,
	}
}

// SetNotificationService sets the notification service for sending webhooks
func (s *StatusPropagationService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

// PropagateStatusToSystem updates a system's status based on its dependencies' statuses.
// Returns true if the system status was changed, false otherwise.
func (s *StatusPropagationService) PropagateStatusToSystem(ctx context.Context, systemID int64) (bool, error) {
	// Get the system
	system, err := s.systemRepo.GetByID(ctx, systemID)
	if err != nil {
		return false, fmt.Errorf("failed to get system: %w", err)
	}
	if system == nil {
		return false, fmt.Errorf("system not found: %d", systemID)
	}

	// Get all dependencies for the system
	deps, err := s.depRepo.GetBySystemID(ctx, systemID)
	if err != nil {
		return false, fmt.Errorf("failed to get dependencies: %w", err)
	}

	// If no dependencies, nothing to propagate
	if len(deps) == 0 {
		return false, nil
	}

	// Collect all dependency statuses
	statuses := make([]domain.Status, len(deps))
	for i, dep := range deps {
		statuses[i] = dep.Status
	}

	// Calculate aggregate status (worst-case)
	aggregateStatus := domain.MaxSeverityStatus(statuses)

	// Check if status changed
	oldStatus := system.Status
	if oldStatus == aggregateStatus {
		return false, nil
	}

	// Update system status
	if err := system.UpdateStatus(aggregateStatus); err != nil {
		return false, fmt.Errorf("failed to update system status: %w", err)
	}

	if err := s.systemRepo.Update(ctx, system); err != nil {
		return false, fmt.Errorf("failed to save system: %w", err)
	}

	// Create status log
	message := fmt.Sprintf("Status propagated from dependencies (worst-case: %s)", aggregateStatus)
	statusLog := domain.NewStatusLog(&systemID, nil, oldStatus, aggregateStatus, message, domain.SourcePropagation)
	if err := s.logRepo.Create(ctx, statusLog); err != nil {
		fmt.Printf("failed to log propagated status change: %v\n", err)
	}

	// Send notifications
	if s.notificationService != nil {
		go s.notificationService.NotifyStatusChange(ctx, statusLog)
	}

	return true, nil
}
