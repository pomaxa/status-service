package domain

import (
	"testing"
	"time"
)

func TestNewSystem_ValidInput(t *testing.T) {
	system, err := NewSystem("Production API", "Main production API server", "https://api.example.com", "Platform Team")
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	if system.Name != "Production API" {
		t.Errorf("Name = %q, want %q", system.Name, "Production API")
	}
	if system.Description != "Main production API server" {
		t.Errorf("Description = %q, want %q", system.Description, "Main production API server")
	}
	if system.URL != "https://api.example.com" {
		t.Errorf("URL = %q, want %q", system.URL, "https://api.example.com")
	}
	if system.Owner != "Platform Team" {
		t.Errorf("Owner = %q, want %q", system.Owner, "Platform Team")
	}
	if system.Status != StatusGreen {
		t.Errorf("Status = %v, want %v (default green)", system.Status, StatusGreen)
	}
	if system.ID != 0 {
		t.Errorf("ID should be 0 before persistence, got %d", system.ID)
	}
	if system.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestNewSystem_EmptyName(t *testing.T) {
	_, err := NewSystem("", "description", "", "")
	if err == nil {
		t.Error("NewSystem should return error for empty name")
	}
	if err != ErrEmptyName {
		t.Errorf("Error = %v, want %v", err, ErrEmptyName)
	}
}

func TestNewSystem_WhitespaceName(t *testing.T) {
	_, err := NewSystem("   ", "description", "", "")
	if err == nil {
		t.Error("NewSystem should return error for whitespace-only name")
	}
}

func TestSystem_UpdateStatus(t *testing.T) {
	system, _ := NewSystem("Test System", "", "", "")

	err := system.UpdateStatus(StatusYellow)
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}

	if system.Status != StatusYellow {
		t.Errorf("Status = %v, want %v", system.Status, StatusYellow)
	}
	if system.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set after status change")
	}
}

func TestSystem_UpdateStatus_SameStatus(t *testing.T) {
	system, _ := NewSystem("Test System", "", "", "")
	originalUpdatedAt := system.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	err := system.UpdateStatus(StatusGreen) // same as default
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}

	// UpdatedAt should still be updated even if status is the same
	if !system.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated even for same status")
	}
}

func TestSystem_UpdateStatus_InvalidStatus(t *testing.T) {
	system, _ := NewSystem("Test System", "", "", "")

	err := system.UpdateStatus(Status("invalid"))
	if err == nil {
		t.Error("UpdateStatus should return error for invalid status")
	}
}

func TestSystem_Update(t *testing.T) {
	system, _ := NewSystem("Original Name", "Original Description", "", "")

	err := system.Update("New Name", "New Description", "https://new.example.com", "New Owner")
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if system.Name != "New Name" {
		t.Errorf("Name = %q, want %q", system.Name, "New Name")
	}
	if system.Description != "New Description" {
		t.Errorf("Description = %q, want %q", system.Description, "New Description")
	}
	if system.URL != "https://new.example.com" {
		t.Errorf("URL = %q, want %q", system.URL, "https://new.example.com")
	}
	if system.Owner != "New Owner" {
		t.Errorf("Owner = %q, want %q", system.Owner, "New Owner")
	}
}

func TestSystem_Update_EmptyName(t *testing.T) {
	system, _ := NewSystem("Original Name", "Original Description", "", "")

	err := system.Update("", "New Description", "", "")
	if err == nil {
		t.Error("Update should return error for empty name")
	}
}

func TestSystem_IsHealthy(t *testing.T) {
	system, _ := NewSystem("Test", "", "", "")

	if !system.IsHealthy() {
		t.Error("System with green status should be healthy")
	}

	system.UpdateStatus(StatusYellow)
	if system.IsHealthy() {
		t.Error("System with yellow status should not be healthy")
	}

	system.UpdateStatus(StatusRed)
	if system.IsHealthy() {
		t.Error("System with red status should not be healthy")
	}
}

func TestSystem_GetSLATarget(t *testing.T) {
	tests := []struct {
		name     string
		target   float64
		expected float64
	}{
		{"default when zero", 0, DefaultSLATarget},
		{"default when negative", -1, DefaultSLATarget},
		{"custom target", 99.5, 99.5},
		{"high target", 99.99, 99.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			system := &System{SLATarget: tt.target}
			result := system.GetSLATarget()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSystem_SetSLATarget(t *testing.T) {
	tests := []struct {
		name     string
		target   float64
		expected float64
	}{
		{"valid target", 99.5, 99.5},
		{"zero defaults", 0, DefaultSLATarget},
		{"negative defaults", -5, DefaultSLATarget},
		{"over 100 defaults", 101, DefaultSLATarget},
		{"exactly 100", 100, 100},
		{"low target", 90, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			system, _ := NewSystem("Test", "", "", "")
			system.SetSLATarget(tt.target)

			if system.SLATarget != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, system.SLATarget)
			}
		})
	}
}

func TestSystem_IsSLAMet(t *testing.T) {
	tests := []struct {
		name          string
		slaTarget     float64
		uptimePercent float64
		expected      bool
	}{
		{"met exactly", 99.9, 99.9, true},
		{"exceeded", 99.9, 99.95, true},
		{"not met", 99.9, 99.8, false},
		{"perfect uptime", 99.9, 100, true},
		{"zero uptime", 99.9, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			system := &System{SLATarget: tt.slaTarget}
			result := system.IsSLAMet(tt.uptimePercent)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
