package http

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"status-incident/internal/domain"

	"github.com/go-chi/chi/v5"
)

// Template data structures
type dashboardData struct {
	Systems    []*systemWithDeps
	Analytics  *domain.Analytics
}

type systemWithDeps struct {
	*domain.System
	Dependencies []*domain.Dependency
	Analytics    *domain.Analytics
}

type systemDetailData struct {
	System       *domain.System
	Dependencies []*domain.Dependency
	Logs         []*domain.StatusLog
	Analytics    *domain.Analytics
}

type adminData struct {
	Systems []*systemWithDeps
}

type logsData struct {
	Logs []*logWithContext
}

type logWithContext struct {
	*domain.StatusLog
	SystemName     string
	DependencyName string
}

type analyticsPageData struct {
	Overall  *domain.Analytics
	Systems  []*systemWithDeps
}

type slaPageData struct {
	Reports     []*domain.SLAReport
	Breaches    []*domain.SLABreachEvent
	Systems     []*systemWithSLA
}

type systemWithSLA struct {
	*domain.System
	SLAStatus *domain.SystemSLAReport
}

type publicStatusData struct {
	Title               string
	Systems             []*systemWithDeps
	ActiveMaintenance   []*maintenanceInfo
	UpcomingMaintenance []*maintenanceInfo
	ActiveIncidents     []*incidentInfo
	UpdatedAt           string
}

type maintenanceInfo struct {
	ID          int64
	Title       string
	Description string
	StartTime   string
	EndTime     string
	Status      string
}

type incidentInfo struct {
	ID        int64
	Title     string
	Status    string
	Severity  string
	Message   string
	CreatedAt string
	UpdatedAt string
}

func (s *Server) getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"statusClass": func(status domain.Status) string {
			switch status {
			case domain.StatusGreen:
				return "status-green"
			case domain.StatusYellow:
				return "status-yellow"
			case domain.StatusRed:
				return "status-red"
			}
			return ""
		},
		"statusIcon": func(status domain.Status) string {
			switch status {
			case domain.StatusGreen:
				return "operational"
			case domain.StatusYellow:
				return "degraded"
			case domain.StatusRed:
				return "outage"
			}
			return ""
		},
		"statusText": func(status domain.Status) string {
			switch status {
			case domain.StatusGreen:
				return "Operational"
			case domain.StatusYellow:
				return "Degraded"
			case domain.StatusRed:
				return "Outage"
			}
			return "Unknown"
		},
		"statusTextClass": func(status domain.Status) string {
			switch status {
			case domain.StatusGreen:
				return "operational"
			case domain.StatusYellow:
				return "degraded"
			case domain.StatusRed:
				return "outage"
			}
			return ""
		},
		"overallStatusClass": func(systems []*systemWithDeps) string {
			worst := domain.StatusGreen
			for _, sys := range systems {
				if sys.Status == domain.StatusRed {
					return "status-red"
				}
				if sys.Status == domain.StatusYellow && worst != domain.StatusRed {
					worst = domain.StatusYellow
				}
				for _, dep := range sys.Dependencies {
					if dep.Status == domain.StatusRed {
						return "status-red"
					}
					if dep.Status == domain.StatusYellow && worst != domain.StatusRed {
						worst = domain.StatusYellow
					}
				}
			}
			switch worst {
			case domain.StatusYellow:
				return "status-yellow"
			default:
				return "status-green"
			}
		},
		"overallStatusText": func(systems []*systemWithDeps) string {
			for _, sys := range systems {
				if sys.Status == domain.StatusRed {
					return "Major Outage"
				}
				for _, dep := range sys.Dependencies {
					if dep.Status == domain.StatusRed {
						return "Partial Outage"
					}
				}
			}
			for _, sys := range systems {
				if sys.Status == domain.StatusYellow {
					return "Degraded Performance"
				}
				for _, dep := range sys.Dependencies {
					if dep.Status == domain.StatusYellow {
						return "Degraded Performance"
					}
				}
			}
			return "All Systems Operational"
		},
		"formatDuration": func(d interface{}) string {
			switch v := d.(type) {
			case int64:
				return formatDurationNs(v)
			default:
				return "N/A"
			}
		},
		"formatPercent": func(p float64) string {
			return formatPercent(p)
		},
		"headersJSON": func(headers map[string]string) string {
			if len(headers) == 0 {
				return ""
			}
			data, err := json.Marshal(headers)
			if err != nil {
				return ""
			}
			return string(data)
		},
	}
}

