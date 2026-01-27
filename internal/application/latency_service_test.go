package application

import (
	"context"
	"status-incident/internal/domain"
	"testing"
	"time"
)

func TestNewLatencyService(t *testing.T) {
	latencyRepo := NewMockLatencyRepository()
	depRepo := NewMockDependencyRepository()

	service := NewLatencyService(latencyRepo, depRepo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestLatencyService_GetDependencyLatencyStats(t *testing.T) {
	latencyRepo := NewMockLatencyRepository()
	depRepo := NewMockDependencyRepository()

	// Add a dependency
	dep, _ := domain.NewDependency(1, "Redis", "Cache")
	dep.ID = 1
	depRepo.Dependencies[1] = dep

	service := NewLatencyService(latencyRepo, depRepo)

	stats, err := service.GetDependencyLatencyStats(context.Background(), 1, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.DependencyName != "Redis" {
		t.Errorf("expected DependencyName 'Redis', got %q", stats.DependencyName)
	}
	if stats.AvgLatencyMs != 50.5 {
		t.Errorf("expected AvgLatencyMs 50.5, got %v", stats.AvgLatencyMs)
	}
}

func TestLatencyService_GetDependencyLatencyStats_NotFound(t *testing.T) {
	latencyRepo := NewMockLatencyRepository()
	depRepo := NewMockDependencyRepository()

	service := NewLatencyService(latencyRepo, depRepo)

	_, err := service.GetDependencyLatencyStats(context.Background(), 999, "7d")
	if err == nil {
		t.Error("expected error for non-existent dependency")
	}
}

func TestLatencyService_GetDependencyUptimeHeatmap(t *testing.T) {
	latencyRepo := NewMockLatencyRepository()
	latencyRepo.GetDailyUptimeFunc = func(ctx context.Context, dependencyID int64, days int) ([]domain.UptimePoint, error) {
		return []domain.UptimePoint{
			{Date: "2024-01-01", UptimePercent: 99.9, Status: "green"},
			{Date: "2024-01-02", UptimePercent: 95.0, Status: "yellow"},
		}, nil
	}

	depRepo := NewMockDependencyRepository()
	service := NewLatencyService(latencyRepo, depRepo)

	heatmap, err := service.GetDependencyUptimeHeatmap(context.Background(), 1, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(heatmap) != 2 {
		t.Errorf("expected 2 points, got %d", len(heatmap))
	}
}

func TestLatencyService_GetDependencyLatencyChart(t *testing.T) {
	latencyRepo := NewMockLatencyRepository()
	latencyRepo.GetAggregatedFunc = func(ctx context.Context, dependencyID int64, start, end time.Time, intervalMinutes int) ([]domain.LatencyPoint, error) {
		return []domain.LatencyPoint{
			{Timestamp: time.Now(), AvgMs: 50.0, MinMs: 10, MaxMs: 100},
		}, nil
	}

	depRepo := NewMockDependencyRepository()
	service := NewLatencyService(latencyRepo, depRepo)

	chart, err := service.GetDependencyLatencyChart(context.Background(), 1, "24h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chart) != 1 {
		t.Errorf("expected 1 point, got %d", len(chart))
	}
}

func TestLatencyService_CleanupOldRecords(t *testing.T) {
	latencyRepo := NewMockLatencyRepository()
	depRepo := NewMockDependencyRepository()
	service := NewLatencyService(latencyRepo, depRepo)

	err := service.CleanupOldRecords(context.Background(), 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLatencyService_GetDependencyLatencyChart_Periods(t *testing.T) {
	// Test different periods through the public API
	tests := []struct {
		period string
	}{
		{"1h"},
		{"6h"},
		{"24h"},
		{"7d"},
		{"30d"},
		{"90d"},
		{"unknown"}, // defaults to 24h
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			latencyRepo := NewMockLatencyRepository()
			latencyRepo.GetAggregatedFunc = func(ctx context.Context, dependencyID int64, start, end time.Time, intervalMinutes int) ([]domain.LatencyPoint, error) {
				return []domain.LatencyPoint{
					{Timestamp: time.Now(), AvgMs: 50.0, MinMs: 10, MaxMs: 100},
				}, nil
			}
			depRepo := NewMockDependencyRepository()
			service := NewLatencyService(latencyRepo, depRepo)

			chart, err := service.GetDependencyLatencyChart(context.Background(), 1, tt.period)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(chart) != 1 {
				t.Errorf("expected 1 point, got %d", len(chart))
			}
		})
	}
}

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		name             string
		period           string
		expectedDuration time.Duration
	}{
		{"1h", "1h", 1 * time.Hour},
		{"6h", "6h", 6 * time.Hour},
		{"24h", "24h", 24 * time.Hour},
		{"1d_alias", "1d", 24 * time.Hour},
		{"7d", "7d", 7 * 24 * time.Hour},
		{"30d", "30d", 30 * 24 * time.Hour},
		{"90d", "90d", 90 * 24 * time.Hour},
		{"unknown_defaults_24h", "unknown", 24 * time.Hour},
		{"empty_defaults_24h", "", 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := parsePeriod(tt.period)
			duration := end.Sub(start)

			// Allow small tolerance for execution time
			tolerance := time.Second
			if duration < tt.expectedDuration-tolerance || duration > tt.expectedDuration+tolerance {
				t.Errorf("period %q: expected duration ~%v, got %v", tt.period, tt.expectedDuration, duration)
			}
		})
	}
}

func TestGetIntervalMinutes(t *testing.T) {
	tests := []struct {
		period          string
		expectedMinutes int
	}{
		{"1h", 1},
		{"6h", 5},
		{"24h", 15},
		{"1d", 15},
		{"7d", 60},
		{"30d", 360},
		{"90d", 1440},
		{"unknown", 15},
		{"", 15},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			result := getIntervalMinutes(tt.period)
			if result != tt.expectedMinutes {
				t.Errorf("period %q: expected %d minutes, got %d", tt.period, tt.expectedMinutes, result)
			}
		})
	}
}

