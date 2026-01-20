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

	changed := dep.RecordCheckSuccess(150)

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
	if dep.LastLatency != 150 {
		t.Errorf("LastLatency = %d, want 150", dep.LastLatency)
	}
}

func TestDependency_RecordCheckSuccess_AlreadyGreen(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	changed := dep.RecordCheckSuccess(100)

	if changed {
		t.Error("RecordCheckSuccess should return false when status doesn't change")
	}
}

func TestDependency_RecordCheckFailure_FirstFailure(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	changed := dep.RecordCheckFailure(5000)

	if dep.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1", dep.ConsecutiveFailures)
	}
	if dep.Status != StatusYellow {
		t.Errorf("Status = %v, want %v (first failure = yellow)", dep.Status, StatusYellow)
	}
	if !changed {
		t.Error("RecordCheckFailure should return true when status changes")
	}
	if dep.LastLatency != 5000 {
		t.Errorf("LastLatency = %d, want 5000", dep.LastLatency)
	}
}

func TestDependency_RecordCheckFailure_ThreeFailures(t *testing.T) {
	dep, _ := NewDependency(1, "Redis", "Cache")

	dep.RecordCheckFailure(100) // 1st: green -> yellow
	dep.RecordCheckFailure(200) // 2nd: yellow -> yellow
	changed := dep.RecordCheckFailure(300) // 3rd: yellow -> red

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
		dep.RecordCheckFailure(100)
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

func TestDependency_SetHeartbeatConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  HeartbeatConfig
		wantErr bool
	}{
		{
			name: "valid GET config",
			config: HeartbeatConfig{
				URL:      "https://api.example.com/health",
				Interval: 60,
				Method:   "GET",
			},
			wantErr: false,
		},
		{
			name: "valid POST with body",
			config: HeartbeatConfig{
				URL:      "https://api.example.com/health",
				Interval: 30,
				Method:   "POST",
				Body:     `{"check": "deep"}`,
			},
			wantErr: false,
		},
		{
			name: "valid with headers",
			config: HeartbeatConfig{
				URL:      "https://api.example.com/health",
				Interval: 60,
				Headers:  map[string]string{"Authorization": "Bearer token"},
			},
			wantErr: false,
		},
		{
			name: "valid expect status",
			config: HeartbeatConfig{
				URL:          "https://api.example.com/health",
				Interval:     60,
				ExpectStatus: "200,201",
			},
			wantErr: false,
		},
		{
			name: "valid expect status wildcard",
			config: HeartbeatConfig{
				URL:          "https://api.example.com/health",
				Interval:     60,
				ExpectStatus: "2xx",
			},
			wantErr: false,
		},
		{
			name: "empty method defaults to GET",
			config: HeartbeatConfig{
				URL:      "https://api.example.com/health",
				Interval: 60,
				Method:   "",
			},
			wantErr: false,
		},
		{
			name: "invalid method",
			config: HeartbeatConfig{
				URL:      "https://api.example.com/health",
				Interval: 60,
				Method:   "INVALID",
			},
			wantErr: true,
		},
		{
			name: "invalid expect status format",
			config: HeartbeatConfig{
				URL:          "https://api.example.com/health",
				Interval:     60,
				ExpectStatus: "abc",
			},
			wantErr: true,
		},
		{
			name: "ftp scheme not allowed",
			config: HeartbeatConfig{
				URL:      "ftp://files.example.com/health",
				Interval: 60,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep, _ := NewDependency(1, "Test", "")
			err := dep.SetHeartbeatConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if dep.HeartbeatURL != tt.config.URL {
				t.Errorf("expected URL %q, got %q", tt.config.URL, dep.HeartbeatURL)
			}
		})
	}
}

func TestIsValidExpectStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"single code", "200", true},
		{"multiple codes", "200,201,204", true},
		{"wildcard 2xx", "2xx", true},
		{"wildcard 3xx", "3xx", true},
		{"wildcard 4xx", "4xx", true},
		{"wildcard 5xx", "5xx", true},
		{"mixed", "200,3xx", true},
		{"with spaces", " 200 , 201 ", true},
		{"invalid wildcard 0xx", "0xx", false},
		{"invalid wildcard 6xx", "6xx", false},
		{"letters", "abc", false},
		{"empty part", "200,,201", false},
		{"only comma", ",", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidExpectStatus(tt.input)
			if result != tt.expected {
				t.Errorf("isValidExpectStatus(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDependency_GetHeartbeatConfig(t *testing.T) {
	dep, _ := NewDependency(1, "Test", "")
	dep.SetHeartbeatConfig(HeartbeatConfig{
		URL:          "https://api.example.com/health",
		Interval:     60,
		Method:       "POST",
		Headers:      map[string]string{"Authorization": "Bearer token"},
		Body:         `{"test": true}`,
		ExpectStatus: "2xx",
		ExpectBody:   `"status":\s*"ok"`,
	})

	config := dep.GetHeartbeatConfig()

	if config.URL != "https://api.example.com/health" {
		t.Errorf("expected URL, got %q", config.URL)
	}
	if config.Interval != 60 {
		t.Errorf("expected interval 60, got %d", config.Interval)
	}
	if config.Method != "POST" {
		t.Errorf("expected method POST, got %q", config.Method)
	}
	if config.Headers["Authorization"] != "Bearer token" {
		t.Error("expected Authorization header")
	}
	if config.Body != `{"test": true}` {
		t.Errorf("expected body, got %q", config.Body)
	}
	if config.ExpectStatus != "2xx" {
		t.Errorf("expected status 2xx, got %q", config.ExpectStatus)
	}
	if config.ExpectBody != `"status":\s*"ok"` {
		t.Errorf("expected body pattern, got %q", config.ExpectBody)
	}
}

func TestDependency_GetHeartbeatConfig_DefaultMethod(t *testing.T) {
	dep, _ := NewDependency(1, "Test", "")
	dep.HeartbeatURL = "https://example.com"
	dep.HeartbeatInterval = 30
	// HeartbeatMethod is empty

	config := dep.GetHeartbeatConfig()

	if config.Method != "GET" {
		t.Errorf("expected default method GET, got %q", config.Method)
	}
}

func TestDependency_UpdateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  Status
		wantErr bool
	}{
		{"green status", StatusGreen, false},
		{"yellow status", StatusYellow, false},
		{"red status", StatusRed, false},
		{"invalid status", Status("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep, _ := NewDependency(1, "Test", "")
			dep.ConsecutiveFailures = 5 // should be reset

			err := dep.UpdateStatus(tt.status)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if dep.Status != tt.status {
				t.Errorf("expected status %q, got %q", tt.status, dep.Status)
			}
			if dep.ConsecutiveFailures != 0 {
				t.Errorf("ConsecutiveFailures should be reset, got %d", dep.ConsecutiveFailures)
			}
		})
	}
}

func TestDependency_Update(t *testing.T) {
	tests := []struct {
		name        string
		newName     string
		newDesc     string
		wantErr     bool
	}{
		{"valid update", "New Name", "New Description", false},
		{"empty name", "", "Description", true},
		{"whitespace name", "   ", "Description", true},
		{"empty description ok", "Name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep, _ := NewDependency(1, "Original", "Original Desc")
			err := dep.Update(tt.newName, tt.newDesc)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if dep.Name != tt.newName {
				t.Errorf("expected name %q, got %q", tt.newName, dep.Name)
			}
		})
	}
}
