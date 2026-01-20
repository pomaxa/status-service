package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"status-incident/internal/domain"
)

// NotificationService handles sending notifications via webhooks
type NotificationService struct {
	webhookRepo domain.WebhookRepository
	systemRepo  domain.SystemRepository
	depRepo     domain.DependencyRepository
	httpClient  *http.Client
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(
	webhookRepo domain.WebhookRepository,
	systemRepo domain.SystemRepository,
	depRepo domain.DependencyRepository,
) *NotificationService {
	return &NotificationService{
		webhookRepo: webhookRepo,
		systemRepo:  systemRepo,
		depRepo:     depRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NotifyStatusChange sends notifications for a status change
func (s *NotificationService) NotifyStatusChange(ctx context.Context, statusLog *domain.StatusLog) {
	webhooks, err := s.webhookRepo.GetEnabled(ctx)
	if err != nil {
		logError("Failed to get webhooks: %v", err)
		return
	}

	if len(webhooks) == 0 {
		return
	}

	// Build payload
	payload := s.buildPayload(ctx, statusLog)
	if payload == nil {
		return
	}

	// Determine system ID for filtering
	var systemID int64
	if statusLog.SystemID != nil {
		systemID = *statusLog.SystemID
	} else if statusLog.DependencyID != nil {
		// Get system ID from dependency
		dep, err := s.depRepo.GetByID(ctx, *statusLog.DependencyID)
		if err == nil && dep != nil {
			systemID = dep.SystemID
		}
	}

	// Send to matching webhooks
	for _, webhook := range webhooks {
		if webhook.ShouldTrigger(domain.EventStatusChange, systemID) {
			go s.sendNotification(webhook, payload)
		}
	}
}

func (s *NotificationService) buildPayload(ctx context.Context, statusLog *domain.StatusLog) *domain.NotificationPayload {
	payload := &domain.NotificationPayload{
		Event:     domain.EventStatusChange,
		Timestamp: statusLog.CreatedAt,
		OldStatus: statusLog.OldStatus,
		NewStatus: statusLog.NewStatus,
		Message:   statusLog.Message,
		Source:    string(statusLog.Source),
	}

	// Add system info
	if statusLog.SystemID != nil {
		system, err := s.systemRepo.GetByID(ctx, *statusLog.SystemID)
		if err == nil && system != nil {
			payload.System = &domain.SystemInfo{
				ID:   system.ID,
				Name: system.Name,
			}
		}
	}

	// Add dependency info
	if statusLog.DependencyID != nil {
		dep, err := s.depRepo.GetByID(ctx, *statusLog.DependencyID)
		if err == nil && dep != nil {
			payload.Dependency = &domain.DepInfo{
				ID:   dep.ID,
				Name: dep.Name,
			}
			// Also get system if not already set
			if payload.System == nil {
				system, err := s.systemRepo.GetByID(ctx, dep.SystemID)
				if err == nil && system != nil {
					payload.System = &domain.SystemInfo{
						ID:   system.ID,
						Name: system.Name,
					}
				}
			}
		}
	}

	return payload
}

func (s *NotificationService) sendNotification(webhook *domain.Webhook, payload *domain.NotificationPayload) {
	var body []byte
	var err error

	switch webhook.Type {
	case domain.WebhookTypeSlack:
		body, err = s.formatSlackPayload(payload)
	case domain.WebhookTypeTelegram:
		body, err = s.formatTelegramPayload(webhook.URL, payload)
	case domain.WebhookTypeDiscord:
		body, err = s.formatDiscordPayload(payload)
	default:
		body, err = json.Marshal(payload)
	}

	if err != nil {
		logError("Failed to format payload for webhook %s: %v", webhook.Name, err)
		return
	}

	url := webhook.URL
	// For Telegram, we need to modify the URL
	if webhook.Type == domain.WebhookTypeTelegram {
		// URL format: https://api.telegram.org/bot{token}/sendMessage
		// or just the token, and we construct the URL
		if !strings.Contains(url, "api.telegram.org") {
			// Assume it's token:chatid format
			parts := strings.SplitN(url, ":", 2)
			if len(parts) == 2 {
				url = fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", parts[0])
			}
		}
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		logError("Failed to create request for webhook %s: %v", webhook.Name, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "StatusIncident-Webhook/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logError("Failed to send webhook %s: %v", webhook.Name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logError("Webhook %s returned status %d", webhook.Name, resp.StatusCode)
	}
}

func (s *NotificationService) formatSlackPayload(payload *domain.NotificationPayload) ([]byte, error) {
	emoji := domain.StatusEmoji(payload.NewStatus)
	statusText := domain.StatusText(payload.NewStatus)

	// Build entity name
	entityName := ""
	if payload.System != nil {
		entityName = payload.System.Name
	}
	if payload.Dependency != nil {
		if entityName != "" {
			entityName += " / " + payload.Dependency.Name
		} else {
			entityName = payload.Dependency.Name
		}
	}

	// Slack color
	color := "good"
	switch payload.NewStatus {
	case domain.StatusYellow:
		color = "warning"
	case domain.StatusRed:
		color = "danger"
	}

	slackPayload := map[string]interface{}{
		"text": fmt.Sprintf("%s *%s* is now *%s*", emoji, entityName, statusText),
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"fields": []map[string]interface{}{
					{"title": "Status", "value": statusText, "short": true},
					{"title": "Source", "value": payload.Source, "short": true},
				},
			},
		},
	}

	if payload.Message != "" {
		slackPayload["attachments"].([]map[string]interface{})[0]["fields"] = append(
			slackPayload["attachments"].([]map[string]interface{})[0]["fields"].([]map[string]interface{}),
			map[string]interface{}{"title": "Message", "value": payload.Message, "short": false},
		)
	}

	return json.Marshal(slackPayload)
}

func (s *NotificationService) formatTelegramPayload(webhookURL string, payload *domain.NotificationPayload) ([]byte, error) {
	emoji := domain.StatusEmoji(payload.NewStatus)
	statusText := domain.StatusText(payload.NewStatus)

	// Build entity name
	entityName := ""
	if payload.System != nil {
		entityName = payload.System.Name
	}
	if payload.Dependency != nil {
		if entityName != "" {
			entityName += " / " + payload.Dependency.Name
		} else {
			entityName = payload.Dependency.Name
		}
	}

	text := fmt.Sprintf("%s <b>%s</b>\nStatus: %s", emoji, entityName, statusText)
	if payload.Message != "" {
		text += fmt.Sprintf("\nMessage: %s", payload.Message)
	}

	// Extract chat_id from URL if present (format: token:chatid)
	chatID := ""
	if !strings.Contains(webhookURL, "api.telegram.org") {
		parts := strings.SplitN(webhookURL, ":", 2)
		if len(parts) == 2 {
			chatID = parts[1]
		}
	}

	telegramPayload := map[string]interface{}{
		"text":       text,
		"parse_mode": "HTML",
	}
	if chatID != "" {
		telegramPayload["chat_id"] = chatID
	}

	return json.Marshal(telegramPayload)
}

func (s *NotificationService) formatDiscordPayload(payload *domain.NotificationPayload) ([]byte, error) {
	emoji := domain.StatusEmoji(payload.NewStatus)
	statusText := domain.StatusText(payload.NewStatus)

	// Build entity name
	entityName := ""
	if payload.System != nil {
		entityName = payload.System.Name
	}
	if payload.Dependency != nil {
		if entityName != "" {
			entityName += " / " + payload.Dependency.Name
		} else {
			entityName = payload.Dependency.Name
		}
	}

	// Discord color (decimal)
	color := 5763719 // green
	switch payload.NewStatus {
	case domain.StatusYellow:
		color = 16776960 // yellow
	case domain.StatusRed:
		color = 15548997 // red
	}

	discordPayload := map[string]interface{}{
		"content": fmt.Sprintf("%s **%s** is now **%s**", emoji, entityName, statusText),
		"embeds": []map[string]interface{}{
			{
				"color": color,
				"fields": []map[string]interface{}{
					{"name": "Status", "value": statusText, "inline": true},
					{"name": "Source", "value": payload.Source, "inline": true},
				},
			},
		},
	}

	if payload.Message != "" {
		discordPayload["embeds"].([]map[string]interface{})[0]["fields"] = append(
			discordPayload["embeds"].([]map[string]interface{})[0]["fields"].([]map[string]interface{}),
			map[string]interface{}{"name": "Message", "value": payload.Message, "inline": false},
		)
	}

	return json.Marshal(discordPayload)
}

// SendTestNotification sends a test notification to a webhook
func (s *NotificationService) SendTestNotification(ctx context.Context, webhookID int64) error {
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return fmt.Errorf("failed to get webhook: %w", err)
	}
	if webhook == nil {
		return fmt.Errorf("webhook not found")
	}

	// Create test payload
	payload := &domain.NotificationPayload{
		Event:     domain.EventStatusChange,
		Timestamp: time.Now(),
		System: &domain.SystemInfo{
			ID:   0,
			Name: "Test System",
		},
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusYellow,
		Message:   "This is a test notification from Status Incident",
		Source:    "manual",
	}

	s.sendNotification(webhook, payload)
	return nil
}

func logError(format string, args ...interface{}) {
	log.Printf("[WEBHOOK ERROR] "+format, args...)
}
