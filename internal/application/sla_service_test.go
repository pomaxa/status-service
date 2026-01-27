package application

import (
	"context"
	"math"
	"status-incident/internal/domain"
	"testing"
	"time"
)

const slaEpsilon = 0.001

func TestSLAService_parsePeriod(t *testing.T) {
	s := &SLAService{}

	tests := []struct {
		name           string
		period         string
		expectedDays   int
	}{
		{"daily", "daily", 1},
		{"1d", "1d", 1},
		{"weekly", "weekly", 7},
		{"7d", "7d", 7},
		{"monthly", "monthly", 30},
		{"30d", "30d", 30},
		{"quarterly", "quarterly", 90},
		{"90d", "90d", 90},
		{"yearly", "yearly", 365},
		{"365d", "365d", 365},
		{"default", "unknown", 30},
		{"empty", "", 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := s.parsePeriod(tt.period)

			// Check that end is approximately now
			if time.Since(end) > time.Second {
				t.Errorf("end time should be approximately now, got %v", end)
			}

			// Check the duration
			duration := end.Sub(start)
			expectedDuration := time.Duration(tt.expectedDays) * 24 * time.Hour

			// Allow small tolerance
			if duration < expectedDuration-time.Minute || duration > expectedDuration+time.Minute {
				t.Errorf("expected duration ~%v, got %v", expectedDuration, duration)
			}
		})
	}
}

func TestSLAService_calculateMTTR(t *testing.T) {
	s := &SLAService{}
	now := time.Now()

	tests := []struct {
		name      string
		incidents []domain.IncidentPeriod
		expected  time.Duration
	}{
		{
			name:      "empty incidents",
			incidents: []domain.IncidentPeriod{},
			expected:  0,
		},
		{
			name: "single resolved incident",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)),
				},
			},
			expected: 1 * time.Hour,
		},
		{
			name: "multiple resolved incidents",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-3 * time.Hour)),
				},
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)),
				},
			},
			expected: 1 * time.Hour, // (1h + 1h) / 2
		},
		{
			name: "mixed resolved and unresolved",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-2 * time.Hour)), // 2h
				},
				{
					StartedAt: now.Add(-1 * time.Hour),
					EndedAt:   nil, // ongoing - not counted
				},
			},
			expected: 2 * time.Hour, // only the resolved one
		},
		{
			name: "all unresolved",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-1 * time.Hour),
					EndedAt:   nil,
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.calculateMTTR(tt.incidents)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSLAService_findLongestOutage(t *testing.T) {
	s := &SLAService{}
	now := time.Now()

	tests := []struct {
		name      string
		incidents []domain.IncidentPeriod
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "empty incidents",
			incidents:   []domain.IncidentPeriod{},
			minExpected: 0,
			maxExpected: 0,
		},
		{
			name: "single resolved incident",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)),
				},
			},
			minExpected: 1 * time.Hour,
			maxExpected: 1 * time.Hour,
		},
		{
			name: "multiple incidents different durations",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-3 * time.Hour)), // 1h
				},
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-30 * time.Minute)), // 1.5h
				},
			},
			minExpected: 90 * time.Minute,
			maxExpected: 90 * time.Minute,
		},
		{
			name: "ongoing incident is longest",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-30 * time.Minute),
					EndedAt:   timePtr(now.Add(-20 * time.Minute)), // 10m
				},
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   nil, // ongoing ~2h
				},
			},
			minExpected: 2*time.Hour - time.Minute, // approximately 2 hours
			maxExpected: 2*time.Hour + time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.findLongestOutage(tt.incidents)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("expected between %v and %v, got %v", tt.minExpected, tt.maxExpected, result)
			}
		})
	}
}

