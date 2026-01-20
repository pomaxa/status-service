package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"status-incident/internal/domain"
)

// Request/Response types
type createSystemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Owner       string `json:"owner"`
}

type updateStatusRequest struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type createDependencyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type setHeartbeatRequest struct {
	URL      string `json:"url"`
	Interval int    `json:"interval"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Helper functions
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, errorResponse{Error: message})
}

// Standalone helper functions for non-Server handlers
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func parseID(r *http.Request, param string) (int64, error) {
	idStr := chi.URLParam(r, param)
	return strconv.ParseInt(idStr, 10, 64)
}

// System handlers

// @Summary List all systems
// @Description Get a list of all systems
// @Tags systems
// @Produce json
// @Success 200 {array} domain.System
// @Router /systems [get]
func (s *Server) apiGetSystems(w http.ResponseWriter, r *http.Request) {
	systems, err := s.systemService.GetAllSystems(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, systems)
}

// @Summary Create a new system
// @Description Create a new system with the provided data
// @Tags systems
// @Accept json
// @Produce json
// @Param system body createSystemRequest true "System data"
// @Success 201 {object} domain.System
// @Failure 400 {object} errorResponse
// @Router /systems [post]
func (s *Server) apiCreateSystem(w http.ResponseWriter, r *http.Request) {
	var req createSystemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	system, err := s.systemService.CreateSystem(r.Context(), req.Name, req.Description, req.URL, req.Owner)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, system)
}

// @Summary Get a system by ID
// @Description Get detailed information about a specific system
// @Tags systems
// @Produce json
// @Param id path int true "System ID"
// @Success 200 {object} domain.System
// @Failure 404 {object} errorResponse
// @Router /systems/{id} [get]
func (s *Server) apiGetSystem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid system ID")
		return
	}

	system, err := s.systemService.GetSystem(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if system == nil {
		s.respondError(w, http.StatusNotFound, "system not found")
		return
	}

	s.respondJSON(w, http.StatusOK, system)
}

func (s *Server) apiUpdateSystem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid system ID")
		return
	}

	var req createSystemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	system, err := s.systemService.UpdateSystem(r.Context(), id, req.Name, req.Description, req.URL, req.Owner)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, system)
}

func (s *Server) apiDeleteSystem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid system ID")
		return
	}

	if err := s.systemService.DeleteSystem(r.Context(), id); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary Update system status
// @Description Update the status of a system (green, yellow, red)
// @Tags systems
// @Accept json
// @Produce json
// @Param id path int true "System ID"
// @Param status body updateStatusRequest true "New status"
// @Success 200 {object} domain.System
// @Failure 400 {object} errorResponse
// @Router /systems/{id}/status [post]
func (s *Server) apiUpdateSystemStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid system ID")
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	system, err := s.systemService.UpdateSystemStatus(r.Context(), id, req.Status, req.Message)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, system)
}

func (s *Server) apiGetSystemLogs(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid system ID")
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := s.systemService.GetSystemLogs(r.Context(), id, limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, logs)
}

func (s *Server) apiGetSystemAnalytics(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid system ID")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	analytics, err := s.analyticsService.GetSystemAnalytics(r.Context(), id, period)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, analytics)
}

// Dependency handlers
func (s *Server) apiGetDependencies(w http.ResponseWriter, r *http.Request) {
	systemID, err := parseID(r, "systemId")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid system ID")
		return
	}

	deps, err := s.depService.GetDependenciesBySystem(r.Context(), systemID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, deps)
}

func (s *Server) apiCreateDependency(w http.ResponseWriter, r *http.Request) {
	systemID, err := parseID(r, "systemId")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid system ID")
		return
	}

	var req createDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dep, err := s.depService.CreateDependency(r.Context(), systemID, req.Name, req.Description)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, dep)
}

func (s *Server) apiGetDependency(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	dep, err := s.depService.GetDependency(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dep == nil {
		s.respondError(w, http.StatusNotFound, "dependency not found")
		return
	}

	s.respondJSON(w, http.StatusOK, dep)
}

func (s *Server) apiUpdateDependency(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	var req createDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dep, err := s.depService.UpdateDependency(r.Context(), id, req.Name, req.Description)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, dep)
}

func (s *Server) apiDeleteDependency(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	if err := s.depService.DeleteDependency(r.Context(), id); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiUpdateDependencyStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dep, err := s.depService.UpdateDependencyStatus(r.Context(), id, req.Status, req.Message)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, dep)
}

func (s *Server) apiSetHeartbeat(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	var req setHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dep, err := s.depService.SetHeartbeat(r.Context(), id, req.URL, req.Interval)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, dep)
}

func (s *Server) apiClearHeartbeat(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	dep, err := s.depService.ClearHeartbeat(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, dep)
}

func (s *Server) apiForceCheck(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	dep, err := s.heartbeatService.ForceCheck(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, dep)
}

func (s *Server) apiGetDependencyLogs(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := s.depService.GetDependencyLogs(r.Context(), id, limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, logs)
}

func (s *Server) apiGetDependencyAnalytics(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid dependency ID")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	analytics, err := s.analyticsService.GetDependencyAnalytics(r.Context(), id, period)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, analytics)
}

// Log handlers
func (s *Server) apiGetAllLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := s.analyticsService.GetAllLogs(r.Context(), limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, logs)
}

// Analytics handlers
func (s *Server) apiGetOverallAnalytics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	analytics, err := s.analyticsService.GetOverallAnalytics(r.Context(), period)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, analytics)
}

// Maintenance handlers

type maintenanceRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	SystemIDs   []int64 `json:"system_ids"`
}

type maintenanceResponse struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	SystemIDs   []int64 `json:"system_ids,omitempty"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func (s *Server) apiGetMaintenances(w http.ResponseWriter, r *http.Request) {
	maintenances, err := s.maintenanceService.GetAllMaintenances(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]maintenanceResponse, len(maintenances))
	for i, m := range maintenances {
		response[i] = toMaintenanceResponse(m)
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) apiGetActiveMaintenances(w http.ResponseWriter, r *http.Request) {
	maintenances, err := s.maintenanceService.GetActiveMaintenances(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]maintenanceResponse, len(maintenances))
	for i, m := range maintenances {
		response[i] = toMaintenanceResponse(m)
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) apiGetUpcomingMaintenances(w http.ResponseWriter, r *http.Request) {
	maintenances, err := s.maintenanceService.GetUpcomingMaintenances(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]maintenanceResponse, len(maintenances))
	for i, m := range maintenances {
		response[i] = toMaintenanceResponse(m)
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) apiCreateMaintenance(w http.ResponseWriter, r *http.Request) {
	var req maintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid start_time format (use RFC3339)")
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid end_time format (use RFC3339)")
		return
	}

	m, err := s.maintenanceService.CreateMaintenance(r.Context(), req.Title, req.Description, startTime, endTime, req.SystemIDs)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, toMaintenanceResponse(m))
}

