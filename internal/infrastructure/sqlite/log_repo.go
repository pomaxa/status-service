package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"status-incident/internal/domain"
	"time"
)

// LogRepo implements domain.StatusLogRepository
type LogRepo struct {
	db *DB
}

// NewLogRepo creates a new LogRepo
func NewLogRepo(db *DB) *LogRepo {
	return &LogRepo{db: db}
}

// Create persists a new status log entry
func (r *LogRepo) Create(ctx context.Context, log *domain.StatusLog) error {
	query := `
		INSERT INTO status_log (system_id, dependency_id, old_status, new_status, message, source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		log.SystemID,
		log.DependencyID,
		log.OldStatus.String(),
		log.NewStatus.String(),
		log.Message,
		string(log.Source),
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create status log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	log.ID = id
	return nil
}

// GetBySystemID retrieves logs for a system
func (r *LogRepo) GetBySystemID(ctx context.Context, systemID int64, limit int) ([]*domain.StatusLog, error) {
	query := `
		SELECT id, system_id, dependency_id, old_status, new_status, message, source, created_at
		FROM status_log
		WHERE system_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, systemID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetByDependencyID retrieves logs for a dependency
func (r *LogRepo) GetByDependencyID(ctx context.Context, dependencyID int64, limit int) ([]*domain.StatusLog, error) {
	query := `
		SELECT id, system_id, dependency_id, old_status, new_status, message, source, created_at
		FROM status_log
		WHERE dependency_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, dependencyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetAll retrieves all logs with optional limit
func (r *LogRepo) GetAll(ctx context.Context, limit int) ([]*domain.StatusLog, error) {
	query := `
		SELECT id, system_id, dependency_id, old_status, new_status, message, source, created_at
		FROM status_log
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetByTimeRange retrieves logs within a time range
func (r *LogRepo) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.StatusLog, error) {
	query := `
		SELECT id, system_id, dependency_id, old_status, new_status, message, source, created_at
		FROM status_log
		WHERE created_at >= ? AND created_at <= ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs by time range: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetSystemLogsByTimeRange retrieves system logs within time range
func (r *LogRepo) GetSystemLogsByTimeRange(ctx context.Context, systemID int64, start, end time.Time) ([]*domain.StatusLog, error) {
	query := `
		SELECT id, system_id, dependency_id, old_status, new_status, message, source, created_at
		FROM status_log
		WHERE system_id = ? AND created_at >= ? AND created_at <= ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, systemID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query system logs by time range: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetDependencyLogsByTimeRange retrieves dependency logs within time range
func (r *LogRepo) GetDependencyLogsByTimeRange(ctx context.Context, dependencyID int64, start, end time.Time) ([]*domain.StatusLog, error) {
	query := `
		SELECT id, system_id, dependency_id, old_status, new_status, message, source, created_at
		FROM status_log
		WHERE dependency_id = ? AND created_at >= ? AND created_at <= ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, dependencyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependency logs by time range: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

func (r *LogRepo) scanLogs(rows *sql.Rows) ([]*domain.StatusLog, error) {
	var logs []*domain.StatusLog

	for rows.Next() {
		var log domain.StatusLog
		var systemID, dependencyID sql.NullInt64
		var oldStatusStr, newStatusStr, sourceStr string

		if err := rows.Scan(
			&log.ID,
			&systemID,
			&dependencyID,
			&oldStatusStr,
			&newStatusStr,
			&log.Message,
			&sourceStr,
			&log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		if systemID.Valid {
			log.SystemID = &systemID.Int64
		}
		if dependencyID.Valid {
			log.DependencyID = &dependencyID.Int64
		}

		oldStatus, _ := domain.NewStatus(oldStatusStr)
		newStatus, _ := domain.NewStatus(newStatusStr)
		log.OldStatus = oldStatus
		log.NewStatus = newStatus
		log.Source = domain.ChangeSource(sourceStr)

		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating logs: %w", err)
	}

	return logs, nil
}
