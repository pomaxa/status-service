package domain

import (
	"testing"
	"time"
)

func TestNewIncident(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		message     string
		severity    IncidentSeverity
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid incident minor",
			title:    "Database Outage",
			message:  "Database is not responding",
			severity: SeverityMinor,
			wantErr:  false,
		},
		{
			name:     "valid incident major",
			title:    "API Degradation",
			message:  "High latency on API calls",
			severity: SeverityMajor,
			wantErr:  false,
		},
		{
			name:     "valid incident critical",
			title:    "Complete Outage",
			message:  "All services down",
			severity: SeverityCritical,
			wantErr:  false,
		},
		{
			name:        "empty title",
			title:       "",
			message:     "Some message",
			severity:    SeverityMinor,
			wantErr:     true,
			errContains: "title is required",
		},
		{
			name:        "empty message",
			title:       "Some Title",
			message:     "",
			severity:    SeverityMinor,
			wantErr:     true,
			errContains: "message is required",
		},
		{
			name:     "invalid severity defaults to minor",
			title:    "Test Incident",
			message:  "Test message",
			severity: IncidentSeverity("invalid"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incident, err := NewIncident(tt.title, tt.message, tt.severity)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if incident.Title != tt.title {
				t.Errorf("expected title %q, got %q", tt.title, incident.Title)
			}
			if incident.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, incident.Message)
			}
			if incident.Status != IncidentInvestigating {
				t.Errorf("expected status investigating, got %q", incident.Status)
			}
			if incident.CreatedAt.IsZero() {
				t.Error("expected CreatedAt to be set")
			}

			// Invalid severity should default to minor
			if tt.severity == IncidentSeverity("invalid") && incident.Severity != SeverityMinor {
				t.Errorf("expected severity to default to minor, got %q", incident.Severity)
			}
		})
	}
}

func TestIncident_SetSystemIDs(t *testing.T) {
	incident, _ := NewIncident("Test", "Test message", SeverityMinor)
	oldUpdatedAt := incident.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	incident.SetSystemIDs([]int64{1, 2, 3})

	if len(incident.SystemIDs) != 3 {
		t.Errorf("expected 3 system IDs, got %d", len(incident.SystemIDs))
	}
	if !incident.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestIncident_Acknowledge(t *testing.T) {
	incident, _ := NewIncident("Test", "Test message", SeverityMinor)

	// First acknowledgment should succeed
	err := incident.Acknowledge("admin@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if incident.AcknowledgedAt == nil {
		t.Error("expected AcknowledgedAt to be set")
	}
	if incident.AcknowledgedBy != "admin@example.com" {
		t.Errorf("expected AcknowledgedBy admin@example.com, got %q", incident.AcknowledgedBy)
	}

	// Second acknowledgment should fail
	err = incident.Acknowledge("other@example.com")
	if err == nil {
		t.Error("expected error for double acknowledgment")
	}
}

func TestIncident_UpdateStatus(t *testing.T) {
	tests := []struct {
		name      string
		fromStatus IncidentStatus
		toStatus   IncidentStatus
		wantErr   bool
	}{
		{"investigating to identified", IncidentInvestigating, IncidentIdentified, false},
		{"identified to monitoring", IncidentIdentified, IncidentMonitoring, false},
		{"monitoring to resolved", IncidentMonitoring, IncidentResolved, false},
		{"invalid status", IncidentInvestigating, IncidentStatus("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incident, _ := NewIncident("Test", "Test message", SeverityMinor)
			incident.Status = tt.fromStatus

			err := incident.UpdateStatus(tt.toStatus)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if incident.Status != tt.toStatus {
				t.Errorf("expected status %q, got %q", tt.toStatus, incident.Status)
			}

			if tt.toStatus == IncidentResolved && incident.ResolvedAt == nil {
				t.Error("expected ResolvedAt to be set when resolved")
			}
		})
	}
}

func TestIncident_UpdateStatus_ResolvedIncident(t *testing.T) {
	incident, _ := NewIncident("Test", "Test message", SeverityMinor)
	incident.Resolve("postmortem")

	err := incident.UpdateStatus(IncidentInvestigating)
	if err == nil {
		t.Error("expected error when updating resolved incident")
	}
}

func TestIncident_Resolve(t *testing.T) {
	incident, _ := NewIncident("Test", "Test message", SeverityMinor)

	err := incident.Resolve("This is the postmortem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if incident.Status != IncidentResolved {
		t.Errorf("expected status resolved, got %q", incident.Status)
	}
	if incident.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set")
	}
	if incident.Postmortem != "This is the postmortem" {
		t.Errorf("expected postmortem to be set, got %q", incident.Postmortem)
	}

	// Cannot resolve again
	err = incident.Resolve("Another postmortem")
	if err == nil {
		t.Error("expected error when resolving already resolved incident")
	}
}