func (s *Server) apiGetMaintenance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid maintenance ID")
		return
	}

	m, err := s.maintenanceService.GetMaintenance(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if m == nil {
		s.respondError(w, http.StatusNotFound, "maintenance not found")
		return
	}

	s.respondJSON(w, http.StatusOK, toMaintenanceResponse(m))
}

func (s *Server) apiUpdateMaintenance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid maintenance ID")
		return
	}

	var req maintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid start_time format (use RFC3339)")
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid end_time format (use RFC3339)")
		return
	}

	m, err := s.maintenanceService.UpdateMaintenance(r.Context(), id, req.Title, req.Description, startTime, endTime, req.SystemIDs)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, toMaintenanceResponse(m))
}

func (s *Server) apiDeleteMaintenance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid maintenance ID")
		return
	}

	if err := s.maintenanceService.DeleteMaintenance(r.Context(), id); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiCancelMaintenance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid maintenance ID")
		return
	}

	m, err := s.maintenanceService.CancelMaintenance(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, toMaintenanceResponse(m))
}

func toMaintenanceResponse(m *domain.Maintenance) maintenanceResponse {
	return maintenanceResponse{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		StartTime:   m.StartTime.Format(time.RFC3339),
		EndTime:     m.EndTime.Format(time.RFC3339),
		SystemIDs:   m.SystemIDs,
		Status:      string(m.Status),
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
	}
}

// Incident handlers

type incidentRequest struct {
	Title     string  `json:"title"`
	Message   string  `json:"message"`
	Severity  string  `json:"severity"`
	SystemIDs []int64 `json:"system_ids"`
}

type incidentStatusRequest struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	By      string `json:"by"`
}

type incidentResolveRequest struct {
	Postmortem string `json:"postmortem"`
	By         string `json:"by"`
}

type incidentAckRequest struct {
	By string `json:"by"`
}

type incidentUpdateRequest struct {
	Message string `json:"message"`
	By      string `json:"by"`
}

type incidentResponse struct {
	ID             int64   `json:"id"`
	Title          string  `json:"title"`
	Status         string  `json:"status"`
	Severity       string  `json:"severity"`
	SystemIDs      []int64 `json:"system_ids,omitempty"`
	Message        string  `json:"message"`
	Postmortem     string  `json:"postmortem,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	ResolvedAt     *string `json:"resolved_at,omitempty"`
	AcknowledgedAt *string `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string  `json:"acknowledged_by,omitempty"`
	Duration       string  `json:"duration"`
}

