package domain

import (
	"context"
	"time"
)

// LatencyRecord represents a single latency measurement
type LatencyRecord struct {
	ID           int64
	DependencyID int64
	LatencyMs    int64
	Success      bool
	StatusCode   int
	CreatedAt    time.Time
}

// LatencyPoint represents aggregated latency data for charting
type LatencyPoint struct {
	Timestamp time.Time `json:"timestamp"`
	AvgMs     float64   `json:"avg_ms"`
	MinMs     int64     `json:"min_ms"`
	MaxMs     int64     `json:"max_ms"`
	Count     int       `json:"count"`
	Failures  int       `json:"failures"`
}

// UptimePoint represents daily uptime for heatmap
type UptimePoint struct {
	Date          string  `json:"date"` // YYYY-MM-DD
	UptimePercent float64 `json:"uptime_percent"`
	TotalChecks   int     `json:"total_checks"`
	FailedChecks  int     `json:"failed_checks"`
	Status        string  `json:"status"` // green, yellow, red based on uptime
}

// LatencyStats holds statistics for a dependency
type LatencyStats struct {
	DependencyID   int64         `json:"dependency_id"`
	DependencyName string        `json:"dependency_name"`
	Period         string        `json:"period"`
	AvgLatencyMs   float64       `json:"avg_latency_ms"`
	MinLatencyMs   int64         `json:"min_latency_ms"`
	MaxLatencyMs   int64         `json:"max_latency_ms"`
	P50LatencyMs   int64         `json:"p50_latency_ms"`
	P95LatencyMs   int64         `json:"p95_latency_ms"`
	P99LatencyMs   int64         `json:"p99_latency_ms"`
	TotalChecks    int           `json:"total_checks"`
	FailedChecks   int           `json:"failed_checks"`
	UptimePercent  float64       `json:"uptime_percent"`
	DataPoints     []LatencyPoint `json:"data_points,omitempty"`
	UptimeHeatmap  []UptimePoint  `json:"uptime_heatmap,omitempty"`
}

// LatencyRepository defines operations for latency data persistence
type LatencyRepository interface {
	// Record stores a new latency measurement
	Record(ctx context.Context, record *LatencyRecord) error

	// GetByDependency retrieves latency records for a dependency within time range
	GetByDependency(ctx context.Context, dependencyID int64, start, end time.Time, limit int) ([]*LatencyRecord, error)

	// GetAggregated retrieves aggregated latency data for charting
	GetAggregated(ctx context.Context, dependencyID int64, start, end time.Time, intervalMinutes int) ([]LatencyPoint, error)

	// GetDailyUptime retrieves daily uptime data for heatmap
	GetDailyUptime(ctx context.Context, dependencyID int64, days int) ([]UptimePoint, error)

	// GetStats retrieves latency statistics
	GetStats(ctx context.Context, dependencyID int64, start, end time.Time) (*LatencyStats, error)

	// Cleanup removes old records
	Cleanup(ctx context.Context, olderThan time.Time) error
}

// GetUptimeStatus returns status color based on uptime percentage
func GetUptimeStatus(uptimePercent float64) string {
	if uptimePercent >= 99.0 {
		return "green"
	} else if uptimePercent >= 95.0 {
		return "yellow"
	}
	return "red"
}