func (s *Server) loadTemplate(name string) (*template.Template, error) {
	layoutPath := filepath.Join(s.templateDir, "layout.html")
	tmplPath := filepath.Join(s.templateDir, name+".html")

	return template.New("layout.html").Funcs(s.getTemplateFuncs()).ParseFiles(layoutPath, tmplPath)
}

func (s *Server) loadStandaloneTemplate(name string) (*template.Template, error) {
	tmplPath := filepath.Join(s.templateDir, name+".html")
	return template.New(name + ".html").Funcs(s.getTemplateFuncs()).ParseFiles(tmplPath)
}

func formatDurationNs(ns int64) string {
	if ns == 0 {
		return "0s"
	}

	seconds := ns / 1e9
	if seconds < 60 {
		return formatSeconds(seconds)
	}

	minutes := seconds / 60
	if minutes < 60 {
		return formatMinutes(minutes, seconds%60)
	}

	hours := minutes / 60
	if hours < 24 {
		return formatHours(hours, minutes%60)
	}

	days := hours / 24
	return formatDays(days, hours%24)
}

func formatSeconds(s int64) string {
	return formatValue(s, "s")
}

func formatMinutes(m, s int64) string {
	if s > 0 {
		return formatValue(m, "m ") + formatSeconds(s)
	}
	return formatValue(m, "m")
}

func formatHours(h, m int64) string {
	if m > 0 {
		return formatValue(h, "h ") + formatValue(m, "m")
	}
	return formatValue(h, "h")
}

func formatDays(d, h int64) string {
	if h > 0 {
		return formatValue(d, "d ") + formatValue(h, "h")
	}
	return formatValue(d, "d")
}

func formatValue(v int64, unit string) string {
	return string(rune('0'+v%10+v/10*10)) + unit
}

func formatPercent(p float64) string {
	if p >= 99.99 {
		return "99.99%"
	}
	return string(rune(int('0'+int(p)/10))) + string(rune(int('0'+int(p)%10))) + "." + string(rune(int('0'+int(p*10)%10))) + string(rune(int('0'+int(p*100)%10))) + "%"
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	systems, err := s.systemService.GetAllSystems(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var systemsWithDeps []*systemWithDeps
	for _, sys := range systems {
		deps, _ := s.depService.GetDependenciesBySystem(r.Context(), sys.ID)
		analytics, _ := s.analyticsService.GetSystemAnalytics(r.Context(), sys.ID, "24h")
		systemsWithDeps = append(systemsWithDeps, &systemWithDeps{
			System:       sys,
			Dependencies: deps,
			Analytics:    analytics,
		})
	}

	overall, _ := s.analyticsService.GetOverallAnalytics(r.Context(), "24h")

	tmpl, err := s.loadTemplate("dashboard")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, dashboardData{
		Systems:   systemsWithDeps,
		Analytics: overall,
	})
}

