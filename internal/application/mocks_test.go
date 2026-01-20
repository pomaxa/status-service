package application

import (
	"context"
	"status-incident/internal/domain"
	"time"
)

// MockSystemRepository is a mock implementation of domain.SystemRepository
type MockSystemRepository struct {
	Systems      map[int64]*domain.System
	CreateFunc   func(ctx context.Context, s *domain.System) error
	GetByIDFunc  func(ctx context.Context, id int64) (*domain.System, error)
	GetAllFunc   func(ctx context.Context) ([]*domain.System, error)
	UpdateFunc   func(ctx context.Context, s *domain.System) error
	DeleteFunc   func(ctx context.Context, id int64) error
}

func NewMockSystemRepository() *MockSystemRepository {
	return &MockSystemRepository{
		Systems: make(map[int64]*domain.System),
	}
}

func (m *MockSystemRepository) Create(ctx context.Context, s *domain.System) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, s)
	}
	s.ID = int64(len(m.Systems) + 1)
	m.Systems[s.ID] = s
	return nil
}

func (m *MockSystemRepository) GetByID(ctx context.Context, id int64) (*domain.System, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return m.Systems[id], nil
}

func (m *MockSystemRepository) GetAll(ctx context.Context) ([]*domain.System, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx)
	}
	var result []*domain.System
	for _, s := range m.Systems {
		result = append(result, s)
	}
	return result, nil
}

func (m *MockSystemRepository) Update(ctx context.Context, s *domain.System) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, s)
	}
	m.Systems[s.ID] = s
	return nil
}

func (m *MockSystemRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	delete(m.Systems, id)
	return nil
}

// MockDependencyRepository is a mock implementation of domain.DependencyRepository
type MockDependencyRepository struct {
	Dependencies    map[int64]*domain.Dependency
	CreateFunc      func(ctx context.Context, d *domain.Dependency) error
	GetByIDFunc     func(ctx context.Context, id int64) (*domain.Dependency, error)
	GetBySystemIDFunc func(ctx context.Context, systemID int64) ([]*domain.Dependency, error)
	GetAllFunc      func(ctx context.Context) ([]*domain.Dependency, error)
	UpdateFunc      func(ctx context.Context, d *domain.Dependency) error
	DeleteFunc      func(ctx context.Context, id int64) error
	GetWithHeartbeatFunc func(ctx context.Context) ([]*domain.Dependency, error)
}

func NewMockDependencyRepository() *MockDependencyRepository {
	return &MockDependencyRepository{
		Dependencies: make(map[int64]*domain.Dependency),
	}
}

func (m *MockDependencyRepository) Create(ctx context.Context, d *domain.Dependency) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, d)
	}
	d.ID = int64(len(m.Dependencies) + 1)
	m.Dependencies[d.ID] = d
	return nil
}

func (m *MockDependencyRepository) GetByID(ctx context.Context, id int64) (*domain.Dependency, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return m.Dependencies[id], nil
}

func (m *MockDependencyRepository) GetBySystemID(ctx context.Context, systemID int64) ([]*domain.Dependency, error) {
	if m.GetBySystemIDFunc != nil {
		return m.GetBySystemIDFunc(ctx, systemID)
	}
	var result []*domain.Dependency
	for _, d := range m.Dependencies {
		if d.SystemID == systemID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *MockDependencyRepository) GetAll(ctx context.Context) ([]*domain.Dependency, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx)
	}
	var result []*domain.Dependency
	for _, d := range m.Dependencies {
		result = append(result, d)
	}
	return result, nil
}

func (m *MockDependencyRepository) Update(ctx context.Context, d *domain.Dependency) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, d)
	}
	m.Dependencies[d.ID] = d
	return nil
}

func (m *MockDependencyRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	delete(m.Dependencies, id)
	return nil
}

func (m *MockDependencyRepository) GetAllWithHeartbeat(ctx context.Context) ([]*domain.Dependency, error) {
	if m.GetWithHeartbeatFunc != nil {
		return m.GetWithHeartbeatFunc(ctx)
	}
	var result []*domain.Dependency
	for _, d := range m.Dependencies {
		if d.HasHeartbeat() {
			result = append(result, d)
		}
	}
	return result, nil
}

