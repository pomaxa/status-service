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
	URL         string // link to the system
	Owner       string // responsible person/team
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

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
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
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
