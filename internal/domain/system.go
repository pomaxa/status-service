package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrEmptyName = errors.New("name cannot be empty")

// System is an entity representing a monitored system/project
type System struct {
	ID          int64
	Name        string
	Description string
	URL         string  // link to the system
	Owner       string  // responsible person/team
	Status      Status
	SLATarget   float64 // SLA target percentage (e.g., 99.9)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DefaultSLATarget is the default SLA target if not specified
const DefaultSLATarget = 99.9

// NewSystem creates a new System with validation
func NewSystem(name, description, url, owner string) (*System, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	now := time.Now()
	return &System{
		ID:          0, // will be set by repository
		Name:        name,
		Description: strings.TrimSpace(description),
		URL:         strings.TrimSpace(url),
		Owner:       strings.TrimSpace(owner),
		Status:      StatusGreen, // default healthy
		SLATarget:   DefaultSLATarget,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetSLATarget returns the SLA target, defaulting if not set
func (s *System) GetSLATarget() float64 {
	if s.SLATarget <= 0 {
		return DefaultSLATarget
	}
	return s.SLATarget
}

// SetSLATarget sets the SLA target with validation
func (s *System) SetSLATarget(target float64) {
	if target <= 0 || target > 100 {
		s.SLATarget = DefaultSLATarget
	} else {
		s.SLATarget = target
	}
	s.UpdatedAt = time.Now()
}

// IsSLAMet checks if the given uptime meets the SLA target
func (s *System) IsSLAMet(uptimePercent float64) bool {
	return uptimePercent >= s.GetSLATarget()
}

// UpdateStatus changes system status with validation
func (s *System) UpdateStatus(status Status) error {
	if !status.IsValid() {
		return ErrInvalidStatus
	}
	s.Status = status
	s.UpdatedAt = time.Now()
	return nil
}

// Update modifies system name, description, URL and owner
func (s *System) Update(name, description, url, owner string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}
	s.Name = name
	s.Description = strings.TrimSpace(description)
	s.URL = strings.TrimSpace(url)
	s.Owner = strings.TrimSpace(owner)
	s.UpdatedAt = time.Now()
	return nil
}

// IsHealthy returns true if system status is green
func (s *System) IsHealthy() bool {
	return s.Status.IsOperational()
}
