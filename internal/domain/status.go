package domain

import (
	"errors"
	"strings"
)

// Status is a value object representing system/dependency health state
type Status string

const (
	StatusGreen  Status = "green"
	StatusYellow Status = "yellow"
	StatusRed    Status = "red"
)

var ErrInvalidStatus = errors.New("invalid status: must be green, yellow, or red")

// NewStatus creates a Status from string with validation
func NewStatus(s string) (Status, error) {
	normalized := Status(strings.ToLower(strings.TrimSpace(s)))
	if !normalized.IsValid() {
		return "", ErrInvalidStatus
	}
	return normalized, nil
}

// String returns string representation
func (s Status) String() string {
	return string(s)
}

// IsValid checks if status is one of allowed values
func (s Status) IsValid() bool {
	switch s {
	case StatusGreen, StatusYellow, StatusRed:
		return true
	}
	return false
}

// IsOperational returns true only for green status
func (s Status) IsOperational() bool {
	return s == StatusGreen
}

// Severity returns numeric severity level (0=green, 1=yellow, 2=red)
func (s Status) Severity() int {
	switch s {
	case StatusGreen:
		return 0
	case StatusYellow:
		return 1
	case StatusRed:
		return 2
	}
	return -1
}
