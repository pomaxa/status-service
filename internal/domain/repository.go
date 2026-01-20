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
	// GetIncidentsBySystemID calculates incident periods for a system within time range
	GetIncidentsBySystemID(ctx context.Context, systemID int64, start, end time.Time) ([]IncidentPeriod, error)

	// GetIncidentsByDependencyID calculates incident periods for a dependency within time range
	GetIncidentsByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) ([]IncidentPeriod, error)

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

// MaintenanceRepository defines operations for Maintenance persistence
type MaintenanceRepository interface {
	// Create persists a new maintenance window and sets its ID
	Create(ctx context.Context, m *Maintenance) error

	// GetByID retrieves a maintenance window by ID
	GetByID(ctx context.Context, id int64) (*Maintenance, error)

	// GetAll retrieves all maintenance windows
	GetAll(ctx context.Context) ([]*Maintenance, error)

	// GetActive retrieves currently active maintenance windows
	GetActive(ctx context.Context) ([]*Maintenance, error)

	// GetUpcoming retrieves scheduled maintenance windows
	GetUpcoming(ctx context.Context) ([]*Maintenance, error)

	// GetByTimeRange retrieves maintenance windows overlapping with time range
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]*Maintenance, error)

	// Update saves changes to an existing maintenance window
	Update(ctx context.Context, m *Maintenance) error

	// Delete removes a maintenance window by ID
	Delete(ctx context.Context, id int64) error
}

// IncidentRepository defines operations for Incident persistence
type IncidentRepository interface {
	// Create persists a new incident and sets its ID
	Create(ctx context.Context, incident *Incident) error

	// GetByID retrieves an incident by ID
	GetByID(ctx context.Context, id int64) (*Incident, error)

	// GetAll retrieves all incidents with optional limit
	GetAll(ctx context.Context, limit int) ([]*Incident, error)

	// GetActive retrieves all unresolved incidents
	GetActive(ctx context.Context) ([]*Incident, error)

	// GetRecent retrieves recent incidents (resolved in last N days)
	GetRecent(ctx context.Context, days int) ([]*Incident, error)

	// Update saves changes to an existing incident
	Update(ctx context.Context, incident *Incident) error

	// Delete removes an incident by ID
	Delete(ctx context.Context, id int64) error

	// CreateUpdate adds a timeline entry to an incident
	CreateUpdate(ctx context.Context, update *IncidentUpdate) error

	// GetUpdates retrieves all updates for an incident
	GetUpdates(ctx context.Context, incidentID int64) ([]*IncidentUpdate, error)
}

// SLAReportRepository defines operations for SLA Report persistence
type SLAReportRepository interface {
	// Create persists a new SLA report
	Create(ctx context.Context, report *SLAReport) error

	// GetByID retrieves an SLA report by ID
	GetByID(ctx context.Context, id int64) (*SLAReport, error)

	// GetAll retrieves all SLA reports with optional limit
	GetAll(ctx context.Context, limit int) ([]*SLAReport, error)

	// GetByPeriod retrieves reports within a time range
	GetByPeriod(ctx context.Context, start, end time.Time) ([]*SLAReport, error)

	// Delete removes an SLA report by ID
	Delete(ctx context.Context, id int64) error
}

// SLABreachRepository defines operations for SLA Breach persistence
type SLABreachRepository interface {
	// Create persists a new SLA breach
	Create(ctx context.Context, breach *SLABreachEvent) error

	// GetByID retrieves an SLA breach by ID
	GetByID(ctx context.Context, id int64) (*SLABreachEvent, error)

	// GetAll retrieves all breaches with optional limit
	GetAll(ctx context.Context, limit int) ([]*SLABreachEvent, error)

	// GetUnacknowledged retrieves all unacknowledged breaches
	GetUnacknowledged(ctx context.Context) ([]*SLABreachEvent, error)

	// GetBySystemID retrieves breaches for a system
	GetBySystemID(ctx context.Context, systemID int64, limit int) ([]*SLABreachEvent, error)

	// Acknowledge marks a breach as acknowledged
	Acknowledge(ctx context.Context, id int64, ackedBy string) error

	// GetByPeriod retrieves breaches within a time range
	GetByPeriod(ctx context.Context, start, end time.Time) ([]*SLABreachEvent, error)
}
