package application

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
)

// HeartbeatService handles heartbeat checking
type HeartbeatService struct {
	depRepo             domain.DependencyRepository
	logRepo             domain.StatusLogRepository
	latencyRepo         domain.LatencyRepository
	checker             domain.HealthChecker
	notificationService *NotificationService
}

// NewHeartbeatService creates a new HeartbeatService
func NewHeartbeatService(
	depRepo domain.DependencyRepository,
	logRepo domain.StatusLogRepository,
	checker domain.HealthChecker,
) *HeartbeatService {
	return &HeartbeatService{
		depRepo: depRepo,
		logRepo: logRepo,
		checker: checker,
	}
}

// SetLatencyRepo sets the latency repository for recording history
func (s *HeartbeatService) SetLatencyRepo(repo domain.LatencyRepository) {
	s.latencyRepo = repo
}

// SetNotificationService sets the notification service for sending webhooks
func (s *HeartbeatService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

// CheckAllDependencies checks all dependencies with heartbeat configured
func (s *HeartbeatService) CheckAllDependencies(ctx context.Context) error {
	deps, err := s.depRepo.GetAllWithHeartbeat(ctx)
	if err != nil {
		return fmt.Errorf("failed to get dependencies: %w", err)
	}

	for _, dep := range deps {
		if dep.NeedsCheck() {
			if err := s.checkDependency(ctx, dep); err != nil {
				// Log error but continue checking other dependencies
				fmt.Printf("heartbeat check failed for dependency %d: %v\n", dep.ID, err)
			}
		}
	}

	return nil
}

// checkDependency performs health check on a single dependency
func (s *HeartbeatService) checkDependency(ctx context.Context, dep *domain.Dependency) error {
	healthy, latencyMs, err := s.checker.Check(ctx, dep.HeartbeatURL)
	if err != nil {
		return fmt.Errorf("check error: %w", err)
	}

	oldStatus := dep.Status
	var statusChanged bool

	if healthy {
		statusChanged = dep.RecordCheckSuccess(latencyMs)
	} else {
		statusChanged = dep.RecordCheckFailure(latencyMs)
	}

	// Record latency history
	if s.latencyRepo != nil {
		record := &domain.LatencyRecord{
			DependencyID: dep.ID,
			LatencyMs:    latencyMs,
			Success:      healthy,
			StatusCode:   200, // TODO: get actual status code from checker
		}
		if !healthy {
			record.StatusCode = 0
		}
		if err := s.latencyRepo.Record(ctx, record); err != nil {
			fmt.Printf("failed to record latency history: %v\n", err)
		}
	}

	// Always update to save LastCheck and LastLatency
	if err := s.depRepo.Update(ctx, dep); err != nil {
		return fmt.Errorf("failed to update dependency: %w", err)
	}

	// Log status change if happened
	if statusChanged {
		var message string
		if healthy {
			message = fmt.Sprintf("Heartbeat check succeeded, service recovered (latency: %dms)", latencyMs)
		} else {
			message = fmt.Sprintf("Heartbeat check failed (%d consecutive failures, latency: %dms)", dep.ConsecutiveFailures, latencyMs)
		}

		log := domain.NewStatusLog(nil, &dep.ID, oldStatus, dep.Status, message, domain.SourceHeartbeat)
		if err := s.logRepo.Create(ctx, log); err != nil {
			fmt.Printf("failed to log heartbeat status change: %v\n", err)
		}

		// Send notifications
		if s.notificationService != nil {
			go s.notificationService.NotifyStatusChange(ctx, log)
		}
	}

	return nil
}

// ForceCheck forces immediate health check for a specific dependency
func (s *HeartbeatService) ForceCheck(ctx context.Context, depID int64) (*domain.Dependency, error) {
	dep, err := s.depRepo.GetByID(ctx, depID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}
	if dep == nil {
		return nil, fmt.Errorf("dependency not found: %d", depID)
	}
	if !dep.HasHeartbeat() {
		return nil, fmt.Errorf("dependency has no heartbeat configured")
	}

	if err := s.checkDependency(ctx, dep); err != nil {
		return nil, err
	}

	return dep, nil
}
