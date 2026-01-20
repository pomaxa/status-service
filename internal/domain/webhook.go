package domain

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

// WebhookType represents the type of webhook
type WebhookType string

const (
	WebhookTypeGeneric  WebhookType = "generic"
	WebhookTypeSlack    WebhookType = "slack"
	WebhookTypeTelegram WebhookType = "telegram"
	WebhookTypeDiscord  WebhookType = "discord"
	WebhookTypeTeams    WebhookType = "teams"
)

// WebhookEvent represents events that trigger webhooks
type WebhookEvent string

const (
	EventStatusChange  WebhookEvent = "status_change"
	EventIncidentStart WebhookEvent = "incident_start"
	EventIncidentEnd   WebhookEvent = "incident_end"
	EventSLABreach     WebhookEvent = "sla_breach"
)

// Webhook represents a notification webhook configuration
type Webhook struct {
	ID        int64
	Name      string
	URL       string
	Type      WebhookType
	Events    []WebhookEvent
	SystemIDs []int64 // nil or empty means all systems
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewWebhook creates a new webhook with validation
func NewWebhook(name, webhookURL string, webhookType WebhookType) (*Webhook, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("webhook name is required")
	}

	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return nil, errors.New("webhook URL is required")
	}

	// Validate URL
	if _, err := url.ParseRequestURI(webhookURL); err != nil {
		return nil, errors.New("invalid webhook URL")
	}

	// Validate type
	if !isValidWebhookType(webhookType) {
		webhookType = WebhookTypeGeneric
	}

	now := time.Now()
	return &Webhook{
		Name:      name,
		URL:       webhookURL,
		Type:      webhookType,
		Events:    []WebhookEvent{EventStatusChange},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func isValidWebhookType(t WebhookType) bool {
	switch t {
	case WebhookTypeGeneric, WebhookTypeSlack, WebhookTypeTelegram, WebhookTypeDiscord, WebhookTypeTeams:
		return true
	}
	return false
}

// Update updates webhook properties
func (w *Webhook) Update(name, webhookURL string, webhookType WebhookType) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("webhook name is required")
	}

	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return errors.New("webhook URL is required")
	}

	if _, err := url.ParseRequestURI(webhookURL); err != nil {
		return errors.New("invalid webhook URL")
	}

	if !isValidWebhookType(webhookType) {
		webhookType = WebhookTypeGeneric
	}

	w.Name = name
	w.URL = webhookURL
	w.Type = webhookType
	w.UpdatedAt = time.Now()
	return nil
}

// SetEvents sets the events that trigger this webhook
func (w *Webhook) SetEvents(events []WebhookEvent) {
	w.Events = events
	w.UpdatedAt = time.Now()
}

// SetSystemIDs sets the systems this webhook monitors (nil for all)
func (w *Webhook) SetSystemIDs(ids []int64) {
	w.SystemIDs = ids
	w.UpdatedAt = time.Now()
}

// Enable enables the webhook
func (w *Webhook) Enable() {
	w.Enabled = true
	w.UpdatedAt = time.Now()
}

// Disable disables the webhook
func (w *Webhook) Disable() {
	w.Enabled = false
	w.UpdatedAt = time.Now()
}

// ShouldTrigger checks if webhook should be triggered for given event and system
func (w *Webhook) ShouldTrigger(event WebhookEvent, systemID int64) bool {
	if !w.Enabled {
		return false
	}

	// Check if event is subscribed
	eventMatch := false
	for _, e := range w.Events {
		if e == event {
			eventMatch = true
			break
		}
	}
	if !eventMatch {
		return false
	}

	// Check if system is subscribed (nil/empty means all)
	if len(w.SystemIDs) == 0 {
		return true
	}
	for _, id := range w.SystemIDs {
		if id == systemID {
			return true
		}
	}
	return false
}

// EventsJSON returns events as JSON string for storage
func (w *Webhook) EventsJSON() string {
	data, _ := json.Marshal(w.Events)
	return string(data)
}

// SystemIDsJSON returns system IDs as JSON string for storage
func (w *Webhook) SystemIDsJSON() *string {
	if len(w.SystemIDs) == 0 {
		return nil
	}
	data, _ := json.Marshal(w.SystemIDs)
	s := string(data)
	return &s
}

// ParseEventsJSON parses events from JSON string
func ParseEventsJSON(data string) []WebhookEvent {
	var events []WebhookEvent
	if err := json.Unmarshal([]byte(data), &events); err != nil {
		return []WebhookEvent{EventStatusChange}
	}
	return events
}

// ParseSystemIDsJSON parses system IDs from JSON string
func ParseSystemIDsJSON(data *string) []int64 {
	if data == nil || *data == "" {
		return nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(*data), &ids); err != nil {
		return nil
	}
	return ids
}

// NotificationPayload represents a notification to be sent
type NotificationPayload struct {
	Event      WebhookEvent `json:"event"`
	Timestamp  time.Time    `json:"timestamp"`
	System     *SystemInfo  `json:"system,omitempty"`
	Dependency *DepInfo     `json:"dependency,omitempty"`
	OldStatus  Status       `json:"old_status"`
	NewStatus  Status       `json:"new_status"`
	Message    string       `json:"message,omitempty"`
	Source     string       `json:"source"`
}

// SystemInfo contains system information for notifications
type SystemInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// DepInfo contains dependency information for notifications
type DepInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// StatusEmoji returns emoji for status
func StatusEmoji(s Status) string {
	switch s {
	case StatusGreen:
		return "🟢"
	case StatusYellow:
		return "🟡"
	case StatusRed:
		return "🔴"
	default:
		return "⚪"
	}
}

// StatusText returns human-readable status text
func StatusText(s Status) string {
	switch s {
	case StatusGreen:
		return "Operational"
	case StatusYellow:
		return "Degraded"
	case StatusRed:
		return "Outage"
	default:
		return "Unknown"
	}
}

// SLABreachPayload represents an SLA breach notification
type SLABreachPayload struct {
	Event       WebhookEvent `json:"event"`
	Timestamp   time.Time    `json:"timestamp"`
	System      *SystemInfo  `json:"system"`
	BreachType  string       `json:"breach_type"`
	SLATarget   float64      `json:"sla_target"`
	ActualValue float64      `json:"actual_value"`
	Period      string       `json:"period"`
	Message     string       `json:"message"`
}
