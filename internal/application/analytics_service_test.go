package application

import (
	"context"
	"status-incident/internal/domain"
	"testing"
	"time"
)

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
