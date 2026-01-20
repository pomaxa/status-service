package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"status-incident/internal/domain"
	"time"
)

// SLAReportRepo implements domain.SLAReportRepository
type SLAReportRepo struct {
	db *DB
}

// NewSLAReportRepo creates a new SLAReportRepo
func NewSLAReportRepo(db *DB) *SLAReportRepo {
	return &SLAReportRepo{db: db}
}

// Create persists a new SLA report
func (r *SLAReportRepo) Create(ctx context.Context, report *domain.SLAReport) error {
	reportData, err := json.Marshal(report.SystemReports)
	if err != nil {
		return fmt.Errorf("failed to marshal report data: %w", err)
	}

	query := `
		INSERT INTO sla_reports (title, period, period_start, period_end, generated_at, generated_by,
			overall_uptime, overall_availability, total_systems, systems_meeting_sla, systems_breaching_sla, report_data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		report.Title,
		report.Period,
		report.PeriodStart,
		report.PeriodEnd,
		report.GeneratedAt,
		report.GeneratedBy,
		report.OverallUptime,
		report.OverallAvailability,
		report.TotalSystems,
		report.SystemsMeetingSLA,
		report.SystemsBreachingSLA,
		string(reportData),
	)
	if err != nil {
		return fmt.Errorf("failed to create SLA report: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	report.ID = id
	return nil
}

// GetByID retrieves an SLA report by ID
func (r *SLAReportRepo) GetByID(ctx context.Context, id int64) (*domain.SLAReport, error) {
	query := `
		SELECT id, title, period, period_start, period_end, generated_at, generated_by,
			overall_uptime, overall_availability, total_systems, systems_meeting_sla, systems_breaching_sla, report_data
		FROM sla_reports
		WHERE id = ?
	`

	var report domain.SLAReport
	var reportData string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&report.ID,
		&report.Title,
		&report.Period,
		&report.PeriodStart,
		&report.PeriodEnd,
		&report.GeneratedAt,
		&report.GeneratedBy,
		&report.OverallUptime,
		&report.OverallAvailability,
		&report.TotalSystems,
		&report.SystemsMeetingSLA,
		&report.SystemsBreachingSLA,
		&reportData,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get SLA report: %w", err)
	}

	if err := json.Unmarshal([]byte(reportData), &report.SystemReports); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report data: %w", err)
	}

	return &report, nil
}

// GetAll retrieves all SLA reports with optional limit
func (r *SLAReportRepo) GetAll(ctx context.Context, limit int) ([]*domain.SLAReport, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, title, period, period_start, period_end, generated_at, generated_by,
			overall_uptime, overall_availability, total_systems, systems_meeting_sla, systems_breaching_sla, report_data
		FROM sla_reports
		ORDER BY generated_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query SLA reports: %w", err)
	}
	defer rows.Close()

	var reports []*domain.SLAReport
	for rows.Next() {
		var report domain.SLAReport
		var reportData string

		if err := rows.Scan(
			&report.ID,
			&report.Title,
			&report.Period,
			&report.PeriodStart,
			&report.PeriodEnd,
			&report.GeneratedAt,
			&report.GeneratedBy,
			&report.OverallUptime,
			&report.OverallAvailability,
			&report.TotalSystems,
			&report.SystemsMeetingSLA,
			&report.SystemsBreachingSLA,
			&reportData,
		); err != nil {
			return nil, fmt.Errorf("failed to scan SLA report: %w", err)
		}

		if err := json.Unmarshal([]byte(reportData), &report.SystemReports); err != nil {
			// Log error but continue
			report.SystemReports = nil
		}

		reports = append(reports, &report)
	}

	return reports, rows.Err()
}

// GetByPeriod retrieves reports within a time range
func (r *SLAReportRepo) GetByPeriod(ctx context.Context, start, end time.Time) ([]*domain.SLAReport, error) {
	query := `
		SELECT id, title, period, period_start, period_end, generated_at, generated_by,
			overall_uptime, overall_availability, total_systems, systems_meeting_sla, systems_breaching_sla, report_data
		FROM sla_reports
		WHERE period_start >= ? AND period_end <= ?
		ORDER BY generated_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query SLA reports by period: %w", err)
	}
	defer rows.Close()

	var reports []*domain.SLAReport
	for rows.Next() {
		var report domain.SLAReport
		var reportData string

		if err := rows.Scan(
			&report.ID,
			&report.Title,
			&report.Period,
			&report.PeriodStart,
			&report.PeriodEnd,
			&report.GeneratedAt,
			&report.GeneratedBy,
			&report.OverallUptime,
			&report.OverallAvailability,
			&report.TotalSystems,
			&report.SystemsMeetingSLA,
			&report.SystemsBreachingSLA,
			&reportData,
		); err != nil {
			return nil, fmt.Errorf("failed to scan SLA report: %w", err)
		}

		if err := json.Unmarshal([]byte(reportData), &report.SystemReports); err != nil {
			report.SystemReports = nil
		}

		reports = append(reports, &report)
	}

	return reports, rows.Err()
}

// Delete removes an SLA report by ID
func (r *SLAReportRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM sla_reports WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete SLA report: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("SLA report not found: %d", id)
	}

	return nil
}

// SLABreachRepo implements domain.SLABreachRepository
type SLABreachRepo struct {
	db *DB
}

// NewSLABreachRepo creates a new SLABreachRepo
func NewSLABreachRepo(db *DB) *SLABreachRepo {
	return &SLABreachRepo{db: db}
}

