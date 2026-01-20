package application

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
	"time"
)

// LatencyService handles latency analytics
type LatencyService struct {
	latencyRepo domain.LatencyRepository
	depRepo     domain.DependencyRepository
}

// NewLatencyService creates a new LatencyService
func NewLatencyService(latencyRepo domain.LatencyRepository, depRepo domain.DependencyRepository) *LatencyService {
	return &LatencyService{
		latencyRepo: latencyRepo,
		depRepo:     depRepo,
	}
}

// GetDependencyLatencyStats retrieves latency statistics for a dependency
func (s *LatencyService) GetDependencyLatencyStats(ctx context.Context, dependencyID int64, period string) (*domain.LatencyStats, error) {
	dep, err := s.depRepo.GetByID(ctx, dependencyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}
	if dep == nil {
		return nil, fmt.Errorf("dependency not found")
	}

	start, end := parsePeriod(period)
	intervalMinutes := getIntervalMinutes(period)

	stats, err := s.latencyRepo.GetStats(ctx, dependencyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats.DependencyName = dep.Name
	stats.Period = period

	// Get data points for chart
	dataPoints, err := s.latencyRepo.GetAggregated(ctx, dependencyID, start, end, intervalMinutes)
	if err == nil {
		stats.DataPoints = dataPoints
	}

	return stats, nil
}

// GetDependencyUptimeHeatmap retrieves uptime heatmap for a dependency
func (s *LatencyService) GetDependencyUptimeHeatmap(ctx context.Context, dependencyID int64, days int) ([]domain.UptimePoint, error) {
	if days <= 0 {
		days = 90
	}
	return s.latencyRepo.GetDailyUptime(ctx, dependencyID, days)
}

// GetDependencyLatencyChart retrieves latency chart data
func (s *LatencyService) GetDependencyLatencyChart(ctx context.Context, dependencyID int64, period string) ([]domain.LatencyPoint, error) {
	start, end := parsePeriod(period)
	intervalMinutes := getIntervalMinutes(period)
	return s.latencyRepo.GetAggregated(ctx, dependencyID, start, end, intervalMinutes)
}

// CleanupOldRecords removes records older than specified days
func (s *LatencyService) CleanupOldRecords(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 90 // default 90 days retention
	}
	olderThan := time.Now().AddDate(0, 0, -retentionDays)
	return s.latencyRepo.Cleanup(ctx, olderThan)
}

// parsePeriod converts period string to time range
func parsePeriod(period string) (start, end time.Time) {
	end = time.Now()

	switch period {
	case "1h":
		start = end.Add(-1 * time.Hour)
	case "6h":
		start = end.Add(-6 * time.Hour)
	case "24h", "1d":
		start = end.Add(-24 * time.Hour)
	case "7d":
		start = end.Add(-7 * 24 * time.Hour)
	case "30d":
		start = end.Add(-30 * 24 * time.Hour)
	case "90d":
		start = end.Add(-90 * 24 * time.Hour)
	default:
		start = end.Add(-24 * time.Hour)
	}

	return start, end
}

// getIntervalMinutes returns aggregation interval based on period
func getIntervalMinutes(period string) int {
	switch period {
	case "1h":
		return 1  // 1 minute intervals
	case "6h":
		return 5  // 5 minute intervals
	case "24h", "1d":
		return 15 // 15 minute intervals
	case "7d":
		return 60 // 1 hour intervals
	case "30d":
		return 360 // 6 hour intervals
	case "90d":
		return 1440 // 1 day intervals
	default:
		return 15
	}
}
