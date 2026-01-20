package application

import (
	"status-incident/internal/domain"
	"testing"
	"time"
)

func TestSLAService_parsePeriod(t *testing.T) {
	s := &SLAService{}

	tests := []struct {
		name           string
		period         string
		expectedDays   int
	}{
		{"daily", "daily", 1},
		{"1d", "1d", 1},
		{"weekly", "weekly", 7},
		{"7d", "7d", 7},
		{"monthly", "monthly", 30},
		{"30d", "30d", 30},
		{"quarterly", "quarterly", 90},
		{"90d", "90d", 90},
		{"yearly", "yearly", 365},
		{"365d", "365d", 365},
		{"default", "unknown", 30},
		{"empty", "", 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := s.parsePeriod(tt.period)

			// Check that end is approximately now
			if time.Since(end) > time.Second {
				t.Errorf("end time should be approximately now, got %v", end)
			}

			// Check the duration
			duration := end.Sub(start)
			expectedDuration := time.Duration(tt.expectedDays) * 24 * time.Hour

			// Allow small tolerance
			if duration < expectedDuration-time.Minute || duration > expectedDuration+time.Minute {
				t.Errorf("expected duration ~%v, got %v", expectedDuration, duration)
			}
		})
	}
}

func TestSLAService_calculateMTTR(t *testing.T) {
	s := &SLAService{}
	now := time.Now()

	tests := []struct {
		name      string
		incidents []domain.IncidentPeriod
		expected  time.Duration
	}{
		{
			name:      "empty incidents",
			incidents: []domain.IncidentPeriod{},
			expected:  0,
		},
		{
			name: "single resolved incident",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)),
				},
			},
			expected: 1 * time.Hour,
		},
		{
			name: "multiple resolved incidents",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-3 * time.Hour)),
				},
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)),
				},
			},
			expected: 1 * time.Hour, // (1h + 1h) / 2
		},
		{
			name: "mixed resolved and unresolved",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-2 * time.Hour)), // 2h
				},
				{
					StartedAt: now.Add(-1 * time.Hour),
					EndedAt:   nil, // ongoing - not counted
				},
			},
			expected: 2 * time.Hour, // only the resolved one
		},
		{
			name: "all unresolved",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-1 * time.Hour),
					EndedAt:   nil,
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.calculateMTTR(tt.incidents)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSLAService_findLongestOutage(t *testing.T) {
	s := &SLAService{}
	now := time.Now()

	tests := []struct {
		name      string
		incidents []domain.IncidentPeriod
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "empty incidents",
			incidents:   []domain.IncidentPeriod{},
			minExpected: 0,
			maxExpected: 0,
		},
		{
			name: "single resolved incident",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)),
				},
			},
			minExpected: 1 * time.Hour,
			maxExpected: 1 * time.Hour,
		},
		{
			name: "multiple incidents different durations",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-3 * time.Hour)), // 1h
				},
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-30 * time.Minute)), // 1.5h
				},
			},
			minExpected: 90 * time.Minute,
			maxExpected: 90 * time.Minute,
		},
		{
			name: "ongoing incident is longest",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-30 * time.Minute),
					EndedAt:   timePtr(now.Add(-20 * time.Minute)), // 10m
				},
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   nil, // ongoing ~2h
				},
			},
			minExpected: 2*time.Hour - time.Minute, // approximately 2 hours
			maxExpected: 2*time.Hour + time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.findLongestOutage(tt.incidents)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("expected between %v and %v, got %v", tt.minExpected, tt.maxExpected, result)
			}
		})
	}
}

func TestSLAService_calculateTotalDowntime(t *testing.T) {
	s := &SLAService{}
	now := time.Now()

	tests := []struct {
		name        string
		incidents   []domain.IncidentPeriod
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "empty incidents",
			incidents:   []domain.IncidentPeriod{},
			minExpected: 0,
			maxExpected: 0,
		},
		{
			name: "single resolved incident",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)),
				},
			},
			minExpected: 1 * time.Hour,
			maxExpected: 1 * time.Hour,
		},
		{
			name: "multiple resolved incidents",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-3 * time.Hour)), // 1h
				},
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)), // 1h
				},
			},
			minExpected: 2 * time.Hour,
			maxExpected: 2 * time.Hour,
		},
		{
			name: "mixed resolved and ongoing",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-3 * time.Hour)), // 1h
				},
				{
					StartedAt: now.Add(-30 * time.Minute),
					EndedAt:   nil, // ~30m ongoing
				},
			},
			minExpected: 90*time.Minute - time.Minute,
			maxExpected: 90*time.Minute + time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.calculateTotalDowntime(tt.incidents)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("expected between %v and %v, got %v", tt.minExpected, tt.maxExpected, result)
			}
		})
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