// MockStatusLogRepository is a mock implementation of domain.StatusLogRepository
type MockStatusLogRepository struct {
	Logs       []*domain.StatusLog
	CreateFunc func(ctx context.Context, log *domain.StatusLog) error
	GetBySystemIDFunc func(ctx context.Context, systemID int64, limit int) ([]*domain.StatusLog, error)
	GetByDependencyIDFunc func(ctx context.Context, dependencyID int64, limit int) ([]*domain.StatusLog, error)
	GetAllFunc func(ctx context.Context, limit int) ([]*domain.StatusLog, error)
}

func NewMockStatusLogRepository() *MockStatusLogRepository {
	return &MockStatusLogRepository{
		Logs: make([]*domain.StatusLog, 0),
	}
}

func (m *MockStatusLogRepository) Create(ctx context.Context, log *domain.StatusLog) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, log)
	}
	log.ID = int64(len(m.Logs) + 1)
	m.Logs = append(m.Logs, log)
	return nil
}

func (m *MockStatusLogRepository) GetBySystemID(ctx context.Context, systemID int64, limit int) ([]*domain.StatusLog, error) {
	if m.GetBySystemIDFunc != nil {
		return m.GetBySystemIDFunc(ctx, systemID, limit)
	}
	var result []*domain.StatusLog
	for _, log := range m.Logs {
		if log.SystemID != nil && *log.SystemID == systemID {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *MockStatusLogRepository) GetByDependencyID(ctx context.Context, dependencyID int64, limit int) ([]*domain.StatusLog, error) {
	if m.GetByDependencyIDFunc != nil {
		return m.GetByDependencyIDFunc(ctx, dependencyID, limit)
	}
	var result []*domain.StatusLog
	for _, log := range m.Logs {
		if log.DependencyID != nil && *log.DependencyID == dependencyID {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *MockStatusLogRepository) GetAll(ctx context.Context, limit int) ([]*domain.StatusLog, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx, limit)
	}
	if limit > 0 && limit < len(m.Logs) {
		return m.Logs[:limit], nil
	}
	return m.Logs, nil
}

func (m *MockStatusLogRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.StatusLog, error) {
	var result []*domain.StatusLog
	for _, log := range m.Logs {
		if log.CreatedAt.After(start) && log.CreatedAt.Before(end) {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *MockStatusLogRepository) GetSystemLogsByTimeRange(ctx context.Context, systemID int64, start, end time.Time) ([]*domain.StatusLog, error) {
	var result []*domain.StatusLog
	for _, log := range m.Logs {
		if log.SystemID != nil && *log.SystemID == systemID && log.CreatedAt.After(start) && log.CreatedAt.Before(end) {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *MockStatusLogRepository) GetDependencyLogsByTimeRange(ctx context.Context, dependencyID int64, start, end time.Time) ([]*domain.StatusLog, error) {
	var result []*domain.StatusLog
	for _, log := range m.Logs {
		if log.DependencyID != nil && *log.DependencyID == dependencyID && log.CreatedAt.After(start) && log.CreatedAt.Before(end) {
			result = append(result, log)
		}
	}
	return result, nil
}

// MockAnalyticsRepository is a mock implementation of domain.AnalyticsRepository
type MockAnalyticsRepository struct {
	GetUptimeBySystemIDFunc     func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error)
	GetUptimeByDependencyIDFunc func(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.Analytics, error)
	GetOverallAnalyticsFunc     func(ctx context.Context, start, end time.Time) (*domain.Analytics, error)
	GetIncidentsBySystemIDFunc  func(ctx context.Context, systemID int64, start, end time.Time) ([]domain.IncidentPeriod, error)
	GetIncidentsByDependencyIDFunc func(ctx context.Context, dependencyID int64, start, end time.Time) ([]domain.IncidentPeriod, error)
}

func NewMockAnalyticsRepository() *MockAnalyticsRepository {
	return &MockAnalyticsRepository{}
}

func (m *MockAnalyticsRepository) GetUptimeBySystemID(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
	if m.GetUptimeBySystemIDFunc != nil {
		return m.GetUptimeBySystemIDFunc(ctx, systemID, start, end)
	}
	return &domain.Analytics{
		UptimePercent:       99.9,
		AvailabilityPercent: 99.95,
		TotalIncidents:      1,
		ResolvedIncidents:   1,
	}, nil
}

func (m *MockAnalyticsRepository) GetUptimeByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.Analytics, error) {
	if m.GetUptimeByDependencyIDFunc != nil {
		return m.GetUptimeByDependencyIDFunc(ctx, dependencyID, start, end)
	}
	return &domain.Analytics{
		UptimePercent:       99.8,
		AvailabilityPercent: 99.9,
	}, nil
}

func (m *MockAnalyticsRepository) GetOverallAnalytics(ctx context.Context, start, end time.Time) (*domain.Analytics, error) {
	if m.GetOverallAnalyticsFunc != nil {
		return m.GetOverallAnalyticsFunc(ctx, start, end)
	}
	return &domain.Analytics{
		UptimePercent:       99.5,
		AvailabilityPercent: 99.7,
	}, nil
}

func (m *MockAnalyticsRepository) GetIncidentsBySystemID(ctx context.Context, systemID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
	if m.GetIncidentsBySystemIDFunc != nil {
		return m.GetIncidentsBySystemIDFunc(ctx, systemID, start, end)
	}
	return []domain.IncidentPeriod{}, nil
}

func (m *MockAnalyticsRepository) GetIncidentsByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
	if m.GetIncidentsByDependencyIDFunc != nil {
		return m.GetIncidentsByDependencyIDFunc(ctx, dependencyID, start, end)
	}
	return []domain.IncidentPeriod{}, nil
}

// MockLatencyRepository is a mock implementation of domain.LatencyRepository
type MockLatencyRepository struct {
	Records      []*domain.LatencyRecord
	RecordFunc   func(ctx context.Context, record *domain.LatencyRecord) error
	GetStatsFunc func(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.LatencyStats, error)
	GetDailyUptimeFunc func(ctx context.Context, dependencyID int64, days int) ([]domain.UptimePoint, error)
	GetAggregatedFunc func(ctx context.Context, dependencyID int64, start, end time.Time, intervalMinutes int) ([]domain.LatencyPoint, error)
}

func NewMockLatencyRepository() *MockLatencyRepository {
	return &MockLatencyRepository{
		Records: make([]*domain.LatencyRecord, 0),
	}
}

func (m *MockLatencyRepository) Record(ctx context.Context, record *domain.LatencyRecord) error {
	if m.RecordFunc != nil {
		return m.RecordFunc(ctx, record)
	}
	record.ID = int64(len(m.Records) + 1)
	m.Records = append(m.Records, record)
	return nil
}

func (m *MockLatencyRepository) GetByDependency(ctx context.Context, dependencyID int64, start, end time.Time, limit int) ([]*domain.LatencyRecord, error) {
	var result []*domain.LatencyRecord
	for _, r := range m.Records {
		if r.DependencyID == dependencyID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockLatencyRepository) GetAggregated(ctx context.Context, dependencyID int64, start, end time.Time, intervalMinutes int) ([]domain.LatencyPoint, error) {
	if m.GetAggregatedFunc != nil {
		return m.GetAggregatedFunc(ctx, dependencyID, start, end, intervalMinutes)
	}
	return []domain.LatencyPoint{}, nil
}

func (m *MockLatencyRepository) GetDailyUptime(ctx context.Context, dependencyID int64, days int) ([]domain.UptimePoint, error) {
	if m.GetDailyUptimeFunc != nil {
		return m.GetDailyUptimeFunc(ctx, dependencyID, days)
	}
	return []domain.UptimePoint{}, nil
}

func (m *MockLatencyRepository) GetStats(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.LatencyStats, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc(ctx, dependencyID, start, end)
	}
	return &domain.LatencyStats{
		DependencyID: dependencyID,
		AvgLatencyMs: 50.5,
		MinLatencyMs: 10,
		MaxLatencyMs: 200,
		P95LatencyMs: 150,
		P99LatencyMs: 180,
		TotalChecks:  1000,
		FailedChecks: 5,
		UptimePercent: 99.5,
	}, nil
}

func (m *MockLatencyRepository) Cleanup(ctx context.Context, olderThan time.Time) error {
	return nil
}

// MockIncidentRepository is a mock implementation of domain.IncidentRepository
type MockIncidentRepository struct {
	Incidents    map[int64]*domain.Incident
	Updates      []*domain.IncidentUpdate
	CreateFunc   func(ctx context.Context, i *domain.Incident) error
	GetByIDFunc  func(ctx context.Context, id int64) (*domain.Incident, error)
}

func NewMockIncidentRepository() *MockIncidentRepository {
	return &MockIncidentRepository{
		Incidents: make(map[int64]*domain.Incident),
		Updates:   make([]*domain.IncidentUpdate, 0),
	}
}

func (m *MockIncidentRepository) Create(ctx context.Context, i *domain.Incident) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, i)
	}
	i.ID = int64(len(m.Incidents) + 1)
	m.Incidents[i.ID] = i
	return nil
}

