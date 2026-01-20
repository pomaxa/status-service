package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"status-incident/internal/domain"
)

// IncidentRepo implements IncidentRepository for SQLite
type IncidentRepo struct {
	db *DB
}

// NewIncidentRepo creates a new IncidentRepo
func NewIncidentRepo(db *DB) *IncidentRepo {
	return &IncidentRepo{db: db}
}

// Create persists a new incident
func (r *IncidentRepo) Create(ctx context.Context, i *domain.Incident) error {
	systemIDsJSON, _ := json.Marshal(i.SystemIDs)

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO incidents (title, status, severity, system_ids, message, postmortem,
			created_at, updated_at, resolved_at, acknowledged_at, acknowledged_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, i.Title, string(i.Status), string(i.Severity), string(systemIDsJSON), i.Message, i.Postmortem,
		i.CreatedAt, i.UpdatedAt, i.ResolvedAt, i.AcknowledgedAt, i.AcknowledgedBy)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	i.ID = id
	return nil
}

// GetByID retrieves an incident by ID
func (r *IncidentRepo) GetByID(ctx context.Context, id int64) (*domain.Incident, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, status, severity, system_ids, message, postmortem,
			created_at, updated_at, resolved_at, acknowledged_at, acknowledged_by
		FROM incidents WHERE id = ?
	`, id)

	return r.scanIncident(row)
}

// GetAll retrieves all incidents with optional limit
func (r *IncidentRepo) GetAll(ctx context.Context, limit int) ([]*domain.Incident, error) {
	query := `
		SELECT id, title, status, severity, system_ids, message, postmortem,
			created_at, updated_at, resolved_at, acknowledged_at, acknowledged_by
		FROM incidents ORDER BY created_at DESC
	`
	if limit > 0 {
		query += " LIMIT ?"
	}

	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = r.db.QueryContext(ctx, query, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanIncidents(rows)
}

// GetActive retrieves all unresolved incidents
func (r *IncidentRepo) GetActive(ctx context.Context) ([]*domain.Incident, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, status, severity, system_ids, message, postmortem,
			created_at, updated_at, resolved_at, acknowledged_at, acknowledged_by
		FROM incidents
		WHERE status != 'resolved'
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'major' THEN 2
				ELSE 3
			END,
			created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanIncidents(rows)
}

// GetRecent retrieves incidents resolved in last N days
func (r *IncidentRepo) GetRecent(ctx context.Context, days int) ([]*domain.Incident, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, status, severity, system_ids, message, postmortem,
			created_at, updated_at, resolved_at, acknowledged_at, acknowledged_by
		FROM incidents
		WHERE status = 'resolved' AND resolved_at >= ?
		ORDER BY resolved_at DESC
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanIncidents(rows)
}

// Update saves changes to an existing incident
func (r *IncidentRepo) Update(ctx context.Context, i *domain.Incident) error {
	systemIDsJSON, _ := json.Marshal(i.SystemIDs)

	_, err := r.db.ExecContext(ctx, `
		UPDATE incidents
		SET title = ?, status = ?, severity = ?, system_ids = ?, message = ?, postmortem = ?,
			updated_at = ?, resolved_at = ?, acknowledged_at = ?, acknowledged_by = ?
		WHERE id = ?
	`, i.Title, string(i.Status), string(i.Severity), string(systemIDsJSON), i.Message, i.Postmortem,
		i.UpdatedAt, i.ResolvedAt, i.AcknowledgedAt, i.AcknowledgedBy, i.ID)

	return err
}

// Delete removes an incident by ID
func (r *IncidentRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM incidents WHERE id = ?", id)
	return err
}

// CreateUpdate adds a timeline entry to an incident
func (r *IncidentRepo) CreateUpdate(ctx context.Context, u *domain.IncidentUpdate) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO incident_updates (incident_id, status, message, created_at, created_by)
		VALUES (?, ?, ?, ?, ?)
	`, u.IncidentID, string(u.Status), u.Message, u.CreatedAt, u.CreatedBy)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = id
	return nil
}

// GetUpdates retrieves all updates for an incident
func (r *IncidentRepo) GetUpdates(ctx context.Context, incidentID int64) ([]*domain.IncidentUpdate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, incident_id, status, message, created_at, created_by
		FROM incident_updates
		WHERE incident_id = ?
		ORDER BY created_at DESC
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var updates []*domain.IncidentUpdate
	for rows.Next() {
		var u domain.IncidentUpdate
		var status string
		err := rows.Scan(&u.ID, &u.IncidentID, &status, &u.Message, &u.CreatedAt, &u.CreatedBy)
		if err != nil {
			return nil, err
		}
		u.Status = domain.IncidentStatus(status)
		updates = append(updates, &u)
	}

	return updates, nil
}

func (r *IncidentRepo) scanIncident(row *sql.Row) (*domain.Incident, error) {
	var i domain.Incident
	var systemIDsJSON string
	var status, severity string

	err := row.Scan(
		&i.ID, &i.Title, &status, &severity, &systemIDsJSON, &i.Message, &i.Postmortem,
		&i.CreatedAt, &i.UpdatedAt, &i.ResolvedAt, &i.AcknowledgedAt, &i.AcknowledgedBy,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if systemIDsJSON != "" && systemIDsJSON != "null" {
		json.Unmarshal([]byte(systemIDsJSON), &i.SystemIDs)
	}
	i.Status = domain.IncidentStatus(status)
	i.Severity = domain.IncidentSeverity(severity)

	return &i, nil
}

func (r *IncidentRepo) scanIncidents(rows *sql.Rows) ([]*domain.Incident, error) {
	var incidents []*domain.Incident

	for rows.Next() {
		var i domain.Incident
		var systemIDsJSON string
		var status, severity string

		err := rows.Scan(
			&i.ID, &i.Title, &status, &severity, &systemIDsJSON, &i.Message, &i.Postmortem,
			&i.CreatedAt, &i.UpdatedAt, &i.ResolvedAt, &i.AcknowledgedAt, &i.AcknowledgedBy,
		)
		if err != nil {
			return nil, err
		}

		if systemIDsJSON != "" && systemIDsJSON != "null" {
			json.Unmarshal([]byte(systemIDsJSON), &i.SystemIDs)
		}
		i.Status = domain.IncidentStatus(status)
		i.Severity = domain.IncidentSeverity(severity)

		incidents = append(incidents, &i)
	}

	return incidents, nil
}
