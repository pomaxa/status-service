package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// ============= SLAReportRepo Tests =============

func TestSLAReportRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	report := domain.NewSLAReport("Monthly SLA Report", "monthly", start, end, "admin")
	report.OverallUptime = 99.5
	report.OverallAvailability = 99.8
	report.TotalSystems = 5
	report.SystemsMeetingSLA = 4
	report.SystemsBreachingSLA = 1

	err := repo.Create(ctx, report)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if report.ID == 0 {
		t.Error("expected report ID to be set after Create()")
	}
}

func TestSLAReportRepo_Create_WithSystemReports(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	report := domain.NewSLAReport("Report with Systems", "monthly", start, end, "admin")

	// Add system reports
	report.AddSystemReport(domain.SystemSLAReport{
		SystemID:          1,
		SystemName:        "API Gateway",
		UptimePercent:     99.9,
		SLATarget:         99.5,
		SLAMet:            true,
		TotalIncidents:    2,
		ResolvedIncidents: 2,
	})
	report.AddSystemReport(domain.SystemSLAReport{
		SystemID:          2,
		SystemName:        "Database",
		UptimePercent:     98.5,
		SLATarget:         99.5,
		SLAMet:            false,
		TotalIncidents:    5,
		ResolvedIncidents: 4,
	})

	report.CalculateOverall()

	if err := repo.Create(ctx, report); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, report.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(retrieved.SystemReports) != 2 {
		t.Errorf("SystemReports count = %d, want 2", len(retrieved.SystemReports))
	}
}

func TestSLAReportRepo_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	report := domain.NewSLAReport("Test Report", "weekly", start, end, "tester")
	report.OverallUptime = 99.0
	if err := repo.Create(ctx, report); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, report.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByID() returned nil")
	}

	if retrieved.ID != report.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, report.ID)
	}
	if retrieved.Title != report.Title {
		t.Errorf("Title = %s, want %s", retrieved.Title, report.Title)
	}
	if retrieved.Period != "weekly" {
		t.Errorf("Period = %s, want weekly", retrieved.Period)
	}
	if retrieved.GeneratedBy != "tester" {
		t.Errorf("GeneratedBy = %s, want tester", retrieved.GeneratedBy)
	}
	if retrieved.OverallUptime != 99.0 {
		t.Errorf("OverallUptime = %f, want 99.0", retrieved.OverallUptime)
	}
}

func TestSLAReportRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestSLAReportRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	// Create multiple reports
	for i := 0; i < 5; i++ {
		start := time.Now().AddDate(0, -i-1, 0)
		end := time.Now().AddDate(0, -i, 0)
		report := domain.NewSLAReport("Report "+string(rune('A'+i)), "monthly", start, end, "admin")
		repo.Create(ctx, report)
		time.Sleep(time.Millisecond)
	}

	reports, err := repo.GetAll(ctx, 10)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(reports) != 5 {
		t.Errorf("GetAll() returned %d reports, want 5", len(reports))
	}
}

func TestSLAReportRepo_GetAll_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	// Create multiple reports
	for i := 0; i < 10; i++ {
		start := time.Now().AddDate(0, -i-1, 0)
		end := time.Now().AddDate(0, -i, 0)
		report := domain.NewSLAReport("Report "+string(rune('A'+i)), "monthly", start, end, "admin")
		repo.Create(ctx, report)
	}

	reports, err := repo.GetAll(ctx, 5)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(reports) != 5 {
		t.Errorf("GetAll(5) returned %d reports, want 5", len(reports))
	}
}

func TestSLAReportRepo_GetByPeriod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	// Create reports at different times
	now := time.Now()

	// Report within query range
	inRangeStart := now.AddDate(0, 0, -15)
	inRangeEnd := now.AddDate(0, 0, -1)
	inRange := domain.NewSLAReport("In Range", "monthly", inRangeStart, inRangeEnd, "admin")
	repo.Create(ctx, inRange)

	// Report outside query range
	outRangeStart := now.AddDate(0, -3, 0)
	outRangeEnd := now.AddDate(0, -2, 0)
	outRange := domain.NewSLAReport("Out of Range", "monthly", outRangeStart, outRangeEnd, "admin")
	repo.Create(ctx, outRange)

	// Query for reports in last month
	queryStart := now.AddDate(0, -1, 0)
	queryEnd := now

	reports, err := repo.GetByPeriod(ctx, queryStart, queryEnd)
	if err != nil {
		t.Fatalf("GetByPeriod() error = %v", err)
	}

	if len(reports) != 1 {
		t.Errorf("GetByPeriod() returned %d reports, want 1", len(reports))
	}

	if len(reports) > 0 && reports[0].Title != "In Range" {
		t.Errorf("report title = %s, want In Range", reports[0].Title)
	}
}

func TestSLAReportRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	report := domain.NewSLAReport("To Delete", "monthly", start, end, "admin")
	if err := repo.Create(ctx, report); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, report.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, report.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil after deletion")
	}
}

func TestSLAReportRepo_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLAReportRepo(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error when deleting non-existent report")
	}
}

