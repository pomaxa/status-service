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
	case domain.WebhookTypeTeams:
		body, err = s.formatTeamsPayload(payload)
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

func (s *NotificationService) formatTeamsPayload(payload *domain.NotificationPayload) ([]byte, error) {
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

	// Teams theme color
	themeColor := "00FF00" // green
	switch payload.NewStatus {
	case domain.StatusYellow:
		themeColor = "FFFF00" // yellow
	case domain.StatusRed:
		themeColor = "FF0000" // red
	}

	// Microsoft Teams Adaptive Card format
	teamsPayload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": themeColor,
		"summary":    fmt.Sprintf("%s is now %s", entityName, statusText),
		"sections": []map[string]interface{}{
			{
				"activityTitle":    fmt.Sprintf("%s %s", emoji, entityName),
				"activitySubtitle": fmt.Sprintf("Status changed to **%s**", statusText),
				"facts": []map[string]interface{}{
					{"name": "Status", "value": statusText},
					{"name": "Source", "value": payload.Source},
					{"name": "Time", "value": payload.Timestamp.Format("2006-01-02 15:04:05")},
				},
				"markdown": true,
			},
		},
	}

	if payload.Message != "" {
		teamsPayload["sections"].([]map[string]interface{})[0]["facts"] = append(
			teamsPayload["sections"].([]map[string]interface{})[0]["facts"].([]map[string]interface{}),
			map[string]interface{}{"name": "Message", "value": payload.Message},
		)
	}

	return json.Marshal(teamsPayload)
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

// NotifySLABreach sends notifications for an SLA breach
func (s *NotificationService) NotifySLABreach(ctx context.Context, breach *domain.SLABreachEvent) {
	webhooks, err := s.webhookRepo.GetEnabled(ctx)
	if err != nil {
		logError("Failed to get webhooks: %v", err)
		return
	}

	if len(webhooks) == 0 {
		return
	}

	// Build payload
	payload := &domain.SLABreachPayload{
		Event:     domain.EventSLABreach,
		Timestamp: breach.DetectedAt,
		System: &domain.SystemInfo{
			ID:   breach.SystemID,
			Name: breach.SystemName,
		},
		BreachType:  breach.BreachType,
		SLATarget:   breach.SLATarget,
		ActualValue: breach.ActualValue,
		Period:      breach.Period,
		Message:     fmt.Sprintf("SLA target %.2f%% not met (actual: %.2f%%)", breach.SLATarget, breach.ActualValue),
	}

	// Send to matching webhooks
	for _, webhook := range webhooks {
		if webhook.ShouldTrigger(domain.EventSLABreach, breach.SystemID) {
			go s.sendSLABreachNotification(webhook, payload)
		}
	}
}

func (s *NotificationService) sendSLABreachNotification(webhook *domain.Webhook, payload *domain.SLABreachPayload) {
	var body []byte
	var err error

	switch webhook.Type {
	case domain.WebhookTypeSlack:
		body, err = s.formatSlackSLABreach(payload)
	case domain.WebhookTypeTelegram:
		body, err = s.formatTelegramSLABreach(webhook.URL, payload)
	case domain.WebhookTypeDiscord:
		body, err = s.formatDiscordSLABreach(payload)
	case domain.WebhookTypeTeams:
		body, err = s.formatTeamsSLABreach(payload)
	default:
		body, err = json.Marshal(payload)
	}

	if err != nil {
		logError("Failed to format SLA breach payload for webhook %s: %v", webhook.Name, err)
		return
	}

	url := webhook.URL
	if webhook.Type == domain.WebhookTypeTelegram {
		if !strings.Contains(url, "api.telegram.org") {
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

func (s *NotificationService) formatSlackSLABreach(payload *domain.SLABreachPayload) ([]byte, error) {
	slackPayload := map[string]interface{}{
		"text": fmt.Sprintf("⚠️ *SLA Breach* - *%s*", payload.System.Name),
		"attachments": []map[string]interface{}{
			{
				"color": "danger",
				"fields": []map[string]interface{}{
					{"title": "System", "value": payload.System.Name, "short": true},
					{"title": "Period", "value": payload.Period, "short": true},
					{"title": "Target", "value": fmt.Sprintf("%.2f%%", payload.SLATarget), "short": true},
					{"title": "Actual", "value": fmt.Sprintf("%.2f%%", payload.ActualValue), "short": true},
					{"title": "Message", "value": payload.Message, "short": false},
				},
			},
		},
	}

	return json.Marshal(slackPayload)
}

func (s *NotificationService) formatTelegramSLABreach(webhookURL string, payload *domain.SLABreachPayload) ([]byte, error) {
	text := fmt.Sprintf("⚠️ <b>SLA Breach - %s</b>\n\nPeriod: %s\nTarget: %.2f%%\nActual: %.2f%%\n\n%s",
		payload.System.Name, payload.Period, payload.SLATarget, payload.ActualValue, payload.Message)

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

func (s *NotificationService) formatDiscordSLABreach(payload *domain.SLABreachPayload) ([]byte, error) {
	discordPayload := map[string]interface{}{
		"content": fmt.Sprintf("⚠️ **SLA Breach** - **%s**", payload.System.Name),
		"embeds": []map[string]interface{}{
			{
				"color": 15548997, // red
				"fields": []map[string]interface{}{
					{"name": "System", "value": payload.System.Name, "inline": true},
					{"name": "Period", "value": payload.Period, "inline": true},
					{"name": "Target", "value": fmt.Sprintf("%.2f%%", payload.SLATarget), "inline": true},
					{"name": "Actual", "value": fmt.Sprintf("%.2f%%", payload.ActualValue), "inline": true},
					{"name": "Message", "value": payload.Message, "inline": false},
				},
			},
		},
	}

	return json.Marshal(discordPayload)
}

func (s *NotificationService) formatTeamsSLABreach(payload *domain.SLABreachPayload) ([]byte, error) {
	teamsPayload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": "FF0000",
		"summary":    fmt.Sprintf("SLA Breach - %s", payload.System.Name),
		"sections": []map[string]interface{}{
			{
				"activityTitle":    fmt.Sprintf("⚠️ SLA Breach - %s", payload.System.Name),
				"activitySubtitle": "SLA target not met",
				"facts": []map[string]interface{}{
					{"name": "System", "value": payload.System.Name},
					{"name": "Period", "value": payload.Period},
					{"name": "Target", "value": fmt.Sprintf("%.2f%%", payload.SLATarget)},
					{"name": "Actual", "value": fmt.Sprintf("%.2f%%", payload.ActualValue)},
					{"name": "Message", "value": payload.Message},
				},
				"markdown": true,
			},
		},
	}

	return json.Marshal(teamsPayload)
}

func logError(format string, args ...interface{}) {
	log.Printf("[WEBHOOK ERROR] "+format, args...)
}
