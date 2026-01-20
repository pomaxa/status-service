package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"status-incident/internal/domain"
)

// WebhookRepo implements domain.WebhookRepository
type WebhookRepo struct {
	db *DB
}

// NewWebhookRepo creates a new WebhookRepo
func NewWebhookRepo(db *DB) *WebhookRepo {
	return &WebhookRepo{db: db}
}

// Create persists a new webhook
func (r *WebhookRepo) Create(ctx context.Context, webhook *domain.Webhook) error {
	query := `
		INSERT INTO webhooks (name, url, type, events, system_ids, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		webhook.Name,
		webhook.URL,
		webhook.Type,
		webhook.EventsJSON(),
		webhook.SystemIDsJSON(),
		webhook.Enabled,
		webhook.CreatedAt,
		webhook.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get webhook ID: %w", err)
	}

	webhook.ID = id
	return nil
}

// GetByID retrieves a webhook by ID
func (r *WebhookRepo) GetByID(ctx context.Context, id int64) (*domain.Webhook, error) {
	query := `
		SELECT id, name, url, type, events, system_ids, enabled, created_at, updated_at
		FROM webhooks
		WHERE id = ?
	`

	webhook := &domain.Webhook{}
	var eventsJSON string
	var systemIDsJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&webhook.ID,
		&webhook.Name,
		&webhook.URL,
		&webhook.Type,
		&eventsJSON,
		&systemIDsJSON,
		&webhook.Enabled,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	webhook.Events = domain.ParseEventsJSON(eventsJSON)
	if systemIDsJSON.Valid {
		webhook.SystemIDs = domain.ParseSystemIDsJSON(&systemIDsJSON.String)
	}

	return webhook, nil
}

// GetAll retrieves all webhooks
func (r *WebhookRepo) GetAll(ctx context.Context) ([]*domain.Webhook, error) {
	query := `
		SELECT id, name, url, type, events, system_ids, enabled, created_at, updated_at
		FROM webhooks
		ORDER BY created_at DESC
	`

	return r.queryWebhooks(ctx, query)
}

// GetEnabled retrieves all enabled webhooks
func (r *WebhookRepo) GetEnabled(ctx context.Context) ([]*domain.Webhook, error) {
	query := `
		SELECT id, name, url, type, events, system_ids, enabled, created_at, updated_at
		FROM webhooks
		WHERE enabled = 1
		ORDER BY created_at DESC
	`

	return r.queryWebhooks(ctx, query)
}

func (r *WebhookRepo) queryWebhooks(ctx context.Context, query string, args ...interface{}) ([]*domain.Webhook, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*domain.Webhook
	for rows.Next() {
		webhook := &domain.Webhook{}
		var eventsJSON string
		var systemIDsJSON sql.NullString

		err := rows.Scan(
			&webhook.ID,
			&webhook.Name,
			&webhook.URL,
			&webhook.Type,
			&eventsJSON,
			&systemIDsJSON,
			&webhook.Enabled,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}

		webhook.Events = domain.ParseEventsJSON(eventsJSON)
		if systemIDsJSON.Valid {
			webhook.SystemIDs = domain.ParseSystemIDsJSON(&systemIDsJSON.String)
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// Update saves changes to an existing webhook
func (r *WebhookRepo) Update(ctx context.Context, webhook *domain.Webhook) error {
	query := `
		UPDATE webhooks
		SET name = ?, url = ?, type = ?, events = ?, system_ids = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		webhook.Name,
		webhook.URL,
		webhook.Type,
		webhook.EventsJSON(),
		webhook.SystemIDsJSON(),
		webhook.Enabled,
		webhook.UpdatedAt,
		webhook.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	return nil
}

// Delete removes a webhook by ID
func (r *WebhookRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM webhooks WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}
