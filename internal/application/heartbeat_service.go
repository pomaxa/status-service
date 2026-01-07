package application

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
)

// HeartbeatService handles heartbeat checking
type HeartbeatService struct {
	depRepo domain.DependencyRepository
	logRepo domain.StatusLogRepository
	checker domain.HealthChecker
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
	healthy, err := s.checker.Check(ctx, dep.HeartbeatURL)
	if err != nil {
		return fmt.Errorf("check error: %w", err)
	}

	oldStatus := dep.Status
	var statusChanged bool

	if healthy {
		statusChanged = dep.RecordCheckSuccess()
	} else {
		statusChanged = dep.RecordCheckFailure()
	}

	// Always update to save LastCheck
	if err := s.depRepo.Update(ctx, dep); err != nil {
		return fmt.Errorf("failed to update dependency: %w", err)
	}

	// Log status change if happened
	if statusChanged {
		var message string
		if healthy {
			message = "Heartbeat check succeeded, service recovered"
		} else {
			message = fmt.Sprintf("Heartbeat check failed (%d consecutive failures)", dep.ConsecutiveFailures)
		}

		log := domain.NewStatusLog(nil, &dep.ID, oldStatus, dep.Status, message, domain.SourceHeartbeat)
		if err := s.logRepo.Create(ctx, log); err != nil {
			fmt.Printf("failed to log heartbeat status change: %v\n", err)
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
