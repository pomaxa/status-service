package domain

import (
	"testing"
	"time"
)

func TestNewSLAReport(t *testing.T) {
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()

	report := NewSLAReport("Monthly Report", "monthly", start, end, "admin")

	if report.Title != "Monthly Report" {
		t.Errorf("expected title 'Monthly Report', got %q", report.Title)
	}
	if report.Period != "monthly" {
		t.Errorf("expected period 'monthly', got %q", report.Period)
	}
	if report.GeneratedBy != "admin" {
		t.Errorf("expected generatedBy 'admin', got %q", report.GeneratedBy)
	}
	if report.GeneratedAt.IsZero() {
		t.Error("expected GeneratedAt to be set")
	}
	if report.SystemReports == nil {
		t.Error("expected SystemReports to be initialized")
	}
	if len(report.SystemReports) != 0 {
		t.Error("expected SystemReports to be empty initially")
	}
}

func TestSLAReport_AddSystemReport(t *testing.T) {
	report := NewSLAReport("Test", "monthly", time.Now(), time.Now(), "admin")

	// Add system meeting SLA
	report.AddSystemReport(SystemSLAReport{
		SystemID:   1,
		SystemName: "API",
		SLAMet:     true,
	})

	if report.TotalSystems != 1 {
		t.Errorf("expected TotalSystems 1, got %d", report.TotalSystems)
	}
	if report.SystemsMeetingSLA != 1 {
		t.Errorf("expected SystemsMeetingSLA 1, got %d", report.SystemsMeetingSLA)
	}
	if report.SystemsBreachingSLA != 0 {
		t.Errorf("expected SystemsBreachingSLA 0, got %d", report.SystemsBreachingSLA)
	}

	// Add system breaching SLA
	report.AddSystemReport(SystemSLAReport{
		SystemID:   2,
		SystemName: "Database",
		SLAMet:     false,
	})

	if report.TotalSystems != 2 {
		t.Errorf("expected TotalSystems 2, got %d", report.TotalSystems)
	}
	if report.SystemsMeetingSLA != 1 {
		t.Errorf("expected SystemsMeetingSLA 1, got %d", report.SystemsMeetingSLA)
	}
	if report.SystemsBreachingSLA != 1 {
		t.Errorf("expected SystemsBreachingSLA 1, got %d", report.SystemsBreachingSLA)
	}
}

func TestSLAReport_CalculateOverall(t *testing.T) {
	t.Run("no systems", func(t *testing.T) {
		report := NewSLAReport("Test", "monthly", time.Now(), time.Now(), "admin")
		report.CalculateOverall()

		if report.OverallUptime != 100.0 {
			t.Errorf("expected OverallUptime 100.0, got %v", report.OverallUptime)
		}
		if report.OverallAvailability != 100.0 {
			t.Errorf("expected OverallAvailability 100.0, got %v", report.OverallAvailability)
		}
	})

	t.Run("with systems", func(t *testing.T) {
		report := NewSLAReport("Test", "monthly", time.Now(), time.Now(), "admin")
		report.AddSystemReport(SystemSLAReport{
			UptimePercent:      99.9,
			AvailabilityPercent: 99.95,
		})
		report.AddSystemReport(SystemSLAReport{
			UptimePercent:      99.5,
			AvailabilityPercent: 99.85,
		})
		report.CalculateOverall()

		expectedUptime := (99.9 + 99.5) / 2 // 99.7
		if report.OverallUptime != expectedUptime {
			t.Errorf("expected OverallUptime %v, got %v", expectedUptime, report.OverallUptime)
		}

		expectedAvail := (99.95 + 99.85) / 2 // 99.9
		if report.OverallAvailability != expectedAvail {
			t.Errorf("expected OverallAvailability %v, got %v", expectedAvail, report.OverallAvailability)
		}
	})
}

func TestGetStatusSummary(t *testing.T) {
	tests := []struct {
		name          string
		uptimePercent float64
		slaTarget     float64
		expected      string
	}{
		{"excellent - far above target", 100.0, 99.0, "Excellent"},
		{"excellent - 0.5 above target", 99.5, 99.0, "Excellent"},
		{"good - exactly at target", 99.0, 99.0, "Good"},
		{"good - slightly above", 99.1, 99.0, "Good"},
		{"good - 0.49 above", 99.49, 99.0, "Good"},
		{"at risk - 0.49 below", 98.51, 99.0, "At Risk"},
		{"at risk - exactly 0.5 below", 98.5, 99.0, "At Risk"},
		{"breached - 0.51 below", 98.49, 99.0, "Breached"},
		{"breached - far below", 90.0, 99.0, "Breached"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStatusSummary(tt.uptimePercent, tt.slaTarget)
			if result != tt.expected {
				t.Errorf("GetStatusSummary(%v, %v) = %q, want %q",
					tt.uptimePercent, tt.slaTarget, result, tt.expected)
			}
		})
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		percent  float64
		expected string
	}{
		{"100%", 100.0, "99.99%+"},
		{"99.999%", 99.999, "99.99%+"},
		{"99.99%", 99.99, "99.99%+"},
		{"99.98%", 99.98, ""},
		{"99.0%", 99.0, ""},
		{"0%", 0.0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUptime(tt.percent)
			if result != tt.expected {
				t.Errorf("FormatUptime(%v) = %q, want %q", tt.percent, result, tt.expected)
			}
		})
	}
}