func TestSLAService_calculateTotalDowntime(t *testing.T) {
	s := &SLAService{}
	now := time.Now()

	tests := []struct {
		name        string
		incidents   []domain.IncidentPeriod
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "empty incidents",
			incidents:   []domain.IncidentPeriod{},
			minExpected: 0,
			maxExpected: 0,
		},
		{
			name: "single resolved incident",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)),
				},
			},
			minExpected: 1 * time.Hour,
			maxExpected: 1 * time.Hour,
		},
		{
			name: "multiple resolved incidents",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-3 * time.Hour)), // 1h
				},
				{
					StartedAt: now.Add(-2 * time.Hour),
					EndedAt:   timePtr(now.Add(-1 * time.Hour)), // 1h
				},
			},
			minExpected: 2 * time.Hour,
			maxExpected: 2 * time.Hour,
		},
		{
			name: "mixed resolved and ongoing",
			incidents: []domain.IncidentPeriod{
				{
					StartedAt: now.Add(-4 * time.Hour),
					EndedAt:   timePtr(now.Add(-3 * time.Hour)), // 1h
				},
				{
					StartedAt: now.Add(-30 * time.Minute),
					EndedAt:   nil, // ~30m ongoing
				},
			},
			minExpected: 90*time.Minute - time.Minute,
			maxExpected: 90*time.Minute + time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.calculateTotalDowntime(tt.incidents)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("expected between %v and %v, got %v", tt.minExpected, tt.maxExpected, result)
			}
		})
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}

// ============= SLA Service Integration Tests with Mocks =============

func TestSLAService_GenerateReport(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	reportRepo := NewMockSLAReportRepository()
	breachRepo := NewMockSLABreachRepository()
	latencyRepo := NewMockLatencyRepository()

	service := NewSLAService(
		systemRepo,
		depRepo,
		analyticsRepo,
		reportRepo,
		breachRepo,
		latencyRepo,
		nil,
	)

	// Create systems
	system1, _ := domain.NewSystem("API Gateway", "", "", "ops@example.com")
	systemRepo.Create(ctx, system1)

	system2, _ := domain.NewSystem("Database", "", "", "dba@example.com")
	systemRepo.Create(ctx, system2)

	// Generate report
	report, err := service.GenerateReport(ctx, "Monthly SLA Report", "monthly", "admin")
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.Title != "Monthly SLA Report" {
		t.Errorf("Title = %s, want Monthly SLA Report", report.Title)
	}

	if report.Period != "monthly" {
		t.Errorf("Period = %s, want monthly", report.Period)
	}

	if report.GeneratedBy != "admin" {
		t.Errorf("GeneratedBy = %s, want admin", report.GeneratedBy)
	}
}

func TestSLAService_GenerateCustomReport(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	reportRepo := NewMockSLAReportRepository()
	breachRepo := NewMockSLABreachRepository()
	latencyRepo := NewMockLatencyRepository()

	service := NewSLAService(
		systemRepo,
		depRepo,
		analyticsRepo,
		reportRepo,
		breachRepo,
		latencyRepo,
		nil,
	)

	// Create system
	system, _ := domain.NewSystem("API Gateway", "", "", "")
	systemRepo.Create(ctx, system)

	// Set analytics to return good metrics
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return &domain.Analytics{
			UptimePercent:       99.9,
			AvailabilityPercent: 99.95,
			TotalIncidents:      2,
			ResolvedIncidents:   2,
		}, nil
	}

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	report, err := service.GenerateCustomReport(ctx, "Custom Report", "custom", start, end, "admin")
	if err != nil {
		t.Fatalf("GenerateCustomReport() error = %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if len(report.SystemReports) == 0 {
		t.Error("expected at least one system report")
	}

	if len(report.SystemReports) > 0 {
		sysReport := report.SystemReports[0]
		if sysReport.UptimePercent != 99.9 {
			t.Errorf("UptimePercent = %f, want 99.9", sysReport.UptimePercent)
		}
	}
}

