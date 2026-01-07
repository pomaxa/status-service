package application

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
	"time"
)

// AnalyticsService handles analytics-related use cases
type AnalyticsService struct {
	analyticsRepo domain.AnalyticsRepository
	logRepo       domain.StatusLogRepository
}

// NewAnalyticsService creates a new AnalyticsService
func NewAnalyticsService(analyticsRepo domain.AnalyticsRepository, logRepo domain.StatusLogRepository) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		logRepo:       logRepo,
	}
}

// GetSystemAnalytics retrieves analytics for a system
func (s *AnalyticsService) GetSystemAnalytics(ctx context.Context, systemID int64, period string) (*domain.Analytics, error) {
	start, end := s.parsePeriod(period)

	analytics, err := s.analyticsRepo.GetUptimeBySystemID(ctx, systemID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get system analytics: %w", err)
	}

	return analytics, nil
}

// GetDependencyAnalytics retrieves analytics for a dependency
func (s *AnalyticsService) GetDependencyAnalytics(ctx context.Context, dependencyID int64, period string) (*domain.Analytics, error) {
	start, end := s.parsePeriod(period)

	analytics, err := s.analyticsRepo.GetUptimeByDependencyID(ctx, dependencyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency analytics: %w", err)
	}

	return analytics, nil
}

// GetOverallAnalytics retrieves aggregate analytics
func (s *AnalyticsService) GetOverallAnalytics(ctx context.Context, period string) (*domain.Analytics, error) {
	start, end := s.parsePeriod(period)

	analytics, err := s.analyticsRepo.GetOverallAnalytics(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall analytics: %w", err)
	}

	return analytics, nil
}

// GetSystemIncidents retrieves incidents for a system
func (s *AnalyticsService) GetSystemIncidents(ctx context.Context, systemID int64, period string) ([]domain.Incident, error) {
	start, end := s.parsePeriod(period)

	incidents, err := s.analyticsRepo.GetIncidentsBySystemID(ctx, systemID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get incidents: %w", err)
	}

	return incidents, nil
}

// GetDependencyIncidents retrieves incidents for a dependency
func (s *AnalyticsService) GetDependencyIncidents(ctx context.Context, dependencyID int64, period string) ([]domain.Incident, error) {
	start, end := s.parsePeriod(period)

	incidents, err := s.analyticsRepo.GetIncidentsByDependencyID(ctx, dependencyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get incidents: %w", err)
	}

	return incidents, nil
}

// GetAllLogs retrieves all status logs with limit
func (s *AnalyticsService) GetAllLogs(ctx context.Context, limit int) ([]*domain.StatusLog, error) {
	if limit <= 0 {
		limit = 100
	}

	logs, err := s.logRepo.GetAll(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	return logs, nil
}

// CreateLog creates a new status log entry (used for import)
func (s *AnalyticsService) CreateLog(ctx context.Context, log *domain.StatusLog) error {
	if err := s.logRepo.Create(ctx, log); err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	return nil
}

// parsePeriod converts period string to time range
func (s *AnalyticsService) parsePeriod(period string) (start, end time.Time) {
	end = time.Now()

	switch period {
	case "1h":
		start = end.Add(-1 * time.Hour)
	case "24h", "1d":
		start = end.Add(-24 * time.Hour)
	case "7d":
		start = end.Add(-7 * 24 * time.Hour)
	case "30d":
		start = end.Add(-30 * 24 * time.Hour)
	case "90d":
		start = end.Add(-90 * 24 * time.Hour)
	default:
		// Default to 24h
		start = end.Add(-24 * time.Hour)
	}

	return start, end
}
