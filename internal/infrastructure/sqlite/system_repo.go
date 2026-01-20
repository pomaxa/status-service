package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"status-incident/internal/domain"
)

// SystemRepo implements domain.SystemRepository
type SystemRepo struct {
	db *DB
}

// NewSystemRepo creates a new SystemRepo
func NewSystemRepo(db *DB) *SystemRepo {
	return &SystemRepo{db: db}
}

// Create persists a new system and sets its ID
func (r *SystemRepo) Create(ctx context.Context, system *domain.System) error {
	query := `
		INSERT INTO systems (name, description, url, owner, status, sla_target, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		system.Name,
		system.Description,
		system.URL,
		system.Owner,
		system.Status.String(),
		system.GetSLATarget(),
		system.CreatedAt,
		system.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create system: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	system.ID = id
	return nil
}

// GetByID retrieves a system by ID
func (r *SystemRepo) GetByID(ctx context.Context, id int64) (*domain.System, error) {
	query := `
		SELECT id, name, description, url, owner, status, sla_target, created_at, updated_at
		FROM systems
		WHERE id = ?
	`

	var system domain.System
	var statusStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&system.ID,
		&system.Name,
		&system.Description,
		&system.URL,
		&system.Owner,
		&statusStr,
		&system.SLATarget,
		&system.CreatedAt,
		&system.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get system: %w", err)
	}

	status, _ := domain.NewStatus(statusStr)
	system.Status = status

	return &system, nil
}

// GetAll retrieves all systems
func (r *SystemRepo) GetAll(ctx context.Context) ([]*domain.System, error) {
	query := `
		SELECT id, name, description, url, owner, status, sla_target, created_at, updated_at
		FROM systems
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query systems: %w", err)
	}
	defer rows.Close()

	var systems []*domain.System
	for rows.Next() {
		var system domain.System
		var statusStr string

		if err := rows.Scan(
			&system.ID,
			&system.Name,
			&system.Description,
			&system.URL,
			&system.Owner,
			&statusStr,
			&system.SLATarget,
			&system.CreatedAt,
			&system.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan system: %w", err)
		}

		status, _ := domain.NewStatus(statusStr)
		system.Status = status
		systems = append(systems, &system)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating systems: %w", err)
	}

	return systems, nil
}

// Update saves changes to an existing system
func (r *SystemRepo) Update(ctx context.Context, system *domain.System) error {
	query := `
		UPDATE systems
		SET name = ?, description = ?, url = ?, owner = ?, status = ?, sla_target = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		system.Name,
		system.Description,
		system.URL,
		system.Owner,
		system.Status.String(),
		system.GetSLATarget(),
		system.UpdatedAt,
		system.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update system: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("system not found: %d", system.ID)
	}

	return nil
}

// Delete removes a system by ID
func (r *SystemRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM systems WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete system: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("system not found: %d", id)
	}

	return nil
}