func TestLatencyService_GetDependencyUptimeHeatmap_DefaultDays(t *testing.T) {
	latencyRepo := NewMockLatencyRepository()
	var capturedDays int
	latencyRepo.GetDailyUptimeFunc = func(ctx context.Context, dependencyID int64, days int) ([]domain.UptimePoint, error) {
		capturedDays = days
		return []domain.UptimePoint{}, nil
	}

	depRepo := NewMockDependencyRepository()
	service := NewLatencyService(latencyRepo, depRepo)

	// Test with zero days - should default to 90
	_, err := service.GetDependencyUptimeHeatmap(context.Background(), 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedDays != 90 {
		t.Errorf("expected default days 90, got %d", capturedDays)
	}

	// Test with negative days - should default to 90
	_, err = service.GetDependencyUptimeHeatmap(context.Background(), 1, -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedDays != 90 {
		t.Errorf("expected default days 90, got %d", capturedDays)
	}

	// Test with positive days - should use provided value
	_, err = service.GetDependencyUptimeHeatmap(context.Background(), 1, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedDays != 30 {
		t.Errorf("expected days 30, got %d", capturedDays)
	}

	// Test boundary: exactly 1 day (should not default)
	_, err = service.GetDependencyUptimeHeatmap(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedDays != 1 {
		t.Errorf("expected days 1, got %d", capturedDays)
	}
}

func TestLatencyService_CleanupOldRecords_DefaultRetention(t *testing.T) {
	latencyRepo := NewMockLatencyRepository()
	var capturedTime time.Time
	latencyRepo.CleanupFunc = func(ctx context.Context, olderThan time.Time) error {
		capturedTime = olderThan
		return nil
	}

	depRepo := NewMockDependencyRepository()
	service := NewLatencyService(latencyRepo, depRepo)

	now := time.Now()

	// Test with zero retention - should default to 90 days
	err := service.CleanupOldRecords(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedTime := now.AddDate(0, 0, -90)
	if capturedTime.Sub(expectedTime) > time.Second || expectedTime.Sub(capturedTime) > time.Second {
		t.Errorf("expected cleanup time ~%v, got %v", expectedTime, capturedTime)
	}

	// Test with negative retention - should default to 90 days
	err = service.CleanupOldRecords(context.Background(), -10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTime.Sub(expectedTime) > time.Second || expectedTime.Sub(capturedTime) > time.Second {
		t.Errorf("expected cleanup time ~%v, got %v", expectedTime, capturedTime)
	}

	// Test with positive retention - should use provided value
	err = service.CleanupOldRecords(context.Background(), 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedTime = now.AddDate(0, 0, -30)
	if capturedTime.Sub(expectedTime) > time.Second || expectedTime.Sub(capturedTime) > time.Second {
		t.Errorf("expected cleanup time ~%v, got %v", expectedTime, capturedTime)
	}

	// Test boundary: exactly 1 day (should not default)
	err = service.CleanupOldRecords(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedTime = now.AddDate(0, 0, -1)
	if capturedTime.Sub(expectedTime) > time.Second || expectedTime.Sub(capturedTime) > time.Second {
		t.Errorf("expected cleanup time ~%v, got %v", expectedTime, capturedTime)
	}
}

func TestLatencyService_GetDependencyLatencyStats_VerifyIntervalPassed(t *testing.T) {
	tests := []struct {
		period          string
		expectedMinutes int
	}{
		{"1h", 1},
		{"6h", 5},
		{"24h", 15},
		{"7d", 60},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			latencyRepo := NewMockLatencyRepository()
			var capturedInterval int
			latencyRepo.GetAggregatedFunc = func(ctx context.Context, dependencyID int64, start, end time.Time, intervalMinutes int) ([]domain.LatencyPoint, error) {
				capturedInterval = intervalMinutes
				return []domain.LatencyPoint{}, nil
			}

			depRepo := NewMockDependencyRepository()
			dep, _ := domain.NewDependency(1, "Redis", "Cache")
			dep.ID = 1
			depRepo.Dependencies[1] = dep

			service := NewLatencyService(latencyRepo, depRepo)

			_, err := service.GetDependencyLatencyStats(context.Background(), 1, tt.period)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedInterval != tt.expectedMinutes {
				t.Errorf("period %q: expected interval %d minutes, got %d", tt.period, tt.expectedMinutes, capturedInterval)
			}
		})
	}
}
