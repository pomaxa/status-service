package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"status-incident/internal/domain"
)

// Export/Import data structures
type ExportData struct {
	ExportedAt   time.Time          `json:"exported_at"`
	Version      string             `json:"version"`
	Systems      []ExportSystem     `json:"systems"`
	Dependencies []ExportDependency `json:"dependencies"`
	Logs         []ExportLog        `json:"logs"`
}

type ExportSystem struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Owner       string    `json:"owner"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ExportDependency struct {
	ID                  int64     `json:"id"`
	SystemID            int64     `json:"system_id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	Status              string    `json:"status"`
	HeartbeatURL        string    `json:"heartbeat_url,omitempty"`
	HeartbeatInterval   int       `json:"heartbeat_interval,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastCheck           time.Time `json:"last_check,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ExportLog struct {
	ID           int64     `json:"id"`
	SystemID     *int64    `json:"system_id,omitempty"`
	DependencyID *int64    `json:"dependency_id,omitempty"`
	OldStatus    string    `json:"old_status"`
	NewStatus    string    `json:"new_status"`
	Message      string    `json:"message,omitempty"`
	Source       string    `json:"source"`
	CreatedAt    time.Time `json:"created_at"`
}

type ImportResult struct {
	SystemsImported      int      `json:"systems_imported"`
	DependenciesImported int      `json:"dependencies_imported"`
	LogsImported         int      `json:"logs_imported"`
	Errors               []string `json:"errors,omitempty"`
}

// apiExportAll exports all data (systems, dependencies, logs)
func (s *Server) apiExportAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all systems
	systems, err := s.systemService.GetAllSystems(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get all logs
	logs, err := s.analyticsService.GetAllLogs(ctx, 10000)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	export := ExportData{
		ExportedAt:   time.Now(),
		Version:      "1.0",
		Systems:      make([]ExportSystem, 0),
		Dependencies: make([]ExportDependency, 0),
		Logs:         make([]ExportLog, 0),
	}

	// Convert systems
	for _, sys := range systems {
		export.Systems = append(export.Systems, ExportSystem{
			ID:          sys.ID,
			Name:        sys.Name,
			Description: sys.Description,
			URL:         sys.URL,
			Owner:       sys.Owner,
			Status:      sys.Status.String(),
			CreatedAt:   sys.CreatedAt,
			UpdatedAt:   sys.UpdatedAt,
		})

		// Get dependencies for each system
		deps, err := s.depService.GetDependenciesBySystem(ctx, sys.ID)
		if err != nil {
			continue
		}

		for _, dep := range deps {
			export.Dependencies = append(export.Dependencies, ExportDependency{
				ID:                  dep.ID,
				SystemID:            dep.SystemID,
				Name:                dep.Name,
				Description:         dep.Description,
				Status:              dep.Status.String(),
				HeartbeatURL:        dep.HeartbeatURL,
				HeartbeatInterval:   dep.HeartbeatInterval,
				ConsecutiveFailures: dep.ConsecutiveFailures,
				LastCheck:           dep.LastCheck,
				CreatedAt:           dep.CreatedAt,
				UpdatedAt:           dep.UpdatedAt,
			})
		}
	}

	// Convert logs
	for _, log := range logs {
		export.Logs = append(export.Logs, ExportLog{
			ID:           log.ID,
			SystemID:     log.SystemID,
			DependencyID: log.DependencyID,
			OldStatus:    log.OldStatus.String(),
			NewStatus:    log.NewStatus.String(),
			Message:      log.Message,
			Source:       string(log.Source),
			CreatedAt:    log.CreatedAt,
		})
	}

	// Set download headers
	filename := fmt.Sprintf("status-incident-export-%s.json", time.Now().Format("2006-01-02-150405"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	s.respondJSON(w, http.StatusOK, export)
}

// apiExportLogs exports only logs
func (s *Server) apiExportLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logs, err := s.analyticsService.GetAllLogs(ctx, 10000)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	exportLogs := make([]ExportLog, 0, len(logs))
	for _, log := range logs {
		exportLogs = append(exportLogs, ExportLog{
			ID:           log.ID,
			SystemID:     log.SystemID,
			DependencyID: log.DependencyID,
			OldStatus:    log.OldStatus.String(),
			NewStatus:    log.NewStatus.String(),
			Message:      log.Message,
			Source:       string(log.Source),
			CreatedAt:    log.CreatedAt,
		})
	}

	export := struct {
		ExportedAt time.Time   `json:"exported_at"`
		Version    string      `json:"version"`
		Logs       []ExportLog `json:"logs"`
	}{
		ExportedAt: time.Now(),
		Version:    "1.0",
		Logs:       exportLogs,
	}

	filename := fmt.Sprintf("status-incident-logs-%s.json", time.Now().Format("2006-01-02-150405"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	s.respondJSON(w, http.StatusOK, export)
}

// apiImportAll imports all data (systems, dependencies, logs)
func (s *Server) apiImportAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var data ExportData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid JSON format")
		return
	}

	result := ImportResult{
		Errors: make([]string, 0),
	}

	// Map old IDs to new IDs
	systemIDMap := make(map[int64]int64)
	depIDMap := make(map[int64]int64)

	// Import systems
	for _, expSys := range data.Systems {
		sys, err := s.systemService.CreateSystem(ctx, expSys.Name, expSys.Description, expSys.URL, expSys.Owner)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("system '%s': %v", expSys.Name, err))
			continue
		}

		// Update status if not green
		if expSys.Status != "green" {
			s.systemService.UpdateSystemStatus(ctx, sys.ID, expSys.Status, "Imported from backup")
		}

		systemIDMap[expSys.ID] = sys.ID
		result.SystemsImported++
	}

	// Import dependencies
	for _, expDep := range data.Dependencies {
		newSystemID, ok := systemIDMap[expDep.SystemID]
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("dependency '%s': system not found", expDep.Name))
			continue
		}

		dep, err := s.depService.CreateDependency(ctx, newSystemID, expDep.Name, expDep.Description)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("dependency '%s': %v", expDep.Name, err))
			continue
		}

		// Set heartbeat if configured
		if expDep.HeartbeatURL != "" {
			s.depService.SetHeartbeat(ctx, dep.ID, expDep.HeartbeatURL, expDep.HeartbeatInterval)
		}

		// Update status if not green
		if expDep.Status != "green" {
			s.depService.UpdateDependencyStatus(ctx, dep.ID, expDep.Status, "Imported from backup")
		}

		depIDMap[expDep.ID] = dep.ID
		result.DependenciesImported++
	}

	// Import logs (create new log entries with mapped IDs)
	for _, expLog := range data.Logs {
		var systemID, depID *int64

		if expLog.SystemID != nil {
			if newID, ok := systemIDMap[*expLog.SystemID]; ok {
				systemID = &newID
			}
		}

		if expLog.DependencyID != nil {
			if newID, ok := depIDMap[*expLog.DependencyID]; ok {
				depID = &newID
			}
		}

		// Skip logs that don't reference imported entities
		if systemID == nil && depID == nil {
			continue
		}

		oldStatus, _ := domain.NewStatus(expLog.OldStatus)
		newStatus, _ := domain.NewStatus(expLog.NewStatus)
		source := domain.ChangeSource(expLog.Source)

		log := domain.NewStatusLog(systemID, depID, oldStatus, newStatus, expLog.Message, source)
		if err := s.analyticsService.CreateLog(ctx, log); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("log: %v", err))
			continue
		}

		result.LogsImported++
	}

	s.respondJSON(w, http.StatusOK, result)
}