func (s *Server) handleSystemDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.Error(w, "invalid system ID", http.StatusBadRequest)
		return
	}

	system, err := s.systemService.GetSystem(r.Context(), id)
	if err != nil || system == nil {
		http.Error(w, "system not found", http.StatusNotFound)
		return
	}

	deps, _ := s.depService.GetDependenciesBySystem(r.Context(), id)
	logs, _ := s.systemService.GetSystemLogs(r.Context(), id, 50)
	analytics, _ := s.analyticsService.GetSystemAnalytics(r.Context(), id, "24h")

	tmpl, err := s.loadTemplate("system")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, systemDetailData{
		System:       system,
		Dependencies: deps,
		Logs:         logs,
		Analytics:    analytics,
	})
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	systems, err := s.systemService.GetAllSystems(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var systemsWithDeps []*systemWithDeps
	for _, sys := range systems {
		deps, _ := s.depService.GetDependenciesBySystem(r.Context(), sys.ID)
		systemsWithDeps = append(systemsWithDeps, &systemWithDeps{
			System:       sys,
			Dependencies: deps,
		})
	}

	tmpl, err := s.loadTemplate("admin")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, adminData{Systems: systemsWithDeps})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.analyticsService.GetAllLogs(r.Context(), 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich logs with system/dependency names
	systemCache := make(map[int64]string)
	depCache := make(map[int64]string)

	var enrichedLogs []*logWithContext
	for _, log := range logs {
		lc := &logWithContext{StatusLog: log}

		if log.SystemID != nil {
			if name, ok := systemCache[*log.SystemID]; ok {
				lc.SystemName = name
			} else {
				if sys, err := s.systemService.GetSystem(r.Context(), *log.SystemID); err == nil && sys != nil {
					lc.SystemName = sys.Name
					systemCache[*log.SystemID] = sys.Name
				}
			}
		}

		if log.DependencyID != nil {
			if name, ok := depCache[*log.DependencyID]; ok {
				lc.DependencyName = name
			} else {
				if dep, err := s.depService.GetDependency(r.Context(), *log.DependencyID); err == nil && dep != nil {
					lc.DependencyName = dep.Name
					depCache[*log.DependencyID] = dep.Name
				}
			}
		}

		enrichedLogs = append(enrichedLogs, lc)
	}

	tmpl, err := s.loadTemplate("logs")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, logsData{Logs: enrichedLogs})
}

func (s *Server) handleAnalyticsPage(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	overall, _ := s.analyticsService.GetOverallAnalytics(r.Context(), period)

	systems, _ := s.systemService.GetAllSystems(r.Context())
	var systemsWithDeps []*systemWithDeps
	for _, sys := range systems {
		deps, _ := s.depService.GetDependenciesBySystem(r.Context(), sys.ID)
		analytics, _ := s.analyticsService.GetSystemAnalytics(r.Context(), sys.ID, period)
		systemsWithDeps = append(systemsWithDeps, &systemWithDeps{
			System:       sys,
			Dependencies: deps,
			Analytics:    analytics,
		})
	}

	tmpl, err := s.loadTemplate("analytics")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, analyticsPageData{
		Overall: overall,
		Systems: systemsWithDeps,
	})
}

// parseID from chi URL params
func parseIDFromChi(r *http.Request, param string) (int64, error) {
	idStr := chi.URLParam(r, param)
	var id int64
	for _, c := range idStr {
		if c < '0' || c > '9' {
			return 0, nil
		}
		id = id*10 + int64(c-'0')
	}
	return id, nil
}

