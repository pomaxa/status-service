package domain

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidSystemID        = errors.New("system ID must be positive")
	ErrInvalidHeartbeatURL    = errors.New("invalid heartbeat URL")
	ErrInvalidHeartbeatInterval = errors.New("heartbeat interval must be positive")
)

// Dependency is an entity representing a subsystem/component of a System
type Dependency struct {
	ID                  int64
	SystemID            int64
	Name                string
	Description         string
	Status              Status
	HeartbeatURL        string
	HeartbeatInterval   int   // seconds
	LastCheck           time.Time
	LastLatency         int64 // milliseconds
	ConsecutiveFailures int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// NewDependency creates a new Dependency with validation
func NewDependency(systemID int64, name, description string) (*Dependency, error) {
	if systemID <= 0 {
		return nil, ErrInvalidSystemID
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	now := time.Now()
	return &Dependency{
		ID:                  0,
		SystemID:            systemID,
		Name:                name,
		Description:         strings.TrimSpace(description),
		Status:              StatusGreen,
		HeartbeatURL:        "",
		HeartbeatInterval:   0,
		ConsecutiveFailures: 0,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// SetHeartbeat configures automatic health checking
func (d *Dependency) SetHeartbeat(heartbeatURL string, intervalSeconds int) error {
	heartbeatURL = strings.TrimSpace(heartbeatURL)

	// Validate URL
	parsed, err := url.Parse(heartbeatURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidHeartbeatURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidHeartbeatURL
	}

	if intervalSeconds <= 0 {
		return ErrInvalidHeartbeatInterval
	}

	d.HeartbeatURL = heartbeatURL
	d.HeartbeatInterval = intervalSeconds
	d.UpdatedAt = time.Now()
	return nil
}

// ClearHeartbeat removes heartbeat configuration
func (d *Dependency) ClearHeartbeat() {
	d.HeartbeatURL = ""
	d.HeartbeatInterval = 0
	d.UpdatedAt = time.Now()
}

// HasHeartbeat returns true if heartbeat URL is configured
func (d *Dependency) HasHeartbeat() bool {
	return d.HeartbeatURL != ""
}

// RecordCheckSuccess records a successful health check with latency
// Returns true if status changed
func (d *Dependency) RecordCheckSuccess(latencyMs int64) bool {
	d.LastCheck = time.Now()
	d.LastLatency = latencyMs
	d.ConsecutiveFailures = 0

	if d.Status != StatusGreen {
		d.Status = StatusGreen
		d.UpdatedAt = time.Now()
		return true
	}
	return false
}

// RecordCheckFailure records a failed health check with latency
// Returns true if status changed
// Logic: 1 failure = yellow, 3+ failures = red
func (d *Dependency) RecordCheckFailure(latencyMs int64) bool {
	d.LastCheck = time.Now()
	d.LastLatency = latencyMs
	d.ConsecutiveFailures++

	oldStatus := d.Status

	if d.ConsecutiveFailures >= 3 {
		d.Status = StatusRed
	} else if d.ConsecutiveFailures >= 1 {
		d.Status = StatusYellow
	}

	if d.Status != oldStatus {
		d.UpdatedAt = time.Now()
		return true
	}
	return false
}

// NeedsCheck returns true if it's time for a health check
func (d *Dependency) NeedsCheck() bool {
	if !d.HasHeartbeat() {
		return false
	}

	if d.LastCheck.IsZero() {
		return true
	}

	nextCheck := d.LastCheck.Add(time.Duration(d.HeartbeatInterval) * time.Second)
	return time.Now().After(nextCheck)
}

// UpdateStatus manually updates dependency status
func (d *Dependency) UpdateStatus(status Status) error {
	if !status.IsValid() {
		return ErrInvalidStatus
	}
	d.Status = status
	d.UpdatedAt = time.Now()
	// Reset consecutive failures on manual update
	d.ConsecutiveFailures = 0
	return nil
}

// Update modifies dependency name and description
func (d *Dependency) Update(name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}
	d.Name = name
	d.Description = strings.TrimSpace(description)
	d.UpdatedAt = time.Now()
	return nil
}
