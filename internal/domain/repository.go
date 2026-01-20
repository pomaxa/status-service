package domain

import (
	"context"
	"time"
)

// SystemRepository defines operations for System persistence
type SystemRepository interface {
	// Create persists a new system and sets its ID
	Create(ctx context.Context, system *System) error

	// GetByID retrieves a system by ID
	GetByID(ctx context.Context, id int64) (*System, error)

	// GetAll retrieves all systems
	GetAll(ctx context.Context) ([]*System, error)

	// Update saves changes to an existing system
	Update(ctx context.Context, system *System) error

	// Delete removes a system by ID
	Delete(ctx context.Context, id int64) error
}

// DependencyRepository defines operations for Dependency persistence
type DependencyRepository interface {
	// Create persists a new dependency and sets its ID
	Create(ctx context.Context, dep *Dependency) error

	// GetByID retrieves a dependency by ID
	GetByID(ctx context.Context, id int64) (*Dependency, error)

	// GetBySystemID retrieves all dependencies for a system
	GetBySystemID(ctx context.Context, systemID int64) ([]*Dependency, error)

	// GetAllWithHeartbeat retrieves all dependencies with heartbeat configured
	GetAllWithHeartbeat(ctx context.Context) ([]*Dependency, error)

	// Update saves changes to an existing dependency
	Update(ctx context.Context, dep *Dependency) error

	// Delete removes a dependency by ID
	Delete(ctx context.Context, id int64) error
}

// StatusLogRepository defines operations for StatusLog persistence
type StatusLogRepository interface {
	// Create persists a new status log entry
	Create(ctx context.Context, log *StatusLog) error

	// GetBySystemID retrieves logs for a system
	GetBySystemID(ctx context.Context, systemID int64, limit int) ([]*StatusLog, error)

	// GetByDependencyID retrieves logs for a dependency
	GetByDependencyID(ctx context.Context, dependencyID int64, limit int) ([]*StatusLog, error)

	// GetAll retrieves all logs with optional limit
	GetAll(ctx context.Context, limit int) ([]*StatusLog, error)

	// GetByTimeRange retrieves logs within a time range
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]*StatusLog, error)

	// GetSystemLogsByTimeRange retrieves system logs within time range
	GetSystemLogsByTimeRange(ctx context.Context, systemID int64, start, end time.Time) ([]*StatusLog, error)

	// GetDependencyLogsByTimeRange retrieves dependency logs within time range
	GetDependencyLogsByTimeRange(ctx context.Context, dependencyID int64, start, end time.Time) ([]*StatusLog, error)
}

// AnalyticsRepository defines operations for analytics queries
type AnalyticsRepository interface {
	// GetIncidentsBySystemID calculates incidents for a system within time range
	GetIncidentsBySystemID(ctx context.Context, systemID int64, start, end time.Time) ([]Incident, error)

	// GetIncidentsByDependencyID calculates incidents for a dependency within time range
	GetIncidentsByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) ([]Incident, error)

	// GetUptimeBySystemID calculates uptime metrics for a system
	GetUptimeBySystemID(ctx context.Context, systemID int64, start, end time.Time) (*Analytics, error)

	// GetUptimeByDependencyID calculates uptime metrics for a dependency
	GetUptimeByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) (*Analytics, error)

	// GetOverallAnalytics returns aggregate analytics for all systems
	GetOverallAnalytics(ctx context.Context, start, end time.Time) (*Analytics, error)
}

// HealthChecker defines interface for checking endpoint health
type HealthChecker interface {
	// Check performs HTTP health check and returns healthy status and response time
	Check(ctx context.Context, url string) (healthy bool, latencyMs int64, err error)
}

// WebhookRepository defines operations for Webhook persistence
type WebhookRepository interface {
	// Create persists a new webhook and sets its ID
	Create(ctx context.Context, webhook *Webhook) error

	// GetByID retrieves a webhook by ID
	GetByID(ctx context.Context, id int64) (*Webhook, error)

	// GetAll retrieves all webhooks
	GetAll(ctx context.Context) ([]*Webhook, error)

	// GetEnabled retrieves all enabled webhooks
	GetEnabled(ctx context.Context) ([]*Webhook, error)

	// Update saves changes to an existing webhook
	Update(ctx context.Context, webhook *Webhook) error

	// Delete removes a webhook by ID
	Delete(ctx context.Context, id int64) error
}