func (m *MockIncidentRepository) GetByID(ctx context.Context, id int64) (*domain.Incident, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return m.Incidents[id], nil
}

func (m *MockIncidentRepository) GetAll(ctx context.Context, limit int) ([]*domain.Incident, error) {
	var result []*domain.Incident
	for _, i := range m.Incidents {
		result = append(result, i)
	}
	return result, nil
}

func (m *MockIncidentRepository) GetActive(ctx context.Context) ([]*domain.Incident, error) {
	var result []*domain.Incident
	for _, i := range m.Incidents {
		if i.IsActive() {
			result = append(result, i)
		}
	}
	return result, nil
}

func (m *MockIncidentRepository) GetRecent(ctx context.Context, limit int) ([]*domain.Incident, error) {
	var result []*domain.Incident
	for _, i := range m.Incidents {
		result = append(result, i)
	}
	return result, nil
}

func (m *MockIncidentRepository) Update(ctx context.Context, i *domain.Incident) error {
	m.Incidents[i.ID] = i
	return nil
}

func (m *MockIncidentRepository) Delete(ctx context.Context, id int64) error {
	delete(m.Incidents, id)
	return nil
}

func (m *MockIncidentRepository) CreateUpdate(ctx context.Context, u *domain.IncidentUpdate) error {
	u.ID = int64(len(m.Updates) + 1)
	m.Updates = append(m.Updates, u)
	return nil
}