func TestSLAService_GetReport(t *testing.T) {
	ctx := context.Background()

	reportRepo := NewMockSLAReportRepository()

	service := NewSLAService(nil, nil, nil, reportRepo, nil, nil, nil)

	// Create a report
	report := domain.NewSLAReport("Test Report", "monthly", time.Now().AddDate(0, -1, 0), time.Now(), "admin")
	reportRepo.Create(ctx, report)

	// Get the report
	retrieved, err := service.GetReport(ctx, report.ID)
	if err != nil {
		t.Fatalf("GetReport() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected non-nil report")
	}

	if retrieved.ID != report.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, report.ID)
	}
}

func TestSLAService_GetAllReports(t *testing.T) {
	ctx := context.Background()

	reportRepo := NewMockSLAReportRepository()

	service := NewSLAService(nil, nil, nil, reportRepo, nil, nil, nil)

	// Create multiple reports
	for i := 0; i < 5; i++ {
		report := domain.NewSLAReport("Report", "monthly", time.Now().AddDate(0, -1, 0), time.Now(), "admin")
		reportRepo.Create(ctx, report)
	}

	reports, err := service.GetAllReports(ctx, 10)
	if err != nil {
		t.Fatalf("GetAllReports() error = %v", err)
	}

	if len(reports) != 5 {
		t.Errorf("GetAllReports() returned %d reports, want 5", len(reports))
	}
}

func TestSLAService_DeleteReport(t *testing.T) {
	ctx := context.Background()

	reportRepo := NewMockSLAReportRepository()

	service := NewSLAService(nil, nil, nil, reportRepo, nil, nil, nil)

	// Create and delete a report
	report := domain.NewSLAReport("To Delete", "monthly", time.Now().AddDate(0, -1, 0), time.Now(), "admin")
	reportRepo.Create(ctx, report)

	err := service.DeleteReport(ctx, report.ID)
	if err != nil {
		t.Fatalf("DeleteReport() error = %v", err)
	}

	// Verify deletion
	retrieved, _ := reportRepo.GetByID(ctx, report.ID)
	if retrieved != nil {
		t.Error("expected report to be deleted")
	}
}

func TestSLAService_CheckForBreaches(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	breachRepo := NewMockSLABreachRepository()

	service := NewSLAService(
		systemRepo,
		nil,
		analyticsRepo,
		nil,
		breachRepo,
		nil,
		nil,
	)

	// Create system with SLA target
	system, _ := domain.NewSystem("API Gateway", "", "", "")
	system.SetSLATarget(99.9)
	systemRepo.Create(ctx, system)

	// Set analytics to return metrics below SLA target
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return &domain.Analytics{
			UptimePercent:       98.5, // Below 99.9 target
			AvailabilityPercent: 99.0,
		}, nil
	}

	breaches, err := service.CheckForBreaches(ctx, "monthly")
	if err != nil {
		t.Fatalf("CheckForBreaches() error = %v", err)
	}

	if len(breaches) != 1 {
		t.Errorf("expected 1 breach, got %d", len(breaches))
	}

	if len(breaches) > 0 {
		breach := breaches[0]
		if breach.BreachType != "uptime" {
			t.Errorf("BreachType = %s, want uptime", breach.BreachType)
		}
		if breach.SLATarget != 99.9 {
			t.Errorf("SLATarget = %f, want 99.9", breach.SLATarget)
		}
		if breach.ActualValue != 98.5 {
			t.Errorf("ActualValue = %f, want 98.5", breach.ActualValue)
		}
	}
}

func TestSLAService_CheckForBreaches_NoBreaches(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	breachRepo := NewMockSLABreachRepository()

	service := NewSLAService(
		systemRepo,
		nil,
		analyticsRepo,
		nil,
		breachRepo,
		nil,
		nil,
	)

	// Create system with SLA target
	system, _ := domain.NewSystem("API Gateway", "", "", "")
	system.SetSLATarget(99.0)
	systemRepo.Create(ctx, system)

	// Set analytics to return metrics above SLA target
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return &domain.Analytics{
			UptimePercent:       99.9, // Above 99.0 target
			AvailabilityPercent: 99.95,
		}, nil
	}

	breaches, err := service.CheckForBreaches(ctx, "monthly")
	if err != nil {
		t.Fatalf("CheckForBreaches() error = %v", err)
	}

	if len(breaches) != 0 {
		t.Errorf("expected 0 breaches, got %d", len(breaches))
	}
}

