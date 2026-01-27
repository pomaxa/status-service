package application

import (
	"context"
	"encoding/json"
	"status-incident/internal/domain"
	"strings"
	"testing"
	"time"
)

func TestNotificationService_formatSlackPayload(t *testing.T) {
	s := &NotificationService{}

	tests := []struct {
		name           string
		payload        *domain.NotificationPayload
		expectedText   string
		expectedColor  string
	}{
		{
			name: "green status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusGreen,
				Source:    "heartbeat",
			},
			expectedText:  "API",
			expectedColor: "good",
		},
		{
			name: "yellow status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusYellow,
				Source:    "manual",
			},
			expectedText:  "API",
			expectedColor: "warning",
		},
		{
			name: "red status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusRed,
				Source:    "manual",
			},
			expectedText:  "API",
			expectedColor: "danger",
		},
		{
			name: "system and dependency",
			payload: &domain.NotificationPayload{
				Event:      domain.EventStatusChange,
				Timestamp:  time.Now(),
				System:     &domain.SystemInfo{ID: 1, Name: "API"},
				Dependency: &domain.DepInfo{ID: 1, Name: "PostgreSQL"},
				NewStatus:  domain.StatusRed,
				Source:     "heartbeat",
			},
			expectedText:  "API / PostgreSQL",
			expectedColor: "danger",
		},
		{
			name: "with message",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusRed,
				Message:   "Connection timeout",
				Source:    "heartbeat",
			},
			expectedText:  "API",
			expectedColor: "danger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := s.formatSlackPayload(tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			// Check text contains entity name
			text, ok := result["text"].(string)
			if !ok {
				t.Fatal("missing text field")
			}
			if !strings.Contains(text, tt.expectedText) {
				t.Errorf("text should contain %q, got %q", tt.expectedText, text)
			}

			// Check color
			attachments, ok := result["attachments"].([]interface{})
			if !ok || len(attachments) == 0 {
				t.Fatal("missing attachments")
			}
			attachment := attachments[0].(map[string]interface{})
			color := attachment["color"].(string)
			if color != tt.expectedColor {
				t.Errorf("expected color %q, got %q", tt.expectedColor, color)
			}
		})
	}
}

func TestNotificationService_formatDiscordPayload(t *testing.T) {
	s := &NotificationService{}

	tests := []struct {
		name          string
		payload       *domain.NotificationPayload
		expectedText  string
		expectedColor int
	}{
		{
			name: "green status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusGreen,
				Source:    "heartbeat",
			},
			expectedText:  "API",
			expectedColor: 5763719, // green
		},
		{
			name: "yellow status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusYellow,
				Source:    "manual",
			},
			expectedText:  "API",
			expectedColor: 16776960, // yellow
		},
		{
			name: "red status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusRed,
				Source:    "manual",
			},
			expectedText:  "API",
			expectedColor: 15548997, // red
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := s.formatDiscordPayload(tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			// Check content contains entity name
			content, ok := result["content"].(string)
			if !ok {
				t.Fatal("missing content field")
			}
			if !strings.Contains(content, tt.expectedText) {
				t.Errorf("content should contain %q, got %q", tt.expectedText, content)
			}

			// Check color
			embeds, ok := result["embeds"].([]interface{})
			if !ok || len(embeds) == 0 {
				t.Fatal("missing embeds")
			}
			embed := embeds[0].(map[string]interface{})
			color := int(embed["color"].(float64))
			if color != tt.expectedColor {
				t.Errorf("expected color %d, got %d", tt.expectedColor, color)
			}
		})
	}
}

func TestNotificationService_formatTelegramPayload(t *testing.T) {
	s := &NotificationService{}

	tests := []struct {
		name          string
		webhookURL    string
		payload       *domain.NotificationPayload
		expectedText  string
		expectedChatID string
	}{
		{
			name:       "simple URL",
			webhookURL: "https://api.telegram.org/bot123/sendMessage",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusRed,
				Source:    "heartbeat",
			},
			expectedText:   "API",
			expectedChatID: "",
		},
		{
			name:       "token:chatid format",
			webhookURL: "12345:@mychannel",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusRed,
				Source:    "heartbeat",
			},
			expectedText:   "API",
			expectedChatID: "@mychannel",
		},
		{
			name:       "with message",
			webhookURL: "https://api.telegram.org/bot123/sendMessage",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusRed,
				Message:   "Database unreachable",
				Source:    "heartbeat",
			},
			expectedText:   "Database unreachable",
			expectedChatID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := s.formatTelegramPayload(tt.webhookURL, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			// Check text contains expected content
			text, ok := result["text"].(string)
			if !ok {
				t.Fatal("missing text field")
			}
			if !strings.Contains(text, tt.expectedText) {
				t.Errorf("text should contain %q, got %q", tt.expectedText, text)
			}

			// Check parse_mode
			parseMode := result["parse_mode"].(string)
			if parseMode != "HTML" {
				t.Errorf("expected parse_mode HTML, got %q", parseMode)
			}

			// Check chat_id if expected
			if tt.expectedChatID != "" {
				chatID, ok := result["chat_id"].(string)
				if !ok {
					t.Fatal("missing chat_id field")
				}
				if chatID != tt.expectedChatID {
					t.Errorf("expected chat_id %q, got %q", tt.expectedChatID, chatID)
				}
			}
		})
	}
}

