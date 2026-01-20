package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"status-incident/internal/application"
)

// SLAHandlers handles SLA-related HTTP requests
type SLAHandlers struct {
	slaService *application.SLAService
}

// NewSLAHandlers creates a new SLAHandlers
func NewSLAHandlers(slaService *application.SLAService) *SLAHandlers {
	return &SLAHandlers{
		slaService: slaService,
	}
}

// GenerateReport creates a new SLA report
// POST /api/sla/reports
func (h *SLAHandlers) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Period      string `json:"period"`
		GeneratedBy string `json:"generated_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		req.Title = "SLA Report"
	}
	if req.Period == "" {
		req.Period = "monthly"
	}
	if req.GeneratedBy == "" {
		req.GeneratedBy = "system"
	}

	report, err := h.slaService.GenerateReport(r.Context(), req.Title, req.Period, req.GeneratedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// GetReports retrieves all SLA reports
// GET /api/sla/reports
func (h *SLAHandlers) GetReports(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	reports, err := h.slaService.GetAllReports(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, reports)
}

// GetReport retrieves a single SLA report
// GET /api/sla/reports/{id}
func (h *SLAHandlers) GetReport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	report, err := h.slaService.GetReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if report == nil {
		writeError(w, http.StatusNotFound, "Report not found")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// DeleteReport removes an SLA report
// DELETE /api/sla/reports/{id}
func (h *SLAHandlers) DeleteReport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	if err := h.slaService.DeleteReport(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetBreaches retrieves all SLA breaches
// GET /api/sla/breaches
func (h *SLAHandlers) GetBreaches(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Check for unacknowledged filter
	if r.URL.Query().Get("unacknowledged") == "true" {
		breaches, err := h.slaService.GetUnacknowledgedBreaches(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, breaches)
		return
	}

	breaches, err := h.slaService.GetBreaches(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, breaches)
}

// AcknowledgeBreach marks a breach as acknowledged
// POST /api/sla/breaches/{id}/acknowledge
func (h *SLAHandlers) AcknowledgeBreach(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid breach ID")
		return
	}

	var req struct {
		AckedBy string `json:"acked_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.AckedBy = "system"
	}
	if req.AckedBy == "" {
		req.AckedBy = "system"
	}

	if err := h.slaService.AcknowledgeBreach(r.Context(), id, req.AckedBy); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

// CheckBreaches manually triggers breach check
// POST /api/sla/breaches/check
func (h *SLAHandlers) CheckBreaches(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "monthly"
	}

	breaches, err := h.slaService.CheckForBreaches(r.Context(), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"breaches_found": len(breaches),
		"breaches":       breaches,
	})
}

// GetSystemSLA returns SLA status for a system
// GET /api/systems/{id}/sla
func (h *SLAHandlers) GetSystemSLA(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "monthly"
	}

	slaStatus, err := h.slaService.GetSystemSLAStatus(r.Context(), id, period)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "System not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, slaStatus)
}

// UpdateSystemSLATarget updates the SLA target for a system
// PUT /api/systems/{id}/sla-target
func (h *SLAHandlers) UpdateSystemSLATarget(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	var req struct {
		SLATarget float64 `json:"sla_target"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SLATarget <= 0 || req.SLATarget > 100 {
		writeError(w, http.StatusBadRequest, "SLA target must be between 0 and 100")
		return
	}

	if err := h.slaService.UpdateSystemSLATarget(r.Context(), id, req.SLATarget); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "System not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "updated",
		"sla_target": req.SLATarget,
	})
}

// GetSystemBreaches retrieves breaches for a specific system
// GET /api/systems/{id}/sla/breaches
func (h *SLAHandlers) GetSystemBreaches(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	breaches, err := h.slaService.GetSystemBreaches(r.Context(), id, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, breaches)
}