// ============= SLABreachRepo Tests =============

func TestSLABreachRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a system first
	sysRepo := NewSystemRepo(db)
	system, _ := domain.NewSystem("Test System", "", "", "")
	sysRepo.Create(context.Background(), system)

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	breach := &domain.SLABreachEvent{
		SystemID:    system.ID,
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.5,
		Period:      "monthly",
		PeriodStart: time.Now().AddDate(0, -1, 0),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}

	err := repo.Create(ctx, breach)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if breach.ID == 0 {
		t.Error("expected breach ID to be set after Create()")
	}
}

func TestSLABreachRepo_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create system
	sysRepo := NewSystemRepo(db)
	system, _ := domain.NewSystem("Breach Test System", "", "", "")
	sysRepo.Create(context.Background(), system)

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	breach := &domain.SLABreachEvent{
		SystemID:    system.ID,
		BreachType:  "availability",
		SLATarget:   99.5,
		ActualValue: 95.0,
		Period:      "weekly",
		PeriodStart: time.Now().AddDate(0, 0, -7),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}
	if err := repo.Create(ctx, breach); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, breach.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByID() returned nil")
	}

	if retrieved.ID != breach.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, breach.ID)
	}
	if retrieved.SystemID != system.ID {
		t.Errorf("SystemID = %d, want %d", retrieved.SystemID, system.ID)
	}
	if retrieved.SystemName != system.Name {
		t.Errorf("SystemName = %s, want %s", retrieved.SystemName, system.Name)
	}
	if retrieved.BreachType != "availability" {
		t.Errorf("BreachType = %s, want availability", retrieved.BreachType)
	}
	if retrieved.SLATarget != 99.5 {
		t.Errorf("SLATarget = %f, want 99.5", retrieved.SLATarget)
	}
	if retrieved.ActualValue != 95.0 {
		t.Errorf("ActualValue = %f, want 95.0", retrieved.ActualValue)
	}
}

func TestSLABreachRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestSLABreachRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create system
	sysRepo := NewSystemRepo(db)
	system, _ := domain.NewSystem("Breach System", "", "", "")
	sysRepo.Create(context.Background(), system)

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	// Create multiple breaches
	for i := 0; i < 5; i++ {
		breach := &domain.SLABreachEvent{
			SystemID:    system.ID,
			BreachType:  "uptime",
			SLATarget:   99.9,
			ActualValue: float64(98 - i),
			Period:      "monthly",
			PeriodStart: time.Now().AddDate(0, -1, 0),
			PeriodEnd:   time.Now(),
			DetectedAt:  time.Now(),
		}
		repo.Create(ctx, breach)
		time.Sleep(time.Millisecond)
	}

	breaches, err := repo.GetAll(ctx, 10)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(breaches) != 5 {
		t.Errorf("GetAll() returned %d breaches, want 5", len(breaches))
	}
}

func TestSLABreachRepo_GetUnacknowledged(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create system
	sysRepo := NewSystemRepo(db)
	system, _ := domain.NewSystem("Ack Test System", "", "", "")
	sysRepo.Create(context.Background(), system)

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	// Create unacknowledged breach
	unacked := &domain.SLABreachEvent{
		SystemID:     system.ID,
		BreachType:   "uptime",
		SLATarget:    99.9,
		ActualValue:  98.0,
		Period:       "monthly",
		PeriodStart:  time.Now().AddDate(0, -1, 0),
		PeriodEnd:    time.Now(),
		DetectedAt:   time.Now(),
		Acknowledged: false,
	}
	repo.Create(ctx, unacked)

	// Create acknowledged breach
	acked := &domain.SLABreachEvent{
		SystemID:     system.ID,
		BreachType:   "uptime",
		SLATarget:    99.9,
		ActualValue:  97.0,
		Period:       "monthly",
		PeriodStart:  time.Now().AddDate(0, -1, 0),
		PeriodEnd:    time.Now(),
		DetectedAt:   time.Now(),
		Acknowledged: true,
		AckedBy:      "admin",
	}
	now := time.Now()
	acked.AckedAt = &now
	repo.Create(ctx, acked)

	// Get unacknowledged
	breaches, err := repo.GetUnacknowledged(ctx)
	if err != nil {
		t.Fatalf("GetUnacknowledged() error = %v", err)
	}

	if len(breaches) != 1 {
		t.Errorf("GetUnacknowledged() returned %d breaches, want 1", len(breaches))
	}

	if len(breaches) > 0 && breaches[0].Acknowledged {
		t.Error("expected unacknowledged breach")
	}
}

