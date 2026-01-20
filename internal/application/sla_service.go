package application

import (
	"context"
	"fmt"
	"status-incident/internal/domain"
	"time"
)

// SLAService handles SLA report generation and breach detection
type SLAService struct {
	systemRepo    domain.SystemRepository
	depRepo       domain.DependencyRepository
	analyticsRepo domain.AnalyticsRepository
	reportRepo    domain.SLAReportRepository
	breachRepo    domain.SLABreachRepository
	latencyRepo   domain.LatencyRepository
	notifService  *NotificationService
}

// NewSLAService creates a new SLAService
func NewSLAService(
	systemRepo domain.SystemRepository,
	depRepo domain.DependencyRepository,
	analyticsRepo domain.AnalyticsRepository,
	reportRepo domain.SLAReportRepository,
	breachRepo domain.SLABreachRepository,
	latencyRepo domain.LatencyRepository,
	notifService *NotificationService,
) *SLAService {
	return &SLAService{
		systemRepo:    systemRepo,
		depRepo:       depRepo,
		analyticsRepo: analyticsRepo,
		reportRepo:    reportRepo,
		breachRepo:    breachRepo,
		latencyRepo:   latencyRepo,
		notifService:  notifService,
	}
}

// GenerateReport creates an SLA report for the specified period
func (s *SLAService) GenerateReport(ctx context.Context, title, period, generatedBy string) (*domain.SLAReport, error) {
	start, end := s.parsePeriod(period)
	return s.GenerateCustomReport(ctx, title, period, start, end, generatedBy)
}

// GenerateCustomReport creates an SLA report for a custom time range
func (s *SLAService) GenerateCustomReport(ctx context.Context, title, period string, start, end time.Time, generatedBy string) (*domain.SLAReport, error) {
	systems, err := s.systemRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get systems: %w", err)
	}

	report := domain.NewSLAReport(title, period, start, end, generatedBy)

	for _, system := range systems {
		systemReport, err := s.generateSystemReport(ctx, system, start, end)
		if err != nil {
			// Log error but continue with other systems
			continue
		}
		report.AddSystemReport(*systemReport)
	}

	report.CalculateOverall()

	// Save report
	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	return report, nil
}

// generateSystemReport creates SLA metrics for a single system
func (s *SLAService) generateSystemReport(ctx context.Context, system *domain.System, start, end time.Time) (*domain.SystemSLAReport, error) {
	// Get uptime analytics
	analytics, err := s.analyticsRepo.GetUptimeBySystemID(ctx, system.ID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get system analytics: %w", err)
	}

	slaTarget := system.GetSLATarget()
	uptimePercent := analytics.UptimePercent
	slaMet := uptimePercent >= slaTarget

	// Get dependencies
	deps, err := s.depRepo.GetBySystemID(ctx, system.ID)
	if err != nil {
		deps = nil // Continue without dependencies
	}

	var depReports []domain.DependencySLAReport
	for _, dep := range deps {
		depReport, err := s.generateDependencyReport(ctx, dep, start, end)
		if err != nil {
			continue
		}
		depReports = append(depReports, *depReport)
	}

	// Calculate MTTR from incident periods
	incidents, _ := s.analyticsRepo.GetIncidentsBySystemID(ctx, system.ID, start, end)
	mttr := s.calculateMTTR(incidents)
	longestOutage := s.findLongestOutage(incidents)
	totalDowntime := s.calculateTotalDowntime(incidents)

	return &domain.SystemSLAReport{
		SystemID:          system.ID,
		SystemName:        system.Name,
		Owner:             system.Owner,
		SLATarget:         slaTarget,
		UptimePercent:     uptimePercent,
		AvailabilityPercent: analytics.AvailabilityPercent,
		TotalIncidents:    analytics.TotalIncidents,
		ResolvedIncidents: analytics.ResolvedIncidents,
		TotalDowntime:     totalDowntime,
		LongestOutage:     longestOutage,
		MTTR:              mttr,
		SLAMet:            slaMet,
		SLADelta:          uptimePercent - slaTarget,
		StatusSummary:     domain.GetStatusSummary(uptimePercent, slaTarget),
		DependencyReports: depReports,
	}, nil
}

// generateDependencyReport creates SLA metrics for a dependency
func (s *SLAService) generateDependencyReport(ctx context.Context, dep *domain.Dependency, start, end time.Time) (*domain.DependencySLAReport, error) {
	// Get dependency analytics
	analytics, err := s.analyticsRepo.GetUptimeByDependencyID(ctx, dep.ID, start, end)
	if err != nil {
		return nil, err
	}

	report := &domain.DependencySLAReport{
		DependencyID:       dep.ID,
		DependencyName:     dep.Name,
		UptimePercent:      analytics.UptimePercent,
		AvailabilityPercent: analytics.AvailabilityPercent,
	}

	// Get latency stats if available
	if s.latencyRepo != nil {
		stats, err := s.latencyRepo.GetStats(ctx, dep.ID, start, end)
		if err == nil && stats != nil {
			report.TotalChecks = stats.TotalChecks
			report.FailedChecks = stats.FailedChecks
			report.AvgLatencyMs = stats.AvgLatencyMs
			report.P95LatencyMs = stats.P95LatencyMs
			report.P99LatencyMs = stats.P99LatencyMs
		}
	}

	return report, nil
}

// calculateMTTR calculates Mean Time To Recovery from incidents
func (s *SLAService) calculateMTTR(incidents []domain.IncidentPeriod) time.Duration {
	if len(incidents) == 0 {
		return 0
	}

	var totalDuration time.Duration
	resolved := 0

	for _, inc := range incidents {
		if inc.EndedAt != nil {
			totalDuration += inc.EndedAt.Sub(inc.StartedAt)
			resolved++
		}
	}

	if resolved == 0 {
		return 0
	}

	return totalDuration / time.Duration(resolved)
}

