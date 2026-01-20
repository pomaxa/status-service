package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"status-incident/internal/domain"
	"time"
)

// APIKeyRepo implements domain.APIKeyRepository
type APIKeyRepo struct {
	db *DB
}

// NewAPIKeyRepo creates a new APIKeyRepo
func NewAPIKeyRepo(db *DB) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

// Create persists a new API key
func (r *APIKeyRepo) Create(ctx context.Context, key *domain.APIKey) error {
	scopesJSON, err := json.Marshal(key.Scopes)
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO api_keys (name, key_value, key_hash, scopes, enabled, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, key.Name, key.Key, key.KeyHash, string(scopesJSON), key.Enabled, key.ExpiresAt, time.Now())
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	key.ID = id
	key.CreatedAt = time.Now()
	return nil
}

// GetByKey retrieves API key by the key value
func (r *APIKeyRepo) GetByKey(ctx context.Context, keyValue string) (*domain.APIKey, error) {
	var key domain.APIKey
	var scopesJSON string
	var expiresAt, lastUsed sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, key_value, key_hash, scopes, enabled, expires_at, last_used, created_at
		FROM api_keys
		WHERE key_value = ?
	`, keyValue).Scan(
		&key.ID,
		&key.Name,
		&key.Key,
		&key.KeyHash,
		&scopesJSON,
		&key.Enabled,
		&expiresAt,
		&lastUsed,
		&key.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(scopesJSON), &key.Scopes); err != nil {
		key.Scopes = []string{"read"}
	}

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if lastUsed.Valid {
		key.LastUsed = &lastUsed.Time
	}

	return &key, nil
}

// GetAll retrieves all API keys
func (r *APIKeyRepo) GetAll(ctx context.Context) ([]*domain.APIKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, key_value, key_hash, scopes, enabled, expires_at, last_used, created_at
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		var key domain.APIKey
		var scopesJSON string
		var expiresAt, lastUsed sql.NullTime

		if err := rows.Scan(
			&key.ID,
			&key.Name,
			&key.Key,
			&key.KeyHash,
			&scopesJSON,
			&key.Enabled,
			&expiresAt,
			&lastUsed,
			&key.CreatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(scopesJSON), &key.Scopes); err != nil {
			key.Scopes = []string{"read"}
		}

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if lastUsed.Valid {
			key.LastUsed = &lastUsed.Time
		}

		keys = append(keys, &key)
	}

	return keys, rows.Err()
}

// Update saves changes to an API key
func (r *APIKeyRepo) Update(ctx context.Context, key *domain.APIKey) error {
	scopesJSON, err := json.Marshal(key.Scopes)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE api_keys
		SET name = ?, scopes = ?, enabled = ?, expires_at = ?
		WHERE id = ?
	`, key.Name, string(scopesJSON), key.Enabled, key.ExpiresAt, key.ID)
	return err
}

// Delete removes an API key
func (r *APIKeyRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	return err
}

// UpdateLastUsed updates the last used timestamp
func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET last_used = ? WHERE id = ?`, time.Now(), id)
	return err
}
