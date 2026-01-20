package http

import (
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

type publicStatusData struct {
	Title     string
	Systems   []*systemWithDeps
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

	tmpl, err := s.loadStandaloneTemplate("public")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, publicStatusData{
		Title:     "System Status",
		Systems:   systemsWithDeps,
		UpdatedAt: formatTimeAgo(),
	})
}

func formatTimeAgo() string {
	return "just now"
}
