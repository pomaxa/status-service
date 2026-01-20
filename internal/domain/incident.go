package domain

import (
	"errors"
	"time"
)

// IncidentStatus represents the current status of an incident
type IncidentStatus string

const (
	IncidentInvestigating IncidentStatus = "investigating"
	IncidentIdentified    IncidentStatus = "identified"
	IncidentMonitoring    IncidentStatus = "monitoring"
	IncidentResolved      IncidentStatus = "resolved"
)

// IncidentSeverity represents the severity level of an incident
type IncidentSeverity string

const (
	SeverityMinor    IncidentSeverity = "minor"
	SeverityMajor    IncidentSeverity = "major"
	SeverityCritical IncidentSeverity = "critical"
)

// Incident represents a service incident
type Incident struct {
	ID          int64
	Title       string
	Status      IncidentStatus
	Severity    IncidentSeverity
	SystemIDs   []int64 // Affected systems (nil = all)
	Message     string  // Initial message
	Postmortem  string  // Post-incident summary
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ResolvedAt  *time.Time
	AcknowledgedAt *time.Time
	AcknowledgedBy string
}

// IncidentUpdate represents a timeline entry for an incident
type IncidentUpdate struct {
	ID         int64
	IncidentID int64
	Status     IncidentStatus
	Message    string
	CreatedAt  time.Time
	CreatedBy  string
}

// NewIncident creates a new incident
func NewIncident(title, message string, severity IncidentSeverity) (*Incident, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	if message == "" {
		return nil, errors.New("message is required")
	}

	// Validate severity
	switch severity {
	case SeverityMinor, SeverityMajor, SeverityCritical:
		// valid
	default:
		severity = SeverityMinor
	}

	now := time.Now()
	return &Incident{
		Title:     title,
		Status:    IncidentInvestigating,
		Severity:  severity,
		Message:   message,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// SetSystemIDs sets the affected systems
func (i *Incident) SetSystemIDs(ids []int64) {
	i.SystemIDs = ids
	i.UpdatedAt = time.Now()
}

// Acknowledge marks the incident as acknowledged
func (i *Incident) Acknowledge(by string) error {
	if i.AcknowledgedAt != nil {
		return errors.New("incident already acknowledged")
	}
	now := time.Now()
	i.AcknowledgedAt = &now
	i.AcknowledgedBy = by
	i.UpdatedAt = now
	return nil
}

// UpdateStatus updates the incident status
func (i *Incident) UpdateStatus(status IncidentStatus) error {
	if i.Status == IncidentResolved {
		return errors.New("cannot update resolved incident")
	}

	switch status {
	case IncidentInvestigating, IncidentIdentified, IncidentMonitoring, IncidentResolved:
		// valid
	default:
		return errors.New("invalid status")
	}

	i.Status = status
	i.UpdatedAt = time.Now()

	if status == IncidentResolved {
		now := time.Now()
		i.ResolvedAt = &now
	}

	return nil
}

// Resolve marks the incident as resolved with optional postmortem
func (i *Incident) Resolve(postmortem string) error {
	if i.Status == IncidentResolved {
		return errors.New("incident already resolved")
	}

	now := time.Now()
	i.Status = IncidentResolved
	i.ResolvedAt = &now
	i.UpdatedAt = now
	i.Postmortem = postmortem
	return nil
}

// IsResolved returns true if the incident is resolved
func (i *Incident) IsResolved() bool {
	return i.Status == IncidentResolved
}

// IsActive returns true if the incident is not resolved
func (i *Incident) IsActive() bool {
	return i.Status != IncidentResolved
}

// Duration returns the duration of the incident
func (i *Incident) Duration() time.Duration {
	if i.ResolvedAt != nil {
		return i.ResolvedAt.Sub(i.CreatedAt)
	}
	return time.Since(i.CreatedAt)
}

// AffectsSystem returns true if this incident affects the given system
func (i *Incident) AffectsSystem(systemID int64) bool {
	if len(i.SystemIDs) == 0 {
		return true // nil means all systems
	}
	for _, id := range i.SystemIDs {
		if id == systemID {
			return true
		}
	}
	return false
}

// NewIncidentUpdate creates a new incident update
func NewIncidentUpdate(incidentID int64, status IncidentStatus, message, createdBy string) (*IncidentUpdate, error) {
	if message == "" {
		return nil, errors.New("message is required")
	}

	return &IncidentUpdate{
		IncidentID: incidentID,
		Status:     status,
		Message:    message,
		CreatedAt:  time.Now(),
		CreatedBy:  createdBy,
	}, nil
}

// SeverityEmoji returns an emoji for the severity
func SeverityEmoji(s IncidentSeverity) string {
	switch s {
	case SeverityMinor:
		return "⚠️"
	case SeverityMajor:
		return "🔶"
	case SeverityCritical:
		return "🔴"
	default:
		return "❓"
	}
}

// StatusEmoji returns an emoji for the incident status
func IncidentStatusEmoji(s IncidentStatus) string {
	switch s {
	case IncidentInvestigating:
		return "🔍"
	case IncidentIdentified:
		return "🎯"
	case IncidentMonitoring:
		return "👀"
	case IncidentResolved:
		return "✅"
	default:
		return "❓"
	}
}
