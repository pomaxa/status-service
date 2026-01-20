package http

import (
	"encoding/json"
	"net/http"
	"status-incident/internal/domain"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// APIKeyHandlers handles API key management endpoints
type APIKeyHandlers struct {
	repo domain.APIKeyRepository
}

// NewAPIKeyHandlers creates a new APIKeyHandlers
func NewAPIKeyHandlers(repo domain.APIKeyRepository) *APIKeyHandlers {
	return &APIKeyHandlers{repo: repo}
}

type createAPIKeyRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresIn *int     `json:"expires_in_days,omitempty"`
}

type apiKeyResponse struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Key       string     `json:"key,omitempty"` // Only returned on creation
	Scopes    []string   `json:"scopes"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ListAPIKeys returns all API keys
func (h *APIKeyHandlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.repo.GetAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list API keys")
		return
	}

	response := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		response = append(response, apiKeyResponse{
			ID:        k.ID,
			Name:      k.Name,
			Scopes:    k.Scopes,
			Enabled:   k.Enabled,
			CreatedAt: k.CreatedAt,
			LastUsed:  k.LastUsed,
			ExpiresAt: k.ExpiresAt,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// CreateAPIKey creates a new API key
func (h *APIKeyHandlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if len(req.Scopes) == 0 {
		req.Scopes = []string{"read"}
	}

	// Generate new key
	keyValue, err := domain.GenerateAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	apiKey := &domain.APIKey{
		Name:    req.Name,
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  req.Scopes,
		Enabled: true,
	}

	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiresAt := time.Now().AddDate(0, 0, *req.ExpiresIn)
		apiKey.ExpiresAt = &expiresAt
	}

	if err := h.repo.Create(r.Context(), apiKey); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	// Return the key value only once (on creation)
	writeJSON(w, http.StatusCreated, apiKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       keyValue, // Only shown on creation!
		Scopes:    apiKey.Scopes,
		Enabled:   apiKey.Enabled,
		CreatedAt: apiKey.CreatedAt,
		ExpiresAt: apiKey.ExpiresAt,
	})
}

// DeleteAPIKey deletes an API key
func (h *APIKeyHandlers) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid API key ID")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete API key")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ToggleAPIKey enables or disables an API key
func (h *APIKeyHandlers) ToggleAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid API key ID")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	key, err := h.repo.GetByKey(r.Context(), idStr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get API key")
		return
	}

	// Need to get by ID, not by key value
	keys, _ := h.repo.GetAll(r.Context())
	var foundKey *domain.APIKey
	for _, k := range keys {
		if k.ID == id {
			foundKey = k
			break
		}
	}

	if foundKey == nil {
		writeError(w, http.StatusNotFound, "API key not found")
		return
	}

	foundKey.Enabled = req.Enabled
	if err := h.repo.Update(r.Context(), foundKey); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update API key")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"enabled": foundKey.Enabled})
	_ = key // suppress unused warning
}