// findLongestOutage finds the longest incident duration
func (s *SLAService) findLongestOutage(incidents []domain.IncidentPeriod) time.Duration {
	var longest time.Duration

	for _, inc := range incidents {
		var duration time.Duration
		if inc.EndedAt != nil {
			duration = inc.EndedAt.Sub(inc.StartedAt)
		} else {
			duration = time.Since(inc.StartedAt)
		}
		if duration > longest {
			longest = duration
		}
	}

	return longest
}

// calculateTotalDowntime sums up all incident durations
func (s *SLAService) calculateTotalDowntime(incidents []domain.IncidentPeriod) time.Duration {
	var total time.Duration

	for _, inc := range incidents {
		if inc.EndedAt != nil {
			total += inc.EndedAt.Sub(inc.StartedAt)
		} else {
			total += time.Since(inc.StartedAt)
		}
	}

	return total
}

// GetReport retrieves an SLA report by ID
func (s *SLAService) GetReport(ctx context.Context, id int64) (*domain.SLAReport, error) {
	return s.reportRepo.GetByID(ctx, id)
}

// GetAllReports retrieves all SLA reports
func (s *SLAService) GetAllReports(ctx context.Context, limit int) ([]*domain.SLAReport, error) {
	return s.reportRepo.GetAll(ctx, limit)
}

// DeleteReport removes an SLA report
func (s *SLAService) DeleteReport(ctx context.Context, id int64) error {
	return s.reportRepo.Delete(ctx, id)
}

// CheckForBreaches checks all systems for SLA breaches and records them
func (s *SLAService) CheckForBreaches(ctx context.Context, period string) ([]*domain.SLABreachEvent, error) {
	start, end := s.parsePeriod(period)

	systems, err := s.systemRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get systems: %w", err)
	}

	var breaches []*domain.SLABreachEvent

	for _, system := range systems {
		analytics, err := s.analyticsRepo.GetUptimeBySystemID(ctx, system.ID, start, end)
		if err != nil {
			continue
		}

		slaTarget := system.GetSLATarget()

		// Check uptime breach
		if analytics.UptimePercent < slaTarget {
			breach := &domain.SLABreachEvent{
				SystemID:    system.ID,
				SystemName:  system.Name,
				BreachType:  "uptime",
				SLATarget:   slaTarget,
				ActualValue: analytics.UptimePercent,
				Period:      period,
				PeriodStart: start,
				PeriodEnd:   end,
				DetectedAt:  time.Now(),
			}

			if err := s.breachRepo.Create(ctx, breach); err == nil {
				breaches = append(breaches, breach)

				// Send notification if available
				if s.notifService != nil {
					s.notifService.NotifySLABreach(ctx, breach)
				}
			}
		}
	}

	return breaches, nil
}

// GetBreaches retrieves SLA breaches
func (s *SLAService) GetBreaches(ctx context.Context, limit int) ([]*domain.SLABreachEvent, error) {
	return s.breachRepo.GetAll(ctx, limit)
}

// GetUnacknowledgedBreaches retrieves unacknowledged breaches
func (s *SLAService) GetUnacknowledgedBreaches(ctx context.Context) ([]*domain.SLABreachEvent, error) {
	return s.breachRepo.GetUnacknowledged(ctx)
}

// GetSystemBreaches retrieves breaches for a system
func (s *SLAService) GetSystemBreaches(ctx context.Context, systemID int64, limit int) ([]*domain.SLABreachEvent, error) {
	return s.breachRepo.GetBySystemID(ctx, systemID, limit)
}

// AcknowledgeBreach marks a breach as acknowledged
func (s *SLAService) AcknowledgeBreach(ctx context.Context, breachID int64, ackedBy string) error {
	return s.breachRepo.Acknowledge(ctx, breachID, ackedBy)
}

// GetSystemSLAStatus returns current SLA status for a system
func (s *SLAService) GetSystemSLAStatus(ctx context.Context, systemID int64, period string) (*domain.SystemSLAReport, error) {
	system, err := s.systemRepo.GetByID(ctx, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get system: %w", err)
	}
	if system == nil {
		return nil, fmt.Errorf("system not found")
	}

	start, end := s.parsePeriod(period)
	return s.generateSystemReport(ctx, system, start, end)
}

// UpdateSystemSLATarget updates the SLA target for a system
func (s *SLAService) UpdateSystemSLATarget(ctx context.Context, systemID int64, target float64) error {
	system, err := s.systemRepo.GetByID(ctx, systemID)
	if err != nil {
		return fmt.Errorf("failed to get system: %w", err)
	}
	if system == nil {
		return fmt.Errorf("system not found")
	}

	system.SetSLATarget(target)
	return s.systemRepo.Update(ctx, system)
}

// parsePeriod converts period string to time range
func (s *SLAService) parsePeriod(period string) (start, end time.Time) {
	end = time.Now()

	switch period {
	case "daily", "1d":
		start = end.Add(-24 * time.Hour)
	case "weekly", "7d":
		start = end.Add(-7 * 24 * time.Hour)
	case "monthly", "30d":
		start = end.Add(-30 * 24 * time.Hour)
	case "quarterly", "90d":
		start = end.Add(-90 * 24 * time.Hour)
	case "yearly", "365d":
		start = end.Add(-365 * 24 * time.Hour)
	default:
		// Default to monthly
		start = end.Add(-30 * 24 * time.Hour)
	}

	return start, end
}
