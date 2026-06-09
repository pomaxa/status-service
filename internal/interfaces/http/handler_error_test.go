package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// ---- Webhook handler: invalid IDs, create variants, repo errors, test success ----

func TestWebhookHandlers_InvalidIDs(t *testing.T) {
	repo := NewMockWebhookRepository()
	h := NewWebhookHandlers(repo, nil)

	handlers := map[string]http.HandlerFunc{
		"get":    h.GetWebhook,
		"update": h.UpdateWebhook,
		"delete": h.DeleteWebhook,
		"test":   h.TestWebhook,
	}
	for name, fn := range handlers {
		r := reqWithID("POST", "/x", "id", "not-a-number", []byte(`{}`))
		w := httptest.NewRecorder()
		fn(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s invalid id: expected 400, got %d", name, w.Code)
		}
	}
}

func TestWebhookHandlers_CreateWithSystemIDsAndDisabled(t *testing.T) {
	repo := NewMockWebhookRepository()
	h := NewWebhookHandlers(repo, nil)

	disabled := false
	body := mustJSON(webhookRequest{
		Name: "wh", URL: "https://hooks.example.com/x", Type: "slack",
		Events: []string{"status_change"}, SystemIDs: []int64{1, 2}, Enabled: &disabled,
	})
	r := httptest.NewRequest("POST", "/api/webhooks", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateWebhook(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	// the created webhook should be disabled with the system ids set
	for _, wh := range repo.Webhooks {
		if wh.Enabled {
			t.Errorf("expected webhook disabled")
		}
		if len(wh.SystemIDs) != 2 {
			t.Errorf("expected 2 system ids, got %d", len(wh.SystemIDs))
		}
	}
}

// ---- Closed-DB write error branches (500) for webhook / apikey / sla / export ----

func TestHandlerWriteErrors_DBClosed(t *testing.T) {
	f := newFullServer(t, false)
	// seed a webhook + apikey so update/delete/toggle reach the repo write
	ctx := context.Background()
	wh, _ := domain.NewWebhook("wh", "https://hooks.example.com/x", domain.WebhookTypeSlack)
	if err := f.webhookRepo.Create(ctx, wh); err != nil {
		t.Fatalf("seed webhook: %v", err)
	}
	keyVal, _ := domain.GenerateAPIKey()
	if err := f.apiKeyRepo.Create(ctx, &domain.APIKey{Name: "k", Key: keyVal, KeyHash: domain.HashAPIKey(keyVal), Enabled: true, Scopes: []string{"read"}}); err != nil {
		t.Fatalf("seed apikey: %v", err)
	}

	if err := f.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{"create_webhook", "POST", "/api/webhooks", mustJSON(webhookRequest{Name: "x", URL: "https://hooks.example.com/y", Type: "slack"})},
		{"update_webhook", "PUT", "/api/webhooks/1", mustJSON(webhookRequest{Name: "x", URL: "https://hooks.example.com/y", Type: "slack"})},
		{"delete_webhook", "DELETE", "/api/webhooks/1", nil},
		{"create_apikey", "POST", "/api/apikeys", []byte(`{"name":"x"}`)},
		{"delete_apikey", "DELETE", "/api/apikeys/1", nil},
		{"toggle_apikey", "PUT", "/api/apikeys/1/toggle", []byte(`{"enabled":false}`)},
		{"generate_report", "POST", "/api/sla/reports", []byte(`{}`)},
		{"ack_breach", "POST", "/api/sla/breaches/1/acknowledge", []byte(`{"acked_by":"x"}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := f.do(tc.method, tc.path, tc.body)
			if w.Code < 400 {
				t.Errorf("%s with closed DB: expected error status, got %d", tc.name, w.Code)
			}
		})
	}
}

// TestImport_PerItemErrors hits the per-item error branches in apiImportAll:
// a system that fails to create (empty name), and a log create that is skipped.
func TestImport_PerItemErrors(t *testing.T) {
	f := newFullServer(t, false)
	now := time.Now()

	data := ExportData{
		Version: "1.0",
		Systems: []ExportSystem{
			{ID: 1, Name: "", Status: "green", CreatedAt: now, UpdatedAt: now}, // empty name -> create error
			{ID: 2, Name: "Good", Status: "green", CreatedAt: now, UpdatedAt: now},
		},
		Dependencies: []ExportDependency{
			{ID: 10, SystemID: 2, Name: "", Status: "green", CreatedAt: now, UpdatedAt: now}, // empty name dep -> create error
		},
		Logs: []ExportLog{
			{ID: 20, SystemID: ptrInt64(2), OldStatus: "green", NewStatus: "yellow", Source: "manual", CreatedAt: now},
		},
	}

	w := f.do("POST", "/api/import", mustJSON(data))
	if w.Code != http.StatusOK {
		t.Fatalf("import: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result ImportResult
	if err := unmarshalBody(w, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Errorf("expected per-item errors recorded")
	}
	if result.SystemsImported != 1 {
		t.Errorf("expected 1 system imported, got %d", result.SystemsImported)
	}
}

// toggleFailRepo lets GetByKey/GetAll succeed but fails on Update, covering the
// ToggleAPIKey 500 (update) branch.
type toggleFailRepo struct {
	key *domain.APIKey
}

func (r *toggleFailRepo) Create(ctx context.Context, k *domain.APIKey) error { return nil }
func (r *toggleFailRepo) GetByKey(ctx context.Context, key string) (*domain.APIKey, error) {
	return nil, nil // no match by the (mis-used) id-as-key lookup; not an error
}
func (r *toggleFailRepo) GetAll(ctx context.Context) ([]*domain.APIKey, error) {
	return []*domain.APIKey{r.key}, nil
}
func (r *toggleFailRepo) Update(ctx context.Context, k *domain.APIKey) error {
	return context.DeadlineExceeded
}
func (r *toggleFailRepo) Delete(ctx context.Context, id int64) error         { return nil }
func (r *toggleFailRepo) UpdateLastUsed(ctx context.Context, id int64) error { return nil }

func TestToggleAPIKey_UpdateError(t *testing.T) {
	repo := &toggleFailRepo{key: &domain.APIKey{ID: 1, Name: "k", Enabled: true, Scopes: []string{"read"}}}
	h := NewAPIKeyHandlers(repo)

	r := reqWithID("PUT", "/x", "id", "1", []byte(`{"enabled":false}`))
	w := httptest.NewRecorder()
	h.ToggleAPIKey(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("toggle update error: expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
