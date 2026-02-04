package domain

import (
	"time"
)

// ChangeSource indicates how status was changed
type ChangeSource string

const (
	SourceManual      ChangeSource = "manual"
	SourceHeartbeat   ChangeSource = "heartbeat"
	SourcePropagation ChangeSource = "propagation"
)

// StatusLog records a status change event
type StatusLog struct {
	ID           int64
	SystemID     *int64 // nullable if change is for dependency
	DependencyID *int64 // nullable if change is for system
	OldStatus    Status
	NewStatus    Status
	Message      string       // user comment or auto-generated message
	Source       ChangeSource // manual or heartbeat
	CreatedAt    time.Time
}

// NewStatusLog creates a new status log entry
func NewStatusLog(systemID, dependencyID *int64, oldStatus, newStatus Status, message string, source ChangeSource) *StatusLog {
	return &StatusLog{
		ID:           0,
		SystemID:     systemID,
		DependencyID: dependencyID,
		OldStatus:    oldStatus,
		NewStatus:    newStatus,
		Message:      message,
		Source:       source,
		CreatedAt:    time.Now(),
	}
}

// IsIncidentStart returns true if this log marks the start of an incident
// (transition from green to non-green)
func (l *StatusLog) IsIncidentStart() bool {
	return l.OldStatus == StatusGreen && l.NewStatus != StatusGreen
}

// IsIncidentEnd returns true if this log marks the end of an incident
// (transition from non-green to green)
func (l *StatusLog) IsIncidentEnd() bool {
	return l.OldStatus != StatusGreen && l.NewStatus == StatusGreen
}

// IncidentPeriod represents a period of degraded/unavailable service (for analytics)
type IncidentPeriod struct {
	ID           int64
	SystemID     *int64
	DependencyID *int64
	StartedAt    time.Time
	EndedAt      *time.Time // nil if still ongoing
	Duration     time.Duration
	MaxSeverity  Status // worst status during incident (yellow or red)
	LogCount     int    // number of status changes during incident
}

// IsResolved returns true if incident period has ended
func (i *IncidentPeriod) IsResolved() bool {
	return i.EndedAt != nil
}

// GetDuration returns incident period duration
// If ongoing, calculates from start to now
func (i *IncidentPeriod) GetDuration() time.Duration {
	if i.EndedAt != nil {
		return i.Duration
	}
	return time.Since(i.StartedAt)
}

// Analytics holds calculated metrics for a system or dependency
type Analytics struct {
	EntityID     int64   // System or Dependency ID
	EntityType   string  // "system" or "dependency"
	EntityName   string  // Name for display
	Period       string  // e.g., "24h", "7d", "30d"
	PeriodStart  time.Time
	PeriodEnd    time.Time

	// Incident metrics
	TotalIncidents   int
	ResolvedIncidents int
	OngoingIncidents int

	// Time metrics
	TotalDowntime     time.Duration // total time in yellow or red
	TotalUnavailable  time.Duration // total time in red only
	MTTR              time.Duration // Mean Time To Recovery
	LongestIncident   time.Duration

	// Uptime/SLA
	UptimePercent     float64 // percent of time in green
	AvailabilityPercent float64 // percent of time not in red (green + yellow)
}

// CalculateUptime calculates uptime percentage
func CalculateUptime(greenDuration, totalDuration time.Duration) float64 {
	if totalDuration == 0 {
		return 100.0
	}
	return float64(greenDuration) / float64(totalDuration) * 100.0
}

// CalculateMTTR calculates Mean Time To Recovery from resolved incident periods
func CalculateMTTR(incidents []IncidentPeriod) time.Duration {
	var totalDuration time.Duration
	var resolvedCount int

	for _, inc := range incidents {
		if inc.IsResolved() {
			totalDuration += inc.Duration
			resolvedCount++
		}
	}

	if resolvedCount == 0 {
		return 0
	}
	return totalDuration / time.Duration(resolvedCount)
}