func TestIncident_IsResolved(t *testing.T) {
	incident, _ := NewIncident("Test", "Test message", SeverityMinor)

	if incident.IsResolved() {
		t.Error("expected IsResolved=false for new incident")
	}

	incident.Resolve("")
	if !incident.IsResolved() {
		t.Error("expected IsResolved=true for resolved incident")
	}
}

func TestIncident_IsActive(t *testing.T) {
	incident, _ := NewIncident("Test", "Test message", SeverityMinor)

	if !incident.IsActive() {
		t.Error("expected IsActive=true for new incident")
	}

	incident.Resolve("")
	if incident.IsActive() {
		t.Error("expected IsActive=false for resolved incident")
	}
}

func TestIncident_Duration(t *testing.T) {
	incident, _ := NewIncident("Test", "Test message", SeverityMinor)

	// Active incident - duration should be positive
	time.Sleep(10 * time.Millisecond)
	duration := incident.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", duration)
	}

	// Resolved incident - duration should be fixed
	incident.Resolve("")
	duration1 := incident.Duration()
	time.Sleep(10 * time.Millisecond)
	duration2 := incident.Duration()

	if duration1 != duration2 {
		t.Errorf("expected fixed duration for resolved incident, got %v and %v", duration1, duration2)
	}
}

func TestIncident_AffectsSystem(t *testing.T) {
	tests := []struct {
		name      string
		systemIDs []int64
		checkID   int64
		expected  bool
	}{
		{"nil affects all", nil, 1, true},
		{"empty affects all", []int64{}, 1, true},
		{"specific system match", []int64{1, 2, 3}, 2, true},
		{"specific system no match", []int64{1, 2, 3}, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incident, _ := NewIncident("Test", "Test message", SeverityMinor)
			incident.SystemIDs = tt.systemIDs

			result := incident.AffectsSystem(tt.checkID)
			if result != tt.expected {
				t.Errorf("AffectsSystem(%d) = %v, want %v", tt.checkID, result, tt.expected)
			}
		})
	}
}

func TestNewIncidentUpdate(t *testing.T) {
	tests := []struct {
		name        string
		incidentID  int64
		status      IncidentStatus
		message     string
		createdBy   string
		wantErr     bool
	}{
		{
			name:       "valid update",
			incidentID: 1,
			status:     IncidentIdentified,
			message:    "Root cause identified",
			createdBy:  "admin",
			wantErr:    false,
		},
		{
			name:       "empty message",
			incidentID: 1,
			status:     IncidentIdentified,
			message:    "",
			createdBy:  "admin",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update, err := NewIncidentUpdate(tt.incidentID, tt.status, tt.message, tt.createdBy)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if update.IncidentID != tt.incidentID {
				t.Errorf("expected IncidentID %d, got %d", tt.incidentID, update.IncidentID)
			}
			if update.Status != tt.status {
				t.Errorf("expected status %q, got %q", tt.status, update.Status)
			}
			if update.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, update.Message)
			}
		})
	}
}

func TestSeverityEmoji(t *testing.T) {
	tests := []struct {
		severity IncidentSeverity
		expected string
	}{
		{SeverityMinor, "⚠️"},
		{SeverityMajor, "🔶"},
		{SeverityCritical, "🔴"},
		{IncidentSeverity("unknown"), "❓"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			result := SeverityEmoji(tt.severity)
			if result != tt.expected {
				t.Errorf("SeverityEmoji(%q) = %q, want %q", tt.severity, result, tt.expected)
			}
		})
	}
}

func TestIncidentStatusEmoji(t *testing.T) {
	tests := []struct {
		status   IncidentStatus
		expected string
	}{
		{IncidentInvestigating, "🔍"},
		{IncidentIdentified, "🎯"},
		{IncidentMonitoring, "👀"},
		{IncidentResolved, "✅"},
		{IncidentStatus("unknown"), "❓"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := IncidentStatusEmoji(tt.status)
			if result != tt.expected {
				t.Errorf("IncidentStatusEmoji(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}