func TestSLAService_GetBreaches(t *testing.T) {
	ctx := context.Background()

	breachRepo := NewMockSLABreachRepository()

	service := NewSLAService(nil, nil, nil, nil, breachRepo, nil, nil)

	// Create breaches
	for i := 0; i < 3; i++ {
		breach := &domain.SLABreachEvent{
			SystemID:    1,
			BreachType:  "uptime",
			SLATarget:   99.9,
			ActualValue: 98.0,
			Period:      "monthly",
			DetectedAt:  time.Now(),
		}
		breachRepo.Create(ctx, breach)
	}

	breaches, err := service.GetBreaches(ctx, 10)
	if err != nil {
		t.Fatalf("GetBreaches() error = %v", err)
	}

	if len(breaches) != 3 {
		t.Errorf("expected 3 breaches, got %d", len(breaches))
	}
}

func TestSLAService_GetUnacknowledgedBreaches(t *testing.T) {
	ctx := context.Background()

	breachRepo := NewMockSLABreachRepository()

	service := NewSLAService(nil, nil, nil, nil, breachRepo, nil, nil)

	// Create unacknowledged breach
	unacked := &domain.SLABreachEvent{
		SystemID:     1,
		BreachType:   "uptime",
		SLATarget:    99.9,
		ActualValue:  98.0,
		Period:       "monthly",
		DetectedAt:   time.Now(),
		Acknowledged: false,
	}
	breachRepo.Create(ctx, unacked)

	// Create acknowledged breach
	acked := &domain.SLABreachEvent{
		SystemID:     1,
		BreachType:   "uptime",
		SLATarget:    99.9,
		ActualValue:  97.0,
		Period:       "monthly",
		DetectedAt:   time.Now(),
		Acknowledged: true,
	}
	breachRepo.Create(ctx, acked)

	breaches, err := service.GetUnacknowledgedBreaches(ctx)
	if err != nil {
		t.Fatalf("GetUnacknowledgedBreaches() error = %v", err)
	}

	if len(breaches) != 1 {
		t.Errorf("expected 1 unacknowledged breach, got %d", len(breaches))
	}
}

func TestSLAService_GetSystemBreaches(t *testing.T) {
	ctx := context.Background()

	breachRepo := NewMockSLABreachRepository()

	service := NewSLAService(nil, nil, nil, nil, breachRepo, nil, nil)

	// Create breaches for different systems
	breach1 := &domain.SLABreachEvent{SystemID: 1, BreachType: "uptime", DetectedAt: time.Now()}
	breach2 := &domain.SLABreachEvent{SystemID: 1, BreachType: "uptime", DetectedAt: time.Now()}
	breach3 := &domain.SLABreachEvent{SystemID: 2, BreachType: "uptime", DetectedAt: time.Now()}

	breachRepo.Create(ctx, breach1)
	breachRepo.Create(ctx, breach2)
	breachRepo.Create(ctx, breach3)

	breaches, err := service.GetSystemBreaches(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetSystemBreaches() error = %v", err)
	}

	if len(breaches) != 2 {
		t.Errorf("expected 2 breaches for system 1, got %d", len(breaches))
	}
}

