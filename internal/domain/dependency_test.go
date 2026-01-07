package domain

import (
	"testing"
	"time"
)

func TestNewDependency_ValidInput(t *testing.T) {
	dep, err := NewDependency(1, "PostgreSQL", "Main database")
	if err != nil {
		t.Fatalf("NewDependency returned error: %v", err)
	}

	if dep.SystemID != 1 {
		t.Errorf("SystemID = %d, want 1", dep.SystemID)
	}
	if dep.Name != "PostgreSQL" {
		t.Errorf("Name = %q, want %q", dep.Name, "PostgreSQL")
	}
	if dep.Status != StatusGreen {
		t.Errorf("Status = %v, want %v", dep.Status, StatusGreen)
	}
	if dep.HeartbeatURL != "" {
		t.Error("HeartbeatURL should be empty by default")
	}
	if dep.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0", dep.ConsecutiveFailures)
	}
}

func TestNewDependency_EmptyName(t *testing.T) {
	_, err := NewDependency(1, "", "description")
	if err == nil {
		t.Error("NewDependency should return error for empty name")
	}
}

func TestNewDependency_ZeroSystemID(t *testing.T) {
	_, err := NewDependency(0, "Redis", "Cache")
	if err == nil {
		t.Error("NewDependency should return error for zero system ID")
	}
}

func TestDependency_SetHeartbeat(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	err := dep.SetHeartbeat("https://api.example.com/health", 60)
	if err != nil {
		t.Fatalf("SetHeartbeat returned error: %v", err)
	}

	if dep.HeartbeatURL != "https://api.example.com/health" {
		t.Errorf("HeartbeatURL = %q, want %q", dep.HeartbeatURL, "https://api.example.com/health")
	}
	if dep.HeartbeatInterval != 60 {
		t.Errorf("HeartbeatInterval = %d, want 60", dep.HeartbeatInterval)
	}
}

func TestDependency_SetHeartbeat_InvalidURL(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	err := dep.SetHeartbeat("not-a-url", 60)
	if err == nil {
		t.Error("SetHeartbeat should return error for invalid URL")
	}
}

func TestDependency_SetHeartbeat_InvalidInterval(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	err := dep.SetHeartbeat("https://api.example.com/health", 0)
	if err == nil {
		t.Error("SetHeartbeat should return error for zero interval")
	}

	err = dep.SetHeartbeat("https://api.example.com/health", -1)
	if err == nil {
		t.Error("SetHeartbeat should return error for negative interval")
	}
}

func TestDependency_ClearHeartbeat(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")
	dep.SetHeartbeat("https://api.example.com/health", 60)

	dep.ClearHeartbeat()

	if dep.HeartbeatURL != "" {
		t.Error("HeartbeatURL should be empty after clearing")
	}
	if dep.HeartbeatInterval != 0 {
		t.Error("HeartbeatInterval should be 0 after clearing")
	}
}

func TestDependency_HasHeartbeat(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	if dep.HasHeartbeat() {
		t.Error("Dependency without heartbeat URL should return false")
	}

	dep.SetHeartbeat("https://api.example.com/health", 60)
	if !dep.HasHeartbeat() {
		t.Error("Dependency with heartbeat URL should return true")
	}
}

func TestDependency_RecordCheckSuccess(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")
	dep.ConsecutiveFailures = 3
	dep.Status = StatusRed

	changed := dep.RecordCheckSuccess()

	if dep.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0", dep.ConsecutiveFailures)
	}
	if dep.Status != StatusGreen {
		t.Errorf("Status = %v, want %v", dep.Status, StatusGreen)
	}
	if !changed {
		t.Error("RecordCheckSuccess should return true when status changes")
	}
	if dep.LastCheck.IsZero() {
		t.Error("LastCheck should be set")
	}
}

func TestDependency_RecordCheckSuccess_AlreadyGreen(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	changed := dep.RecordCheckSuccess()

	if changed {
		t.Error("RecordCheckSuccess should return false when status doesn't change")
	}
}

func TestDependency_RecordCheckFailure_FirstFailure(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	changed := dep.RecordCheckFailure()

	if dep.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1", dep.ConsecutiveFailures)
	}
	if dep.Status != StatusYellow {
		t.Errorf("Status = %v, want %v (first failure = yellow)", dep.Status, StatusYellow)
	}
	if !changed {
		t.Error("RecordCheckFailure should return true when status changes")
	}
}

func TestDependency_RecordCheckFailure_ThreeFailures(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	dep.RecordCheckFailure() // 1st: green -> yellow
	dep.RecordCheckFailure() // 2nd: yellow -> yellow
	changed := dep.RecordCheckFailure() // 3rd: yellow -> red

	if dep.ConsecutiveFailures != 3 {
		t.Errorf("ConsecutiveFailures = %d, want 3", dep.ConsecutiveFailures)
	}
	if dep.Status != StatusRed {
		t.Errorf("Status = %v, want %v (3 failures = red)", dep.Status, StatusRed)
	}
	if !changed {
		t.Error("RecordCheckFailure should return true when status changes to red")
	}
}

func TestDependency_RecordCheckFailure_MoreThanThree(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	for i := 0; i < 5; i++ {
		dep.RecordCheckFailure()
	}

	if dep.Status != StatusRed {
		t.Errorf("Status = %v, want %v", dep.Status, StatusRed)
	}
	if dep.ConsecutiveFailures != 5 {
		t.Errorf("ConsecutiveFailures = %d, want 5", dep.ConsecutiveFailures)
	}
}

func TestDependency_NeedsCheck(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	// No heartbeat configured
	if dep.NeedsCheck() {
		t.Error("Dependency without heartbeat should not need check")
	}

	// With heartbeat, never checked
	dep.SetHeartbeat("https://api.example.com/health", 60)
	if !dep.NeedsCheck() {
		t.Error("Dependency with heartbeat that was never checked should need check")
	}

	// Just checked
	dep.LastCheck = time.Now()
	if dep.NeedsCheck() {
		t.Error("Dependency just checked should not need check")
	}

	// Checked more than interval ago
	dep.LastCheck = time.Now().Add(-61 * time.Second)
	if !dep.NeedsCheck() {
		t.Error("Dependency checked more than interval ago should need check")
	}
}