func (s *Server) handlePublicStatus(w http.ResponseWriter, r *http.Request) {
	systems, err := s.systemService.GetAllSystems(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var systemsWithDeps []*systemWithDeps
	for _, sys := range systems {
		deps, _ := s.depService.GetDependenciesBySystem(r.Context(), sys.ID)
		systemsWithDeps = append(systemsWithDeps, &systemWithDeps{
			System:       sys,
			Dependencies: deps,
		})
	}

	// Get maintenance info
	var activeMaintenance []*maintenanceInfo
	var upcomingMaintenance []*maintenanceInfo

	if s.maintenanceService != nil {
		actives, _ := s.maintenanceService.GetActiveMaintenances(r.Context())
		for _, m := range actives {
			activeMaintenance = append(activeMaintenance, &maintenanceInfo{
				ID:          m.ID,
				Title:       m.Title,
				Description: m.Description,
				StartTime:   m.StartTime.Format("Jan 2, 15:04"),
				EndTime:     m.EndTime.Format("Jan 2, 15:04"),
				Status:      string(m.Status),
			})
		}

		upcoming, _ := s.maintenanceService.GetUpcomingMaintenances(r.Context())
		for _, m := range upcoming {
			upcomingMaintenance = append(upcomingMaintenance, &maintenanceInfo{
				ID:          m.ID,
				Title:       m.Title,
				Description: m.Description,
				StartTime:   m.StartTime.Format("Jan 2, 15:04"),
				EndTime:     m.EndTime.Format("Jan 2, 15:04"),
				Status:      string(m.Status),
			})
		}
	}

	// Get active incidents
	var activeIncidents []*incidentInfo
	if s.incidentService != nil {
		incidents, _ := s.incidentService.GetActiveIncidents(r.Context())
		for _, inc := range incidents {
			activeIncidents = append(activeIncidents, &incidentInfo{
				ID:        inc.ID,
				Title:     inc.Title,
				Status:    string(inc.Status),
				Severity:  string(inc.Severity),
				Message:   inc.Message,
				CreatedAt: inc.CreatedAt.Format("Jan 2, 15:04"),
				UpdatedAt: inc.UpdatedAt.Format("Jan 2, 15:04"),
			})
		}
	}

	tmpl, err := s.loadStandaloneTemplate("public")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, publicStatusData{
		Title:               "System Status",
		Systems:             systemsWithDeps,
		ActiveMaintenance:   activeMaintenance,
		UpcomingMaintenance: upcomingMaintenance,
		ActiveIncidents:     activeIncidents,
		UpdatedAt:           formatTimeAgo(),
	})
}

func formatTimeAgo() string {
	return "just now"
}

