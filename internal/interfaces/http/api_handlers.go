package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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