func (m *MockIncidentRepository) GetUpdates(ctx context.Context, incidentID int64) ([]*domain.IncidentUpdate, error) {
	var result []*domain.IncidentUpdate
	for _, u := range m.Updates {
		if u.IncidentID == incidentID {
			result = append(result, u)
		}
	}
	return result, nil
}

// MockIncidentUpdateRepository is a mock implementation
type MockIncidentUpdateRepository struct {
	Updates []*domain.IncidentUpdate
}

func NewMockIncidentUpdateRepository() *MockIncidentUpdateRepository {
	return &MockIncidentUpdateRepository{
		Updates: make([]*domain.IncidentUpdate, 0),
	}
}

func (m *MockIncidentUpdateRepository) Create(ctx context.Context, u *domain.IncidentUpdate) error {
	u.ID = int64(len(m.Updates) + 1)
	m.Updates = append(m.Updates, u)
	return nil
}

func (m *MockIncidentUpdateRepository) GetByIncidentID(ctx context.Context, incidentID int64) ([]*domain.IncidentUpdate, error) {
	var result []*domain.IncidentUpdate
	for _, u := range m.Updates {
		if u.IncidentID == incidentID {
			result = append(result, u)
		}
	}
	return result, nil
}

// MockMaintenanceRepository is a mock implementation
type MockMaintenanceRepository struct {
	Maintenances map[int64]*domain.Maintenance
}

func NewMockMaintenanceRepository() *MockMaintenanceRepository {
	return &MockMaintenanceRepository{
		Maintenances: make(map[int64]*domain.Maintenance),
	}
}

func (m *MockMaintenanceRepository) Create(ctx context.Context, maint *domain.Maintenance) error {
	maint.ID = int64(len(m.Maintenances) + 1)
	m.Maintenances[maint.ID] = maint
	return nil
}

