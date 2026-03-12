// Health endpoint example for Go applications.
//
// This demonstrates how to implement health check endpoints that work with
// the Status Incident Service's heartbeat monitoring feature.
//
// Run with:
//
//	go run health_endpoint.go
//
// Then configure Status Incident to monitor: http://localhost:8080/health
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks,omitempty"`
	Message string            `json:"message,omitempty"`
}

// HealthChecker performs health checks on dependencies.
type HealthChecker struct {
	db    *sql.DB
	redis interface{ Ping(context.Context) error }
}

// NewHealthChecker creates a new health checker.
// Replace with your actual database and cache connections.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		db:    nil, // Replace with: sql.Open("postgres", connStr)
		redis: nil, // Replace with: redis.NewClient(&redis.Options{...})
	}
}

// Check performs all health checks.
func (h *HealthChecker) Check(ctx context.Context) HealthResponse {
	response := HealthResponse{
		Status: "ok",
		Checks: make(map[string]string),
	}
	healthy := true

	// Database check
	if h.db != nil {
		if err := h.db.PingContext(ctx); err != nil {
			response.Checks["database"] = "error: " + err.Error()
			healthy = false
		} else {
			response.Checks["database"] = "ok"
		}
	} else {
		response.Checks["database"] = "ok (mock)"
	}

	// Redis check
	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			response.Checks["redis"] = "error: " + err.Error()
			healthy = false
		} else {
			response.Checks["redis"] = "ok"
		}
	} else {
		response.Checks["redis"] = "ok (mock)"
	}

	if !healthy {
		response.Status = "error"
		response.Message = "One or more dependencies are unhealthy"
	}

	return response
}

// HealthHandler handles /health requests.
func HealthHandler(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := checker.Check(ctx)

		w.Header().Set("Content-Type", "application/json")
		if response.Status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(response)
	}
}

// LivenessHandler handles /health/live requests (Kubernetes liveness probe).
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ReadinessHandler handles /health/ready requests (Kubernetes readiness probe).
func ReadinessHandler(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := checker.Check(ctx)

		w.Header().Set("Content-Type", "application/json")
		if response.Status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(map[string]string{"status": response.Status})
	}
}

func main() {
	checker := NewHealthChecker()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler(checker))
	mux.HandleFunc("/health/live", LivenessHandler)
	mux.HandleFunc("/health/ready", ReadinessHandler(checker))

	log.Println("Starting health endpoint example on http://localhost:8080")
	log.Println("Endpoints:")
	log.Println("  - GET /health       - Full health check")
	log.Println("  - GET /health/live  - Liveness probe")
	log.Println("  - GET /health/ready - Readiness probe")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
