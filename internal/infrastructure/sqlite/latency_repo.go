package sqlite

import (
	"context"
	"database/sql"
	"status-incident/internal/domain"
	"time"
)

// LatencyRepo implements domain.LatencyRepository
type LatencyRepo struct {
	db *DB
}

// NewLatencyRepo creates a new LatencyRepo
func NewLatencyRepo(db *DB) *LatencyRepo {
	return &LatencyRepo{db: db}
}

// Record stores a new latency measurement
func (r *LatencyRepo) Record(ctx context.Context, record *domain.LatencyRecord) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO latency_history (dependency_id, latency_ms, success, status_code, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, record.DependencyID, record.LatencyMs, record.Success, record.StatusCode, time.Now())
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	record.ID = id
	record.CreatedAt = time.Now()
	return nil
}

// GetByDependency retrieves latency records for a dependency within time range
func (r *LatencyRepo) GetByDependency(ctx context.Context, dependencyID int64, start, end time.Time, limit int) ([]*domain.LatencyRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, dependency_id, latency_ms, success, status_code, created_at
		FROM latency_history
		WHERE dependency_id = ? AND created_at BETWEEN ? AND ?
		ORDER BY created_at DESC
		LIMIT ?
	`, dependencyID, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*domain.LatencyRecord
	for rows.Next() {
		var r domain.LatencyRecord
		if err := rows.Scan(&r.ID, &r.DependencyID, &r.LatencyMs, &r.Success, &r.StatusCode, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, &r)
	}
	return records, rows.Err()
}

// GetAggregated retrieves aggregated latency data for charting
func (r *LatencyRepo) GetAggregated(ctx context.Context, dependencyID int64, start, end time.Time, intervalMinutes int) ([]domain.LatencyPoint, error) {
	// Group by time intervals
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			datetime((strftime('%s', created_at) / (?*60)) * (?*60), 'unixepoch') as interval_start,
			AVG(latency_ms) as avg_ms,
			MIN(latency_ms) as min_ms,
			MAX(latency_ms) as max_ms,
			COUNT(*) as total,
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failures
		FROM latency_history
		WHERE dependency_id = ? AND created_at BETWEEN ? AND ?
		GROUP BY interval_start
		ORDER BY interval_start
	`, intervalMinutes, intervalMinutes, dependencyID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []domain.LatencyPoint
	for rows.Next() {
		var p domain.LatencyPoint
		var timestampStr string
		if err := rows.Scan(&timestampStr, &p.AvgMs, &p.MinMs, &p.MaxMs, &p.Count, &p.Failures); err != nil {
			return nil, err
		}
		p.Timestamp, _ = time.Parse("2006-01-02 15:04:05", timestampStr)
		points = append(points, p)
	}
	return points, rows.Err()
}

// GetDailyUptime retrieves daily uptime data for heatmap
func (r *LatencyRepo) GetDailyUptime(ctx context.Context, dependencyID int64, days int) ([]domain.UptimePoint, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			date(created_at) as day,
			COUNT(*) as total,
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failures
		FROM latency_history
		WHERE dependency_id = ? AND created_at >= ?
		GROUP BY day
		ORDER BY day
	`, dependencyID, startDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pointMap := make(map[string]domain.UptimePoint)
	for rows.Next() {
		var date string
		var total, failures int
		if err := rows.Scan(&date, &total, &failures); err != nil {
			return nil, err
		}
		uptime := 100.0
		if total > 0 {
			uptime = float64(total-failures) / float64(total) * 100
		}
		pointMap[date] = domain.UptimePoint{
			Date:          date,
			UptimePercent: uptime,
			TotalChecks:   total,
			FailedChecks:  failures,
			Status:        domain.GetUptimeStatus(uptime),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fill in missing days
	var points []domain.UptimePoint
	for i := days; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		if p, ok := pointMap[date]; ok {
			points = append(points, p)
		} else {
			points = append(points, domain.UptimePoint{
				Date:          date,
				UptimePercent: 100.0,
				TotalChecks:   0,
				FailedChecks:  0,
				Status:        "green",
			})
		}
	}
	return points, nil
}

// GetStats retrieves latency statistics
func (r *LatencyRepo) GetStats(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.LatencyStats, error) {
	var stats domain.LatencyStats
	stats.DependencyID = dependencyID

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(AVG(latency_ms), 0),
			COALESCE(MIN(latency_ms), 0),
			COALESCE(MAX(latency_ms), 0),
			COUNT(*),
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END)
		FROM latency_history
		WHERE dependency_id = ? AND created_at BETWEEN ? AND ?
	`, dependencyID, start, end).Scan(&stats.AvgLatencyMs, &stats.MinLatencyMs, &stats.MaxLatencyMs, &stats.TotalChecks, &stats.FailedChecks)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if stats.TotalChecks > 0 {
		stats.UptimePercent = float64(stats.TotalChecks-stats.FailedChecks) / float64(stats.TotalChecks) * 100
	} else {
		stats.UptimePercent = 100.0
	}

	// Get percentiles using SQLite window functions (or approximation)
	// For simplicity, we'll calculate P50, P95, P99 from sorted data
	rows, err := r.db.QueryContext(ctx, `
		SELECT latency_ms FROM latency_history
		WHERE dependency_id = ? AND created_at BETWEEN ? AND ? AND success = 1
		ORDER BY latency_ms
	`, dependencyID, start, end)
	if err != nil {
		return &stats, nil // Return stats without percentiles
	}
	defer rows.Close()

	var latencies []int64
	for rows.Next() {
		var l int64
		if err := rows.Scan(&l); err == nil {
			latencies = append(latencies, l)
		}
	}

	if len(latencies) > 0 {
		stats.P50LatencyMs = latencies[len(latencies)*50/100]
		stats.P95LatencyMs = latencies[len(latencies)*95/100]
		if len(latencies) > 100 {
			stats.P99LatencyMs = latencies[len(latencies)*99/100]
		} else {
			stats.P99LatencyMs = latencies[len(latencies)-1]
		}
	}

	return &stats, nil
}

// Cleanup removes old records
func (r *LatencyRepo) Cleanup(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM latency_history WHERE created_at < ?`, olderThan)
	return err
}
