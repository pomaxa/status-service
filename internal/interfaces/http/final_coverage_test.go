package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// TestActiveMaintenances_WithData covers the response-mapping loop in
// apiGetActiveMaintenances.
func TestActiveMaintenances_WithData(t *testing.T) {
	f := newFullServer(t, false)

	// active window: started in the past, ends in the future
	w := f.do("POST", "/api/maintenances", mustJSON(maintenanceRequest{
		Title:     "Active",
		StartTime: time.Now().Add(-time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Add(time.Hour).Format(time.RFC3339),
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create active maintenance: %d %s", w.Code, w.Body.String())
	}

	w = f.do("GET", "/api/maintenances/active", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("active maintenances: expected 200, got %d", w.Code)
	}
}

// TestMaintenanceValidationErrors covers the service-error 400 branches in
// apiCreateMaintenance and apiUpdateMaintenance (end before start).
func TestMaintenanceValidationErrors(t *testing.T) {
	f := newFullServer(t, false)

	start := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	end := time.Now().Add(time.Hour).Format(time.RFC3339) // before start

	// Create with end < start -> 400 from CreateMaintenance
	w := f.do("POST", "/api/maintenances", mustJSON(maintenanceRequest{Title: "Bad", StartTime: start, EndTime: end}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create bad window: expected 400, got %d", w.Code)
	}

	// Seed a valid maintenance, then update it with end < start -> 400.
	goodStart := time.Now().Add(time.Hour).Format(time.RFC3339)
	goodEnd := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	w = f.do("POST", "/api/maintenances", mustJSON(maintenanceRequest{Title: "Good", StartTime: goodStart, EndTime: goodEnd}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create good window: %d %s", w.Code, w.Body.String())
	}
	w = f.do("PUT", "/api/maintenances/1", mustJSON(maintenanceRequest{Title: "Bad", StartTime: start, EndTime: end}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update bad window: expected 400, got %d", w.Code)
	}
}

// TestCreateIncident_ValidationError covers the apiCreateIncident service-error
// 400 branch (empty title).
func TestCreateIncident_ValidationError(t *testing.T) {
	f := newFullServer(t, false)
	w := f.do("POST", "/api/incidents", mustJSON(incidentRequest{Title: "", Message: "x"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create incident empty title: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestAcknowledgeBreach_DefaultActorValidBody covers the branch where the JSON
// body decodes successfully but acked_by is empty (defaulted to "system").
func TestAcknowledgeBreach_DefaultActorValidBody(t *testing.T) {
	f := newFullServer(t, false)
	sys, _ := f.seedSystem(t, "AckDefSys")
	ctx := context.Background()

	breach := &domain.SLABreachEvent{
		SystemID: sys.ID, SystemName: sys.Name, BreachType: "uptime",
		SLATarget: 99.9, ActualValue: 90, Period: "monthly",
		PeriodStart: time.Now().Add(-time.Hour), PeriodEnd: time.Now(), DetectedAt: time.Now(),
	}
	if err := f.breachRepo.Create(ctx, breach); err != nil {
		t.Fatalf("create breach: %v", err)
	}

	// valid JSON, empty acked_by -> default branch then success
	w := f.do("POST", "/api/sla/breaches/1/acknowledge", []byte(`{"acked_by":""}`))
	if w.Code != http.StatusOK {
		t.Fatalf("ack default actor: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestPublicStatus_DBClosed covers the handlePublicStatus 500 branch when
// GetAllSystems fails.
func TestPublicStatus_DBClosed(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "PubErr")
	if err := f.db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	w := f.do("GET", "/status", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("public status with closed DB: expected 500, got %d", w.Code)
	}
}

// ---- Webhook Update/Delete repo-error branches via a failing mock ----

// failingWebhookRepo wraps the in-memory mock but errors on Update / Delete so
// the 500 branches in UpdateWebhook / DeleteWebhook are exercised after a
// successful GetByID.
type failingWebhookRepo struct {
	*MockWebhookRepository
	failUpdate bool
	failDelete bool
}

func (f *failingWebhookRepo) Update(ctx context.Context, w *domain.Webhook) error {
	if f.failUpdate {
		return errors.New("update failed")
	}
	return f.MockWebhookRepository.Update(ctx, w)
}

func (f *failingWebhookRepo) Delete(ctx context.Context, id int64) error {
	if f.failDelete {
		return errors.New("delete failed")
	}
	return f.MockWebhookRepository.Delete(ctx, id)
}

func TestWebhookHandlers_UpdateDeleteRepoErrors(t *testing.T) {
	base := NewMockWebhookRepository()
	wh, _ := domain.NewWebhook("wh", "https://hooks.example.com/x", domain.WebhookTypeSlack)
	base.Create(context.Background(), wh)

	repo := &failingWebhookRepo{MockWebhookRepository: base, failUpdate: true, failDelete: true}
	h := NewWebhookHandlers(repo, nil)

	// Update -> 500 (GetByID ok, Update fails)
	r := reqWithID("PUT", "/x", "id", intToStr(wh.ID), mustJSON(webhookRequest{
		Name: "x", URL: "https://hooks.example.com/y", Type: "slack",
	}))
	w := httptest.NewRecorder()
	h.UpdateWebhook(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("update repo error: expected 500, got %d", w.Code)
	}

	// Delete -> 500 (GetByID ok, Delete fails)
	r = reqWithID("DELETE", "/x", "id", intToStr(wh.ID), nil)
	w = httptest.NewRecorder()
	h.DeleteWebhook(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("delete repo error: expected 500, got %d", w.Code)
	}
}

// errNotifWebhookRepo + a notification service that fails: covers the
// TestWebhook 404 branch (SendTestNotification error). The notification service
// blocks the private host, so SendTestNotification returns an error.
func TestWebhookHandlers_TestWebhookNotifyError(t *testing.T) {
	f := newFullServer(t, false)
	ctx := context.Background()

	wh, _ := domain.NewWebhook("wh", "https://hooks.example.com/x", domain.WebhookTypeSlack)
	if err := f.webhookRepo.Create(ctx, wh); err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	// Route through the full server's webhook handlers (real notification service).
	w := f.do("POST", "/api/webhooks/"+intToStr(wh.ID)+"/test", nil)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Fatalf("test webhook: unexpected %d: %s", w.Code, w.Body.String())
	}

	// TestWebhook against a valid-but-nonexistent ID: SendTestNotification
	// returns "webhook not found", driving the handler's 404 branch.
	w = f.do("POST", "/api/webhooks/999999/test", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("test webhook missing: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestImport_LogCreateError drops the status_log table so that during import the
// system create succeeds but the log create fails, covering the CreateLog error
// branch in apiImportAll.
func TestImport_LogCreateError(t *testing.T) {
	f := newFullServer(t, false)

	// Drop the logs table so CreateLog fails while systems still insert fine.
	if _, err := f.db.Exec("DROP TABLE status_log"); err != nil {
		t.Fatalf("drop status_log: %v", err)
	}

	now := time.Now()
	data := ExportData{
		Version: "1.0",
		Systems: []ExportSystem{{ID: 1, Name: "S", Status: "green", CreatedAt: now, UpdatedAt: now}},
		Logs: []ExportLog{
			{ID: 2, SystemID: ptrInt64(1), OldStatus: "green", NewStatus: "yellow", Source: "manual", CreatedAt: now},
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
	if result.SystemsImported != 1 {
		t.Errorf("expected 1 system imported, got %d", result.SystemsImported)
	}
	// the log create should have failed and been recorded as an error
	if result.LogsImported != 0 {
		t.Errorf("expected 0 logs imported, got %d", result.LogsImported)
	}
	if len(result.Errors) == 0 {
		t.Errorf("expected a log create error recorded")
	}
}

// TestExportAll_LogsQueryError drops status_log so apiExportAll fails at the
// GetAllLogs step (after systems succeed), covering its 500 branch.
func TestExportAll_LogsQueryError(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "ExpLogErr")
	if _, err := f.db.Exec("DROP TABLE status_log"); err != nil {
		t.Fatalf("drop status_log: %v", err)
	}
	w := f.do("GET", "/api/export", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("export with broken logs table: expected 500, got %d", w.Code)
	}
}

// TestExportAll_DependencyQueryError drops the dependencies table so that during
// export each system's dependency lookup fails and is skipped (continue branch),
// while systems and logs still export successfully (200).
func TestExportAll_DependencyQueryError(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "ExpDepErr")
	if _, err := f.db.Exec("DROP TABLE dependencies"); err != nil {
		t.Fatalf("drop dependencies: %v", err)
	}
	w := f.do("GET", "/api/export", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("export with broken dependencies table: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
