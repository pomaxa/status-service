package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"status-incident/internal/domain"
)

// DependencyRepo implements domain.DependencyRepository
type DependencyRepo struct {
	db *DB
}

// NewDependencyRepo creates a new DependencyRepo
func NewDependencyRepo(db *DB) *DependencyRepo {
	return &DependencyRepo{db: db}
}

// Create persists a new dependency and sets its ID
func (r *DependencyRepo) Create(ctx context.Context, dep *domain.Dependency) error {
	query := `
		INSERT INTO dependencies (system_id, name, description, status, heartbeat_url,
			heartbeat_interval, heartbeat_method, heartbeat_headers, heartbeat_body,
			heartbeat_expect_status, heartbeat_expect_body, last_check, last_latency,
			last_status_code, consecutive_failures, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var lastCheck interface{}
	if !dep.LastCheck.IsZero() {
		lastCheck = dep.LastCheck
	}

	headersJSON := encodeHeaders(dep.HeartbeatHeaders)

	result, err := r.db.ExecContext(ctx, query,
		dep.SystemID,
		dep.Name,
		dep.Description,
		dep.Status.String(),
		nullString(dep.HeartbeatURL),
		dep.HeartbeatInterval,
		nullString(dep.HeartbeatMethod),
		headersJSON,
		dep.HeartbeatBody,
		dep.HeartbeatExpectStatus,
		dep.HeartbeatExpectBody,
		lastCheck,
		dep.LastLatency,
		dep.LastStatusCode,
		dep.ConsecutiveFailures,
		dep.CreatedAt,
		dep.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create dependency: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	dep.ID = id
	return nil
}

// GetByID retrieves a dependency by ID
func (r *DependencyRepo) GetByID(ctx context.Context, id int64) (*domain.Dependency, error) {
	query := `
		SELECT id, system_id, name, description, status, heartbeat_url,
			heartbeat_interval, heartbeat_method, heartbeat_headers, heartbeat_body,
			heartbeat_expect_status, heartbeat_expect_body, last_check, last_latency,
			last_status_code, consecutive_failures, created_at, updated_at
		FROM dependencies
		WHERE id = ?
	`

	dep, err := r.scanDependency(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency: %w", err)
	}

	return dep, nil
}

// GetBySystemID retrieves all dependencies for a system
func (r *DependencyRepo) GetBySystemID(ctx context.Context, systemID int64) ([]*domain.Dependency, error) {
	query := `
		SELECT id, system_id, name, description, status, heartbeat_url,
			heartbeat_interval, heartbeat_method, heartbeat_headers, heartbeat_body,
			heartbeat_expect_status, heartbeat_expect_body, last_check, last_latency,
			last_status_code, consecutive_failures, created_at, updated_at
		FROM dependencies
		WHERE system_id = ?
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependencies: %w", err)
	}
	defer rows.Close()

	return r.scanDependencies(rows)
}

