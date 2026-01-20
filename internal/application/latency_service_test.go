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