// handleMetrics returns Prometheus-compatible metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	systems, _ := s.systemService.GetAllSystems(ctx)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// System status metrics
	w.Write([]byte("# HELP status_incident_system_status System status (0=green, 1=yellow, 2=red)\n"))
	w.Write([]byte("# TYPE status_incident_system_status gauge\n"))

	w.Write([]byte("# HELP status_incident_system_sla_target SLA target percentage\n"))
	w.Write([]byte("# TYPE status_incident_system_sla_target gauge\n"))

	// Dependency metrics
	w.Write([]byte("# HELP status_incident_dependency_status Dependency status (0=green, 1=yellow, 2=red)\n"))
	w.Write([]byte("# TYPE status_incident_dependency_status gauge\n"))

	w.Write([]byte("# HELP status_incident_dependency_latency_ms Last check latency in milliseconds\n"))
	w.Write([]byte("# TYPE status_incident_dependency_latency_ms gauge\n"))

	w.Write([]byte("# HELP status_incident_dependency_consecutive_failures Number of consecutive check failures\n"))
	w.Write([]byte("# TYPE status_incident_dependency_consecutive_failures gauge\n"))

	// Counter metrics
	w.Write([]byte("# HELP status_incident_systems_total Total number of systems\n"))
	w.Write([]byte("# TYPE status_incident_systems_total gauge\n"))

	w.Write([]byte("# HELP status_incident_dependencies_total Total number of dependencies\n"))
	w.Write([]byte("# TYPE status_incident_dependencies_total gauge\n"))

	// Incident metrics
	w.Write([]byte("# HELP status_incident_incidents_active Number of active incidents\n"))
	w.Write([]byte("# TYPE status_incident_incidents_active gauge\n"))

	w.Write([]byte("# HELP status_incident_incidents_total Total number of incidents\n"))
	w.Write([]byte("# TYPE status_incident_incidents_total gauge\n"))

	w.Write([]byte("# HELP status_incident_incidents_by_severity Incidents by severity\n"))
	w.Write([]byte("# TYPE status_incident_incidents_by_severity gauge\n"))

	w.Write([]byte("# HELP status_incident_incidents_by_status Incidents by status\n"))
	w.Write([]byte("# TYPE status_incident_incidents_by_status gauge\n"))

	// Maintenance metrics
	w.Write([]byte("# HELP status_incident_maintenances_active Number of active maintenance windows\n"))
	w.Write([]byte("# TYPE status_incident_maintenances_active gauge\n"))

	w.Write([]byte("# HELP status_incident_maintenances_scheduled Number of scheduled maintenance windows\n"))
	w.Write([]byte("# TYPE status_incident_maintenances_scheduled gauge\n"))

	// SLA metrics
	w.Write([]byte("# HELP status_incident_sla_breaches_unacknowledged Number of unacknowledged SLA breaches\n"))
	w.Write([]byte("# TYPE status_incident_sla_breaches_unacknowledged gauge\n"))

	// Uptime metrics
	w.Write([]byte("# HELP status_incident_uptime_24h System uptime percentage over last 24 hours\n"))
	w.Write([]byte("# TYPE status_incident_uptime_24h gauge\n"))

	totalDeps := 0

	// System and dependency metrics
	for _, sys := range systems {
		sysIDStr := intToStr(sys.ID)

		statusVal := statusToInt(sys.Status)
		w.Write([]byte(formatMetricLine("status_incident_system_status", statusVal,
			"system_id", sysIDStr,
			"system_name", sys.Name)))

		w.Write([]byte(formatMetricLine("status_incident_system_sla_target", sys.SLATarget,
			"system_id", sysIDStr,
			"system_name", sys.Name)))

		// Get uptime for this system
		if analytics, err := s.analyticsService.GetSystemAnalytics(ctx, sys.ID, "24h"); err == nil {
			w.Write([]byte(formatMetricLine("status_incident_uptime_24h", analytics.UptimePercent,
				"system_id", sysIDStr,
				"system_name", sys.Name)))
		}

		deps, _ := s.depService.GetDependenciesBySystem(ctx, sys.ID)
		totalDeps += len(deps)

		for _, dep := range deps {
			depIDStr := intToStr(dep.ID)

			depStatusVal := statusToInt(dep.Status)
			w.Write([]byte(formatMetricLine("status_incident_dependency_status", depStatusVal,
				"system_id", sysIDStr,
				"system_name", sys.Name,
				"dependency_id", depIDStr,
				"dependency_name", dep.Name)))

			if dep.LastLatency > 0 {
				w.Write([]byte(formatMetricLine("status_incident_dependency_latency_ms", dep.LastLatency,
					"system_id", sysIDStr,
					"system_name", sys.Name,
					"dependency_id", depIDStr,
					"dependency_name", dep.Name)))
			}

			w.Write([]byte(formatMetricLine("status_incident_dependency_consecutive_failures", dep.ConsecutiveFailures,
				"system_id", sysIDStr,
				"system_name", sys.Name,
				"dependency_id", depIDStr,
				"dependency_name", dep.Name)))
		}
	}

	// Write totals
	w.Write([]byte(formatMetricLine("status_incident_systems_total", len(systems))))
	w.Write([]byte(formatMetricLine("status_incident_dependencies_total", totalDeps)))

	// Incident metrics
	if s.incidentService != nil {
		activeIncidents, _ := s.incidentService.GetActiveIncidents(ctx)
		w.Write([]byte(formatMetricLine("status_incident_incidents_active", len(activeIncidents))))

		allIncidents, _ := s.incidentService.GetAllIncidents(ctx, 1000)
		w.Write([]byte(formatMetricLine("status_incident_incidents_total", len(allIncidents))))

		// Count by severity and status
		severityCounts := map[string]int{"minor": 0, "major": 0, "critical": 0}
		statusCounts := map[string]int{"investigating": 0, "identified": 0, "monitoring": 0, "resolved": 0}

		for _, inc := range allIncidents {
			severityCounts[string(inc.Severity)]++
			statusCounts[string(inc.Status)]++
		}

		for severity, count := range severityCounts {
			w.Write([]byte(formatMetricLine("status_incident_incidents_by_severity", count,
				"severity", severity)))
		}

		for status, count := range statusCounts {
			w.Write([]byte(formatMetricLine("status_incident_incidents_by_status", count,
				"status", status)))
		}
	}

	// Maintenance metrics
	if s.maintenanceService != nil {
		activeMaints, _ := s.maintenanceService.GetActiveMaintenances(ctx)
		w.Write([]byte(formatMetricLine("status_incident_maintenances_active", len(activeMaints))))

		upcomingMaints, _ := s.maintenanceService.GetUpcomingMaintenances(ctx)
		w.Write([]byte(formatMetricLine("status_incident_maintenances_scheduled", len(upcomingMaints))))
	}

	// SLA breach metrics
	if s.slaService != nil {
		breaches, _ := s.slaService.GetUnacknowledgedBreaches(ctx)
		w.Write([]byte(formatMetricLine("status_incident_sla_breaches_unacknowledged", len(breaches))))
	}
}

