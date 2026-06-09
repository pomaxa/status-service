package application

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"status-incident/internal/domain"
	"strings"
	"testing"
	"time"
)

// ============= Formatter message-branch coverage =============

func TestNotificationService_formatDiscordPayload_WithMessage(t *testing.T) {
	s := &NotificationService{}
	payload := &domain.NotificationPayload{
		Event:     domain.EventStatusChange,
		Timestamp: time.Now(),
		System:    &domain.SystemInfo{ID: 1, Name: "API"},
		NewStatus: domain.StatusRed,
		Message:   "Connection timeout",
		Source:    "heartbeat",
	}
	body, err := s.formatDiscordPayload(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), "Connection timeout") {
		t.Error("expected message embedded in Discord payload")
	}
}

func TestNotificationService_formatTeamsPayload_WithMessage(t *testing.T) {
	s := &NotificationService{}
	payload := &domain.NotificationPayload{
		Event:      domain.EventStatusChange,
		Timestamp:  time.Now(),
		System:     &domain.SystemInfo{ID: 1, Name: "API"},
		Dependency: &domain.DepInfo{ID: 2, Name: "DB"},
		NewStatus:  domain.StatusYellow,
		Message:    "Latency spike",
		Source:     "heartbeat",
	}
	body, err := s.formatTeamsPayload(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), "Latency spike") {
		t.Error("expected message embedded in Teams payload")
	}
	if !strings.Contains(string(body), "API / DB") {
		t.Error("expected combined entity name")
	}
}

func TestNotificationService_formatTelegramPayload_DependencyOnly(t *testing.T) {
	s := &NotificationService{}
	payload := &domain.NotificationPayload{
		Event:      domain.EventStatusChange,
		Timestamp:  time.Now(),
		Dependency: &domain.DepInfo{ID: 2, Name: "DB"},
		NewStatus:  domain.StatusGreen,
		Source:     "heartbeat",
	}
	body, err := s.formatTelegramPayload("https://api.telegram.org/bot1/sendMessage", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), "DB") {
		t.Error("expected dependency name")
	}
}

func TestNotificationService_formatDiscordPayload_DependencyOnly(t *testing.T) {
	s := &NotificationService{}
	payload := &domain.NotificationPayload{
		Event:      domain.EventStatusChange,
		Timestamp:  time.Now(),
		Dependency: &domain.DepInfo{ID: 2, Name: "DB"},
		NewStatus:  domain.StatusGreen,
		Source:     "heartbeat",
	}
	if _, err := s.formatDiscordPayload(payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotificationService_formatTeamsPayload_DependencyOnly(t *testing.T) {
	s := &NotificationService{}
	payload := &domain.NotificationPayload{
		Event:      domain.EventStatusChange,
		Timestamp:  time.Now(),
		Dependency: &domain.DepInfo{ID: 2, Name: "DB"},
		NewStatus:  domain.StatusGreen,
		Source:     "heartbeat",
	}
	if _, err := s.formatTeamsPayload(payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============= sendNotification (synchronous, via test server) =============

func newTestNotificationService(client *http.Client) *NotificationService {
	s := NewNotificationService(NewMockWebhookRepository(), NewMockSystemRepository(), NewMockDependencyRepository())
	if client != nil {
		s.httpClient = client
	}
	return s
}

func statusChangePayload() *domain.NotificationPayload {
	return &domain.NotificationPayload{
		Event:     domain.EventStatusChange,
		Timestamp: time.Now(),
		System:    &domain.SystemInfo{ID: 1, Name: "API"},
		NewStatus: domain.StatusRed,
		Message:   "down",
		Source:    "manual",
	}
}

func TestNotificationService_sendNotification_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := newTestNotificationService(srv.Client())
	for _, wt := range []domain.WebhookType{
		domain.WebhookTypeSlack,
		domain.WebhookTypeDiscord,
		domain.WebhookTypeTeams,
		domain.WebhookTypeGeneric,
	} {
		webhook := &domain.Webhook{Name: "wh", URL: srv.URL, Type: wt, Enabled: true}
		s.sendNotification(webhook, statusChangePayload())
	}
}

func TestNotificationService_sendNotification_4xxResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	s := newTestNotificationService(srv.Client())
	webhook := &domain.Webhook{Name: "wh", URL: srv.URL, Type: domain.WebhookTypeGeneric, Enabled: true}
	s.sendNotification(webhook, statusChangePayload())
}

func TestNotificationService_sendNotification_BlockedURL(t *testing.T) {
	s := newTestNotificationService(nil)
	// localhost is blocked by validateWebhookURL
	webhook := &domain.Webhook{Name: "wh", URL: "http://localhost:9999/hook", Type: domain.WebhookTypeGeneric, Enabled: true}
	s.sendNotification(webhook, statusChangePayload())
}

func TestNotificationService_sendNotification_RequestError(t *testing.T) {
	s := newTestNotificationService(nil)
	// Valid public host but unreachable -> httpClient.Do error path.
	webhook := &domain.Webhook{Name: "wh", URL: "https://example.com/hook", Type: domain.WebhookTypeGeneric, Enabled: true}
	// Short timeout so it fails fast.
	s.httpClient = &http.Client{Timeout: 1 * time.Millisecond}
	s.sendNotification(webhook, statusChangePayload())
}

func TestNotificationService_sendNotification_TelegramURLRewrite(t *testing.T) {
	// token:chatid format triggers the api.telegram.org rewrite, which then
	// resolves to a public host and is validated.
	s := newTestNotificationService(&http.Client{Timeout: 1 * time.Millisecond})
	webhook := &domain.Webhook{Name: "tg", URL: "123456:@channel", Type: domain.WebhookTypeTelegram, Enabled: true}
	s.sendNotification(webhook, statusChangePayload())
}

// ============= sendSLABreachNotification (synchronous, via test server) =============

func slaBreachPayload() *domain.SLABreachPayload {
	return &domain.SLABreachPayload{
		Event:       domain.EventSLABreach,
		Timestamp:   time.Now(),
		System:      &domain.SystemInfo{ID: 1, Name: "API"},
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.0,
		Period:      "monthly",
		Message:     "SLA breach",
	}
}

func TestNotificationService_sendSLABreachNotification_AllTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := newTestNotificationService(srv.Client())
	for _, wt := range []domain.WebhookType{
		domain.WebhookTypeSlack,
		domain.WebhookTypeDiscord,
		domain.WebhookTypeTeams,
		domain.WebhookTypeGeneric,
	} {
		webhook := &domain.Webhook{Name: "wh", URL: srv.URL, Type: wt, Enabled: true}
		s.sendSLABreachNotification(webhook, slaBreachPayload())
	}
}

func TestNotificationService_sendSLABreachNotification_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := newTestNotificationService(srv.Client())
	webhook := &domain.Webhook{Name: "wh", URL: srv.URL, Type: domain.WebhookTypeGeneric, Enabled: true}
	s.sendSLABreachNotification(webhook, slaBreachPayload())
}