func TestSLAService_AcknowledgeBreach(t *testing.T) {
	ctx := context.Background()

	breachRepo := NewMockSLABreachRepository()

	service := NewSLAService(nil, nil, nil, nil, breachRepo, nil, nil)

	// Create breach
	breach := &domain.SLABreachEvent{
		SystemID:     1,
		BreachType:   "uptime",
		SLATarget:    99.9,
		ActualValue:  98.0,
		Period:       "monthly",
		DetectedAt:   time.Now(),
		Acknowledged: false,
	}
	breachRepo.Create(ctx, breach)

	err := service.AcknowledgeBreach(ctx, breach.ID, "admin")
	if err != nil {
		t.Fatalf("AcknowledgeBreach() error = %v", err)
	}

	// Verify acknowledgment
	retrieved, _ := breachRepo.GetByID(ctx, breach.ID)
	if !retrieved.Acknowledged {
		t.Error("expected breach to be acknowledged")
	}
	if retrieved.AckedBy != "admin" {
		t.Errorf("AckedBy = %s, want admin", retrieved.AckedBy)
	}
}

func TestSLAService_GetSystemSLAStatus(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	latencyRepo := NewMockLatencyRepository()

	service := NewSLAService(
		systemRepo,
		depRepo,
		analyticsRepo,
		nil,
		nil,
		latencyRepo,
		nil,
	)

	// Create system
	system, _ := domain.NewSystem("API Gateway", "", "", "ops@example.com")
	system.SetSLATarget(99.9)
	systemRepo.Create(ctx, system)

	// Set analytics
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		return &domain.Analytics{
			UptimePercent:       99.95,
			AvailabilityPercent: 99.98,
			TotalIncidents:      1,
			ResolvedIncidents:   1,
		}, nil
	}

	status, err := service.GetSystemSLAStatus(ctx, system.ID, "monthly")
	if err != nil {
		t.Fatalf("GetSystemSLAStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("expected non-nil status")
	}

	if status.SystemID != system.ID {
		t.Errorf("SystemID = %d, want %d", status.SystemID, system.ID)
	}

	if status.SystemName != "API Gateway" {
		t.Errorf("SystemName = %s, want API Gateway", status.SystemName)
	}

	if !status.SLAMet {
		t.Error("expected SLAMet = true (99.95 >= 99.9)")
	}
}

func TestSLAService_GetSystemSLAStatus_NotFound(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()

	service := NewSLAService(systemRepo, nil, nil, nil, nil, nil, nil)

	_, err := service.GetSystemSLAStatus(ctx, 999, "monthly")
	if err == nil {
		t.Error("expected error for non-existent system")
	}
}

func TestSLAService_UpdateSystemSLATarget(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()

	service := NewSLAService(systemRepo, nil, nil, nil, nil, nil, nil)

	// Create system
	system, _ := domain.NewSystem("API Gateway", "", "", "")
	system.SetSLATarget(99.0)
	systemRepo.Create(ctx, system)

	err := service.UpdateSystemSLATarget(ctx, system.ID, 99.9)
	if err != nil {
		t.Fatalf("UpdateSystemSLATarget() error = %v", err)
	}

	// Verify update
	retrieved, _ := systemRepo.GetByID(ctx, system.ID)
	if retrieved.GetSLATarget() != 99.9 {
		t.Errorf("SLATarget = %f, want 99.9", retrieved.GetSLATarget())
	}
}

func TestSLAService_UpdateSystemSLATarget_NotFound(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()

	service := NewSLAService(systemRepo, nil, nil, nil, nil, nil, nil)

	err := service.UpdateSystemSLATarget(ctx, 999, 99.9)
	if err == nil {
		t.Error("expected error for non-existent system")
	}
}

// ============= Mock Repositories for SLA Service =============

type MockSLAReportRepository struct {
	Reports map[int64]*domain.SLAReport
}

func NewMockSLAReportRepository() *MockSLAReportRepository {
	return &MockSLAReportRepository{
		Reports: make(map[int64]*domain.SLAReport),
	}
}

func (m *MockSLAReportRepository) Create(ctx context.Context, r *domain.SLAReport) error {
	r.ID = int64(len(m.Reports) + 1)
	m.Reports[r.ID] = r
	return nil
}

func (m *MockSLAReportRepository) GetByID(ctx context.Context, id int64) (*domain.SLAReport, error) {
	return m.Reports[id], nil
}

