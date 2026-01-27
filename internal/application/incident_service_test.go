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

// Boundary tests for systemIDs parameter
func TestIncidentService_CreateIncident_EmptySystemIDs(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	// Test with empty slice (not nil) - should NOT set system IDs
	incident, err := service.CreateIncident(
		context.Background(),
		"Outage",
		"Message",
		domain.SeverityMajor,
		[]int64{}, // Empty slice
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if incident == nil {
		t.Fatal("expected non-nil incident")
	}
	// Empty slice should result in no system IDs being set
	if len(incident.SystemIDs) != 0 {
		t.Errorf("expected 0 system IDs for empty slice, got %d", len(incident.SystemIDs))
	}
}

func TestIncidentService_CreateIncident_SingleSystemID(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	// Test with exactly one system ID (boundary)
	incident, err := service.CreateIncident(
		context.Background(),
		"Single System Issue",
		"Only one system affected",
		domain.SeverityMinor,
		[]int64{42}, // Exactly one
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if incident == nil {
		t.Fatal("expected non-nil incident")
	}
	if len(incident.SystemIDs) != 1 {
		t.Errorf("expected 1 system ID, got %d", len(incident.SystemIDs))
	}
	if incident.SystemIDs[0] != 42 {
		t.Errorf("expected system ID 42, got %d", incident.SystemIDs[0])
	}
}

func TestIncidentService_CreateIncident_SystemIDsContentValidation(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()
	service := NewIncidentService(incidentRepo)

	expectedIDs := []int64{1, 2, 3, 5, 8}
	incident, err := service.CreateIncident(
		context.Background(),
		"Multi-system Outage",
		"Multiple systems affected",
		domain.SeverityCritical,
		expectedIDs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(incident.SystemIDs) != len(expectedIDs) {
		t.Fatalf("expected %d system IDs, got %d", len(expectedIDs), len(incident.SystemIDs))
	}

	// Validate each system ID
	for i, expectedID := range expectedIDs {
		if incident.SystemIDs[i] != expectedID {
			t.Errorf("systemIDs[%d]: expected %d, got %d", i, expectedID, incident.SystemIDs[i])
		}
	}
}

// Boundary tests for GetRecentIncidents days parameter
func TestIncidentService_GetRecentIncidents_BoundaryConditions(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()

	var capturedDays int
	incidentRepo.GetRecentFunc = func(ctx context.Context, days int) ([]*domain.Incident, error) {
		capturedDays = days
		return []*domain.Incident{}, nil
	}

	service := NewIncidentService(incidentRepo)

	tests := []struct {
		name         string
		inputDays    int
		expectedDays int
	}{
		{"zero defaults to 7", 0, 7},
		{"negative defaults to 7", -5, 7},
		{"exactly 1 is valid", 1, 1},
		{"positive value used", 14, 14},
		{"large value used", 365, 365},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetRecentIncidents(context.Background(), tt.inputDays)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if capturedDays != tt.expectedDays {
				t.Errorf("expected days %d, got %d", tt.expectedDays, capturedDays)
			}
		})
	}
}

func TestIncidentService_GetAllIncidents_ContentValidation(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()

	incident1, _ := domain.NewIncident("First Incident", "First message", domain.SeverityMinor)
	incident1.ID = 1
	incident2, _ := domain.NewIncident("Second Incident", "Second message", domain.SeverityCritical)
	incident2.ID = 2

	incidentRepo.Incidents[1] = incident1
	incidentRepo.Incidents[2] = incident2

	service := NewIncidentService(incidentRepo)

	incidents, err := service.GetAllIncidents(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(incidents) != 2 {
		t.Fatalf("expected 2 incidents, got %d", len(incidents))
	}

	// Create a map for easier verification (order may not be guaranteed)
	incidentMap := make(map[int64]*domain.Incident)
	for _, inc := range incidents {
		incidentMap[inc.ID] = inc
	}

	// Validate first incident
	if inc, ok := incidentMap[1]; !ok {
		t.Error("incident ID 1 not found")
	} else {
		if inc.Title != "First Incident" {
			t.Errorf("incident 1: expected title 'First Incident', got %q", inc.Title)
		}
		if inc.Severity != domain.SeverityMinor {
			t.Errorf("incident 1: expected severity Minor, got %q", inc.Severity)
		}
	}

	// Validate second incident
	if inc, ok := incidentMap[2]; !ok {
		t.Error("incident ID 2 not found")
	} else {
		if inc.Title != "Second Incident" {
			t.Errorf("incident 2: expected title 'Second Incident', got %q", inc.Title)
		}
		if inc.Severity != domain.SeverityCritical {
			t.Errorf("incident 2: expected severity Critical, got %q", inc.Severity)
		}
	}
}

func TestIncidentService_GetIncidentUpdates_ContentValidation(t *testing.T) {
	incidentRepo := NewMockIncidentRepository()

	update1, _ := domain.NewIncidentUpdate(1, domain.IncidentInvestigating, "Started investigation", "admin")
	update2, _ := domain.NewIncidentUpdate(1, domain.IncidentIdentified, "Found root cause", "engineer")

	incidentRepo.Updates = append(incidentRepo.Updates, update1, update2)

	service := NewIncidentService(incidentRepo)

	updates, err := service.GetIncidentUpdates(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(updates))
	}

	// Validate first update
	if updates[0].Message != "Started investigation" {
		t.Errorf("update[0]: expected message 'Started investigation', got %q", updates[0].Message)
	}
	if updates[0].Status != domain.IncidentInvestigating {
		t.Errorf("update[0]: expected status Investigating, got %q", updates[0].Status)
	}
	if updates[0].CreatedBy != "admin" {
		t.Errorf("update[0]: expected CreatedBy 'admin', got %q", updates[0].CreatedBy)
	}

	// Validate second update
	if updates[1].Message != "Found root cause" {
		t.Errorf("update[1]: expected message 'Found root cause', got %q", updates[1].Message)
	}
	if updates[1].Status != domain.IncidentIdentified {
		t.Errorf("update[1]: expected status Identified, got %q", updates[1].Status)
	}
}