func TestNotificationService_sendSLABreachNotification_Blocked(t *testing.T) {
	s := newTestNotificationService(nil)
	webhook := &domain.Webhook{Name: "wh", URL: "http://127.0.0.1/hook", Type: domain.WebhookTypeGeneric, Enabled: true}
	s.sendSLABreachNotification(webhook, slaBreachPayload())
}

func TestNotificationService_sendSLABreachNotification_TelegramRewrite(t *testing.T) {
	s := newTestNotificationService(&http.Client{Timeout: 1 * time.Millisecond})
	webhook := &domain.Webhook{Name: "tg", URL: "999:@chan", Type: domain.WebhookTypeTelegram, Enabled: true}
	s.sendSLABreachNotification(webhook, slaBreachPayload())
}

func TestNotificationService_sendSLABreachNotification_RequestError(t *testing.T) {
	s := newTestNotificationService(&http.Client{Timeout: 1 * time.Millisecond})
	webhook := &domain.Webhook{Name: "wh", URL: "https://example.com/hook", Type: domain.WebhookTypeGeneric, Enabled: true}
	s.sendSLABreachNotification(webhook, slaBreachPayload())
}

// ============= NotifyStatusChange branches =============

func TestNotificationService_NotifyStatusChange_GetWebhooksError(t *testing.T) {
	webhookRepo := NewMockWebhookRepository()
	// Replace with a failing webhook repo via wrapper.
	s := NewNotificationService(&failingWebhookRepo{}, NewMockSystemRepository(), NewMockDependencyRepository())
	_ = webhookRepo
	s.NotifyStatusChange(context.Background(), &domain.StatusLog{
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		CreatedAt: time.Now(),
	})
}