func (m *MockSLAReportRepository) GetAll(ctx context.Context, limit int) ([]*domain.SLAReport, error) {
	var result []*domain.SLAReport
	for _, r := range m.Reports {
		result = append(result, r)
	}
	return result, nil
}

func (m *MockSLAReportRepository) GetByPeriod(ctx context.Context, start, end time.Time) ([]*domain.SLAReport, error) {
	var result []*domain.SLAReport
	for _, r := range m.Reports {
		if r.PeriodStart.After(start) && r.PeriodEnd.Before(end) {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockSLAReportRepository) Delete(ctx context.Context, id int64) error {
	delete(m.Reports, id)
	return nil
}

type MockSLABreachRepository struct {
	Breaches map[int64]*domain.SLABreachEvent
}

func NewMockSLABreachRepository() *MockSLABreachRepository {
	return &MockSLABreachRepository{
		Breaches: make(map[int64]*domain.SLABreachEvent),
	}
}

func (m *MockSLABreachRepository) Create(ctx context.Context, b *domain.SLABreachEvent) error {
	b.ID = int64(len(m.Breaches) + 1)
	m.Breaches[b.ID] = b
	return nil
}

func (m *MockSLABreachRepository) GetByID(ctx context.Context, id int64) (*domain.SLABreachEvent, error) {
	return m.Breaches[id], nil
}

func (m *MockSLABreachRepository) GetAll(ctx context.Context, limit int) ([]*domain.SLABreachEvent, error) {
	var result []*domain.SLABreachEvent
	for _, b := range m.Breaches {
		result = append(result, b)
	}
	return result, nil
}

func (m *MockSLABreachRepository) GetUnacknowledged(ctx context.Context) ([]*domain.SLABreachEvent, error) {
	var result []*domain.SLABreachEvent
	for _, b := range m.Breaches {
		if !b.Acknowledged {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *MockSLABreachRepository) GetBySystemID(ctx context.Context, systemID int64, limit int) ([]*domain.SLABreachEvent, error) {
	var result []*domain.SLABreachEvent
	for _, b := range m.Breaches {
		if b.SystemID == systemID {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *MockSLABreachRepository) Acknowledge(ctx context.Context, id int64, ackedBy string) error {
	if b, ok := m.Breaches[id]; ok {
		b.Acknowledged = true
		b.AckedBy = ackedBy
		now := time.Now()
		b.AckedAt = &now
	}
	return nil
}

func (m *MockSLABreachRepository) GetByPeriod(ctx context.Context, start, end time.Time) ([]*domain.SLABreachEvent, error) {
	var result []*domain.SLABreachEvent
	for _, b := range m.Breaches {
		if b.PeriodStart.After(start) && b.PeriodEnd.Before(end) {
			result = append(result, b)
		}
	}
	return result, nil
}

// ============= Float Comparison and Content Validation Tests =============

func TestSLAService_CheckForBreaches_FloatComparison(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	breachRepo := NewMockSLABreachRepository()

	service := NewSLAService(
		systemRepo,
		nil,
		analyticsRepo,
		nil,
		breachRepo,
		nil,
		nil,
	)

	tests := []struct {
		name           string
		slaTarget      float64
		actualUptime   float64
		expectBreach   bool
	}{
		{"exactly at target", 99.9, 99.9, false},
		{"slightly above", 99.9, 99.91, false},
		{"slightly below", 99.9, 99.89, true},
		{"well below", 99.9, 98.5, true},
		{"well above", 99.9, 99.99, false},
		{"100% target met", 100.0, 100.0, false},
		{"100% target missed", 100.0, 99.99, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous state
			breachRepo.Breaches = make(map[int64]*domain.SLABreachEvent)
			systemRepo.Systems = make(map[int64]*domain.System)

			// Create system with specific target
			system, _ := domain.NewSystem("Test System", "", "", "")
			system.SetSLATarget(tt.slaTarget)
			systemRepo.Create(ctx, system)

			// Set analytics
			analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
				return &domain.Analytics{
					UptimePercent: tt.actualUptime,
				}, nil
			}

			breaches, err := service.CheckForBreaches(ctx, "monthly")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectBreach && len(breaches) == 0 {
				t.Errorf("expected breach for uptime %.2f%% vs target %.2f%%", tt.actualUptime, tt.slaTarget)
			}
			if !tt.expectBreach && len(breaches) > 0 {
				t.Errorf("unexpected breach for uptime %.2f%% vs target %.2f%%", tt.actualUptime, tt.slaTarget)
			}

			// Validate breach content
			if len(breaches) > 0 {
				breach := breaches[0]
				if math.Abs(breach.SLATarget-tt.slaTarget) > slaEpsilon {
					t.Errorf("SLATarget: expected %.2f, got %.2f", tt.slaTarget, breach.SLATarget)
				}
				if math.Abs(breach.ActualValue-tt.actualUptime) > slaEpsilon {
					t.Errorf("ActualValue: expected %.2f, got %.2f", tt.actualUptime, breach.ActualValue)
				}
			}
		})
	}
}

func TestSLAService_GetBreaches_ContentValidation(t *testing.T) {
	ctx := context.Background()

	breachRepo := NewMockSLABreachRepository()
	service := NewSLAService(nil, nil, nil, nil, breachRepo, nil, nil)

	now := time.Now()

	// Create breaches with specific content
	breach1 := &domain.SLABreachEvent{
		SystemID:    1,
		SystemName:  "API Gateway",
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.5,
		Period:      "monthly",
		DetectedAt:  now.Add(-time.Hour),
	}
	breach2 := &domain.SLABreachEvent{
		SystemID:    2,
		SystemName:  "Database",
		BreachType:  "uptime",
		SLATarget:   99.99,
		ActualValue: 99.5,
		Period:      "weekly",
		DetectedAt:  now,
	}

	breachRepo.Create(ctx, breach1)
	breachRepo.Create(ctx, breach2)

	breaches, err := service.GetBreaches(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(breaches) != 2 {
		t.Fatalf("expected 2 breaches, got %d", len(breaches))
	}

	// Create map for easier lookup
	breachMap := make(map[int64]*domain.SLABreachEvent)
	for _, b := range breaches {
		breachMap[b.SystemID] = b
	}

	// Validate breach 1
	if b, ok := breachMap[1]; !ok {
		t.Error("breach for system 1 not found")
	} else {
		if b.SystemName != "API Gateway" {
			t.Errorf("breach 1: expected SystemName 'API Gateway', got %q", b.SystemName)
		}
		if math.Abs(b.SLATarget-99.9) > slaEpsilon {
			t.Errorf("breach 1: expected SLATarget 99.9, got %f", b.SLATarget)
		}
		if math.Abs(b.ActualValue-98.5) > slaEpsilon {
			t.Errorf("breach 1: expected ActualValue 98.5, got %f", b.ActualValue)
		}
	}

	// Validate breach 2
	if b, ok := breachMap[2]; !ok {
		t.Error("breach for system 2 not found")
	} else {
		if b.SystemName != "Database" {
			t.Errorf("breach 2: expected SystemName 'Database', got %q", b.SystemName)
		}
		if b.Period != "weekly" {
			t.Errorf("breach 2: expected Period 'weekly', got %q", b.Period)
		}
	}
}

func TestSLAService_GenerateReport_SystemReportsContentValidation(t *testing.T) {
	ctx := context.Background()

	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	reportRepo := NewMockSLAReportRepository()
	latencyRepo := NewMockLatencyRepository()

	service := NewSLAService(
		systemRepo,
		depRepo,
		analyticsRepo,
		reportRepo,
		nil,
		latencyRepo,
		nil,
	)

	// Create systems with different targets
	system1, _ := domain.NewSystem("API Gateway", "", "", "ops@example.com")
	system1.SetSLATarget(99.9)
	systemRepo.Create(ctx, system1)

	system2, _ := domain.NewSystem("Database", "", "", "dba@example.com")
	system2.SetSLATarget(99.99)
	systemRepo.Create(ctx, system2)

	// Set up different analytics for each system
	analyticsRepo.GetUptimeBySystemIDFunc = func(ctx context.Context, systemID int64, start, end time.Time) (*domain.Analytics, error) {
		if systemID == system1.ID {
			return &domain.Analytics{
				UptimePercent:       99.95,
				AvailabilityPercent: 99.98,
				TotalIncidents:      2,
			}, nil
		}
		return &domain.Analytics{
			UptimePercent:       99.5, // Below target
			AvailabilityPercent: 99.7,
			TotalIncidents:      5,
		}, nil
	}

	report, err := service.GenerateReport(ctx, "Test Report", "monthly", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.SystemReports) != 2 {
		t.Fatalf("expected 2 system reports, got %d", len(report.SystemReports))
	}

	// Create map for easier lookup
	sysReportMap := make(map[int64]domain.SystemSLAReport)
	for _, sr := range report.SystemReports {
		sysReportMap[sr.SystemID] = sr
	}

	// Validate system 1 report
	if sr, ok := sysReportMap[system1.ID]; !ok {
		t.Errorf("system report for %q not found", system1.Name)
	} else {
		if sr.SystemName != "API Gateway" {
			t.Errorf("system 1: expected name 'API Gateway', got %q", sr.SystemName)
		}
		if math.Abs(sr.SLATarget-99.9) > slaEpsilon {
			t.Errorf("system 1: expected SLATarget 99.9, got %f", sr.SLATarget)
		}
		if !sr.SLAMet {
			t.Errorf("system 1: expected SLAMet=true (99.95 >= 99.9)")
		}
		if sr.TotalIncidents != 2 {
			t.Errorf("system 1: expected TotalIncidents 2, got %d", sr.TotalIncidents)
		}
	}

	// Validate system 2 report
	if sr, ok := sysReportMap[system2.ID]; !ok {
		t.Errorf("system report for %q not found", system2.Name)
	} else {
		if sr.SystemName != "Database" {
			t.Errorf("system 2: expected name 'Database', got %q", sr.SystemName)
		}
		if sr.SLAMet {
			t.Errorf("system 2: expected SLAMet=false (99.5 < 99.99)")
		}
	}
}

func TestSLAService_calculateMTTR_EdgeCases(t *testing.T) {
	s := &SLAService{}
	now := time.Now()

	// Test very small durations
	t.Run("very short incident", func(t *testing.T) {
		incidents := []domain.IncidentPeriod{
			{
				StartedAt: now.Add(-100 * time.Millisecond),
				EndedAt:   timePtr(now),
			},
		}
		result := s.calculateMTTR(incidents)
		expected := 100 * time.Millisecond
		tolerance := 10 * time.Millisecond
		if result < expected-tolerance || result > expected+tolerance {
			t.Errorf("expected ~%v, got %v", expected, result)
		}
	})

	// Test very long durations
	t.Run("very long incident", func(t *testing.T) {
		incidents := []domain.IncidentPeriod{
			{
				StartedAt: now.Add(-48 * time.Hour),
				EndedAt:   timePtr(now),
			},
		}
		result := s.calculateMTTR(incidents)
		if result != 48*time.Hour {
			t.Errorf("expected 48h, got %v", result)
		}
	})

	// Test average calculation precision
	t.Run("average calculation", func(t *testing.T) {
		incidents := []domain.IncidentPeriod{
			{StartedAt: now.Add(-3 * time.Hour), EndedAt: timePtr(now.Add(-2 * time.Hour))}, // 1h
			{StartedAt: now.Add(-1 * time.Hour), EndedAt: timePtr(now.Add(-30 * time.Minute))}, // 30m
		}
		result := s.calculateMTTR(incidents)
		expected := 45 * time.Minute // (60m + 30m) / 2
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})
}