func statusToInt(status domain.Status) int {
	switch status {
	case domain.StatusGreen:
		return 0
	case domain.StatusYellow:
		return 1
	case domain.StatusRed:
		return 2
	}
	return -1
}

func intToStr(i int64) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

func formatMetricLine(name string, value interface{}, labels ...string) string {
	result := name
	if len(labels) > 0 {
		result += "{"
		for i := 0; i < len(labels); i += 2 {
			if i > 0 {
				result += ","
			}
			result += labels[i] + "=\"" + escapeLabel(labels[i+1]) + "\""
		}
		result += "}"
	}
	switch v := value.(type) {
	case int:
		result += " " + intToStrPlain(v)
	case int64:
		result += " " + intToStr(v)
	case float64:
		result += " " + formatFloat(v)
	}
	result += "\n"
	return result
}

func intToStrPlain(i int) string {
	if i == 0 {
		return "0"
	}
	negative := i < 0
	if negative {
		i = -i
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func formatFloat(f float64) string {
	// Simple float formatting
	intPart := int64(f)
	fracPart := int64((f - float64(intPart)) * 100)
	if fracPart < 0 {
		fracPart = -fracPart
	}
	return intToStr(intPart) + "." + padLeft(intToStrPlain(int(fracPart)), 2, '0')
}

func padLeft(s string, length int, pad byte) string {
	for len(s) < length {
		s = string(pad) + s
	}
	return s
}

func escapeLabel(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			result = append(result, '\\', '\\')
		case '"':
			result = append(result, '\\', '"')
		case '\n':
			result = append(result, '\\', 'n')
		default:
			result = append(result, c)
		}
	}
	return string(result)
}

func (s *Server) handleSLAPage(w http.ResponseWriter, r *http.Request) {
	if s.slaService == nil {
		http.Error(w, "SLA service not available", http.StatusServiceUnavailable)
		return
	}

	reports, _ := s.slaService.GetAllReports(r.Context(), 10)
	breaches, _ := s.slaService.GetUnacknowledgedBreaches(r.Context())

	systems, _ := s.systemService.GetAllSystems(r.Context())
	var systemsWithSLA []*systemWithSLA
	for _, sys := range systems {
		slaStatus, _ := s.slaService.GetSystemSLAStatus(r.Context(), sys.ID, "monthly")
		systemsWithSLA = append(systemsWithSLA, &systemWithSLA{
			System:    sys,
			SLAStatus: slaStatus,
		})
	}

	tmpl, err := s.loadTemplate("sla")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, slaPageData{
		Reports:  reports,
		Breaches: breaches,
		Systems:  systemsWithSLA,
	})
}