func TestNotificationService_formatTeamsPayload(t *testing.T) {
	s := &NotificationService{}

	tests := []struct {
		name           string
		payload        *domain.NotificationPayload
		expectedText   string
		expectedColor  string
	}{
		{
			name: "green status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusGreen,
				Source:    "heartbeat",
			},
			expectedText:  "API",
			expectedColor: "00FF00",
		},
		{
			name: "yellow status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusYellow,
				Source:    "manual",
			},
			expectedText:  "API",
			expectedColor: "FFFF00",
		},
		{
			name: "red status",
			payload: &domain.NotificationPayload{
				Event:     domain.EventStatusChange,
				Timestamp: time.Now(),
				System:    &domain.SystemInfo{ID: 1, Name: "API"},
				NewStatus: domain.StatusRed,
				Source:    "manual",
			},
			expectedText:  "API",
			expectedColor: "FF0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := s.formatTeamsPayload(tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			// Check @type
			atType := result["@type"].(string)
			if atType != "MessageCard" {
				t.Errorf("expected @type MessageCard, got %q", atType)
			}

			// Check themeColor
			color := result["themeColor"].(string)
			if color != tt.expectedColor {
				t.Errorf("expected themeColor %q, got %q", tt.expectedColor, color)
			}

			// Check summary contains entity name
			summary := result["summary"].(string)
			if !strings.Contains(summary, tt.expectedText) {
				t.Errorf("summary should contain %q, got %q", tt.expectedText, summary)
			}
		})
	}
}

