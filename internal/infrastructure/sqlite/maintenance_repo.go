package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"status-incident/internal/domain"
)

// MaintenanceRepo implements MaintenanceRepository for SQLite
type MaintenanceRepo struct {
	db *DB
}

// NewMaintenanceRepo creates a new MaintenanceRepo
func NewMaintenanceRepo(db *DB) *MaintenanceRepo {
	return &MaintenanceRepo{db: db}
}

// Create persists a new maintenance window
func (r *MaintenanceRepo) Create(ctx context.Context, m *domain.Maintenance) error {
	systemIDsJSON, _ := json.Marshal(m.SystemIDs)

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO maintenances (title, description, start_time, end_time, system_ids, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, m.Title, m.Description, m.StartTime, m.EndTime, string(systemIDsJSON), string(m.Status), m.CreatedAt, m.UpdatedAt)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}

// GetByID retrieves a maintenance window by ID
func (r *MaintenanceRepo) GetByID(ctx context.Context, id int64) (*domain.Maintenance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, start_time, end_time, system_ids, status, created_at, updated_at
		FROM maintenances WHERE id = ?
	`, id)

	return r.scanMaintenance(row)
}

// GetAll retrieves all maintenance windows
func (r *MaintenanceRepo) GetAll(ctx context.Context) ([]*domain.Maintenance, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, start_time, end_time, system_ids, status, created_at, updated_at
		FROM maintenances ORDER BY start_time DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMaintenances(rows)
}

// GetActive retrieves currently active maintenance windows
func (r *MaintenanceRepo) GetActive(ctx context.Context) ([]*domain.Maintenance, error) {
	now := time.Now()
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, start_time, end_time, system_ids, status, created_at, updated_at
		FROM maintenances
		WHERE status != 'cancelled' AND start_time <= ? AND end_time >= ?
		ORDER BY start_time ASC
	`, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	maintenances, err := r.scanMaintenances(rows)
	if err != nil {
		return nil, err
	}

	// Refresh statuses
	for _, m := range maintenances {
		m.RefreshStatus()
	}

	return maintenances, nil
}

// GetUpcoming retrieves scheduled maintenance windows
func (r *MaintenanceRepo) GetUpcoming(ctx context.Context) ([]*domain.Maintenance, error) {
	now := time.Now()
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, start_time, end_time, system_ids, status, created_at, updated_at
		FROM maintenances
		WHERE status = 'scheduled' AND start_time > ?
		ORDER BY start_time ASC
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMaintenances(rows)
}

// GetByTimeRange retrieves maintenance windows overlapping with time range
func (r *MaintenanceRepo) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.Maintenance, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, start_time, end_time, system_ids, status, created_at, updated_at
		FROM maintenances
		WHERE status != 'cancelled' AND start_time <= ? AND end_time >= ?
		ORDER BY start_time ASC
	`, end, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMaintenances(rows)
}

// Update saves changes to an existing maintenance window
func (r *MaintenanceRepo) Update(ctx context.Context, m *domain.Maintenance) error {
	systemIDsJSON, _ := json.Marshal(m.SystemIDs)

	_, err := r.db.ExecContext(ctx, `
		UPDATE maintenances
		SET title = ?, description = ?, start_time = ?, end_time = ?,
		    system_ids = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, m.Title, m.Description, m.StartTime, m.EndTime,
		string(systemIDsJSON), string(m.Status), m.UpdatedAt, m.ID)

	return err
}

// Delete removes a maintenance window by ID
func (r *MaintenanceRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM maintenances WHERE id = ?", id)
	return err
}

func (r *MaintenanceRepo) scanMaintenance(row *sql.Row) (*domain.Maintenance, error) {
	var m domain.Maintenance
	var systemIDsJSON string
	var status string

	err := row.Scan(
		&m.ID, &m.Title, &m.Description, &m.StartTime, &m.EndTime,
		&systemIDsJSON, &status, &m.CreatedAt, &m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if systemIDsJSON != "" && systemIDsJSON != "null" {
		json.Unmarshal([]byte(systemIDsJSON), &m.SystemIDs)
	}
	m.Status = domain.MaintenanceStatus(status)
	m.RefreshStatus()

	return &m, nil
}

func (r *MaintenanceRepo) scanMaintenances(rows *sql.Rows) ([]*domain.Maintenance, error) {
	var maintenances []*domain.Maintenance

	for rows.Next() {
		var m domain.Maintenance
		var systemIDsJSON string
		var status string

		err := rows.Scan(
			&m.ID, &m.Title, &m.Description, &m.StartTime, &m.EndTime,
			&systemIDsJSON, &status, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if systemIDsJSON != "" && systemIDsJSON != "null" {
			json.Unmarshal([]byte(systemIDsJSON), &m.SystemIDs)
		}
		m.Status = domain.MaintenanceStatus(status)

		maintenances = append(maintenances, &m)
	}

	return maintenances, nil
}
