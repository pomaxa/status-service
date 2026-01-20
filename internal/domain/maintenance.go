package domain

import (
	"errors"
	"time"
)

// MaintenanceStatus represents the status of a maintenance window
type MaintenanceStatus string

const (
	MaintenanceScheduled  MaintenanceStatus = "scheduled"
	MaintenanceInProgress MaintenanceStatus = "in_progress"
	MaintenanceCompleted  MaintenanceStatus = "completed"
	MaintenanceCancelled  MaintenanceStatus = "cancelled"
)

// Maintenance represents a scheduled maintenance window
type Maintenance struct {
	ID          int64
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	SystemIDs   []int64           // nil = all systems
	Status      MaintenanceStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewMaintenance creates a new maintenance window
func NewMaintenance(title, description string, startTime, endTime time.Time) (*Maintenance, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	if startTime.IsZero() {
		return nil, errors.New("start time is required")
	}
	if endTime.IsZero() {
		return nil, errors.New("end time is required")
	}
	if endTime.Before(startTime) {
		return nil, errors.New("end time must be after start time")
	}

	now := time.Now()
	status := MaintenanceScheduled
	if now.After(startTime) && now.Before(endTime) {
		status = MaintenanceInProgress
	} else if now.After(endTime) {
		status = MaintenanceCompleted
	}

	return &Maintenance{
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Update updates the maintenance window details
func (m *Maintenance) Update(title, description string, startTime, endTime time.Time) error {
	if title == "" {
		return errors.New("title is required")
	}
	if endTime.Before(startTime) {
		return errors.New("end time must be after start time")
	}

	m.Title = title
	m.Description = description
	m.StartTime = startTime
	m.EndTime = endTime
	m.UpdatedAt = time.Now()
	m.RefreshStatus()
	return nil
}

// SetSystemIDs sets the affected systems
func (m *Maintenance) SetSystemIDs(ids []int64) {
	m.SystemIDs = ids
	m.UpdatedAt = time.Now()
}

// RefreshStatus updates the status based on current time
func (m *Maintenance) RefreshStatus() {
	if m.Status == MaintenanceCancelled {
		return // Don't change cancelled status
	}

	now := time.Now()
	if now.Before(m.StartTime) {
		m.Status = MaintenanceScheduled
	} else if now.After(m.EndTime) {
		m.Status = MaintenanceCompleted
	} else {
		m.Status = MaintenanceInProgress
	}
}

// Cancel marks the maintenance as cancelled
func (m *Maintenance) Cancel() {
	m.Status = MaintenanceCancelled
	m.UpdatedAt = time.Now()
}

// IsActive returns true if maintenance is currently in progress
func (m *Maintenance) IsActive() bool {
	m.RefreshStatus()
	return m.Status == MaintenanceInProgress
}

// AffectsSystem returns true if this maintenance affects the given system
func (m *Maintenance) AffectsSystem(systemID int64) bool {
	if len(m.SystemIDs) == 0 {
		return true // nil means all systems
	}
	for _, id := range m.SystemIDs {
		if id == systemID {
			return true
		}
	}
	return false
}

// IsUpcoming returns true if maintenance is scheduled for the future
func (m *Maintenance) IsUpcoming() bool {
	return m.Status == MaintenanceScheduled && time.Now().Before(m.StartTime)
}

// TimeUntilStart returns duration until maintenance starts (negative if already started)
func (m *Maintenance) TimeUntilStart() time.Duration {
	return m.StartTime.Sub(time.Now())
}

// TimeUntilEnd returns duration until maintenance ends (negative if already ended)
func (m *Maintenance) TimeUntilEnd() time.Duration {
	return m.EndTime.Sub(time.Now())
}
