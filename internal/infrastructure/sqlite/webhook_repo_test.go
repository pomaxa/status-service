package sqlite

import (
	"context"
	"testing"

	"status-incident/internal/domain"
)

func TestWebhookRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	webhook, err := domain.NewWebhook("Test Webhook", "https://example.com/webhook", domain.WebhookTypeSlack)
	if err != nil {
		t.Fatalf("failed to create webhook: %v", err)
	}

	err = repo.Create(ctx, webhook)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if webhook.ID == 0 {
		t.Error("expected webhook ID to be set after Create()")
	}
}

func TestWebhookRepo_Create_WithEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	webhook, _ := domain.NewWebhook("Events Webhook", "https://example.com/webhook", domain.WebhookTypeGeneric)
	webhook.SetEvents([]domain.WebhookEvent{
		domain.EventStatusChange,
		domain.EventIncidentStart,
		domain.EventSLABreach,
	})

	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(retrieved.Events) != 3 {
		t.Errorf("Events count = %d, want 3", len(retrieved.Events))
	}
}

func TestWebhookRepo_Create_WithSystemIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create systems
	sysRepo := NewSystemRepo(db)
	sys1, _ := domain.NewSystem("System 1", "", "", "")
	sys2, _ := domain.NewSystem("System 2", "", "", "")
	sysRepo.Create(context.Background(), sys1)
	sysRepo.Create(context.Background(), sys2)

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	webhook, _ := domain.NewWebhook("Targeted Webhook", "https://example.com/webhook", domain.WebhookTypeSlack)
	webhook.SetSystemIDs([]int64{sys1.ID, sys2.ID})

	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(retrieved.SystemIDs) != 2 {
		t.Errorf("SystemIDs count = %d, want 2", len(retrieved.SystemIDs))
	}
}

func TestWebhookRepo_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	webhook, _ := domain.NewWebhook("Test Webhook", "https://example.com/webhook", domain.WebhookTypeTelegram)
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByID() returned nil")
	}

	if retrieved.ID != webhook.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, webhook.ID)
	}
	if retrieved.Name != webhook.Name {
		t.Errorf("Name = %s, want %s", retrieved.Name, webhook.Name)
	}
	if retrieved.URL != webhook.URL {
		t.Errorf("URL = %s, want %s", retrieved.URL, webhook.URL)
	}
	if retrieved.Type != domain.WebhookTypeTelegram {
		t.Errorf("Type = %s, want telegram", retrieved.Type)
	}
}

func TestWebhookRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestWebhookRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	// Create multiple webhooks
	webhooks := []struct {
		name    string
		url     string
		wtype   domain.WebhookType
		enabled bool
	}{
		{"Webhook 1", "https://example.com/1", domain.WebhookTypeSlack, true},
		{"Webhook 2", "https://example.com/2", domain.WebhookTypeDiscord, false},
		{"Webhook 3", "https://example.com/3", domain.WebhookTypeGeneric, true},
	}

	for _, w := range webhooks {
		webhook, _ := domain.NewWebhook(w.name, w.url, w.wtype)
		if !w.enabled {
			webhook.Disable()
		}
		if err := repo.Create(ctx, webhook); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(all) != 3 {
		t.Errorf("GetAll() returned %d webhooks, want 3", len(all))
	}
}

func TestWebhookRepo_GetEnabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	// Create enabled webhook
	enabled, _ := domain.NewWebhook("Enabled Webhook", "https://example.com/enabled", domain.WebhookTypeSlack)
	if err := repo.Create(ctx, enabled); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create disabled webhook
	disabled, _ := domain.NewWebhook("Disabled Webhook", "https://example.com/disabled", domain.WebhookTypeSlack)
	disabled.Disable()
	if err := repo.Create(ctx, disabled); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get enabled only
	enabledWebhooks, err := repo.GetEnabled(ctx)
	if err != nil {
		t.Fatalf("GetEnabled() error = %v", err)
	}

	if len(enabledWebhooks) != 1 {
		t.Errorf("GetEnabled() returned %d webhooks, want 1", len(enabledWebhooks))
	}

	if enabledWebhooks[0].Name != "Enabled Webhook" {
		t.Errorf("webhook name = %s, want Enabled Webhook", enabledWebhooks[0].Name)
	}
}

func TestWebhookRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	webhook, _ := domain.NewWebhook("Original Name", "https://original.com/webhook", domain.WebhookTypeGeneric)
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	webhook.Update("Updated Name", "https://updated.com/webhook", domain.WebhookTypeSlack)
	webhook.SetEvents([]domain.WebhookEvent{domain.EventIncidentStart, domain.EventIncidentEnd})
	webhook.Disable()

	if err := repo.Update(ctx, webhook); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, err := repo.GetByID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %s, want Updated Name", retrieved.Name)
	}
	if retrieved.URL != "https://updated.com/webhook" {
		t.Errorf("URL = %s, want https://updated.com/webhook", retrieved.URL)
	}
	if retrieved.Type != domain.WebhookTypeSlack {
		t.Errorf("Type = %s, want slack", retrieved.Type)
	}
	if len(retrieved.Events) != 2 {
		t.Errorf("Events count = %d, want 2", len(retrieved.Events))
	}
	if retrieved.Enabled {
		t.Error("expected Enabled = false")
	}
}

func TestWebhookRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	webhook, _ := domain.NewWebhook("To Delete", "https://example.com/delete", domain.WebhookTypeGeneric)
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, webhook.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil after deletion")
	}
}

func TestWebhookRepo_WebhookTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	types := []domain.WebhookType{
		domain.WebhookTypeGeneric,
		domain.WebhookTypeSlack,
		domain.WebhookTypeTelegram,
		domain.WebhookTypeDiscord,
		domain.WebhookTypeTeams,
	}

	for _, wtype := range types {
		t.Run(string(wtype), func(t *testing.T) {
			webhook, _ := domain.NewWebhook("Type Test "+string(wtype), "https://example.com/"+string(wtype), wtype)
			if err := repo.Create(ctx, webhook); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			retrieved, err := repo.GetByID(ctx, webhook.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if retrieved.Type != wtype {
				t.Errorf("Type = %s, want %s", retrieved.Type, wtype)
			}
		})
	}
}

func TestWebhookRepo_EventTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	events := []domain.WebhookEvent{
		domain.EventStatusChange,
		domain.EventIncidentStart,
		domain.EventIncidentEnd,
		domain.EventSLABreach,
	}

	webhook, _ := domain.NewWebhook("All Events", "https://example.com/events", domain.WebhookTypeGeneric)
	webhook.SetEvents(events)

	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(retrieved.Events) != len(events) {
		t.Errorf("Events count = %d, want %d", len(retrieved.Events), len(events))
	}

	eventMap := make(map[domain.WebhookEvent]bool)
	for _, e := range retrieved.Events {
		eventMap[e] = true
	}

	for _, expected := range events {
		if !eventMap[expected] {
			t.Errorf("missing event: %s", expected)
		}
	}
}

func TestWebhookRepo_NilSystemIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	// Webhook with nil SystemIDs (applies to all systems)
	webhook, _ := domain.NewWebhook("All Systems", "https://example.com/all", domain.WebhookTypeGeneric)
	// Don't call SetSystemIDs - leave it nil

	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.SystemIDs != nil && len(retrieved.SystemIDs) != 0 {
		t.Errorf("SystemIDs = %v, want nil or empty", retrieved.SystemIDs)
	}
}

func TestWebhookRepo_EnableDisable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWebhookRepo(db)
	ctx := context.Background()

	webhook, _ := domain.NewWebhook("Toggle Test", "https://example.com/toggle", domain.WebhookTypeGeneric)

	// Initially enabled
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, _ := repo.GetByID(ctx, webhook.ID)
	if !retrieved.Enabled {
		t.Error("expected initially enabled")
	}

	// Disable
	webhook.Disable()
	repo.Update(ctx, webhook)

	retrieved, _ = repo.GetByID(ctx, webhook.ID)
	if retrieved.Enabled {
		t.Error("expected disabled after Disable()")
	}

	// Re-enable
	webhook.Enable()
	repo.Update(ctx, webhook)

	retrieved, _ = repo.GetByID(ctx, webhook.ID)
	if !retrieved.Enabled {
		t.Error("expected enabled after Enable()")
	}
}
