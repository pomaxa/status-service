package main

// @title Status Incident API
// @version 1.0
// @description Internal service for monitoring system status and tracking incidents
// @host localhost:8080
// @BasePath /api

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"status-incident/internal/application"
	"status-incident/internal/infrastructure/http_checker"
	"status-incident/internal/infrastructure/sqlite"
	httpserver "status-incident/internal/interfaces/http"
	"status-incident/internal/interfaces/background"

	_ "status-incident/docs" // Swagger docs
)

// Build information (set via ldflags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Parse flags
	addr := flag.String("addr", ":8080", "HTTP server address")
	dbPath := flag.String("db", "status.db", "SQLite database path")
	templateDir := flag.String("templates", "templates", "Templates directory")
	heartbeatInterval := flag.Duration("heartbeat", 60*time.Second, "Heartbeat check interval")
	showVersion := flag.Bool("version", false, "Show version and exit")

	// Auth flags
	authEnabled := flag.Bool("auth", false, "Enable authentication")
	authUser := flag.String("auth-user", "admin", "Admin username")
	authPass := flag.String("auth-pass", "", "Admin password (required if auth enabled)")

	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("Status Incident Service\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Commit:     %s\n", Commit)
		fmt.Printf("  Build time: %s\n", BuildTime)
		os.Exit(0)
	}

	log.Printf("Starting Status Incident Service v%s (commit: %s)", Version, Commit)

	// Initialize database
	db, err := sqlite.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	systemRepo := sqlite.NewSystemRepo(db)
	depRepo := sqlite.NewDependencyRepo(db)
	logRepo := sqlite.NewLogRepo(db)
	analyticsRepo := sqlite.NewAnalyticsRepo(db)
	webhookRepo := sqlite.NewWebhookRepo(db)
	maintenanceRepo := sqlite.NewMaintenanceRepo(db)
	incidentRepo := sqlite.NewIncidentRepo(db)
	apiKeyRepo := sqlite.NewAPIKeyRepo(db)
	latencyRepo := sqlite.NewLatencyRepo(db)
	slaReportRepo := sqlite.NewSLAReportRepo(db)
	slaBreachRepo := sqlite.NewSLABreachRepo(db)

	// Initialize health checker
	checker := http_checker.New(10 * time.Second)

	// Initialize services
	systemService := application.NewSystemService(systemRepo, logRepo)
	depService := application.NewDependencyService(depRepo, logRepo)
	heartbeatService := application.NewHeartbeatService(depRepo, logRepo, checker)
	analyticsService := application.NewAnalyticsService(analyticsRepo, logRepo)
	maintenanceService := application.NewMaintenanceService(maintenanceRepo)
	incidentService := application.NewIncidentService(incidentRepo)
	latencyService := application.NewLatencyService(latencyRepo, depRepo)
	notificationService := application.NewNotificationService(webhookRepo, systemRepo, depRepo)
	slaService := application.NewSLAService(
		systemRepo, depRepo, analyticsRepo,
		slaReportRepo, slaBreachRepo, latencyRepo,
		notificationService,
	)

	// Initialize status propagation service
	propagationService := application.NewStatusPropagationService(systemRepo, depRepo, logRepo)
	propagationService.SetNotificationService(notificationService)

	// Set notification service on other services
	systemService.SetNotificationService(notificationService)
	depService.SetNotificationService(notificationService)
	heartbeatService.SetNotificationService(notificationService)
	heartbeatService.SetLatencyRepo(latencyRepo)

	// Set propagation service on services that can trigger status changes
	depService.SetPropagationService(propagationService)
	heartbeatService.SetPropagationService(propagationService)

	// Initialize webhook handlers
	webhookHandlers := httpserver.NewWebhookHandlers(webhookRepo, notificationService)

	// Initialize SLA handlers
	slaHandlers := httpserver.NewSLAHandlers(slaService)

	// Initialize auth middleware
	var authMiddleware *httpserver.AuthMiddleware
	var apiKeyHandlers *httpserver.APIKeyHandlers

	if *authEnabled {
		if *authPass == "" {
			log.Fatal("Auth password is required when auth is enabled (use -auth-pass)")
		}
		authMiddleware = httpserver.NewAuthMiddleware(true, *authUser, *authPass, apiKeyRepo)
		apiKeyHandlers = httpserver.NewAPIKeyHandlers(apiKeyRepo)
		log.Printf("Authentication enabled (user: %s)", *authUser)
	}

	// Initialize HTTP server
	server := httpserver.NewServer(
		systemService,
		depService,
		heartbeatService,
		analyticsService,
		maintenanceService,
		incidentService,
		latencyService,
		slaService,
		webhookHandlers,
		slaHandlers,
		apiKeyHandlers,
		authMiddleware,
		*templateDir,
	)

	// Initialize heartbeat worker
	heartbeatWorker := background.NewHeartbeatWorker(heartbeatService, *heartbeatInterval)

	// Start heartbeat worker
	ctx, cancel := context.WithCancel(context.Background())
	heartbeatWorker.Start(ctx)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         *addr,
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Printf("HTTP server listening on %s", *addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// Stop heartbeat worker
	cancel()
	heartbeatWorker.Stop()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Shutdown complete")
}
