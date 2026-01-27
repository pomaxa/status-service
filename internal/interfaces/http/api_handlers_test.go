package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"status-incident/internal/application"
	"status-incident/internal/domain"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// ============= Test Infrastructure =============

// MockSystemRepository for testing
type MockSystemRepository struct {
	Systems map[int64]*domain.System
}

func NewMockSystemRepository() *MockSystemRepository {
	return &MockSystemRepository{Systems: make(map[int64]*domain.System)}
}

func (m *MockSystemRepository) Create(ctx context.Context, s *domain.System) error {
	s.ID = int64(len(m.Systems) + 1)
	m.Systems[s.ID] = s
	return nil
}

func (m *MockSystemRepository) GetByID(ctx context.Context, id int64) (*domain.System, error) {
	return m.Systems[id], nil
}

func (m *MockSystemRepository) GetAll(ctx context.Context) ([]*domain.System, error) {
	var result []*domain.System
	for _, s := range m.Systems {
		result = append(result, s)
	}
	return result, nil
}

func (m *MockSystemRepository) Update(ctx context.Context, s *domain.System) error {
	m.Systems[s.ID] = s
	return nil
}

func (m *MockSystemRepository) Delete(ctx context.Context, id int64) error {
	delete(m.Systems, id)
	return nil
}

// MockStatusLogRepository for testing
type MockStatusLogRepository struct {
	Logs []*domain.StatusLog
}

func NewMockStatusLogRepository() *MockStatusLogRepository {
	return &MockStatusLogRepository{Logs: make([]*domain.StatusLog, 0)}
}

func (m *MockStatusLogRepository) Create(ctx context.Context, log *domain.StatusLog) error {
	log.ID = int64(len(m.Logs) + 1)
	m.Logs = append(m.Logs, log)
	return nil
}

