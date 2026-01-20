package domain

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidSystemID          = errors.New("system ID must be positive")
	ErrInvalidHeartbeatURL      = errors.New("invalid heartbeat URL")
	ErrInvalidHeartbeatInterval = errors.New("heartbeat interval must be positive")
	ErrInvalidHeartbeatMethod   = errors.New("invalid HTTP method")
	ErrInvalidExpectStatus      = errors.New("invalid expected status code format")
)

// HeartbeatConfig contains all configuration for health checks
type HeartbeatConfig struct {
	URL          string            `json:"url"`
	Interval     int               `json:"interval"` // seconds
	Method       string            `json:"method,omitempty"` // GET, POST, PUT, HEAD
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
	ExpectStatus string            `json:"expect_status,omitempty"` // "200", "200,201", "2xx"
	ExpectBody   string            `json:"expect_body,omitempty"`   // regex pattern
}

// ValidHTTPMethods lists allowed HTTP methods for health checks
var ValidHTTPMethods = map[string]bool{
	"GET":  true,
	"POST": true,
	"PUT":  true,
	"HEAD": true,
}

// Dependency is an entity representing a subsystem/component of a System
type Dependency struct {
	ID                  int64
	SystemID            int64
	Name                string
	Description         string
	Status              Status
	HeartbeatURL        string
	HeartbeatInterval   int   // seconds
	HeartbeatMethod     string // GET, POST, PUT, HEAD (default: GET)
	HeartbeatHeaders    map[string]string // custom headers (e.g., Authorization)
	HeartbeatBody       string // request body for POST/PUT
	HeartbeatExpectStatus string // expected status codes: "200", "200,201", "2xx" (default: 2xx)
	HeartbeatExpectBody string // regex pattern to match in response body
	LastCheck           time.Time
	LastLatency         int64 // milliseconds
	LastStatusCode      int   // last HTTP status code received
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

// SetHeartbeat configures automatic health checking (legacy method for backward compatibility)
func (d *Dependency) SetHeartbeat(heartbeatURL string, intervalSeconds int) error {
	return d.SetHeartbeatConfig(HeartbeatConfig{
		URL:      heartbeatURL,
		Interval: intervalSeconds,
	})
}

// SetHeartbeatConfig configures automatic health checking with advanced options
func (d *Dependency) SetHeartbeatConfig(config HeartbeatConfig) error {
	config.URL = strings.TrimSpace(config.URL)

	// Validate URL
	parsed, err := url.Parse(config.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidHeartbeatURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidHeartbeatURL
	}

	if config.Interval <= 0 {
		return ErrInvalidHeartbeatInterval
	}

	// Validate and normalize HTTP method
	method := strings.ToUpper(strings.TrimSpace(config.Method))
	if method == "" {
		method = "GET"
	}
	if !ValidHTTPMethods[method] {
		return ErrInvalidHeartbeatMethod
	}

	// Validate expect_status format if provided
	if config.ExpectStatus != "" {
		if !isValidExpectStatus(config.ExpectStatus) {
			return ErrInvalidExpectStatus
		}
	}

	d.HeartbeatURL = config.URL
	d.HeartbeatInterval = config.Interval
	d.HeartbeatMethod = method
	d.HeartbeatHeaders = config.Headers
	d.HeartbeatBody = config.Body
	d.HeartbeatExpectStatus = config.ExpectStatus
	d.HeartbeatExpectBody = config.ExpectBody
	d.UpdatedAt = time.Now()
	return nil
}

// isValidExpectStatus checks if the expect_status format is valid
// Valid formats: "200", "200,201,204", "2xx", "2xx,3xx"
func isValidExpectStatus(s string) bool {
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}
		// Check for wildcard format like "2xx"
		if len(part) == 3 && part[1] == 'x' && part[2] == 'x' {
			if part[0] < '1' || part[0] > '5' {
				return false
			}
			continue
		}
		// Check for numeric status code
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// GetHeartbeatConfig returns the current heartbeat configuration
func (d *Dependency) GetHeartbeatConfig() HeartbeatConfig {
	method := d.HeartbeatMethod
	if method == "" {
		method = "GET"
	}
	return HeartbeatConfig{
		URL:          d.HeartbeatURL,
		Interval:     d.HeartbeatInterval,
		Method:       method,
		Headers:      d.HeartbeatHeaders,
		Body:         d.HeartbeatBody,
		ExpectStatus: d.HeartbeatExpectStatus,
		ExpectBody:   d.HeartbeatExpectBody,
	}
}

// ClearHeartbeat removes heartbeat configuration
func (d *Dependency) ClearHeartbeat() {
	d.HeartbeatURL = ""
	d.HeartbeatInterval = 0
	d.HeartbeatMethod = ""
	d.HeartbeatHeaders = nil
	d.HeartbeatBody = ""
	d.HeartbeatExpectStatus = ""
	d.HeartbeatExpectBody = ""
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
