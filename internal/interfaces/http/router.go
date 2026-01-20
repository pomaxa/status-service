package http

import (
	"net/http"
	"status-incident/internal/application"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Server represents the HTTP server
type Server struct {
	router             *chi.Mux
	systemService      *application.SystemService
	depService         *application.DependencyService
	heartbeatService   *application.HeartbeatService
	analyticsService   *application.AnalyticsService
	maintenanceService *application.MaintenanceService
	incidentService    *application.IncidentService
	latencyService     *application.LatencyService
	slaService         *application.SLAService
	webhookHandlers    *WebhookHandlers
	slaHandlers        *SLAHandlers
	apiKeyHandlers     *APIKeyHandlers
	authMiddleware     *AuthMiddleware
	templateDir        string
}

// NewServer creates a new HTTP server
func NewServer(
	systemService *application.SystemService,
	depService *application.DependencyService,
	heartbeatService *application.HeartbeatService,
	analyticsService *application.AnalyticsService,
	maintenanceService *application.MaintenanceService,
	incidentService *application.IncidentService,
	latencyService *application.LatencyService,
	slaService *application.SLAService,
	webhookHandlers *WebhookHandlers,
	slaHandlers *SLAHandlers,
	apiKeyHandlers *APIKeyHandlers,
	authMiddleware *AuthMiddleware,
	templateDir string,
) *Server {
	s := &Server{
		router:             chi.NewRouter(),
		systemService:      systemService,
		depService:         depService,
		heartbeatService:   heartbeatService,
		analyticsService:   analyticsService,
		maintenanceService: maintenanceService,
		incidentService:    incidentService,
		latencyService:     latencyService,
		slaService:         slaService,
		webhookHandlers:    webhookHandlers,
		slaHandlers:        slaHandlers,
		apiKeyHandlers:     apiKeyHandlers,
		authMiddleware:     authMiddleware,
		templateDir:        templateDir,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RealIP)

	// Static files
	fs := http.FileServer(http.Dir("static"))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fs))

	// Swagger documentation
	s.router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Public routes (no auth required)
	s.router.Get("/status", s.handlePublicStatus)
	s.router.Get("/metrics", s.handleMetrics)

	// Auth routes
	if s.authMiddleware != nil && s.authMiddleware.IsEnabled() {
		s.router.Get("/login", s.authMiddleware.LoginHandler)
		s.router.Post("/login", s.authMiddleware.LoginHandler)
		s.router.Get("/logout", s.authMiddleware.LogoutHandler)
	}

	// Protected Web UI routes (admin)
	s.router.Group(func(r chi.Router) {
		if s.authMiddleware != nil && s.authMiddleware.IsEnabled() {
			r.Use(s.authMiddleware.RequireAuth)
		}
		r.Get("/", s.handleDashboard)
		r.Get("/systems/{id}", s.handleSystemDetail)
		r.Get("/admin", s.handleAdmin)
		r.Get("/logs", s.handleLogs)
		r.Get("/analytics", s.handleAnalyticsPage)
		r.Get("/sla", s.handleSLAPage)
	})

	// REST API routes
	s.router.Route("/api", func(r chi.Router) {
		r.Use(jsonContentType)
		if s.authMiddleware != nil && s.authMiddleware.IsEnabled() {
			r.Use(s.authMiddleware.RequireAPIAuth)
		}

		// Systems
		r.Get("/systems", s.apiGetSystems)
		r.Post("/systems", s.apiCreateSystem)
		r.Get("/systems/{id}", s.apiGetSystem)
		r.Put("/systems/{id}", s.apiUpdateSystem)
		r.Delete("/systems/{id}", s.apiDeleteSystem)
		r.Post("/systems/{id}/status", s.apiUpdateSystemStatus)
		r.Get("/systems/{id}/logs", s.apiGetSystemLogs)
		r.Get("/systems/{id}/analytics", s.apiGetSystemAnalytics)

		// Dependencies
		r.Get("/systems/{systemId}/dependencies", s.apiGetDependencies)
		r.Post("/systems/{systemId}/dependencies", s.apiCreateDependency)
		r.Get("/dependencies/{id}", s.apiGetDependency)
		r.Put("/dependencies/{id}", s.apiUpdateDependency)
		r.Delete("/dependencies/{id}", s.apiDeleteDependency)
		r.Post("/dependencies/{id}/status", s.apiUpdateDependencyStatus)
		r.Post("/dependencies/{id}/heartbeat", s.apiSetHeartbeat)
		r.Delete("/dependencies/{id}/heartbeat", s.apiClearHeartbeat)
		r.Post("/dependencies/{id}/check", s.apiForceCheck)
		r.Get("/dependencies/{id}/logs", s.apiGetDependencyLogs)
		r.Get("/dependencies/{id}/analytics", s.apiGetDependencyAnalytics)
		r.Get("/dependencies/{id}/latency", s.apiGetDependencyLatency)
		r.Get("/dependencies/{id}/uptime", s.apiGetDependencyUptime)

		// Logs
		r.Get("/logs", s.apiGetAllLogs)

		// Analytics
		r.Get("/analytics", s.apiGetOverallAnalytics)

		// Export/Import
		r.Get("/export", s.apiExportAll)
		r.Get("/export/logs", s.apiExportLogs)
		r.Post("/import", s.apiImportAll)

		// Webhooks
		r.Get("/webhooks", s.webhookHandlers.ListWebhooks)
		r.Post("/webhooks", s.webhookHandlers.CreateWebhook)
		r.Get("/webhooks/{id}", s.webhookHandlers.GetWebhook)
		r.Put("/webhooks/{id}", s.webhookHandlers.UpdateWebhook)
		r.Delete("/webhooks/{id}", s.webhookHandlers.DeleteWebhook)
		r.Post("/webhooks/{id}/test", s.webhookHandlers.TestWebhook)

		// Maintenance windows
		r.Get("/maintenances", s.apiGetMaintenances)
		r.Post("/maintenances", s.apiCreateMaintenance)
		r.Get("/maintenances/active", s.apiGetActiveMaintenances)
		r.Get("/maintenances/upcoming", s.apiGetUpcomingMaintenances)
		r.Get("/maintenances/{id}", s.apiGetMaintenance)
		r.Put("/maintenances/{id}", s.apiUpdateMaintenance)
		r.Delete("/maintenances/{id}", s.apiDeleteMaintenance)
		r.Post("/maintenances/{id}/cancel", s.apiCancelMaintenance)

		// Incidents
		r.Get("/incidents", s.apiGetIncidents)
		r.Post("/incidents", s.apiCreateIncident)
		r.Get("/incidents/active", s.apiGetActiveIncidents)
		r.Get("/incidents/recent", s.apiGetRecentIncidents)
		r.Get("/incidents/{id}", s.apiGetIncident)
		r.Delete("/incidents/{id}", s.apiDeleteIncident)
		r.Post("/incidents/{id}/acknowledge", s.apiAcknowledgeIncident)
		r.Post("/incidents/{id}/status", s.apiUpdateIncidentStatus)
		r.Post("/incidents/{id}/resolve", s.apiResolveIncident)
		r.Get("/incidents/{id}/updates", s.apiGetIncidentUpdates)
		r.Post("/incidents/{id}/updates", s.apiAddIncidentUpdate)

		// API Keys (only if auth is enabled)
		if s.apiKeyHandlers != nil {
			r.Get("/apikeys", s.apiKeyHandlers.ListAPIKeys)
			r.Post("/apikeys", s.apiKeyHandlers.CreateAPIKey)
			r.Delete("/apikeys/{id}", s.apiKeyHandlers.DeleteAPIKey)
			r.Put("/apikeys/{id}/toggle", s.apiKeyHandlers.ToggleAPIKey)
		}

		// SLA Reports and Breaches
		if s.slaHandlers != nil {
			r.Post("/sla/reports", s.slaHandlers.GenerateReport)
			r.Get("/sla/reports", s.slaHandlers.GetReports)
			r.Get("/sla/reports/{id}", s.slaHandlers.GetReport)
			r.Delete("/sla/reports/{id}", s.slaHandlers.DeleteReport)
			r.Get("/sla/breaches", s.slaHandlers.GetBreaches)
			r.Post("/sla/breaches/check", s.slaHandlers.CheckBreaches)
			r.Post("/sla/breaches/{id}/acknowledge", s.slaHandlers.AcknowledgeBreach)
			r.Get("/systems/{id}/sla", s.slaHandlers.GetSystemSLA)
			r.Put("/systems/{id}/sla-target", s.slaHandlers.UpdateSystemSLATarget)
			r.Get("/systems/{id}/sla/breaches", s.slaHandlers.GetSystemBreaches)
		}
	})
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