type incidentUpdateResponse struct {
	ID         int64  `json:"id"`
	IncidentID int64  `json:"incident_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	CreatedAt  string `json:"created_at"`
	CreatedBy  string `json:"created_by"`
}

func (s *Server) apiGetIncidents(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	incidents, err := s.incidentService.GetAllIncidents(r.Context(), limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]incidentResponse, len(incidents))
	for i, inc := range incidents {
		response[i] = toIncidentResponse(inc)
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) apiGetActiveIncidents(w http.ResponseWriter, r *http.Request) {
	incidents, err := s.incidentService.GetActiveIncidents(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]incidentResponse, len(incidents))
	for i, inc := range incidents {
		response[i] = toIncidentResponse(inc)
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) apiGetRecentIncidents(w http.ResponseWriter, r *http.Request) {
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	incidents, err := s.incidentService.GetRecentIncidents(r.Context(), days)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]incidentResponse, len(incidents))
	for i, inc := range incidents {
		response[i] = toIncidentResponse(inc)
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) apiCreateIncident(w http.ResponseWriter, r *http.Request) {
	var req incidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	severity := domain.IncidentSeverity(req.Severity)
	if severity == "" {
		severity = domain.SeverityMinor
	}

	incident, err := s.incidentService.CreateIncident(r.Context(), req.Title, req.Message, severity, req.SystemIDs)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, toIncidentResponse(incident))
}

func (s *Server) apiGetIncident(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid incident ID")
		return
	}

	incident, err := s.incidentService.GetIncident(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if incident == nil {
		s.respondError(w, http.StatusNotFound, "incident not found")
		return
	}

	s.respondJSON(w, http.StatusOK, toIncidentResponse(incident))
}

func (s *Server) apiDeleteIncident(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid incident ID")
		return
	}

	if err := s.incidentService.DeleteIncident(r.Context(), id); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiAcknowledgeIncident(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid incident ID")
		return
	}

	var req incidentAckRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.By == "" {
		req.By = "unknown"
	}

	incident, err := s.incidentService.AcknowledgeIncident(r.Context(), id, req.By)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, toIncidentResponse(incident))
}

func (s *Server) apiUpdateIncidentStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid incident ID")
		return
	}

	var req incidentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status := domain.IncidentStatus(req.Status)
	if req.By == "" {
		req.By = "unknown"
	}

	incident, err := s.incidentService.UpdateIncidentStatus(r.Context(), id, status, req.Message, req.By)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, toIncidentResponse(incident))
}

func (s *Server) apiResolveIncident(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid incident ID")
		return
	}

	var req incidentResolveRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.By == "" {
		req.By = "unknown"
	}

	incident, err := s.incidentService.ResolveIncident(r.Context(), id, req.Postmortem, req.By)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, toIncidentResponse(incident))
}

func (s *Server) apiGetIncidentUpdates(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid incident ID")
		return
	}

	updates, err := s.incidentService.GetIncidentUpdates(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]incidentUpdateResponse, len(updates))
	for i, u := range updates {
		response[i] = incidentUpdateResponse{
			ID:         u.ID,
			IncidentID: u.IncidentID,
			Status:     string(u.Status),
			Message:    u.Message,
			CreatedAt:  u.CreatedAt.Format(time.RFC3339),
			CreatedBy:  u.CreatedBy,
		}
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) apiAddIncidentUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid incident ID")
		return
	}

	var req incidentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.By == "" {
		req.By = "unknown"
	}

	update, err := s.incidentService.AddIncidentUpdate(r.Context(), id, req.Message, req.By)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, incidentUpdateResponse{
		ID:         update.ID,
		IncidentID: update.IncidentID,
		Status:     string(update.Status),
		Message:    update.Message,
		CreatedAt:  update.CreatedAt.Format(time.RFC3339),
		CreatedBy:  update.CreatedBy,
	})
}

func toIncidentResponse(i *domain.Incident) incidentResponse {
	resp := incidentResponse{
		ID:             i.ID,
		Title:          i.Title,
		Status:         string(i.Status),
		Severity:       string(i.Severity),
		SystemIDs:      i.SystemIDs,
		Message:        i.Message,
		Postmortem:     i.Postmortem,
		CreatedAt:      i.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      i.UpdatedAt.Format(time.RFC3339),
		AcknowledgedBy: i.AcknowledgedBy,
		Duration:       formatDuration(i.Duration()),
	}

	if i.ResolvedAt != nil {
		t := i.ResolvedAt.Format(time.RFC3339)
		resp.ResolvedAt = &t
	}
	if i.AcknowledgedAt != nil {
		t := i.AcknowledgedAt.Format(time.RFC3339)
		resp.AcknowledgedAt = &t
	}

	return resp
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		return strconv.Itoa(int(d.Minutes())) + "m"
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m > 0 {
			return strconv.Itoa(h) + "h " + strconv.Itoa(m) + "m"
		}
		return strconv.Itoa(h) + "h"
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h"
	}
	return strconv.Itoa(days) + "d"
}