func (m *MockMaintenanceRepository) GetByID(ctx context.Context, id int64) (*domain.Maintenance, error) {
	return m.Maintenances[id], nil
}

func (m *MockMaintenanceRepository) GetAll(ctx context.Context) ([]*domain.Maintenance, error) {
	var result []*domain.Maintenance
	for _, maint := range m.Maintenances {
		result = append(result, maint)
	}
	return result, nil
}

func (m *MockMaintenanceRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.Maintenance, error) {
	var result []*domain.Maintenance
	for _, maint := range m.Maintenances {
		if maint.StartTime.Before(end) && maint.EndTime.After(start) {
			result = append(result, maint)
		}
	}
	return result, nil
}

func (m *MockMaintenanceRepository) GetActive(ctx context.Context) ([]*domain.Maintenance, error) {
	var result []*domain.Maintenance
	for _, maint := range m.Maintenances {
		if maint.IsActive() {
			result = append(result, maint)
		}
	}
	return result, nil
}

func (m *MockMaintenanceRepository) GetUpcoming(ctx context.Context) ([]*domain.Maintenance, error) {
	var result []*domain.Maintenance
	for _, maint := range m.Maintenances {
		if maint.IsUpcoming() {
			result = append(result, maint)
		}
	}
	return result, nil
}

func (m *MockMaintenanceRepository) Update(ctx context.Context, maint *domain.Maintenance) error {
	m.Maintenances[maint.ID] = maint
	return nil
}

func (m *MockMaintenanceRepository) Delete(ctx context.Context, id int64) error {
	delete(m.Maintenances, id)
	return nil
}

func (m *MockMaintenanceRepository) GetBySystemID(ctx context.Context, systemID int64) ([]*domain.Maintenance, error) {
	var result []*domain.Maintenance
	for _, maint := range m.Maintenances {
		if maint.AffectsSystem(systemID) {
			result = append(result, maint)
		}
	}
	return result, nil
}

// MockWebhookRepository is a mock implementation
type MockWebhookRepository struct {
	Webhooks map[int64]*domain.Webhook
}

func NewMockWebhookRepository() *MockWebhookRepository {
	return &MockWebhookRepository{
		Webhooks: make(map[int64]*domain.Webhook),
	}
}

func (m *MockWebhookRepository) Create(ctx context.Context, w *domain.Webhook) error {
	w.ID = int64(len(m.Webhooks) + 1)
	m.Webhooks[w.ID] = w
	return nil
}

func (m *MockWebhookRepository) GetByID(ctx context.Context, id int64) (*domain.Webhook, error) {
	return m.Webhooks[id], nil
}

func (m *MockWebhookRepository) GetAll(ctx context.Context) ([]*domain.Webhook, error) {
	var result []*domain.Webhook
	for _, w := range m.Webhooks {
		result = append(result, w)
	}
	return result, nil
}

func (m *MockWebhookRepository) GetEnabled(ctx context.Context) ([]*domain.Webhook, error) {
	var result []*domain.Webhook
	for _, w := range m.Webhooks {
		if w.Enabled {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *MockWebhookRepository) Update(ctx context.Context, w *domain.Webhook) error {
	m.Webhooks[w.ID] = w
	return nil
}

func (m *MockWebhookRepository) Delete(ctx context.Context, id int64) error {
	delete(m.Webhooks, id)
	return nil
}

// MockHealthChecker is a mock implementation of domain.HealthChecker
type MockHealthChecker struct {
	CheckFunc           func(ctx context.Context, url string) (healthy bool, latencyMs int64, err error)
	CheckWithConfigFunc func(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult
}

func NewMockHealthChecker() *MockHealthChecker {
	return &MockHealthChecker{}
}

func (m *MockHealthChecker) Check(ctx context.Context, url string) (healthy bool, latencyMs int64, err error) {
	if m.CheckFunc != nil {
		return m.CheckFunc(ctx, url)
	}
	return true, 50, nil
}

func (m *MockHealthChecker) CheckWithConfig(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
	if m.CheckWithConfigFunc != nil {
		return m.CheckWithConfigFunc(ctx, config)
	}
	return domain.HealthCheckResult{
		Healthy:    true,
		LatencyMs:  50,
		StatusCode: 200,
	}
}
