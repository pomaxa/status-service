package application

import (
	"context"
	"status-incident/internal/domain"
	"testing"
)

func TestNewIncidentService(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()

	service := NewIncidentService(incidentRepo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestIncidentService_CreateIncident(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	incident, err := service.CreateIncident(
		context.Background(),
		"API Outage",
		"The API is not responding",
		domain.SeverityCritical,
		[]int64{1, 2},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if incident == nil {
		t.Fatal("expected non-nil incident")
	}
	if incident.Title != "API Outage" {
		t.Errorf("expected title 'API Outage', got %q", incident.Title)
	}
	if incident.Severity != domain.SeverityCritical {
		t.Errorf("expected severity critical, got %q", incident.Severity)
	}
	if len(incident.SystemIDs) != 2 {
		t.Errorf("expected 2 system IDs, got %d", len(incident.SystemIDs))
	}
}

func TestIncidentService_CreateIncident_NoSystems(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	incident, err := service.CreateIncident(
		context.Background(),
		"General Outage",
		"Unknown issue",
		domain.SeverityMajor,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if incident == nil {
		t.Fatal("expected non-nil incident")
	}
}

func TestIncidentService_CreateIncident_EmptyTitle(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	_, err := service.CreateIncident(
		context.Background(),
		"",
		"Message",
		domain.SeverityMinor,
		nil,
	)
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestIncidentService_GetIncident(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident, _ := domain.NewIncident("Test", "Test message", domain.SeverityMinor)
	incident.ID = 1
	incidentRepo.Incidents[1] = incident

	service := NewIncidentService(incidentRepo)

	result, err := service.GetIncident(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil incident")
	}
	if result.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", result.Title)
	}
}

func TestIncidentService_GetAllIncidents(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident1, _ := domain.NewIncident("Incident 1", "Message", domain.SeverityMinor)
	incident1.ID = 1
	incident2, _ := domain.NewIncident("Incident 2", "Message", domain.SeverityMajor)
	incident2.ID = 2
	incidentRepo.Incidents[1] = incident1
	incidentRepo.Incidents[2] = incident2

	service := NewIncidentService(incidentRepo)

	incidents, err := service.GetAllIncidents(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(incidents) != 2 {
		t.Errorf("expected 2 incidents, got %d", len(incidents))
	}
}

func TestIncidentService_GetActiveIncidents(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident1, _ := domain.NewIncident("Active", "Message", domain.SeverityMinor)
	incident1.ID = 1
	incident2, _ := domain.NewIncident("Resolved", "Message", domain.SeverityMinor)
	incident2.ID = 2
	incident2.Resolve("Fixed")
	incidentRepo.Incidents[1] = incident1
	incidentRepo.Incidents[2] = incident2

	service := NewIncidentService(incidentRepo)

	incidents, err := service.GetActiveIncidents(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(incidents) != 1 {
		t.Errorf("expected 1 active incident, got %d", len(incidents))
	}
}

func TestIncidentService_GetRecentIncidents(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	// Test with default days (0 -> 7)
	_, err := service.GetRecentIncidents(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test with specific days
	_, err = service.GetRecentIncidents(context.Background(), 14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIncidentService_GetIncidentUpdates(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	update, _ := domain.NewIncidentUpdate(1, domain.IncidentInvestigating, "Looking into it", "admin")
	incidentRepo.Updates = append(incidentRepo.Updates, update)

	service := NewIncidentService(incidentRepo)

	updates, err := service.GetIncidentUpdates(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updates) != 1 {
		t.Errorf("expected 1 update, got %d", len(updates))
	}
}

func TestIncidentService_AcknowledgeIncident(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident, _ := domain.NewIncident("Test", "Message", domain.SeverityMinor)
	incident.ID = 1
	incidentRepo.Incidents[1] = incident

	service := NewIncidentService(incidentRepo)

	result, err := service.AcknowledgeIncident(context.Background(), 1, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AcknowledgedAt == nil {
		t.Error("expected incident to be acknowledged")
	}
	if result.AcknowledgedBy != "admin" {
		t.Errorf("expected acknowledged by 'admin', got %q", result.AcknowledgedBy)
	}
}

func TestIncidentService_AcknowledgeIncident_NotFound(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	_, err := service.AcknowledgeIncident(context.Background(), 999, "admin")
	if err == nil {
		t.Error("expected error for non-existent incident")
	}
}

func TestIncidentService_UpdateIncidentStatus(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident, _ := domain.NewIncident("Test", "Message", domain.SeverityMinor)
	incident.ID = 1
	incidentRepo.Incidents[1] = incident

	service := NewIncidentService(incidentRepo)

	result, err := service.UpdateIncidentStatus(
		context.Background(),
		1,
		domain.IncidentIdentified,
		"Root cause identified",
		"admin",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != domain.IncidentIdentified {
		t.Errorf("expected status identified, got %q", result.Status)
	}

	// Check that update was added
	if len(incidentRepo.Updates) != 1 {
		t.Errorf("expected 1 update, got %d", len(incidentRepo.Updates))
	}
}

func TestIncidentService_UpdateIncidentStatus_NotFound(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	_, err := service.UpdateIncidentStatus(
		context.Background(),
		999,
		domain.IncidentIdentified,
		"Test",
		"admin",
	)
	if err == nil {
		t.Error("expected error for non-existent incident")
	}
}

func TestIncidentService_AddIncidentUpdate(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident, _ := domain.NewIncident("Test", "Message", domain.SeverityMinor)
	incident.ID = 1
	incidentRepo.Incidents[1] = incident

	service := NewIncidentService(incidentRepo)

	update, err := service.AddIncidentUpdate(
		context.Background(),
		1,
		"Still investigating",
		"admin",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if update == nil {
		t.Fatal("expected non-nil update")
	}
	if update.Message != "Still investigating" {
		t.Errorf("expected message 'Still investigating', got %q", update.Message)
	}
}

func TestIncidentService_AddIncidentUpdate_NotFound(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	_, err := service.AddIncidentUpdate(context.Background(), 999, "Test", "admin")
	if err == nil {
		t.Error("expected error for non-existent incident")
	}
}

func TestIncidentService_ResolveIncident(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident, _ := domain.NewIncident("Test", "Message", domain.SeverityMinor)
	incident.ID = 1
	incidentRepo.Incidents[1] = incident

	service := NewIncidentService(incidentRepo)

	result, err := service.ResolveIncident(context.Background(), 1, "Fixed the bug", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != domain.IncidentResolved {
		t.Errorf("expected status resolved, got %q", result.Status)
	}
	if result.Postmortem != "Fixed the bug" {
		t.Errorf("expected postmortem 'Fixed the bug', got %q", result.Postmortem)
	}
}

func TestIncidentService_ResolveIncident_NoPostmortem(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident, _ := domain.NewIncident("Test", "Message", domain.SeverityMinor)
	incident.ID = 1
	incidentRepo.Incidents[1] = incident

	service := NewIncidentService(incidentRepo)

	result, err := service.ResolveIncident(context.Background(), 1, "", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != domain.IncidentResolved {
		t.Errorf("expected status resolved, got %q", result.Status)
	}
}

func TestIncidentService_ResolveIncident_NotFound(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	_, err := service.ResolveIncident(context.Background(), 999, "Test", "admin")
	if err == nil {
		t.Error("expected error for non-existent incident")
	}
}

func TestIncidentService_DeleteIncident(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	incident, _ := domain.NewIncident("Test", "Message", domain.SeverityMinor)
	incident.ID = 1
	incidentRepo.Incidents[1] = incident

	service := NewIncidentService(incidentRepo)

	err := service.DeleteIncident(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := incidentRepo.Incidents[1]; exists {
		t.Error("expected incident to be deleted")
	}
}
