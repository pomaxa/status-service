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
	router           *chi.Mux
	systemService    *application.SystemService
	depService       *application.DependencyService
	heartbeatService *application.HeartbeatService
	analyticsService *application.AnalyticsService
	templateDir      string
}

// NewServer creates a new HTTP server
func NewServer(
	systemService *application.SystemService,
	depService *application.DependencyService,
	heartbeatService *application.HeartbeatService,
	analyticsService *application.AnalyticsService,
	templateDir string,
) *Server {
	s := &Server{
		router:           chi.NewRouter(),
		systemService:    systemService,
		depService:       depService,
		heartbeatService: heartbeatService,
		analyticsService: analyticsService,
		templateDir:      templateDir,
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

	// Web UI routes
	s.router.Get("/", s.handleDashboard)
	s.router.Get("/systems/{id}", s.handleSystemDetail)
	s.router.Get("/admin", s.handleAdmin)
	s.router.Get("/logs", s.handleLogs)
	s.router.Get("/analytics", s.handleAnalyticsPage)

	// REST API routes
	s.router.Route("/api", func(r chi.Router) {
		r.Use(jsonContentType)

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

		// Logs
		r.Get("/logs", s.apiGetAllLogs)

		// Analytics
		r.Get("/analytics", s.apiGetOverallAnalytics)

		// Export/Import
		r.Get("/export", s.apiExportAll)
		r.Get("/export/logs", s.apiExportLogs)
		r.Post("/import", s.apiImportAll)
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