func TestSLABreachRepo_GetBySystemID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create multiple systems
	sysRepo := NewSystemRepo(db)
	system1, _ := domain.NewSystem("System 1", "", "", "")
	system2, _ := domain.NewSystem("System 2", "", "", "")
	sysRepo.Create(context.Background(), system1)
	sysRepo.Create(context.Background(), system2)

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	// Create breaches for system1
	for i := 0; i < 3; i++ {
		breach := &domain.SLABreachEvent{
			SystemID:    system1.ID,
			BreachType:  "uptime",
			SLATarget:   99.9,
			ActualValue: 98.0,
			Period:      "monthly",
			PeriodStart: time.Now().AddDate(0, -1, 0),
			PeriodEnd:   time.Now(),
			DetectedAt:  time.Now(),
		}
		repo.Create(ctx, breach)
	}

	// Create breach for system2
	breach2 := &domain.SLABreachEvent{
		SystemID:    system2.ID,
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 97.0,
		Period:      "monthly",
		PeriodStart: time.Now().AddDate(0, -1, 0),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}
	repo.Create(ctx, breach2)

	// Get breaches for system1
	breaches, err := repo.GetBySystemID(ctx, system1.ID, 10)
	if err != nil {
		t.Fatalf("GetBySystemID() error = %v", err)
	}

	if len(breaches) != 3 {
		t.Errorf("GetBySystemID() returned %d breaches, want 3", len(breaches))
	}

	for _, b := range breaches {
		if b.SystemID != system1.ID {
			t.Errorf("breach SystemID = %d, want %d", b.SystemID, system1.ID)
		}
	}
}

func TestSLABreachRepo_Acknowledge(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create system
	sysRepo := NewSystemRepo(db)
	system, _ := domain.NewSystem("Ack System", "", "", "")
	sysRepo.Create(context.Background(), system)

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	breach := &domain.SLABreachEvent{
		SystemID:     system.ID,
		BreachType:   "uptime",
		SLATarget:    99.9,
		ActualValue:  98.0,
		Period:       "monthly",
		PeriodStart:  time.Now().AddDate(0, -1, 0),
		PeriodEnd:    time.Now(),
		DetectedAt:   time.Now(),
		Acknowledged: false,
	}
	if err := repo.Create(ctx, breach); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Acknowledge
	err := repo.Acknowledge(ctx, breach.ID, "admin")
	if err != nil {
		t.Fatalf("Acknowledge() error = %v", err)
	}

	// Verify
	retrieved, err := repo.GetByID(ctx, breach.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if !retrieved.Acknowledged {
		t.Error("expected Acknowledged = true")
	}
	if retrieved.AckedBy != "admin" {
		t.Errorf("AckedBy = %s, want admin", retrieved.AckedBy)
	}
	if retrieved.AckedAt == nil {
		t.Error("expected AckedAt to be set")
	}
}

func TestSLABreachRepo_Acknowledge_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	err := repo.Acknowledge(ctx, 99999, "admin")
	if err == nil {
		t.Error("expected error when acknowledging non-existent breach")
	}
}

func TestSLABreachRepo_GetByPeriod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create system
	sysRepo := NewSystemRepo(db)
	system, _ := domain.NewSystem("Period Test System", "", "", "")
	sysRepo.Create(context.Background(), system)

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	now := time.Now()

	// Create breach in query range
	inRange := &domain.SLABreachEvent{
		SystemID:    system.ID,
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.0,
		Period:      "weekly",
		PeriodStart: now.AddDate(0, 0, -7),
		PeriodEnd:   now,
		DetectedAt:  now,
	}
	repo.Create(ctx, inRange)

	// Create breach outside query range
	outRange := &domain.SLABreachEvent{
		SystemID:    system.ID,
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 97.0,
		Period:      "monthly",
		PeriodStart: now.AddDate(0, -2, 0),
		PeriodEnd:   now.AddDate(0, -1, 0),
		DetectedAt:  now.AddDate(0, -1, 0),
	}
	repo.Create(ctx, outRange)

	// Query for breaches in last 2 weeks
	queryStart := now.AddDate(0, 0, -14)
	queryEnd := now.AddDate(0, 0, 1)

	breaches, err := repo.GetByPeriod(ctx, queryStart, queryEnd)
	if err != nil {
		t.Fatalf("GetByPeriod() error = %v", err)
	}

	if len(breaches) != 1 {
		t.Errorf("GetByPeriod() returned %d breaches, want 1", len(breaches))
	}
}

func TestSLABreachRepo_BreachTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create system
	sysRepo := NewSystemRepo(db)
	system, _ := domain.NewSystem("Breach Types System", "", "", "")
	sysRepo.Create(context.Background(), system)

	repo := NewSLABreachRepo(db)
	ctx := context.Background()

	breachTypes := []string{"uptime", "availability", "response_time"}

	for _, btype := range breachTypes {
		t.Run(btype, func(t *testing.T) {
			breach := &domain.SLABreachEvent{
				SystemID:    system.ID,
				BreachType:  btype,
				SLATarget:   99.9,
				ActualValue: 98.0,
				Period:      "monthly",
				PeriodStart: time.Now().AddDate(0, -1, 0),
				PeriodEnd:   time.Now(),
				DetectedAt:  time.Now(),
			}

			if err := repo.Create(ctx, breach); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			retrieved, err := repo.GetByID(ctx, breach.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if retrieved.BreachType != btype {
				t.Errorf("BreachType = %s, want %s", retrieved.BreachType, btype)
			}
		})
	}
}