// GetAllWithHeartbeat retrieves all dependencies with heartbeat configured
func (r *DependencyRepo) GetAllWithHeartbeat(ctx context.Context) ([]*domain.Dependency, error) {
	query := `
		SELECT id, system_id, name, description, status, heartbeat_url,
			heartbeat_interval, heartbeat_method, heartbeat_headers, heartbeat_body,
			heartbeat_expect_status, heartbeat_expect_body, last_check, last_latency,
			last_status_code, consecutive_failures, created_at, updated_at
		FROM dependencies
		WHERE heartbeat_url IS NOT NULL AND heartbeat_url != ''
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependencies with heartbeat: %w", err)
	}
	defer rows.Close()

	return r.scanDependencies(rows)
}

// Update saves changes to an existing dependency
func (r *DependencyRepo) Update(ctx context.Context, dep *domain.Dependency) error {
	query := `
		UPDATE dependencies
		SET name = ?, description = ?, status = ?, heartbeat_url = ?,
			heartbeat_interval = ?, heartbeat_method = ?, heartbeat_headers = ?,
			heartbeat_body = ?, heartbeat_expect_status = ?, heartbeat_expect_body = ?,
			last_check = ?, last_latency = ?, last_status_code = ?, consecutive_failures = ?, updated_at = ?
		WHERE id = ?
	`

	var lastCheck interface{}
	if !dep.LastCheck.IsZero() {
		lastCheck = dep.LastCheck
	}

	headersJSON := encodeHeaders(dep.HeartbeatHeaders)

	result, err := r.db.ExecContext(ctx, query,
		dep.Name,
		dep.Description,
		dep.Status.String(),
		nullString(dep.HeartbeatURL),
		dep.HeartbeatInterval,
		nullString(dep.HeartbeatMethod),
		headersJSON,
		dep.HeartbeatBody,
		dep.HeartbeatExpectStatus,
		dep.HeartbeatExpectBody,
		lastCheck,
		dep.LastLatency,
		dep.LastStatusCode,
		dep.ConsecutiveFailures,
		dep.UpdatedAt,
		dep.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update dependency: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("dependency not found: %d", dep.ID)
	}

	return nil
}

// Delete removes a dependency by ID
func (r *DependencyRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM dependencies WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dependency: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("dependency not found: %d", id)
	}

	return nil
}

func (r *DependencyRepo) scanDependency(row *sql.Row) (*domain.Dependency, error) {
	var dep domain.Dependency
	var statusStr string
	var heartbeatURL, heartbeatMethod, heartbeatHeaders sql.NullString
	var lastCheck sql.NullTime

	err := row.Scan(
		&dep.ID,
		&dep.SystemID,
		&dep.Name,
		&dep.Description,
		&statusStr,
		&heartbeatURL,
		&dep.HeartbeatInterval,
		&heartbeatMethod,
		&heartbeatHeaders,
		&dep.HeartbeatBody,
		&dep.HeartbeatExpectStatus,
		&dep.HeartbeatExpectBody,
		&lastCheck,
		&dep.LastLatency,
		&dep.LastStatusCode,
		&dep.ConsecutiveFailures,
		&dep.CreatedAt,
		&dep.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	status, _ := domain.NewStatus(statusStr)
	dep.Status = status

	if heartbeatURL.Valid {
		dep.HeartbeatURL = heartbeatURL.String
	}
	if heartbeatMethod.Valid {
		dep.HeartbeatMethod = heartbeatMethod.String
	}
	if heartbeatHeaders.Valid {
		dep.HeartbeatHeaders = decodeHeaders(heartbeatHeaders.String)
	}
	if lastCheck.Valid {
		dep.LastCheck = lastCheck.Time
	}

	return &dep, nil
}

func (r *DependencyRepo) scanDependencies(rows *sql.Rows) ([]*domain.Dependency, error) {
	var deps []*domain.Dependency

	for rows.Next() {
		var dep domain.Dependency
		var statusStr string
		var heartbeatURL, heartbeatMethod, heartbeatHeaders sql.NullString
		var lastCheck sql.NullTime

		if err := rows.Scan(
			&dep.ID,
			&dep.SystemID,
			&dep.Name,
			&dep.Description,
			&statusStr,
			&heartbeatURL,
			&dep.HeartbeatInterval,
			&heartbeatMethod,
			&heartbeatHeaders,
			&dep.HeartbeatBody,
			&dep.HeartbeatExpectStatus,
			&dep.HeartbeatExpectBody,
			&lastCheck,
			&dep.LastLatency,
			&dep.LastStatusCode,
			&dep.ConsecutiveFailures,
			&dep.CreatedAt,
			&dep.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}

		status, _ := domain.NewStatus(statusStr)
		dep.Status = status

		if heartbeatURL.Valid {
			dep.HeartbeatURL = heartbeatURL.String
		}
		if heartbeatMethod.Valid {
			dep.HeartbeatMethod = heartbeatMethod.String
		}
		if heartbeatHeaders.Valid {
			dep.HeartbeatHeaders = decodeHeaders(heartbeatHeaders.String)
		}
		if lastCheck.Valid {
			dep.LastCheck = lastCheck.Time
		}

		deps = append(deps, &dep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dependencies: %w", err)
	}

	return deps, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func encodeHeaders(headers map[string]string) interface{} {
	if len(headers) == 0 {
		return nil
	}
	data, _ := json.Marshal(headers)
	return string(data)
}

func decodeHeaders(data string) map[string]string {
	if data == "" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(data), &headers); err != nil {
		return nil
	}
	return headers
}
