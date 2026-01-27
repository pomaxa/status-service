package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

func TestIncidentRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	incident, err := domain.NewIncident("Database Outage", "Database is not responding", domain.SeverityMajor)
	if err != nil {
		t.Fatalf("failed to create incident: %v", err)
	}

	err = repo.Create(ctx, incident)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if incident.ID == 0 {
		t.Error("expected incident ID to be set after Create()")
	}
}

func TestIncidentRepo_Create_WithSystemIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create some systems first
	sysRepo := NewSystemRepo(db)
	sys1, _ := domain.NewSystem("System 1", "", "", "")
	sys2, _ := domain.NewSystem("System 2", "", "", "")
	sysRepo.Create(context.Background(), sys1)
	sysRepo.Create(context.Background(), sys2)

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	incident, _ := domain.NewIncident("Multi-system Outage", "Multiple systems affected", domain.SeverityCritical)
	incident.SystemIDs = []int64{sys1.ID, sys2.ID}

	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(retrieved.SystemIDs) != 2 {
		t.Errorf("SystemIDs count = %d, want 2", len(retrieved.SystemIDs))
	}
}

func TestIncidentRepo_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	incident, _ := domain.NewIncident("Test Incident", "Test message", domain.SeverityMinor)
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByID() returned nil")
	}

	if retrieved.ID != incident.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, incident.ID)
	}
	if retrieved.Title != incident.Title {
		t.Errorf("Title = %s, want %s", retrieved.Title, incident.Title)
	}
	if retrieved.Message != incident.Message {
		t.Errorf("Message = %s, want %s", retrieved.Message, incident.Message)
	}
	if retrieved.Status != incident.Status {
		t.Errorf("Status = %s, want %s", retrieved.Status, incident.Status)
	}
	if retrieved.Severity != incident.Severity {
		t.Errorf("Severity = %s, want %s", retrieved.Severity, incident.Severity)
	}
}

func TestIncidentRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestIncidentRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	// Create multiple incidents
	for i := 0; i < 5; i++ {
		incident, _ := domain.NewIncident("Incident "+string(rune('A'+i)), "Message", domain.SeverityMinor)
		if err := repo.Create(ctx, incident); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// Get all
	all, err := repo.GetAll(ctx, 0) // 0 = no limit
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(all) != 5 {
		t.Errorf("GetAll() returned %d incidents, want 5", len(all))
	}
}

func TestIncidentRepo_GetAll_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	// Create multiple incidents
	for i := 0; i < 5; i++ {
		incident, _ := domain.NewIncident("Incident "+string(rune('A'+i)), "Message", domain.SeverityMinor)
		repo.Create(ctx, incident)
		time.Sleep(time.Millisecond)
	}

	// Get with limit
	limited, err := repo.GetAll(ctx, 3)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(limited) != 3 {
		t.Errorf("GetAll(3) returned %d incidents, want 3", len(limited))
	}
}

func TestIncidentRepo_GetActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	// Create active incidents with different severities
	critical, _ := domain.NewIncident("Critical Incident", "Critical", domain.SeverityCritical)
	major, _ := domain.NewIncident("Major Incident", "Major", domain.SeverityMajor)
	minor, _ := domain.NewIncident("Minor Incident", "Minor", domain.SeverityMinor)

	// Create resolved incident
	resolved, _ := domain.NewIncident("Resolved Incident", "Was resolved", domain.SeverityMajor)
	resolved.Resolve("Fixed the issue")

	for _, inc := range []*domain.Incident{critical, major, minor, resolved} {
		if err := repo.Create(ctx, inc); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get active
	active, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}

	if len(active) != 3 {
		t.Errorf("GetActive() returned %d incidents, want 3", len(active))
	}

	// Should be ordered by severity (critical first)
	if active[0].Severity != domain.SeverityCritical {
		t.Errorf("first incident severity = %s, want critical", active[0].Severity)
	}
	if active[1].Severity != domain.SeverityMajor {
		t.Errorf("second incident severity = %s, want major", active[1].Severity)
	}
	if active[2].Severity != domain.SeverityMinor {
		t.Errorf("third incident severity = %s, want minor", active[2].Severity)
	}
}

func TestIncidentRepo_GetRecent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	// Create resolved incident
	incident, _ := domain.NewIncident("Recent Resolved", "Was resolved recently", domain.SeverityMinor)
	incident.Resolve("Fixed")
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create active incident (should not appear in recent resolved)
	active, _ := domain.NewIncident("Still Active", "Still investigating", domain.SeverityMajor)
	if err := repo.Create(ctx, active); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get recent (last 7 days)
	recent, err := repo.GetRecent(ctx, 7)
	if err != nil {
		t.Fatalf("GetRecent() error = %v", err)
	}

	if len(recent) != 1 {
		t.Errorf("GetRecent() returned %d incidents, want 1", len(recent))
	}

	if recent[0].Title != "Recent Resolved" {
		t.Errorf("recent incident title = %s, want Recent Resolved", recent[0].Title)
	}
}

func TestIncidentRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	incident, _ := domain.NewIncident("Original Title", "Original message", domain.SeverityMinor)
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	incident.Title = "Updated Title"
	incident.Message = "Updated message"
	incident.Severity = domain.SeverityMajor
	incident.UpdateStatus(domain.IncidentIdentified)
	incident.Postmortem = "Root cause found"

	if err := repo.Update(ctx, incident); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, err := repo.GetByID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Title = %s, want Updated Title", retrieved.Title)
	}
	if retrieved.Message != "Updated message" {
		t.Errorf("Message = %s, want Updated message", retrieved.Message)
	}
	if retrieved.Severity != domain.SeverityMajor {
		t.Errorf("Severity = %s, want major", retrieved.Severity)
	}
	if retrieved.Status != domain.IncidentIdentified {
		t.Errorf("Status = %s, want identified", retrieved.Status)
	}
	if retrieved.Postmortem != "Root cause found" {
		t.Errorf("Postmortem = %s, want Root cause found", retrieved.Postmortem)
	}
}

func TestIncidentRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	incident, _ := domain.NewIncident("To Delete", "Will be deleted", domain.SeverityMinor)
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, incident.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil after deletion")
	}
}

func TestIncidentRepo_CreateUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	// Create incident
	incident, _ := domain.NewIncident("Incident with Updates", "Initial message", domain.SeverityMajor)
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create update
	update, err := domain.NewIncidentUpdate(incident.ID, domain.IncidentIdentified, "Root cause identified", "admin")
	if err != nil {
		t.Fatalf("NewIncidentUpdate() error = %v", err)
	}

	if err := repo.CreateUpdate(ctx, update); err != nil {
		t.Fatalf("CreateUpdate() error = %v", err)
	}

	if update.ID == 0 {
		t.Error("expected update ID to be set after CreateUpdate()")
	}
}

func TestIncidentRepo_GetUpdates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	// Create incident
	incident, _ := domain.NewIncident("Incident with Multiple Updates", "Initial message", domain.SeverityCritical)
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create multiple updates
	updates := []struct {
		status  domain.IncidentStatus
		message string
		author  string
	}{
		{domain.IncidentInvestigating, "Investigating the issue", "alice"},
		{domain.IncidentIdentified, "Root cause identified", "bob"},
		{domain.IncidentMonitoring, "Fix deployed, monitoring", "charlie"},
	}

	for _, u := range updates {
		update, _ := domain.NewIncidentUpdate(incident.ID, u.status, u.message, u.author)
		if err := repo.CreateUpdate(ctx, update); err != nil {
			t.Fatalf("CreateUpdate() error = %v", err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// Get updates
	retrieved, err := repo.GetUpdates(ctx, incident.ID)
	if err != nil {
		t.Fatalf("GetUpdates() error = %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("GetUpdates() returned %d updates, want 3", len(retrieved))
	}

	// Should be ordered by created_at DESC (most recent first)
	if retrieved[0].Message != "Fix deployed, monitoring" {
		t.Errorf("first update message = %s, want 'Fix deployed, monitoring'", retrieved[0].Message)
	}
}

func TestIncidentRepo_GetUpdates_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	// Create incident without updates
	incident, _ := domain.NewIncident("No Updates", "Initial message", domain.SeverityMinor)
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updates, err := repo.GetUpdates(ctx, incident.ID)
	if err != nil {
		t.Fatalf("GetUpdates() error = %v", err)
	}

	if updates != nil && len(updates) != 0 {
		t.Errorf("GetUpdates() returned %d updates, want 0", len(updates))
	}
}

func TestIncidentRepo_Acknowledge(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	incident, _ := domain.NewIncident("Needs Ack", "Needs acknowledgement", domain.SeverityMajor)
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Acknowledge
	incident.Acknowledge("admin")
	if err := repo.Update(ctx, incident); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, err := repo.GetByID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.AcknowledgedAt == nil {
		t.Error("expected AcknowledgedAt to be set")
	}
	if retrieved.AcknowledgedBy != "admin" {
		t.Errorf("AcknowledgedBy = %s, want admin", retrieved.AcknowledgedBy)
	}
}

func TestIncidentRepo_Resolve(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	incident, _ := domain.NewIncident("To Resolve", "Will be resolved", domain.SeverityMinor)
	if err := repo.Create(ctx, incident); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Resolve
	incident.Resolve("Issue fixed, deployed hotfix")
	if err := repo.Update(ctx, incident); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, err := repo.GetByID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Status != domain.IncidentResolved {
		t.Errorf("Status = %s, want resolved", retrieved.Status)
	}
	if retrieved.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set")
	}
	if retrieved.Postmortem != "Issue fixed, deployed hotfix" {
		t.Errorf("Postmortem = %s, want 'Issue fixed, deployed hotfix'", retrieved.Postmortem)
	}
}

func TestIncidentRepo_SeverityTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewIncidentRepo(db)
	ctx := context.Background()

	severities := []domain.IncidentSeverity{
		domain.SeverityMinor,
		domain.SeverityMajor,
		domain.SeverityCritical,
	}

	for _, severity := range severities {
		t.Run(string(severity), func(t *testing.T) {
			incident, _ := domain.NewIncident("Severity Test", "Testing "+string(severity), severity)
			if err := repo.Create(ctx, incident); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			retrieved, err := repo.GetByID(ctx, incident.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if retrieved.Severity != severity {
				t.Errorf("Severity = %s, want %s", retrieved.Severity, severity)
			}
		})
	}
}