// Create persists a new SLA breach
func (r *SLABreachRepo) Create(ctx context.Context, breach *domain.SLABreachEvent) error {
	query := `
		INSERT INTO sla_breaches (system_id, breach_type, sla_target, actual_value, period,
			period_start, period_end, detected_at, acknowledged, acked_by, acked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		breach.SystemID,
		breach.BreachType,
		breach.SLATarget,
		breach.ActualValue,
		breach.Period,
		breach.PeriodStart,
		breach.PeriodEnd,
		breach.DetectedAt,
		breach.Acknowledged,
		breach.AckedBy,
		breach.AckedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create SLA breach: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	breach.ID = id
	return nil
}

// GetByID retrieves an SLA breach by ID
func (r *SLABreachRepo) GetByID(ctx context.Context, id int64) (*domain.SLABreachEvent, error) {
	query := `
		SELECT b.id, b.system_id, s.name, b.breach_type, b.sla_target, b.actual_value, b.period,
			b.period_start, b.period_end, b.detected_at, b.acknowledged, b.acked_by, b.acked_at
		FROM sla_breaches b
		LEFT JOIN systems s ON b.system_id = s.id
		WHERE b.id = ?
	`

	var breach domain.SLABreachEvent
	var systemName sql.NullString
	var ackedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&breach.ID,
		&breach.SystemID,
		&systemName,
		&breach.BreachType,
		&breach.SLATarget,
		&breach.ActualValue,
		&breach.Period,
		&breach.PeriodStart,
		&breach.PeriodEnd,
		&breach.DetectedAt,
		&breach.Acknowledged,
		&breach.AckedBy,
		&ackedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get SLA breach: %w", err)
	}

	if systemName.Valid {
		breach.SystemName = systemName.String
	}
	if ackedAt.Valid {
		breach.AckedAt = &ackedAt.Time
	}

	return &breach, nil
}

// GetAll retrieves all breaches with optional limit
func (r *SLABreachRepo) GetAll(ctx context.Context, limit int) ([]*domain.SLABreachEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT b.id, b.system_id, s.name, b.breach_type, b.sla_target, b.actual_value, b.period,
			b.period_start, b.period_end, b.detected_at, b.acknowledged, b.acked_by, b.acked_at
		FROM sla_breaches b
		LEFT JOIN systems s ON b.system_id = s.id
		ORDER BY b.detected_at DESC
		LIMIT ?
	`

	return r.scanBreaches(ctx, query, limit)
}

// GetUnacknowledged retrieves all unacknowledged breaches
func (r *SLABreachRepo) GetUnacknowledged(ctx context.Context) ([]*domain.SLABreachEvent, error) {
	query := `
		SELECT b.id, b.system_id, s.name, b.breach_type, b.sla_target, b.actual_value, b.period,
			b.period_start, b.period_end, b.detected_at, b.acknowledged, b.acked_by, b.acked_at
		FROM sla_breaches b
		LEFT JOIN systems s ON b.system_id = s.id
		WHERE b.acknowledged = 0
		ORDER BY b.detected_at DESC
	`

	return r.scanBreaches(ctx, query)
}

// GetBySystemID retrieves breaches for a system
func (r *SLABreachRepo) GetBySystemID(ctx context.Context, systemID int64, limit int) ([]*domain.SLABreachEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT b.id, b.system_id, s.name, b.breach_type, b.sla_target, b.actual_value, b.period,
			b.period_start, b.period_end, b.detected_at, b.acknowledged, b.acked_by, b.acked_at
		FROM sla_breaches b
		LEFT JOIN systems s ON b.system_id = s.id
		WHERE b.system_id = ?
		ORDER BY b.detected_at DESC
		LIMIT ?
	`

	return r.scanBreaches(ctx, query, systemID, limit)
}

// Acknowledge marks a breach as acknowledged
func (r *SLABreachRepo) Acknowledge(ctx context.Context, id int64, ackedBy string) error {
	query := `
		UPDATE sla_breaches
		SET acknowledged = 1, acked_by = ?, acked_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, ackedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to acknowledge breach: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("breach not found: %d", id)
	}

	return nil
}

// GetByPeriod retrieves breaches within a time range
func (r *SLABreachRepo) GetByPeriod(ctx context.Context, start, end time.Time) ([]*domain.SLABreachEvent, error) {
	query := `
		SELECT b.id, b.system_id, s.name, b.breach_type, b.sla_target, b.actual_value, b.period,
			b.period_start, b.period_end, b.detected_at, b.acknowledged, b.acked_by, b.acked_at
		FROM sla_breaches b
		LEFT JOIN systems s ON b.system_id = s.id
		WHERE b.period_start >= ? AND b.period_end <= ?
		ORDER BY b.detected_at DESC
	`

	return r.scanBreaches(ctx, query, start, end)
}

// scanBreaches is a helper to scan breach rows
func (r *SLABreachRepo) scanBreaches(ctx context.Context, query string, args ...interface{}) ([]*domain.SLABreachEvent, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query breaches: %w", err)
	}
	defer rows.Close()

	var breaches []*domain.SLABreachEvent
	for rows.Next() {
		var breach domain.SLABreachEvent
		var systemName sql.NullString
		var ackedAt sql.NullTime

		if err := rows.Scan(
			&breach.ID,
			&breach.SystemID,
			&systemName,
			&breach.BreachType,
			&breach.SLATarget,
			&breach.ActualValue,
			&breach.Period,
			&breach.PeriodStart,
			&breach.PeriodEnd,
			&breach.DetectedAt,
			&breach.Acknowledged,
			&breach.AckedBy,
			&ackedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan breach: %w", err)
		}

		if systemName.Valid {
			breach.SystemName = systemName.String
		}
		if ackedAt.Valid {
			breach.AckedAt = &ackedAt.Time
		}

		breaches = append(breaches, &breach)
	}

	return breaches, rows.Err()
}
