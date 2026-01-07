package domain

import (
	"testing"
	"time"
)

func TestStatusLog_IsIncidentStart(t *testing.T) {
	systemID := int64(1)
	tests := []struct {
		name      string
		oldStatus Status
		newStatus Status
		want      bool
	}{
		{"green to yellow", StatusGreen, StatusYellow, true},
		{"green to red", StatusGreen, StatusRed, true},
		{"yellow to red", StatusYellow, StatusRed, false},
		{"red to yellow", StatusRed, StatusYellow, false},
		{"yellow to green", StatusYellow, StatusGreen, false},
		{"green to green", StatusGreen, StatusGreen, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := NewStatusLog(&systemID, nil, tt.oldStatus, tt.newStatus, "", SourceManual)
			if got := log.IsIncidentStart(); got != tt.want {
				t.Errorf("IsIncidentStart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusLog_IsIncidentEnd(t *testing.T) {
	systemID := int64(1)
	tests := []struct {
		name      string
		oldStatus Status
		newStatus Status
		want      bool
	}{
		{"yellow to green", StatusYellow, StatusGreen, true},
		{"red to green", StatusRed, StatusGreen, true},
		{"green to yellow", StatusGreen, StatusYellow, false},
		{"yellow to red", StatusYellow, StatusRed, false},
		{"red to yellow", StatusRed, StatusYellow, false},
		{"green to green", StatusGreen, StatusGreen, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := NewStatusLog(&systemID, nil, tt.oldStatus, tt.newStatus, "", SourceManual)
			if got := log.IsIncidentEnd(); got != tt.want {
				t.Errorf("IsIncidentEnd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIncident_IsResolved(t *testing.T) {
	now := time.Now()

	// Ongoing incident
	ongoing := Incident{
		StartedAt: now.Add(-1 * time.Hour),
		EndedAt:   nil,
	}
	if ongoing.IsResolved() {
		t.Error("Ongoing incident should not be resolved")
	}

	// Resolved incident
	endTime := now
	resolved := Incident{
		StartedAt: now.Add(-1 * time.Hour),
		EndedAt:   &endTime,
		Duration:  1 * time.Hour,
	}
	if !resolved.IsResolved() {
		t.Error("Resolved incident should be resolved")
	}
}

func TestIncident_GetDuration(t *testing.T) {
	now := time.Now()

	// Resolved incident - should return stored duration
	endTime := now
	resolved := Incident{
		StartedAt: now.Add(-2 * time.Hour),
		EndedAt:   &endTime,
		Duration:  2 * time.Hour,
	}
	if resolved.GetDuration() != 2*time.Hour {
		t.Errorf("GetDuration() = %v, want %v", resolved.GetDuration(), 2*time.Hour)
	}

	// Ongoing incident - should calculate from start to now
	ongoing := Incident{
		StartedAt: now.Add(-30 * time.Minute),
		EndedAt:   nil,
	}
	duration := ongoing.GetDuration()
	// Should be approximately 30 minutes (allow some tolerance)
	if duration < 29*time.Minute || duration > 31*time.Minute {
		t.Errorf("GetDuration() = %v, want ~30m", duration)
	}
}

func TestCalculateUptime(t *testing.T) {
	tests := []struct {
		name          string
		greenDuration time.Duration
		totalDuration time.Duration
		want          float64
	}{
		{"100% uptime", 24 * time.Hour, 24 * time.Hour, 100.0},
		{"50% uptime", 12 * time.Hour, 24 * time.Hour, 50.0},
		{"0% uptime", 0, 24 * time.Hour, 0.0},
		{"zero total duration", 0, 0, 100.0},
		{"99.9% uptime", 23*time.Hour + 58*time.Minute + 33*time.Second + 600*time.Millisecond, 24 * time.Hour, 99.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateUptime(tt.greenDuration, tt.totalDuration)
			// Allow small floating point tolerance
			if got < tt.want-0.01 || got > tt.want+0.01 {
				t.Errorf("CalculateUptime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateMTTR(t *testing.T) {
	now := time.Now()
	end1 := now.Add(-2 * time.Hour)
	end2 := now.Add(-1 * time.Hour)

	tests := []struct {
		name      string
		incidents []Incident
		want      time.Duration
	}{
		{
			name:      "no incidents",
			incidents: []Incident{},
			want:      0,
		},
		{
			name: "one resolved incident",
			incidents: []Incident{
				{StartedAt: now.Add(-3 * time.Hour), EndedAt: &end1, Duration: 1 * time.Hour},
			},
			want: 1 * time.Hour,
		},
		{
			name: "two resolved incidents",
			incidents: []Incident{
				{StartedAt: now.Add(-5 * time.Hour), EndedAt: &end1, Duration: 2 * time.Hour},
				{StartedAt: now.Add(-2 * time.Hour), EndedAt: &end2, Duration: 1 * time.Hour},
			},
			want: 90 * time.Minute, // (2h + 1h) / 2 = 1.5h
		},
		{
			name: "only ongoing incidents",
			incidents: []Incident{
				{StartedAt: now.Add(-1 * time.Hour), EndedAt: nil},
			},
			want: 0,
		},
		{
			name: "mixed resolved and ongoing",
			incidents: []Incident{
				{StartedAt: now.Add(-3 * time.Hour), EndedAt: &end1, Duration: 1 * time.Hour},
				{StartedAt: now.Add(-30 * time.Minute), EndedAt: nil},
			},
			want: 1 * time.Hour, // only counts resolved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMTTR(tt.incidents)
			if got != tt.want {
				t.Errorf("CalculateMTTR() = %v, want %v", got, tt.want)
			}
		})
	}
}