func TestNotificationService_formatSlackSLABreach(t *testing.T) {
	s := &NotificationService{}

	payload := &domain.SLABreachPayload{
		Event:     domain.EventSLABreach,
		Timestamp: time.Now(),
		System:    &domain.SystemInfo{ID: 1, Name: "API"},
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.5,
		Period:      "monthly",
		Message:     "SLA target 99.90% not met (actual: 98.50%)",
	}

	body, err := s.formatSlackSLABreach(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Check text contains system name
	text := result["text"].(string)
	if !strings.Contains(text, "API") {
		t.Errorf("text should contain system name, got %q", text)
	}
	if !strings.Contains(text, "SLA Breach") {
		t.Errorf("text should contain 'SLA Breach', got %q", text)
	}

	// Check color is danger
	attachments := result["attachments"].([]interface{})
	attachment := attachments[0].(map[string]interface{})
	if attachment["color"] != "danger" {
		t.Errorf("expected color danger, got %q", attachment["color"])
	}
}

func TestNotificationService_formatDiscordSLABreach(t *testing.T) {
	s := &NotificationService{}

	payload := &domain.SLABreachPayload{
		Event:       domain.EventSLABreach,
		Timestamp:   time.Now(),
		System:      &domain.SystemInfo{ID: 1, Name: "API"},
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.5,
		Period:      "monthly",
		Message:     "SLA target 99.90% not met (actual: 98.50%)",
	}

	body, err := s.formatDiscordSLABreach(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Check content
	content := result["content"].(string)
	if !strings.Contains(content, "API") {
		t.Errorf("content should contain system name, got %q", content)
	}

	// Check color is red
	embeds := result["embeds"].([]interface{})
	embed := embeds[0].(map[string]interface{})
	if int(embed["color"].(float64)) != 15548997 {
		t.Errorf("expected red color 15548997, got %v", embed["color"])
	}
}

func TestNotificationService_formatTeamsSLABreach(t *testing.T) {
	s := &NotificationService{}

	payload := &domain.SLABreachPayload{
		Event:       domain.EventSLABreach,
		Timestamp:   time.Now(),
		System:      &domain.SystemInfo{ID: 1, Name: "API"},
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.5,
		Period:      "monthly",
		Message:     "SLA target 99.90% not met (actual: 98.50%)",
	}

	body, err := s.formatTeamsSLABreach(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Check type
	if result["@type"] != "MessageCard" {
		t.Errorf("expected @type MessageCard, got %v", result["@type"])
	}

	// Check themeColor is red
	if result["themeColor"] != "FF0000" {
		t.Errorf("expected themeColor FF0000, got %v", result["themeColor"])
	}

	// Check summary
	summary := result["summary"].(string)
	if !strings.Contains(summary, "API") {
		t.Errorf("summary should contain system name, got %q", summary)
	}
}

func TestNotificationService_formatTelegramSLABreach(t *testing.T) {
	s := &NotificationService{}

	tests := []struct {
		name           string
		webhookURL     string
		payload        *domain.SLABreachPayload
		expectedChatID string
	}{
		{
			name:       "simple URL",
			webhookURL: "https://api.telegram.org/bot123/sendMessage",
			payload: &domain.SLABreachPayload{
				Event:       domain.EventSLABreach,
				Timestamp:   time.Now(),
				System:      &domain.SystemInfo{ID: 1, Name: "API"},
				BreachType:  "uptime",
				SLATarget:   99.9,
				ActualValue: 98.5,
				Period:      "monthly",
				Message:     "SLA breach message",
			},
			expectedChatID: "",
		},
		{
			name:       "token:chatid format",
			webhookURL: "12345:@mychannel",
			payload: &domain.SLABreachPayload{
				Event:       domain.EventSLABreach,
				Timestamp:   time.Now(),
				System:      &domain.SystemInfo{ID: 1, Name: "API"},
				BreachType:  "uptime",
				SLATarget:   99.9,
				ActualValue: 98.5,
				Period:      "monthly",
				Message:     "SLA breach message",
			},
			expectedChatID: "@mychannel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := s.formatTelegramSLABreach(tt.webhookURL, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			// Check text contains system name
			text := result["text"].(string)
			if !strings.Contains(text, "API") {
				t.Errorf("text should contain system name, got %q", text)
			}

			// Check parse_mode
			parseMode := result["parse_mode"].(string)
			if parseMode != "HTML" {
				t.Errorf("expected parse_mode HTML, got %q", parseMode)
			}

			// Check chat_id if expected
			if tt.expectedChatID != "" {
				chatID, ok := result["chat_id"].(string)
				if !ok {
					t.Fatal("missing chat_id field")
				}
				if chatID != tt.expectedChatID {
					t.Errorf("expected chat_id %q, got %q", tt.expectedChatID, chatID)
				}
			}
		})
	}
}

func TestNotificationService_buildPayload(t *testing.T) {
	ctx := context.Background()

	// Set up mock repositories
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()

	// Create test system
	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(ctx, system)

	// Create test dependency
	dep := &domain.Dependency{
		SystemID: system.ID,
		Name:     "Test Dependency",
	}
	depRepo.Create(ctx, dep)

	service := NewNotificationService(nil, systemRepo, depRepo)

	tests := []struct {
		name           string
		statusLog      *domain.StatusLog
		expectedSystem bool
		expectedDep    bool
	}{
		{
			name: "system status log",
			statusLog: &domain.StatusLog{
				SystemID:  &system.ID,
				OldStatus: domain.StatusGreen,
				NewStatus: domain.StatusRed,
				Message:   "Test message",
				Source:    domain.SourceManual,
				CreatedAt: time.Now(),
			},
			expectedSystem: true,
			expectedDep:    false,
		},
		{
			name: "dependency status log",
			statusLog: &domain.StatusLog{
				DependencyID: &dep.ID,
				OldStatus:    domain.StatusGreen,
				NewStatus:    domain.StatusYellow,
				Message:      "Dependency degraded",
				Source:       domain.SourceHeartbeat,
				CreatedAt:    time.Now(),
			},
			expectedSystem: true, // Should also get system from dependency
			expectedDep:    true,
		},
		{
			name: "system and dependency status log",
			statusLog: &domain.StatusLog{
				SystemID:     &system.ID,
				DependencyID: &dep.ID,
				OldStatus:    domain.StatusYellow,
				NewStatus:    domain.StatusGreen,
				Message:      "Recovered",
				Source:       domain.SourceHeartbeat,
				CreatedAt:    time.Now(),
			},
			expectedSystem: true,
			expectedDep:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := service.buildPayload(ctx, tt.statusLog)

			if payload == nil {
				t.Fatal("expected non-nil payload")
			}

			if payload.Event != domain.EventStatusChange {
				t.Errorf("expected event %s, got %s", domain.EventStatusChange, payload.Event)
			}

			if payload.OldStatus != tt.statusLog.OldStatus {
				t.Errorf("expected OldStatus %s, got %s", tt.statusLog.OldStatus, payload.OldStatus)
			}

			if payload.NewStatus != tt.statusLog.NewStatus {
				t.Errorf("expected NewStatus %s, got %s", tt.statusLog.NewStatus, payload.NewStatus)
			}

			if tt.expectedSystem && payload.System == nil {
				t.Error("expected System to be set")
			}

			if tt.expectedDep && payload.Dependency == nil {
				t.Error("expected Dependency to be set")
			}
		})
	}
}

func TestNotificationService_SendTestNotification(t *testing.T) {
	ctx := context.Background()

	webhookRepo := NewMockWebhookRepository()
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()

	service := NewNotificationService(webhookRepo, systemRepo, depRepo)

	// Test with non-existent webhook
	err := service.SendTestNotification(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent webhook")
	}
	if !strings.Contains(err.Error(), "webhook not found") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Create webhook and test
	webhook := &domain.Webhook{
		Name:    "Test Webhook",
		URL:     "https://example.com/webhook",
		Type:    domain.WebhookTypeGeneric,
		Enabled: true,
		Events:  []domain.WebhookEvent{domain.EventStatusChange},
	}
	webhookRepo.Create(ctx, webhook)

	// Note: This will attempt to send (and fail) but should not return error
	// because sendNotification handles errors internally
	err = service.SendTestNotification(ctx, webhook.ID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNotificationService_NotifyStatusChange_NoWebhooks(t *testing.T) {
	ctx := context.Background()

	webhookRepo := NewMockWebhookRepository()
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()

	service := NewNotificationService(webhookRepo, systemRepo, depRepo)

	// Create system
	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(ctx, system)

	statusLog := &domain.StatusLog{
		SystemID:  &system.ID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "System down",
		Source:    domain.SourceManual,
		CreatedAt: time.Now(),
	}

	// Should not panic when no webhooks exist
	service.NotifyStatusChange(ctx, statusLog)
}

func TestNotificationService_NotifyStatusChange_WithWebhooks(t *testing.T) {
	ctx := context.Background()

	webhookRepo := NewMockWebhookRepository()
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()

	service := NewNotificationService(webhookRepo, systemRepo, depRepo)

	// Create system
	system, _ := domain.NewSystem("Test System", "", "", "")
	systemRepo.Create(ctx, system)

	// Create enabled webhook
	webhook := &domain.Webhook{
		Name:    "Test Webhook",
		URL:     "https://example.com/webhook",
		Type:    domain.WebhookTypeSlack,
		Enabled: true,
		Events:  []domain.WebhookEvent{domain.EventStatusChange},
	}
	webhookRepo.Create(ctx, webhook)

	statusLog := &domain.StatusLog{
		SystemID:  &system.ID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Message:   "System down",
		Source:    domain.SourceManual,
		CreatedAt: time.Now(),
	}

	// Should not panic, sends notification asynchronously
	service.NotifyStatusChange(ctx, statusLog)
	// Give time for goroutine to start
	time.Sleep(10 * time.Millisecond)
}

func TestNotificationService_NotifySLABreach_NoWebhooks(t *testing.T) {
	ctx := context.Background()

	webhookRepo := NewMockWebhookRepository()
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()

	service := NewNotificationService(webhookRepo, systemRepo, depRepo)

	breach := &domain.SLABreachEvent{
		SystemID:    1,
		SystemName:  "Test System",
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.5,
		Period:      "monthly",
		PeriodStart: time.Now().AddDate(0, -1, 0),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}

	// Should not panic when no webhooks exist
	service.NotifySLABreach(ctx, breach)
}

func TestNotificationService_NotifySLABreach_WithWebhooks(t *testing.T) {
	ctx := context.Background()

	webhookRepo := NewMockWebhookRepository()
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()

	service := NewNotificationService(webhookRepo, systemRepo, depRepo)

	// Create enabled webhook
	webhook := &domain.Webhook{
		Name:    "SLA Webhook",
		URL:     "https://example.com/webhook",
		Type:    domain.WebhookTypeSlack,
		Enabled: true,
		Events:  []domain.WebhookEvent{domain.EventSLABreach},
	}
	webhookRepo.Create(ctx, webhook)

	breach := &domain.SLABreachEvent{
		SystemID:    1,
		SystemName:  "Test System",
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.5,
		Period:      "monthly",
		PeriodStart: time.Now().AddDate(0, -1, 0),
		PeriodEnd:   time.Now(),
		DetectedAt:  time.Now(),
	}

	// Should not panic, sends notification asynchronously
	service.NotifySLABreach(ctx, breach)
	// Give time for goroutine to start
	time.Sleep(10 * time.Millisecond)
}

func TestNotificationService_DependencyOnly(t *testing.T) {
	s := &NotificationService{}

	payload := &domain.NotificationPayload{
		Event:      domain.EventStatusChange,
		Timestamp:  time.Now(),
		Dependency: &domain.DepInfo{ID: 1, Name: "PostgreSQL"},
		NewStatus:  domain.StatusRed,
		Source:     "heartbeat",
	}

	body, err := s.formatSlackPayload(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	text := result["text"].(string)
	if !strings.Contains(text, "PostgreSQL") {
		t.Errorf("text should contain dependency name, got %q", text)
	}
}