func (m *MockStatusLogRepository) GetBySystemID(ctx context.Context, systemID int64, limit int) ([]*domain.StatusLog, error) {
	var result []*domain.StatusLog
	for _, log := range m.Logs {
		if log.SystemID != nil && *log.SystemID == systemID {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *MockStatusLogRepository) GetByDependencyID(ctx context.Context, dependencyID int64, limit int) ([]*domain.StatusLog, error) {
	return nil, nil
}

func (m *MockStatusLogRepository) GetAll(ctx context.Context, limit int) ([]*domain.StatusLog, error) {
	return m.Logs, nil
}

func (m *MockStatusLogRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.StatusLog, error) {
	return m.Logs, nil
}

func (m *MockStatusLogRepository) GetSystemLogsByTimeRange(ctx context.Context, systemID int64, start, end time.Time) ([]*domain.StatusLog, error) {
	return nil, nil
}

func (m *MockStatusLogRepository) GetDependencyLogsByTimeRange(ctx context.Context, dependencyID int64, start, end time.Time) ([]*domain.StatusLog, error) {
	return nil, nil
}

// MockDependencyRepository for testing
type MockDependencyRepository struct {
	Dependencies map[int64]*domain.Dependency
}

func NewMockDependencyRepository() *MockDependencyRepository {
	return &MockDependencyRepository{Dependencies: make(map[int64]*domain.Dependency)}
}

func (m *MockDependencyRepository) Create(ctx context.Context, d *domain.Dependency) error {
	d.ID = int64(len(m.Dependencies) + 1)
	m.Dependencies[d.ID] = d
	return nil
}

func (m *MockDependencyRepository) GetByID(ctx context.Context, id int64) (*domain.Dependency, error) {
	return m.Dependencies[id], nil
}

func (m *MockDependencyRepository) GetBySystemID(ctx context.Context, systemID int64) ([]*domain.Dependency, error) {
	var result []*domain.Dependency
	for _, d := range m.Dependencies {
		if d.SystemID == systemID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *MockDependencyRepository) GetAll(ctx context.Context) ([]*domain.Dependency, error) {
	var result []*domain.Dependency
	for _, d := range m.Dependencies {
		result = append(result, d)
	}
	return result, nil
}

func (m *MockDependencyRepository) Update(ctx context.Context, d *domain.Dependency) error {
	m.Dependencies[d.ID] = d
	return nil
}

func (m *MockDependencyRepository) Delete(ctx context.Context, id int64) error {
	delete(m.Dependencies, id)
	return nil
}

func (m *MockDependencyRepository) GetAllWithHeartbeat(ctx context.Context) ([]*domain.Dependency, error) {
	var result []*domain.Dependency
	for _, d := range m.Dependencies {
		if d.HasHeartbeat() {
			result = append(result, d)
		}
	}
	return result, nil
}

// MockAnalyticsRepository for testing
type MockAnalyticsRepository struct{}

func NewMockAnalyticsRepository() *MockAnalyticsRepository {
	return &MockAnalyticsRepository{}
}

func (m *MockAnalyticsRepository) GetUptimeBySystemID(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
	return &domain.Analytics{
		UptimePercent:       99.9,
		AvailabilityPercent: 99.95,
		TotalIncidents:      1,
		ResolvedIncidents:   1,
	}, nil
}

func (m *MockAnalyticsRepository) GetUptimeByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) (*domain.Analytics, error) {
	return &domain.Analytics{
		UptimePercent:       99.8,
		AvailabilityPercent: 99.9,
	}, nil
}

func (m *MockAnalyticsRepository) GetOverallAnalytics(ctx context.Context, start, end time.Time) (*domain.Analytics, error) {
	return &domain.Analytics{
		UptimePercent:       99.5,
		AvailabilityPercent: 99.7,
	}, nil
}

func (m *MockAnalyticsRepository) GetIncidentsBySystemID(ctx context.Context, systemID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
	return []domain.IncidentPeriod{}, nil
}

func (m *MockAnalyticsRepository) GetIncidentsByDependencyID(ctx context.Context, dependencyID int64, start, end time.Time) ([]domain.IncidentPeriod, error) {
	return []domain.IncidentPeriod{}, nil
}

// Helper to create test server
func setupTestServer() (*Server, *MockSystemRepository, *MockDependencyRepository) {
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()
	logRepo := NewMockStatusLogRepository()
	analyticsRepo := NewMockAnalyticsRepository()

	systemService := application.NewSystemService(systemRepo, logRepo)
	depService := application.NewDependencyService(depRepo, logRepo)
	analyticsService := application.NewAnalyticsService(analyticsRepo, logRepo)

	server := &Server{
		router:           chi.NewRouter(),
		systemService:    systemService,
		depService:       depService,
		analyticsService: analyticsService,
	}

	return server, systemRepo, depRepo
}

// ============= System API Tests =============

func TestAPIGetSystems(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	// Create some systems
	system1, _ := domain.NewSystem("API Gateway", "Main API", "https://api.example.com", "team-a")
	system2, _ := domain.NewSystem("Database", "Primary DB", "", "team-b")
	systemRepo.Create(context.Background(), system1)
	systemRepo.Create(context.Background(), system2)

	req := httptest.NewRequest("GET", "/api/systems", nil)
	w := httptest.NewRecorder()

	server.apiGetSystems(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var systems []*domain.System
	if err := json.Unmarshal(w.Body.Bytes(), &systems); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(systems) != 2 {
		t.Errorf("expected 2 systems, got %d", len(systems))
	}
}

func TestAPICreateSystem(t *testing.T) {
	server, _, _ := setupTestServer()

	reqBody := createSystemRequest{
		Name:        "New System",
		Description: "A new test system",
		URL:         "https://test.example.com",
		Owner:       "team-c",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/systems", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.apiCreateSystem(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var system domain.System
	if err := json.Unmarshal(w.Body.Bytes(), &system); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if system.Name != "New System" {
		t.Errorf("expected name 'New System', got '%s'", system.Name)
	}
}

func TestAPICreateSystem_InvalidBody(t *testing.T) {
	server, _, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/systems", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.apiCreateSystem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPICreateSystem_EmptyName(t *testing.T) {
	server, _, _ := setupTestServer()

	reqBody := createSystemRequest{
		Name:        "", // Empty name should fail
		Description: "A test system",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/systems", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.apiCreateSystem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPIGetSystem(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	system, _ := domain.NewSystem("Test System", "Description", "", "owner")
	systemRepo.Create(context.Background(), system)

	// Create request with chi URL params
	req := httptest.NewRequest("GET", "/api/systems/1", nil)
	w := httptest.NewRecorder()

	// Add chi URL parameter context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetSystem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var retrieved domain.System
	if err := json.Unmarshal(w.Body.Bytes(), &retrieved); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if retrieved.Name != "Test System" {
		t.Errorf("expected name 'Test System', got '%s'", retrieved.Name)
	}
}

func TestAPIGetSystem_NotFound(t *testing.T) {
	server, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/systems/999", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetSystem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestAPIGetSystem_InvalidID(t *testing.T) {
	server, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/systems/invalid", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetSystem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPIUpdateSystem(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	system, _ := domain.NewSystem("Original Name", "Original desc", "", "owner")
	systemRepo.Create(context.Background(), system)

	reqBody := createSystemRequest{
		Name:        "Updated Name",
		Description: "Updated description",
		URL:         "https://updated.example.com",
		Owner:       "new-owner",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/systems/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiUpdateSystem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var updated domain.System
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", updated.Name)
	}
}

func TestAPIDeleteSystem(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	system, _ := domain.NewSystem("To Delete", "desc", "", "owner")
	systemRepo.Create(context.Background(), system)

	req := httptest.NewRequest("DELETE", "/api/systems/1", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiDeleteSystem(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify deletion
	if _, exists := systemRepo.Systems[1]; exists {
		t.Error("expected system to be deleted")
	}
}

func TestAPIUpdateSystemStatus(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	system, _ := domain.NewSystem("Test System", "desc", "", "owner")
	systemRepo.Create(context.Background(), system)

	reqBody := updateStatusRequest{
		Status:  "yellow",
		Message: "Degraded performance",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/systems/1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiUpdateSystemStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var updated domain.System
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if updated.Status != domain.StatusYellow {
		t.Errorf("expected status yellow, got %s", updated.Status)
	}
}

func TestAPIUpdateSystemStatus_InvalidStatus(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	system, _ := domain.NewSystem("Test System", "desc", "", "owner")
	systemRepo.Create(context.Background(), system)

	reqBody := updateStatusRequest{
		Status:  "invalid",
		Message: "test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/systems/1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiUpdateSystemStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// ============= Dependency API Tests =============

func TestAPIGetDependencies(t *testing.T) {
	server, systemRepo, depRepo := setupTestServer()

	// Create system
	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	// Create dependencies
	dep1 := &domain.Dependency{SystemID: system.ID, Name: "Database", Status: domain.StatusGreen}
	dep2 := &domain.Dependency{SystemID: system.ID, Name: "Cache", Status: domain.StatusGreen}
	depRepo.Create(context.Background(), dep1)
	depRepo.Create(context.Background(), dep2)

	req := httptest.NewRequest("GET", "/api/systems/1/dependencies", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("systemId", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetDependencies(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var deps []*domain.Dependency
	if err := json.Unmarshal(w.Body.Bytes(), &deps); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestAPICreateDependency(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	reqBody := createDependencyRequest{
		Name:        "New Dependency",
		Description: "A test dependency",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/systems/1/dependencies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("systemId", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiCreateDependency(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var dep domain.Dependency
	if err := json.Unmarshal(w.Body.Bytes(), &dep); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if dep.Name != "New Dependency" {
		t.Errorf("expected name 'New Dependency', got '%s'", dep.Name)
	}
}

func TestAPIGetDependency(t *testing.T) {
	server, systemRepo, depRepo := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	dep := &domain.Dependency{SystemID: system.ID, Name: "Test Dep", Status: domain.StatusGreen}
	depRepo.Create(context.Background(), dep)

	req := httptest.NewRequest("GET", "/api/dependencies/1", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetDependency(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAPIGetDependency_NotFound(t *testing.T) {
	server, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/dependencies/999", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetDependency(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestAPIDeleteDependency(t *testing.T) {
	server, systemRepo, depRepo := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	dep := &domain.Dependency{SystemID: system.ID, Name: "To Delete", Status: domain.StatusGreen}
	depRepo.Create(context.Background(), dep)

	req := httptest.NewRequest("DELETE", "/api/dependencies/1", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiDeleteDependency(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

// ============= Analytics API Tests =============

func TestAPIGetOverallAnalytics(t *testing.T) {
	server, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/analytics?period=24h", nil)
	w := httptest.NewRecorder()

	server.apiGetOverallAnalytics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var analytics domain.Analytics
	if err := json.Unmarshal(w.Body.Bytes(), &analytics); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
}

func TestAPIGetSystemAnalytics(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	req := httptest.NewRequest("GET", "/api/systems/1/analytics?period=24h", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetSystemAnalytics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// ============= Helper Function Tests =============

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "< 1m"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h 30m"},
		{2 * time.Hour, "2h"},
		{25 * time.Hour, "1d 1h"},
		{48 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("expected key=value, got key=%s", result["key"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var result errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Error != "test error message" {
		t.Errorf("expected error 'test error message', got '%s'", result.Error)
	}
}

// ============= Additional Tests =============

func TestAPIGetAllLogs(t *testing.T) {
	server, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/logs?limit=50", nil)
	w := httptest.NewRecorder()

	server.apiGetAllLogs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAPIGetSystemLogs(t *testing.T) {
	server, systemRepo, _ := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	req := httptest.NewRequest("GET", "/api/systems/1/logs?limit=50", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetSystemLogs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAPIGetDependencyLogs(t *testing.T) {
	server, systemRepo, depRepo := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	dep := &domain.Dependency{SystemID: system.ID, Name: "Test Dep", Status: domain.StatusGreen}
	depRepo.Create(context.Background(), dep)

	req := httptest.NewRequest("GET", "/api/dependencies/1/logs?limit=50", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetDependencyLogs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAPIUpdateDependency(t *testing.T) {
	server, systemRepo, depRepo := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	dep := &domain.Dependency{SystemID: system.ID, Name: "Original Name", Status: domain.StatusGreen}
	depRepo.Create(context.Background(), dep)

	reqBody := createDependencyRequest{
		Name:        "Updated Name",
		Description: "Updated description",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/dependencies/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiUpdateDependency(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAPIUpdateDependencyStatus(t *testing.T) {
	server, systemRepo, depRepo := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	dep := &domain.Dependency{SystemID: system.ID, Name: "Test Dep", Status: domain.StatusGreen}
	depRepo.Create(context.Background(), dep)

	reqBody := updateStatusRequest{
		Status:  "yellow",
		Message: "Degraded",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/dependencies/1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiUpdateDependencyStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAPIGetDependencyAnalytics(t *testing.T) {
	server, systemRepo, depRepo := setupTestServer()

	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(context.Background(), system)

	dep := &domain.Dependency{SystemID: system.ID, Name: "Test Dep", Status: domain.StatusGreen}
	depRepo.Create(context.Background(), dep)

	req := httptest.NewRequest("GET", "/api/dependencies/1/analytics?period=24h", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	server.apiGetDependencyAnalytics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// ============= Webhook Handler Tests =============

// MockWebhookRepository for testing
type MockWebhookRepository struct {
	Webhooks map[int64]*domain.Webhook
}

func NewMockWebhookRepository() *MockWebhookRepository {
	return &MockWebhookRepository{Webhooks: make(map[int64]*domain.Webhook)}
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

func TestWebhookHandlers_ListWebhooks(t *testing.T) {
	webhookRepo := NewMockWebhookRepository()
	handlers := NewWebhookHandlers(webhookRepo, nil)

	// Create webhooks
	webhook1, _ := domain.NewWebhook("Webhook 1", "https://example.com/1", domain.WebhookTypeSlack)
	webhook2, _ := domain.NewWebhook("Webhook 2", "https://example.com/2", domain.WebhookTypeDiscord)
	webhookRepo.Create(context.Background(), webhook1)
	webhookRepo.Create(context.Background(), webhook2)

	req := httptest.NewRequest("GET", "/api/webhooks", nil)
	w := httptest.NewRecorder()

	handlers.ListWebhooks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var webhooks []webhookResponse
	if err := json.Unmarshal(w.Body.Bytes(), &webhooks); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(webhooks) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(webhooks))
	}
}

func TestWebhookHandlers_CreateWebhook(t *testing.T) {
	webhookRepo := NewMockWebhookRepository()
	handlers := NewWebhookHandlers(webhookRepo, nil)

	reqBody := webhookRequest{
		Name:   "New Webhook",
		URL:    "https://example.com/webhook",
		Type:   "slack",
		Events: []string{"status_change"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.CreateWebhook(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response webhookResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Name != "New Webhook" {
		t.Errorf("expected name 'New Webhook', got '%s'", response.Name)
	}
}

func TestWebhookHandlers_CreateWebhook_InvalidBody(t *testing.T) {
	webhookRepo := NewMockWebhookRepository()
	handlers := NewWebhookHandlers(webhookRepo, nil)

	req := httptest.NewRequest("POST", "/api/webhooks", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	handlers.CreateWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestWebhookHandlers_GetWebhook(t *testing.T) {
	webhookRepo := NewMockWebhookRepository()
	handlers := NewWebhookHandlers(webhookRepo, nil)

	webhook, _ := domain.NewWebhook("Test Webhook", "https://example.com/webhook", domain.WebhookTypeSlack)
	webhookRepo.Create(context.Background(), webhook)

	req := httptest.NewRequest("GET", "/api/webhooks/1", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handlers.GetWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestWebhookHandlers_GetWebhook_NotFound(t *testing.T) {
	webhookRepo := NewMockWebhookRepository()
	handlers := NewWebhookHandlers(webhookRepo, nil)

	req := httptest.NewRequest("GET", "/api/webhooks/999", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handlers.GetWebhook(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestWebhookHandlers_DeleteWebhook(t *testing.T) {
	webhookRepo := NewMockWebhookRepository()
	handlers := NewWebhookHandlers(webhookRepo, nil)

	webhook, _ := domain.NewWebhook("To Delete", "https://example.com/webhook", domain.WebhookTypeSlack)
	webhookRepo.Create(context.Background(), webhook)

	req := httptest.NewRequest("DELETE", "/api/webhooks/1", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handlers.DeleteWebhook(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestWebhookHandlers_UpdateWebhook(t *testing.T) {
	webhookRepo := NewMockWebhookRepository()
	handlers := NewWebhookHandlers(webhookRepo, nil)

	webhook, _ := domain.NewWebhook("Original", "https://example.com/webhook", domain.WebhookTypeSlack)
	webhookRepo.Create(context.Background(), webhook)

	enabled := true
	reqBody := webhookRequest{
		Name:    "Updated",
		URL:     "https://example.com/updated",
		Type:    "discord",
		Events:  []string{"status_change", "sla_breach"},
		Enabled: &enabled,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/webhooks/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handlers.UpdateWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response webhookResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Name != "Updated" {
		t.Errorf("expected name 'Updated', got '%s'", response.Name)
	}
}

func TestToWebhookResponse(t *testing.T) {
	webhook, _ := domain.NewWebhook("Test", "https://example.com", domain.WebhookTypeSlack)
	webhook.ID = 1
	webhook.SetEvents([]domain.WebhookEvent{domain.EventStatusChange})
	webhook.SetSystemIDs([]int64{1, 2, 3})

	response := toWebhookResponse(webhook)

	if response.ID != 1 {
		t.Errorf("expected ID 1, got %d", response.ID)
	}
	if response.Name != "Test" {
		t.Errorf("expected name 'Test', got '%s'", response.Name)
	}
	if response.Type != "slack" {
		t.Errorf("expected type 'slack', got '%s'", response.Type)
	}
	if len(response.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(response.Events))
	}
	if len(response.SystemIDs) != 3 {
		t.Errorf("expected 3 system IDs, got %d", len(response.SystemIDs))
	}
}

func TestJsonResponse(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	jsonResponse(w, data)

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("expected key=value, got key=%s", result["key"])
	}
}

func TestJsonError(t *testing.T) {
	w := httptest.NewRecorder()

	jsonError(w, "test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result["error"] != "test error" {
		t.Errorf("expected error 'test error', got '%s'", result["error"])
	}
}

// ============= Router Tests =============

func TestJsonContentType(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})

	middleware := jsonContentType(nextHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
}

func TestToMaintenanceResponse(t *testing.T) {
	now := time.Now()
	m := &domain.Maintenance{
		ID:          1,
		Title:       "Test Maintenance",
		Description: "Test description",
		StartTime:   now,
		EndTime:     now.Add(2 * time.Hour),
		SystemIDs:   []int64{1, 2},
		Status:      domain.MaintenanceScheduled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	response := toMaintenanceResponse(m)

	if response.ID != 1 {
		t.Errorf("expected ID 1, got %d", response.ID)
	}
	if response.Title != "Test Maintenance" {
		t.Errorf("expected title 'Test Maintenance', got '%s'", response.Title)
	}
	if response.Status != "scheduled" {
		t.Errorf("expected status 'scheduled', got '%s'", response.Status)
	}
	if len(response.SystemIDs) != 2 {
		t.Errorf("expected 2 system IDs, got %d", len(response.SystemIDs))
	}
}

func TestToIncidentResponse(t *testing.T) {
	now := time.Now()
	resolvedAt := now.Add(time.Hour)
	ackedAt := now.Add(30 * time.Minute)

	inc := &domain.Incident{
		ID:             1,
		Title:          "Test Incident",
		Status:         domain.IncidentResolved,
		Severity:       domain.SeverityMajor,
		SystemIDs:      []int64{1, 2, 3},
		Message:        "Test message",
		Postmortem:     "Test postmortem",
		CreatedAt:      now,
		UpdatedAt:      now,
		ResolvedAt:     &resolvedAt,
		AcknowledgedAt: &ackedAt,
		AcknowledgedBy: "admin",
	}

	response := toIncidentResponse(inc)

	if response.ID != 1 {
		t.Errorf("expected ID 1, got %d", response.ID)
	}
	if response.Title != "Test Incident" {
		t.Errorf("expected title 'Test Incident', got '%s'", response.Title)
	}
	if response.Status != "resolved" {
		t.Errorf("expected status 'resolved', got '%s'", response.Status)
	}
	if response.Severity != "major" {
		t.Errorf("expected severity 'major', got '%s'", response.Severity)
	}
	if response.AcknowledgedBy != "admin" {
		t.Errorf("expected acknowledged_by 'admin', got '%s'", response.AcknowledgedBy)
	}
	if response.ResolvedAt == nil {
		t.Error("expected resolved_at to be set")
	}
	if response.AcknowledgedAt == nil {
		t.Error("expected acknowledged_at to be set")
	}
}
