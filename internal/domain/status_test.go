package domain

import (
	"testing"
)

func TestNewStatus_ValidStatuses(t *testing.T) {
	tests := []struct {
		input    string
		expected Status
	}{
		{"green", StatusGreen},
		{"yellow", StatusYellow},
		{"red", StatusRed},
		{"GREEN", StatusGreen},
		{"Yellow", StatusYellow},
		{"RED", StatusRed},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			status, err := NewStatus(tt.input)
			if err != nil {
				t.Errorf("NewStatus(%q) returned error: %v", tt.input, err)
			}
			if status != tt.expected {
				t.Errorf("NewStatus(%q) = %v, want %v", tt.input, status, tt.expected)
			}
		})
	}
}

func TestNewStatus_InvalidStatus(t *testing.T) {
	invalidStatuses := []string{"blue", "orange", "", "invalid", "greenish"}

	for _, input := range invalidStatuses {
		t.Run(input, func(t *testing.T) {
			_, err := NewStatus(input)
			if err == nil {
				t.Errorf("NewStatus(%q) should return error for invalid status", input)
			}
		})
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusGreen, "green"},
		{StatusYellow, "yellow"},
		{StatusRed, "red"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.status.String() != tt.expected {
				t.Errorf("Status.String() = %q, want %q", tt.status.String(), tt.expected)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	if !StatusGreen.IsValid() {
		t.Error("StatusGreen should be valid")
	}
	if !StatusYellow.IsValid() {
		t.Error("StatusYellow should be valid")
	}
	if !StatusRed.IsValid() {
		t.Error("StatusRed should be valid")
	}

	invalidStatus := Status("invalid")
	if invalidStatus.IsValid() {
		t.Error("Invalid status should not be valid")
	}
}

func TestStatus_IsOperational(t *testing.T) {
	if !StatusGreen.IsOperational() {
		t.Error("StatusGreen should be operational")
	}
	if StatusYellow.IsOperational() {
		t.Error("StatusYellow should not be fully operational")
	}
	if StatusRed.IsOperational() {
		t.Error("StatusRed should not be operational")
	}
}

func TestStatus_Severity(t *testing.T) {
	if StatusGreen.Severity() >= StatusYellow.Severity() {
		t.Error("Green should have lower severity than Yellow")
	}
	if StatusYellow.Severity() >= StatusRed.Severity() {
		t.Error("Yellow should have lower severity than Red")
	}
}
