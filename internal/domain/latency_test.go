package domain

import (
	"testing"
)

func TestGetUptimeStatus(t *testing.T) {
	tests := []struct {
		name          string
		uptimePercent float64
		expected      string
	}{
		{"100% is green", 100.0, "green"},
		{"99.9% is green", 99.9, "green"},
		{"99.0% is green", 99.0, "green"},
		{"98.9% is yellow", 98.9, "yellow"},
		{"95.0% is yellow", 95.0, "yellow"},
		{"94.9% is red", 94.9, "red"},
		{"90.0% is red", 90.0, "red"},
		{"0% is red", 0.0, "red"},
		{"negative is red", -1.0, "red"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUptimeStatus(tt.uptimePercent)
			if result != tt.expected {
				t.Errorf("GetUptimeStatus(%v) = %q, want %q", tt.uptimePercent, result, tt.expected)
			}
		})
	}
}
