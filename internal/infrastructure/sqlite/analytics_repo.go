package sqlite

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
	"time"
)

// AnalyticsRepo implements domain.AnalyticsRepository
type AnalyticsRepo struct {
	db      *DB
	logRepo *LogRepo
}

// NewAnalyticsRepo creates a new AnalyticsRepo
func NewAnalyticsRepo(db *DB) *AnalyticsRepo {
	return &AnalyticsRepo{
		db:      db,
		logRepo: NewLogRepo(db),
	}
}

// GetIncidentsBySystemID calculates incident periods for a system within time range
func (r *AnalyticsRepo) GetIncidentsBySystemID(ctx context.Context, systemID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
	logs, err := r.logRepo.GetSystemLogsByTimeRange(ctx, systemID, start, end)
	if err != nil {
		return nil, err
	}

	return r.calculateIncidents(logs, &systemID, nil), nil
}

// GetIncidentsByDependencyID calculates incident periods for a dependency within time range
func (r *AnalyticsRepo) GetIncidentsByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
	logs, err := r.logRepo.GetDependencyLogsByTimeRange(ctx, dependencyID, start, end)
	if err != nil {
		return nil, err
	}

	return r.calculateIncidents(logs, nil, &dependencyID), nil
}

// GetUptimeBySystemID calculates uptime metrics for a system
func (r *AnalyticsRepo) GetUptimeBySystemID(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
	// Get system info
	var name string
	err := r.db.QueryRowContext(ctx, "SELECT name FROM systems WHERE id = ?", systemID).Scan(&name)
	if err != nil {
		return nil, fmt.Errorf("failed to get system: %w", err)
	}

	logs, err := r.logRepo.GetSystemLogsByTimeRange(ctx, systemID, start, end)
	if err != nil {
		return nil, err
	}

	incidents := r.calculateIncidents(logs, &systemID, nil)

	return r.buildAnalytics(systemID, "system", name, start, end, logs, incidents), nil
}

// GetUptimeByDependencyID calculates uptime metrics for a dependency
func (r *AnalyticsRepo) GetUptimeByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.Analytics, error) {
	// Get dependency info
	var name string
	err := r.db.QueryRowContext(ctx, "SELECT name FROM dependencies WHERE id = ?", dependencyID).Scan(&name)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}

	logs, err := r.logRepo.GetDependencyLogsByTimeRange(ctx, dependencyID, start, end)
	if err != nil {
		return nil, err
	}

	incidents := r.calculateIncidents(logs, nil, &dependencyID)

	return r.buildAnalytics(dependencyID, "dependency", name, start, end, logs, incidents), nil
}

// GetOverallAnalytics returns aggregate analytics for all systems
// It calculates metrics per-system and then averages them to ensure correlation
func (r *AnalyticsRepo) GetOverallAnalytics(ctx context.Context, start, end time.Time) (*domain.Analytics, error) {
	// Get all system IDs
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM systems")
	if err != nil {
		return nil, fmt.Errorf("failed to get systems: %w", err)
	}
	defer rows.Close()

	var systemIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan system id: %w", err)
		}
		systemIDs = append(systemIDs, id)
	}

	if len(systemIDs) == 0 {
		// No systems, return 100% uptime
		return &domain.Analytics{
			EntityID:            0,
			EntityType:          "overall",
			EntityName:          "All Systems",
			PeriodStart:         start,
			PeriodEnd:           end,
			UptimePercent:       100,
			AvailabilityPercent: 100,
		}, nil
	}

	// Calculate metrics for each system and aggregate
	var totalUptime, totalAvailability float64
	var totalIncidents, resolvedIncidents, ongoingIncidents int
	var totalDowntime, totalUnavailable, longestIncident time.Duration

	for _, sysID := range systemIDs {
		analytics, err := r.GetUptimeBySystemID(ctx, sysID, start, end)
		if err != nil {
			continue // Skip systems that fail
		}

		totalUptime += analytics.UptimePercent
		totalAvailability += analytics.AvailabilityPercent
		totalIncidents += analytics.TotalIncidents
		resolvedIncidents += analytics.ResolvedIncidents
		ongoingIncidents += analytics.OngoingIncidents
		totalDowntime += analytics.TotalDowntime
		totalUnavailable += analytics.TotalUnavailable
		if analytics.LongestIncident > longestIncident {
			longestIncident = analytics.LongestIncident
		}
	}

	n := float64(len(systemIDs))
	avgUptime := totalUptime / n
	avgAvailability := totalAvailability / n

	totalDuration := end.Sub(start)
	var period string
	hours := totalDuration.Hours()
	switch {
	case hours <= 1:
		period = "1h"
	case hours <= 24:
		period = "24h"
	case hours <= 24*7:
		period = "7d"
	case hours <= 24*30:
		period = "30d"
	default:
		period = fmt.Sprintf("%.0fd", hours/24)
	}

	return &domain.Analytics{
		EntityID:            0,
		EntityType:          "overall",
		EntityName:          "All Systems",
		Period:              period,
		PeriodStart:         start,
		PeriodEnd:           end,
		TotalIncidents:      totalIncidents,
		ResolvedIncidents:   resolvedIncidents,
		OngoingIncidents:    ongoingIncidents,
		TotalDowntime:       totalDowntime,
		TotalUnavailable:    totalUnavailable,
		LongestIncident:     longestIncident,
		UptimePercent:       avgUptime,
		AvailabilityPercent: avgAvailability,
	}, nil
}

