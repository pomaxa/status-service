package domain

import (
	"testing"
)

func TestNewWebhook(t *testing.T) {
	tests := []struct {
		name        string
		webhookName string
		url         string
		webhookType WebhookType
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid generic webhook",
			webhookName: "My Webhook",
			url:         "https://example.com/webhook",
			webhookType: WebhookTypeGeneric,
			wantErr:     false,
		},
		{
			name:        "valid slack webhook",
			webhookName: "Slack Alerts",
			url:         "https://hooks.slack.com/services/xxx",
			webhookType: WebhookTypeSlack,
			wantErr:     false,
		},
		{
			name:        "valid telegram webhook",
			webhookName: "Telegram Bot",
			url:         "https://api.telegram.org/bot123/sendMessage",
			webhookType: WebhookTypeTelegram,
			wantErr:     false,
		},
		{
			name:        "valid discord webhook",
			webhookName: "Discord Channel",
			url:         "https://discord.com/api/webhooks/xxx",
			webhookType: WebhookTypeDiscord,
			wantErr:     false,
		},
		{
			name:        "valid teams webhook",
			webhookName: "Teams Channel",
			url:         "https://outlook.office.com/webhook/xxx",
			webhookType: WebhookTypeTeams,
			wantErr:     false,
		},
		{
			name:        "empty name",
			webhookName: "",
			url:         "https://example.com/webhook",
			webhookType: WebhookTypeGeneric,
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "whitespace name",
			webhookName: "   ",
			url:         "https://example.com/webhook",
			webhookType: WebhookTypeGeneric,
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "empty URL",
			webhookName: "My Webhook",
			url:         "",
			webhookType: WebhookTypeGeneric,
			wantErr:     true,
			errContains: "URL is required",
		},
		{
			name:        "invalid URL",
			webhookName: "My Webhook",
			url:         "not-a-url",
			webhookType: WebhookTypeGeneric,
			wantErr:     true,
			errContains: "invalid webhook URL",
		},
		{
			name:        "invalid type defaults to generic",
			webhookName: "My Webhook",
			url:         "https://example.com/webhook",
			webhookType: WebhookType("invalid"),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook, err := NewWebhook(tt.webhookName, tt.url, tt.webhookType)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if webhook.Name != tt.webhookName {
				t.Errorf("expected name %q, got %q", tt.webhookName, webhook.Name)
			}
			if webhook.URL != tt.url {
				t.Errorf("expected URL %q, got %q", tt.url, webhook.URL)
			}
			if !webhook.Enabled {
				t.Error("expected Enabled=true by default")
			}
			if len(webhook.Events) != 1 || webhook.Events[0] != EventStatusChange {
				t.Error("expected default event to be status_change")
			}
			if webhook.CreatedAt.IsZero() {
				t.Error("expected CreatedAt to be set")
			}

			// Invalid type should default to generic
			if tt.webhookType == WebhookType("invalid") && webhook.Type != WebhookTypeGeneric {
				t.Errorf("expected type to default to generic, got %q", webhook.Type)
			}
		})
	}
}

func TestWebhook_Update(t *testing.T) {
	webhook, _ := NewWebhook("Original", "https://example.com/original", WebhookTypeGeneric)

	tests := []struct {
		name        string
		newName     string
		newURL      string
		newType     WebhookType
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid update",
			newName: "Updated",
			newURL:  "https://example.com/updated",
			newType: WebhookTypeSlack,
			wantErr: false,
		},
		{
			name:        "empty name",
			newName:     "",
			newURL:      "https://example.com/updated",
			newType:     WebhookTypeSlack,
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "empty URL",
			newName:     "Updated",
			newURL:      "",
			newType:     WebhookTypeSlack,
			wantErr:     true,
			errContains: "URL is required",
		},
		{
			name:        "invalid URL",
			newName:     "Updated",
			newURL:      "invalid",
			newType:     WebhookTypeSlack,
			wantErr:     true,
			errContains: "invalid webhook URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := webhook.Update(tt.newName, tt.newURL, tt.newType)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if webhook.Name != tt.newName {
				t.Errorf("expected name %q, got %q", tt.newName, webhook.Name)
			}
			if webhook.URL != tt.newURL {
				t.Errorf("expected URL %q, got %q", tt.newURL, webhook.URL)
			}
		})
	}
}

func TestWebhook_SetEvents(t *testing.T) {
	webhook, _ := NewWebhook("Test", "https://example.com/webhook", WebhookTypeGeneric)
	oldUpdatedAt := webhook.UpdatedAt

	events := []WebhookEvent{EventStatusChange, EventIncidentStart, EventIncidentEnd}
	webhook.SetEvents(events)

	if len(webhook.Events) != 3 {
		t.Errorf("expected 3 events, got %d", len(webhook.Events))
	}
	if webhook.UpdatedAt == oldUpdatedAt {
		t.Error("expected UpdatedAt to change")
	}
}

func TestWebhook_SetSystemIDs(t *testing.T) {
	webhook, _ := NewWebhook("Test", "https://example.com/webhook", WebhookTypeGeneric)

	// Set specific systems
	webhook.SetSystemIDs([]int64{1, 2, 3})
	if len(webhook.SystemIDs) != 3 {
		t.Errorf("expected 3 system IDs, got %d", len(webhook.SystemIDs))
	}

	// Set to nil (all systems)
	webhook.SetSystemIDs(nil)
	if webhook.SystemIDs != nil {
		t.Error("expected SystemIDs to be nil")
	}
}