func TestNotificationService_NotifyStatusChange_DependencyResolvesSystem(t *testing.T) {
	ctx := context.Background()
	webhookRepo := NewMockWebhookRepository()
	systemRepo := NewMockSystemRepository()
	depRepo := NewMockDependencyRepository()

	system, _ := domain.NewSystem("API", "", "", "")
	systemRepo.Create(ctx, system)
	dep, _ := domain.NewDependency(system.ID, "DB", "")
	depRepo.Create(ctx, dep)

	webhook := &domain.Webhook{Name: "wh", URL: "https://example.com/hook", Type: domain.WebhookTypeGeneric, Enabled: true, Events: []domain.WebhookEvent{domain.EventStatusChange}}
	webhookRepo.Create(ctx, webhook)

	s := NewNotificationService(webhookRepo, systemRepo, depRepo)
	s.httpClient = &http.Client{Timeout: 1 * time.Millisecond}

	// DependencyID set -> systemID resolved from dependency.
	s.NotifyStatusChange(ctx, &domain.StatusLog{
		DependencyID: &dep.ID,
		OldStatus:    domain.StatusGreen,
		NewStatus:    domain.StatusRed,
		CreatedAt:    time.Now(),
		Source:       domain.SourceHeartbeat,
	})
	time.Sleep(10 * time.Millisecond)
}

func TestNotificationService_NotifySLABreach_GetWebhooksError(t *testing.T) {
	s := NewNotificationService(&failingWebhookRepo{}, NewMockSystemRepository(), NewMockDependencyRepository())
	s.NotifySLABreach(context.Background(), &domain.SLABreachEvent{
		SystemID:   1,
		SystemName: "API",
		DetectedAt: time.Now(),
	})
}

func TestNotificationService_NotifySLABreach_TriggersWebhook(t *testing.T) {
	ctx := context.Background()
	webhookRepo := NewMockWebhookRepository()
	webhook := &domain.Webhook{Name: "wh", URL: "https://example.com/hook", Type: domain.WebhookTypeSlack, Enabled: true, Events: []domain.WebhookEvent{domain.EventSLABreach}}
	webhookRepo.Create(ctx, webhook)

	s := NewNotificationService(webhookRepo, NewMockSystemRepository(), NewMockDependencyRepository())
	s.httpClient = &http.Client{Timeout: 1 * time.Millisecond}

	s.NotifySLABreach(ctx, &domain.SLABreachEvent{
		SystemID:    1,
		SystemName:  "API",
		BreachType:  "uptime",
		SLATarget:   99.9,
		ActualValue: 98.0,
		Period:      "monthly",
		DetectedAt:  time.Now(),
	})
	time.Sleep(10 * time.Millisecond)
}

// SendTestNotification: GetByID error path.
func TestNotificationService_SendTestNotification_GetError(t *testing.T) {
	s := NewNotificationService(&failingWebhookRepo{}, NewMockSystemRepository(), NewMockDependencyRepository())
	if err := s.SendTestNotification(context.Background(), 1); err == nil {
		t.Error("expected get webhook error")
	}
}

// ============= isPrivateIP / validateWebhookURL coverage =============

func TestIsPrivateIP_Cases(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},   // loopback
		{"10.1.2.3", true},    // private 10/8
		{"172.16.5.5", true},  // private 172.16/12
		{"192.168.1.1", true}, // private 192.168/16
		{"169.254.1.1", true}, // link-local
		{"::1", true},         // ipv6 loopback
		{"8.8.8.8", false},    // public
		{"1.1.1.1", false},    // public
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := isPrivateIP(ip); got != tt.expected {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
	if isPrivateIP(nil) {
		t.Error("isPrivateIP(nil) should be false")
	}
}

func TestValidateWebhookURL_Cases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"parse error", "://bad-url", true},
		{"localhost blocked", "http://localhost/x", true},
		{"empty host blocked", "https://", true},
		{"private ip literal blocked", "http://10.0.0.1/x", true},
		{"public ip literal ok", "http://8.8.8.8/x", false},
		{"public hostname ok", "https://example.com/x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.url, err)
			}
		})
	}
}

// failingWebhookRepo returns an error from GetEnabled/GetByID for error-path tests.
type failingWebhookRepo struct{}

func (f *failingWebhookRepo) Create(ctx context.Context, w *domain.Webhook) error { return nil }
func (f *failingWebhookRepo) GetByID(ctx context.Context, id int64) (*domain.Webhook, error) {
	return nil, errors.New("get error")
}
func (f *failingWebhookRepo) GetAll(ctx context.Context) ([]*domain.Webhook, error) {
	return nil, errors.New("getall error")
}
func (f *failingWebhookRepo) GetEnabled(ctx context.Context) ([]*domain.Webhook, error) {
	return nil, errors.New("getenabled error")
}
func (f *failingWebhookRepo) Update(ctx context.Context, w *domain.Webhook) error { return nil }
func (f *failingWebhookRepo) Delete(ctx context.Context, id int64) error          { return nil }
