package application

import (
	"context"
	"math"
	"status-incident/internal/domain"
	"testing"
	"time"
)

const floatEpsilon = 0.001

func TestNewAnalyticsService(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()

	service := NewAnalyticsService(analyticsRepo, logRepo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestAnalyticsService_GetSystemAnalytics(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewAnalyticsService(analyticsRepo, logRepo)

	analytics, err := service.GetSystemAnalytics(context.Background(), 1, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analytics == nil {
		t.Fatal("expected non-nil analytics")
	}
	if analytics.UptimePercent != 99.9 {
		t.Errorf("expected UptimePercent 99.9, got %v", analytics.UptimePercent)
	}
}

func TestAnalyticsService_GetDependencyAnalytics(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewAnalyticsService(analyticsRepo, logRepo)

	analytics, err := service.GetDependencyAnalytics(context.Background(), 1, "30d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analytics == nil {
		t.Fatal("expected non-nil analytics")
	}
	if analytics.UptimePercent != 99.8 {
		t.Errorf("expected UptimePercent 99.8, got %v", analytics.UptimePercent)
	}
}

func TestAnalyticsService_GetOverallAnalytics(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewAnalyticsService(analyticsRepo, logRepo)

	analytics, err := service.GetOverallAnalytics(context.Background(), "monthly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analytics == nil {
		t.Fatal("expected non-nil analytics")
	}
	if analytics.UptimePercent != 99.5 {
		t.Errorf("expected UptimePercent 99.5, got %v", analytics.UptimePercent)
	}
}

func TestAnalyticsService_GetSystemIncidents(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewAnalyticsService(analyticsRepo, logRepo)

	incidents, err := service.GetSystemIncidents(context.Background(), 1, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if incidents == nil {
		t.Fatal("expected non-nil incidents slice")
	}
}

func TestAnalyticsService_GetDependencyIncidents(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewAnalyticsService(analyticsRepo, logRepo)

	incidents, err := service.GetDependencyIncidents(context.Background(), 1, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if incidents == nil {
		t.Fatal("expected non-nil incidents slice")
	}
}

func TestAnalyticsService_GetAllLogs(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()

	// Add some logs
	systemID := int64(1)
	logRepo.Logs = append(logRepo.Logs, &domain.StatusLog{
		ID:        1,
		SystemID:  &systemID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusYellow,
		CreatedAt: time.Now(),
	})

	service := NewAnalyticsService(analyticsRepo, logRepo)

	logs, err := service.GetAllLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestAnalyticsService_CreateLog(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()
	service := NewAnalyticsService(analyticsRepo, logRepo)

	systemID := int64(1)
	log := &domain.StatusLog{
		SystemID:  &systemID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "Test incident",
		Source:    domain.SourceManual,
	}

	err := service.CreateLog(context.Background(), log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.ID == 0 {
		t.Error("expected log ID to be set")
	}
}

func TestAnalyticsService_parsePeriod(t *testing.T) {
	service := &AnalyticsService{}

	tests := []struct {
		name             string
		period           string
		expectedDuration time.Duration
	}{
		{"1h", "1h", 1 * time.Hour},
		{"24h", "24h", 24 * time.Hour},
		{"1d", "1d", 24 * time.Hour},
		{"7d", "7d", 7 * 24 * time.Hour},
		{"30d", "30d", 30 * 24 * time.Hour},
		{"90d", "90d", 90 * 24 * time.Hour},
		{"unknown_defaults_24h", "unknown", 24 * time.Hour},
		{"365d_defaults_24h", "365d", 24 * time.Hour},
		{"empty_defaults_24h", "", 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := service.parsePeriod(tt.period)
			duration := end.Sub(start)

			if duration < tt.expectedDuration-time.Minute || duration > tt.expectedDuration+time.Minute {
				t.Errorf("expected duration ~%v, got %v", tt.expectedDuration, duration)
			}
		})
	}
}

func TestAnalyticsService_GetSystemAnalytics_FloatComparison(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()

	// Set up mock to return specific uptime values for testing float comparison
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return &domain.Analytics{
			UptimePercent:     99.99999,
			TotalIncidents:    1,
			TotalDowntime:     14*time.Minute + 24*time.Second, // ~14.4 minutes
		}, nil
	}

	service := NewAnalyticsService(analyticsRepo, logRepo)
	analytics, err := service.GetSystemAnalytics(context.Background(), 1, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Use epsilon for float comparison
	expectedUptime := 99.99999
	if math.Abs(analytics.UptimePercent-expectedUptime) > floatEpsilon {
		t.Errorf("expected UptimePercent ~%.5f, got %.5f (diff: %v)",
			expectedUptime, analytics.UptimePercent, math.Abs(analytics.UptimePercent-expectedUptime))
	}

	expectedDowntime := 14*time.Minute + 24*time.Second
	if analytics.TotalDowntime != expectedDowntime {
		t.Errorf("expected TotalDowntime %v, got %v", expectedDowntime, analytics.TotalDowntime)
	}
}

func TestAnalyticsService_GetAllLogs_DefaultLimit(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()

	var capturedLimit int
	logRepo.GetAllFunc = func(ctx context.Context, limit int) ([]*domain.StatusLog, error) {
		capturedLimit = limit
		return []*domain.StatusLog{}, nil
	}

	service := NewAnalyticsService(analyticsRepo, logRepo)

	// Test with zero limit - should default to 100
	_, err := service.GetAllLogs(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 100 {
		t.Errorf("expected default limit 100, got %d", capturedLimit)
	}

	// Test with negative limit - should default to 100
	_, err = service.GetAllLogs(context.Background(), -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 100 {
		t.Errorf("expected default limit 100, got %d", capturedLimit)
	}

	// Test with positive limit - should use provided value
	_, err = service.GetAllLogs(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 50 {
		t.Errorf("expected limit 50, got %d", capturedLimit)
	}

	// Test boundary: exactly 1 (should not default)
	_, err = service.GetAllLogs(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 1 {
		t.Errorf("expected limit 1, got %d", capturedLimit)
	}
}

func TestAnalyticsService_GetAllLogs_ContentValidation(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()

	systemID1 := int64(1)
	systemID2 := int64(2)
	now := time.Now()

	// Add logs with specific content
	logRepo.Logs = append(logRepo.Logs,
		&domain.StatusLog{
			ID:        1,
			SystemID:  &systemID1,
			OldStatus: domain.StatusGreen,
			NewStatus: domain.StatusYellow,
			Message:   "First log",
			Source:    domain.SourceManual,
			CreatedAt: now.Add(-time.Hour),
		},
		&domain.StatusLog{
			ID:        2,
			SystemID:  &systemID2,
			OldStatus: domain.StatusYellow,
			NewStatus: domain.StatusRed,
			Message:   "Second log",
			Source:    domain.SourceHeartbeat,
			CreatedAt: now,
		},
	)

	service := NewAnalyticsService(analyticsRepo, logRepo)
	logs, err := service.GetAllLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate length
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}

	// Validate content of first log
	if logs[0].ID != 1 {
		t.Errorf("log[0]: expected ID 1, got %d", logs[0].ID)
	}
	if logs[0].SystemID == nil || *logs[0].SystemID != systemID1 {
		t.Errorf("log[0]: expected SystemID %d, got %v", systemID1, logs[0].SystemID)
	}
	if logs[0].OldStatus != domain.StatusGreen {
		t.Errorf("log[0]: expected OldStatus Green, got %v", logs[0].OldStatus)
	}
	if logs[0].NewStatus != domain.StatusYellow {
		t.Errorf("log[0]: expected NewStatus Yellow, got %v", logs[0].NewStatus)
	}
	if logs[0].Message != "First log" {
		t.Errorf("log[0]: expected Message 'First log', got %q", logs[0].Message)
	}

	// Validate content of second log
	if logs[1].ID != 2 {
		t.Errorf("log[1]: expected ID 2, got %d", logs[1].ID)
	}
	if logs[1].SystemID == nil || *logs[1].SystemID != systemID2 {
		t.Errorf("log[1]: expected SystemID %d, got %v", systemID2, logs[1].SystemID)
	}
	if logs[1].Source != domain.SourceHeartbeat {
		t.Errorf("log[1]: expected Source Heartbeat, got %v", logs[1].Source)
	}
}

func TestAnalyticsService_GetSystemIncidents_ContentValidation(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()

	// Set up mock with specific incident data
	now := time.Now()
	endTime1 := now.Add(-time.Hour)
	endTime2 := now
	analyticsRepo.GetIncidentsBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
		return []domain.IncidentPeriod{
			{
				ID:          1,
				StartedAt:   now.Add(-2 * time.Hour),
				EndedAt:     &endTime1,
				Duration:    60 * time.Minute,
				MaxSeverity: domain.StatusYellow,
			},
			{
				ID:          2,
				StartedAt:   now.Add(-30 * time.Minute),
				EndedAt:     &endTime2,
				Duration:    30 * time.Minute,
				MaxSeverity: domain.StatusRed,
			},
		}, nil
	}

	service := NewAnalyticsService(analyticsRepo, logRepo)
	incidents, err := service.GetSystemIncidents(context.Background(), 1, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate length
	if len(incidents) != 2 {
		t.Fatalf("expected 2 incidents, got %d", len(incidents))
	}

	// Validate content of incidents
	if incidents[0].Duration != 60*time.Minute {
		t.Errorf("incident[0]: expected Duration 60m, got %v", incidents[0].Duration)
	}
	if incidents[0].MaxSeverity != domain.StatusYellow {
		t.Errorf("incident[0]: expected MaxSeverity Yellow, got %v", incidents[0].MaxSeverity)
	}

	if incidents[1].Duration != 30*time.Minute {
		t.Errorf("incident[1]: expected Duration 30m, got %v", incidents[1].Duration)
	}
	if incidents[1].MaxSeverity != domain.StatusRed {
		t.Errorf("incident[1]: expected MaxSeverity Red, got %v", incidents[1].MaxSeverity)
	}
}

func TestAnalyticsService_ZeroRecords(t *testing.T) {
	analyticsRepo := NewMockAnalyticsRepository()
	logRepo := NewMockStatusLogRepository()

	// Mock returns empty data
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return &domain.Analytics{
			UptimePercent:    100.0, // No downtime
			TotalIncidents:   0,
			TotalDowntime:    0,
		}, nil
	}
	analyticsRepo.GetIncidentsBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
		return []domain.IncidentPeriod{}, nil // Empty slice, not nil
	}

	service := NewAnalyticsService(analyticsRepo, logRepo)

	// Test analytics with zero incidents
	analytics, err := service.GetSystemAnalytics(context.Background(), 1, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(analytics.UptimePercent-100.0) > floatEpsilon {
		t.Errorf("expected 100%% uptime, got %.2f%%", analytics.UptimePercent)
	}
	if analytics.TotalIncidents != 0 {
		t.Errorf("expected 0 incidents, got %d", analytics.TotalIncidents)
	}

	// Test incidents with zero results
	incidents, err := service.GetSystemIncidents(context.Background(), 1, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if incidents == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(incidents) != 0 {
		t.Errorf("expected 0 incidents, got %d", len(incidents))
	}
}
