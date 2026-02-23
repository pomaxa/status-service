// Example usage of the Status Incident Go client.
//
// Run with:
//
//	go run main.go client.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	ctx := context.Background()

	// Initialize the client
	client := NewClient("http://localhost:8080",
		WithAPIKey("your-api-key"), // Optional: set your API key
	)

	// Example: Create a system
	system, err := client.CreateSystem(ctx, &CreateSystemRequest{
		Name:        "Payment Service",
		Description: "Handles all payment processing",
		URL:         "https://payments.example.com",
		Owner:       "Payments Team",
	})
	if err != nil {
		log.Fatalf("Failed to create system: %v", err)
	}
	fmt.Printf("Created system: %s (ID: %d)\n", system.Name, system.ID)

	// Example: Add a dependency with heartbeat monitoring
	dependency, err := client.CreateDependency(ctx, system.ID, &CreateDependencyRequest{
		Name:        "PostgreSQL",
		Description: "Primary database",
	})
	if err != nil {
		log.Fatalf("Failed to create dependency: %v", err)
	}
	fmt.Printf("Created dependency: %s\n", dependency.Name)

	// Configure automatic health checking
	_, err = client.ConfigureHeartbeat(ctx, dependency.ID, &HeartbeatConfigRequest{
		URL:          "https://payments.example.com/health",
		Interval:     60,
		ExpectStatus: "200",
	})
	if err != nil {
		log.Fatalf("Failed to configure heartbeat: %v", err)
	}
	fmt.Println("Configured heartbeat monitoring")

	// Example: Update system status
	_, err = client.UpdateSystemStatus(ctx, system.ID, &UpdateStatusRequest{
		Status:  "yellow",
		Message: "High latency detected",
	})
	if err != nil {
		log.Fatalf("Failed to update status: %v", err)
	}
	fmt.Println("Updated system status to yellow")

	// Example: Create an incident
	incident, err := client.CreateIncident(ctx, &CreateIncidentRequest{
		Title:     "Database Connection Issues",
		Message:   "Experiencing intermittent connection failures to the primary database",
		Severity:  "major",
		SystemIDs: []int{system.ID},
	})
	if err != nil {
		log.Fatalf("Failed to create incident: %v", err)
	}
	fmt.Printf("Created incident: %s (ID: %d)\n", incident.Title, incident.ID)

	// Add an update to the incident
	err = client.AddIncidentUpdate(ctx, incident.ID, &IncidentUpdateRequest{
		Message: "Identified root cause: connection pool exhaustion",
		By:      "ops-team",
	})
	if err != nil {
		log.Fatalf("Failed to add incident update: %v", err)
	}
	fmt.Println("Added incident update")

	// Resolve the incident
	_, err = client.ResolveIncident(ctx, incident.ID, &ResolveIncidentRequest{
		Message:    "Increased connection pool size and deployed fix",
		Postmortem: "Connection pool was undersized for traffic load",
	})
	if err != nil {
		log.Fatalf("Failed to resolve incident: %v", err)
	}
	fmt.Println("Incident resolved")

	// Example: Set up Slack notifications
	webhook, err := client.CreateWebhook(ctx, &CreateWebhookRequest{
		Name:    "Slack Alerts",
		URL:     "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
		Type:    "slack",
		Events:  []string{"system.status_changed", "incident.created", "incident.resolved"},
		Enabled: true,
	})
	if err != nil {
		log.Fatalf("Failed to create webhook: %v", err)
	}
	fmt.Printf("Created webhook: %s\n", webhook.Name)

	// Example: Schedule maintenance
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	maintenance, err := client.CreateMaintenance(ctx, &CreateMaintenanceRequest{
		Title:       "Database Upgrade",
		Description: "Upgrading PostgreSQL to version 16",
		StartTime:   startTime,
		EndTime:     endTime,
		SystemIDs:   []int{system.ID},
	})
	if err != nil {
		log.Fatalf("Failed to schedule maintenance: %v", err)
	}
	fmt.Printf("Scheduled maintenance: %s\n", maintenance.Title)

	// Example: Generate SLA report
	report, err := client.GenerateSLAReport(ctx, &SLAReportRequest{
		Title:       "Monthly SLA Report - February 2026",
		Period:      "monthly",
		GeneratedBy: "automation",
	})
	if err != nil {
		log.Fatalf("Failed to generate SLA report: %v", err)
	}
	fmt.Printf("Generated SLA report: %v\n", report)

	// Example: Get analytics
	analytics, err := client.GetAnalytics(ctx, "24h")
	if err != nil {
		log.Fatalf("Failed to get analytics: %v", err)
	}
	fmt.Printf("Analytics: %v\n", analytics)

	// Example: List all systems
	systems, err := client.ListSystems(ctx)
	if err != nil {
		log.Fatalf("Failed to list systems: %v", err)
	}
	fmt.Printf("\nAll systems (%d):\n", len(systems))
	for _, s := range systems {
		fmt.Printf("  - %s (status: %s)\n", s.Name, s.Status)
	}

	fmt.Println("\nAll examples completed successfully!")
}
