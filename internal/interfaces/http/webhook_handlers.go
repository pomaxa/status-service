package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"status-incident/internal/application"
	"status-incident/internal/domain"
)

// WebhookHandlers contains HTTP handlers for webhook operations
type WebhookHandlers struct {
	webhookRepo         domain.WebhookRepository
	notificationService *application.NotificationService
}

// NewWebhookHandlers creates new WebhookHandlers
func NewWebhookHandlers(
	webhookRepo domain.WebhookRepository,
	notificationService *application.NotificationService,
) *WebhookHandlers {
	return &WebhookHandlers{
		webhookRepo:         webhookRepo,
		notificationService: notificationService,
	}
}

// webhookRequest represents a webhook create/update request
type webhookRequest struct {
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Type      string   `json:"type"`
	Events    []string `json:"events"`
	SystemIDs []int64  `json:"system_ids"`
	Enabled   *bool    `json:"enabled"`
}

// webhookResponse represents a webhook in API responses
type webhookResponse struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Type      string   `json:"type"`
	Events    []string `json:"events"`
	SystemIDs []int64  `json:"system_ids,omitempty"`
	Enabled   bool     `json:"enabled"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func toWebhookResponse(w *domain.Webhook) webhookResponse {
	events := make([]string, len(w.Events))
	for i, e := range w.Events {
		events[i] = string(e)
	}

	return webhookResponse{
		ID:        w.ID,
		Name:      w.Name,
		URL:       w.URL,
		Type:      string(w.Type),
		Events:    events,
		SystemIDs: w.SystemIDs,
		Enabled:   w.Enabled,
		CreatedAt: w.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: w.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// ListWebhooks handles GET /api/webhooks
// @Summary List all webhooks
// @Tags webhooks
// @Produce json
// @Success 200 {array} webhookResponse
// @Router /api/webhooks [get]
func (h *WebhookHandlers) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	webhooks, err := h.webhookRepo.GetAll(r.Context())
	if err != nil {
		jsonError(w, "Failed to get webhooks", http.StatusInternalServerError)
		return
	}

	response := make([]webhookResponse, len(webhooks))
	for i, wh := range webhooks {
		response[i] = toWebhookResponse(wh)
	}

	jsonResponse(w, response)
}

// CreateWebhook handles POST /api/webhooks
// @Summary Create a new webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param webhook body webhookRequest true "Webhook data"
// @Success 201 {object} webhookResponse
// @Failure 400 {object} errorResponse
// @Router /api/webhooks [post]
func (h *WebhookHandlers) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	webhook, err := domain.NewWebhook(req.Name, req.URL, domain.WebhookType(req.Type))
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set events
	if len(req.Events) > 0 {
		events := make([]domain.WebhookEvent, len(req.Events))
		for i, e := range req.Events {
			events[i] = domain.WebhookEvent(e)
		}
		webhook.SetEvents(events)
	}

	// Set system IDs
	if len(req.SystemIDs) > 0 {
		webhook.SetSystemIDs(req.SystemIDs)
	}

	// Set enabled
	if req.Enabled != nil && !*req.Enabled {
		webhook.Disable()
	}

	if err := h.webhookRepo.Create(r.Context(), webhook); err != nil {
		jsonError(w, "Failed to create webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, toWebhookResponse(webhook))
}

// GetWebhook handles GET /api/webhooks/{id}
// @Summary Get a webhook by ID
// @Tags webhooks
// @Produce json
// @Param id path int true "Webhook ID"
// @Success 200 {object} webhookResponse
// @Failure 404 {object} errorResponse
// @Router /api/webhooks/{id} [get]
func (h *WebhookHandlers) GetWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	webhook, err := h.webhookRepo.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, "Failed to get webhook", http.StatusInternalServerError)
		return
	}
	if webhook == nil {
		jsonError(w, "Webhook not found", http.StatusNotFound)
		return
	}

	jsonResponse(w, toWebhookResponse(webhook))
}

// UpdateWebhook handles PUT /api/webhooks/{id}
// @Summary Update a webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path int true "Webhook ID"
// @Param webhook body webhookRequest true "Webhook data"
// @Success 200 {object} webhookResponse
// @Failure 400,404 {object} errorResponse
// @Router /api/webhooks/{id} [put]
func (h *WebhookHandlers) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	webhook, err := h.webhookRepo.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, "Failed to get webhook", http.StatusInternalServerError)
		return
	}
	if webhook == nil {
		jsonError(w, "Webhook not found", http.StatusNotFound)
		return
	}

	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := webhook.Update(req.Name, req.URL, domain.WebhookType(req.Type)); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update events
	if len(req.Events) > 0 {
		events := make([]domain.WebhookEvent, len(req.Events))
		for i, e := range req.Events {
			events[i] = domain.WebhookEvent(e)
		}
		webhook.SetEvents(events)
	}

	// Update system IDs
	webhook.SetSystemIDs(req.SystemIDs)

	// Update enabled
	if req.Enabled != nil {
		if *req.Enabled {
			webhook.Enable()
		} else {
			webhook.Disable()
		}
	}

	if err := h.webhookRepo.Update(r.Context(), webhook); err != nil {
		jsonError(w, "Failed to update webhook", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, toWebhookResponse(webhook))
}

// DeleteWebhook handles DELETE /api/webhooks/{id}
// @Summary Delete a webhook
// @Tags webhooks
// @Param id path int true "Webhook ID"
// @Success 204
// @Failure 404 {object} errorResponse
// @Router /api/webhooks/{id} [delete]
func (h *WebhookHandlers) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	webhook, err := h.webhookRepo.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, "Failed to get webhook", http.StatusInternalServerError)
		return
	}
	if webhook == nil {
		jsonError(w, "Webhook not found", http.StatusNotFound)
		return
	}

	if err := h.webhookRepo.Delete(r.Context(), id); err != nil {
		jsonError(w, "Failed to delete webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestWebhook handles POST /api/webhooks/{id}/test
// @Summary Send a test notification to a webhook
// @Tags webhooks
// @Param id path int true "Webhook ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} errorResponse
// @Router /api/webhooks/{id}/test [post]
func (h *WebhookHandlers) TestWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	if err := h.notificationService.SendTestNotification(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, map[string]string{"status": "sent"})
}