func TestWebhook_EnableDisable(t *testing.T) {
	webhook, _ := NewWebhook("Test", "https://example.com/webhook", WebhookTypeGeneric)

	if !webhook.Enabled {
		t.Error("expected Enabled=true by default")
	}

	webhook.Disable()
	if webhook.Enabled {
		t.Error("expected Enabled=false after Disable()")
	}

	webhook.Enable()
	if !webhook.Enabled {
		t.Error("expected Enabled=true after Enable()")
	}
}

func TestWebhook_ShouldTrigger(t *testing.T) {
	tests := []struct {
		name      string
		webhook   *Webhook
		event     WebhookEvent
		systemID  int64
		expected  bool
	}{
		{
			name: "disabled webhook",
			webhook: func() *Webhook {
				w, _ := NewWebhook("Test", "https://example.com", WebhookTypeGeneric)
				w.Disable()
				return w
			}(),
			event:    EventStatusChange,
			systemID: 1,
			expected: false,
		},
		{
			name: "event not subscribed",
			webhook: func() *Webhook {
				w, _ := NewWebhook("Test", "https://example.com", WebhookTypeGeneric)
				w.SetEvents([]WebhookEvent{EventIncidentStart})
				return w
			}(),
			event:    EventStatusChange,
			systemID: 1,
			expected: false,
		},
		{
			name: "event subscribed, all systems",
			webhook: func() *Webhook {
				w, _ := NewWebhook("Test", "https://example.com", WebhookTypeGeneric)
				return w
			}(),
			event:    EventStatusChange,
			systemID: 1,
			expected: true,
		},
		{
			name: "event subscribed, specific system match",
			webhook: func() *Webhook {
				w, _ := NewWebhook("Test", "https://example.com", WebhookTypeGeneric)
				w.SetSystemIDs([]int64{1, 2, 3})
				return w
			}(),
			event:    EventStatusChange,
			systemID: 2,
			expected: true,
		},
		{
			name: "event subscribed, specific system no match",
			webhook: func() *Webhook {
				w, _ := NewWebhook("Test", "https://example.com", WebhookTypeGeneric)
				w.SetSystemIDs([]int64{1, 2, 3})
				return w
			}(),
			event:    EventStatusChange,
			systemID: 5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.webhook.ShouldTrigger(tt.event, tt.systemID)
			if result != tt.expected {
				t.Errorf("ShouldTrigger() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWebhook_EventsJSON(t *testing.T) {
	webhook, _ := NewWebhook("Test", "https://example.com", WebhookTypeGeneric)
	webhook.SetEvents([]WebhookEvent{EventStatusChange, EventIncidentStart})

	json := webhook.EventsJSON()
	if json != `["status_change","incident_start"]` {
		t.Errorf("unexpected JSON: %s", json)
	}
}

func TestWebhook_SystemIDsJSON(t *testing.T) {
	webhook, _ := NewWebhook("Test", "https://example.com", WebhookTypeGeneric)

	// Test nil system IDs
	json := webhook.SystemIDsJSON()
	if json != nil {
		t.Error("expected nil for empty system IDs")
	}

	// Test with system IDs
	webhook.SetSystemIDs([]int64{1, 2, 3})
	json = webhook.SystemIDsJSON()
	if json == nil || *json != "[1,2,3]" {
		t.Errorf("unexpected JSON: %v", json)
	}
}

func TestParseEventsJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WebhookEvent
	}{
		{
			name:     "valid events",
			input:    `["status_change","incident_start"]`,
			expected: []WebhookEvent{EventStatusChange, EventIncidentStart},
		},
		{
			name:     "invalid JSON",
			input:    "invalid",
			expected: []WebhookEvent{EventStatusChange},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []WebhookEvent{EventStatusChange},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseEventsJSON(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d events, got %d", len(tt.expected), len(result))
			}
		})
	}
}

func TestParseSystemIDsJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected []int64
	}{
		{
			name:     "valid IDs",
			input:    strPtr("[1,2,3]"),
			expected: []int64{1, 2, 3},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty string",
			input:    strPtr(""),
			expected: nil,
		},
		{
			name:     "invalid JSON",
			input:    strPtr("invalid"),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSystemIDsJSON(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d IDs, got %d", len(tt.expected), len(result))
			}
		})
	}
}

func TestStatusEmoji(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusGreen, "🟢"},
		{StatusYellow, "🟡"},
		{StatusRed, "🔴"},
		{Status("unknown"), "⚪"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := StatusEmoji(tt.status)
			if result != tt.expected {
				t.Errorf("StatusEmoji(%s) = %s, want %s", tt.status, result, tt.expected)
			}
		})
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusGreen, "Operational"},
		{StatusYellow, "Degraded"},
		{StatusRed, "Outage"},
		{Status("unknown"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := StatusText(tt.status)
			if result != tt.expected {
				t.Errorf("StatusText(%s) = %s, want %s", tt.status, result, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