func (r *AnalyticsRepo) calculateIncidents(logs []*domain.StatusLog, systemID, dependencyID *int64) []domain.IncidentPeriod {
	var incidents []domain.IncidentPeriod
	var currentIncident *domain.IncidentPeriod

	for _, log := range logs {
		if log.IsIncidentStart() {
			// Start new incident period
			currentIncident = &domain.IncidentPeriod{
				SystemID:     systemID,
				DependencyID: dependencyID,
				StartedAt:    log.CreatedAt,
				MaxSeverity:  log.NewStatus,
				LogCount:     1,
			}
		} else if currentIncident != nil {
			currentIncident.LogCount++

			// Update max severity
			if log.NewStatus.Severity() > currentIncident.MaxSeverity.Severity() {
				currentIncident.MaxSeverity = log.NewStatus
			}

			if log.IsIncidentEnd() {
				// End incident period
				endTime := log.CreatedAt
				currentIncident.EndedAt = &endTime
				currentIncident.Duration = endTime.Sub(currentIncident.StartedAt)
				incidents = append(incidents, *currentIncident)
				currentIncident = nil
			}
		}
	}

	// Add ongoing incident period if exists
	if currentIncident != nil {
		incidents = append(incidents, *currentIncident)
	}

	return incidents
}

func (r *AnalyticsRepo) buildAnalytics(entityID int64, entityType, entityName string, start, end time.Time, logs []*domain.StatusLog, incidents []domain.IncidentPeriod) *domain.Analytics {
	totalDuration := end.Sub(start)

	var totalDowntime, totalUnavailable, longestIncident time.Duration
	var resolvedIncidents, ongoingIncidents int

	for _, inc := range incidents {
		duration := inc.GetDuration()

		// Cap duration at period bounds
		if inc.StartedAt.Before(start) {
			duration -= start.Sub(inc.StartedAt)
		}
		if inc.EndedAt != nil && inc.EndedAt.After(end) {
			duration -= inc.EndedAt.Sub(end)
		} else if inc.EndedAt == nil && time.Now().After(end) {
			duration = end.Sub(inc.StartedAt)
			if inc.StartedAt.Before(start) {
				duration = totalDuration
			}
		}

		totalDowntime += duration

		if inc.MaxSeverity == domain.StatusRed {
			totalUnavailable += duration
		}

		if duration > longestIncident {
			longestIncident = duration
		}

		if inc.IsResolved() {
			resolvedIncidents++
		} else {
			ongoingIncidents++
		}
	}

	greenDuration := totalDuration - totalDowntime
	if greenDuration < 0 {
		greenDuration = 0
	}

	availableDuration := totalDuration - totalUnavailable

	analytics := &domain.Analytics{
		EntityID:            entityID,
		EntityType:          entityType,
		EntityName:          entityName,
		PeriodStart:         start,
		PeriodEnd:           end,
		TotalIncidents:      len(incidents),
		ResolvedIncidents:   resolvedIncidents,
		OngoingIncidents:    ongoingIncidents,
		TotalDowntime:       totalDowntime,
		TotalUnavailable:    totalUnavailable,
		MTTR:                domain.CalculateMTTR(incidents),
		LongestIncident:     longestIncident,
		UptimePercent:       domain.CalculateUptime(greenDuration, totalDuration),
		AvailabilityPercent: domain.CalculateUptime(availableDuration, totalDuration),
	}

	// Determine period string
	hours := totalDuration.Hours()
	switch {
	case hours <= 24:
		analytics.Period = "24h"
	case hours <= 24*7:
		analytics.Period = "7d"
	case hours <= 24*30:
		analytics.Period = "30d"
	default:
		analytics.Period = fmt.Sprintf("%.0fd", hours/24)
	}

	return analytics
}
