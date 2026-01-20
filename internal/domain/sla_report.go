package domain

import (
	"time"
)

// SLAReport represents a generated SLA report
type SLAReport struct {
	ID           int64
	Title        string
	Period       string // "monthly", "weekly", "custom"
	PeriodStart  time.Time
	PeriodEnd    time.Time
	GeneratedAt  time.Time
	GeneratedBy  string

	// Overall metrics
	OverallUptime      float64
	OverallAvailability float64
	TotalSystems       int
	SystemsMeetingSLA  int
	SystemsBreachingSLA int

	// Per-system details
	SystemReports []SystemSLAReport
}

// SystemSLAReport represents SLA metrics for a single system
type SystemSLAReport struct {
	SystemID       int64
	SystemName     string
	Owner          string
	SLATarget      float64

	// Uptime metrics
	UptimePercent      float64 // time in green status
	AvailabilityPercent float64 // time not in red status

	// Incident metrics
	TotalIncidents     int
	ResolvedIncidents  int
	TotalDowntime      time.Duration
	LongestOutage      time.Duration
	MTTR               time.Duration // Mean Time To Recovery

	// Status
	SLAMet        bool
	SLADelta      float64 // positive = above target, negative = below
	StatusSummary string  // "Excellent", "Good", "At Risk", "Breached"

	// Dependencies
	DependencyReports []DependencySLAReport
}

// DependencySLAReport represents SLA metrics for a dependency
type DependencySLAReport struct {
	DependencyID   int64
	DependencyName string

	UptimePercent      float64
	AvailabilityPercent float64
	TotalChecks        int
	FailedChecks       int
	AvgLatencyMs       float64
	P95LatencyMs       int64
	P99LatencyMs       int64
}

// SLABreachEvent represents an SLA breach that occurred
type SLABreachEvent struct {
	ID           int64
	SystemID     int64
	SystemName   string
	BreachType   string // "uptime", "availability", "response_time"
	SLATarget    float64
	ActualValue  float64
	Period       string
	PeriodStart  time.Time
	PeriodEnd    time.Time
	DetectedAt   time.Time
	Acknowledged bool
	AckedBy      string
	AckedAt      *time.Time
}

// NewSLAReport creates a new SLA report
func NewSLAReport(title, period string, start, end time.Time, generatedBy string) *SLAReport {
	return &SLAReport{
		Title:        title,
		Period:       period,
		PeriodStart:  start,
		PeriodEnd:    end,
		GeneratedAt:  time.Now(),
		GeneratedBy:  generatedBy,
		SystemReports: make([]SystemSLAReport, 0),
	}
}

// AddSystemReport adds a system report to the SLA report
func (r *SLAReport) AddSystemReport(sr SystemSLAReport) {
	r.SystemReports = append(r.SystemReports, sr)
	r.TotalSystems++
	if sr.SLAMet {
		r.SystemsMeetingSLA++
	} else {
		r.SystemsBreachingSLA++
	}
}

// CalculateOverall calculates overall metrics from system reports
func (r *SLAReport) CalculateOverall() {
	if len(r.SystemReports) == 0 {
		r.OverallUptime = 100.0
		r.OverallAvailability = 100.0
		return
	}

	var totalUptime, totalAvailability float64
	for _, sr := range r.SystemReports {
		totalUptime += sr.UptimePercent
		totalAvailability += sr.AvailabilityPercent
	}

	r.OverallUptime = totalUptime / float64(len(r.SystemReports))
	r.OverallAvailability = totalAvailability / float64(len(r.SystemReports))
}

// GetStatusSummary returns a human-readable status based on SLA compliance
func GetStatusSummary(uptimePercent, slaTarget float64) string {
	delta := uptimePercent - slaTarget
	if delta >= 0.5 {
		return "Excellent"
	} else if delta >= 0 {
		return "Good"
	} else if delta >= -0.5 {
		return "At Risk"
	}
	return "Breached"
}

// FormatUptime formats uptime percentage for display
func FormatUptime(percent float64) string {
	if percent >= 99.99 {
		return "99.99%+"
	}
	return ""
}
