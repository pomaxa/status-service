# Go Integration Examples

This directory contains Go examples for integrating with the Status Incident Service.

## Files

- `client.go` - Full-featured Go client for the Status Incident API
- `main.go` - Example usage of the client
- `health_endpoint.go` - Health check endpoint implementation
- `go.mod` - Go module file

## Quick Start

### Using the Client

```go
package main

import (
    "context"
    "log"
    
    statusincident "github.com/example/status-incident-go"
)

func main() {
    ctx := context.Background()

    // Initialize the client
    client := statusincident.NewClient("http://localhost:8080",
        statusincident.WithAPIKey("your-api-key"), // Optional
    )

    // Create a system
    system, err := client.CreateSystem(ctx, &statusincident.CreateSystemRequest{
        Name:        "My Service",
        Description: "Production API service",
        URL:         "https://api.example.com",
        Owner:       "Backend Team",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Update system status
    client.UpdateSystemStatus(ctx, system.ID, &statusincident.UpdateStatusRequest{
        Status:  "yellow",
        Message: "High latency detected",
    })

    // Create an incident
    incident, _ := client.CreateIncident(ctx, &statusincident.CreateIncidentRequest{
        Title:     "Database Issues",
        Message:   "Connection timeouts occurring",
        Severity:  "major",
        SystemIDs: []int{system.ID},
    })

    // Resolve the incident
    client.ResolveIncident(ctx, incident.ID, &statusincident.ResolveIncidentRequest{
        Message: "Issue resolved after database restart",
    })
}
```

### Setting Up Webhooks

```go
// Configure Slack notifications
client.CreateWebhook(ctx, &statusincident.CreateWebhookRequest{
    Name:    "Slack Alerts",
    URL:     "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    Type:    "slack",
    Events:  []string{"system.status_changed", "incident.created", "incident.resolved"},
    Enabled: true,
})

// Configure Discord notifications
client.CreateWebhook(ctx, &statusincident.CreateWebhookRequest{
    Name:    "Discord Alerts",
    URL:     "https://discord.com/api/webhooks/YOUR/WEBHOOK",
    Type:    "discord",
    Events:  []string{"incident.created", "incident.resolved"},
    Enabled: true,
})
```

### Heartbeat Monitoring

```go
// Add a dependency to your system
dependency, _ := client.CreateDependency(ctx, system.ID, &statusincident.CreateDependencyRequest{
    Name:        "PostgreSQL",
    Description: "Primary database",
})

// Configure automatic health checks
client.ConfigureHeartbeat(ctx, dependency.ID, &statusincident.HeartbeatConfigRequest{
    URL:          "https://api.example.com/health",
    Interval:     60, // Check every 60 seconds
    ExpectStatus: "200",
    ExpectBody:   ".*ok.*", // Regex to match response body
})

// Force an immediate check
client.ForceCheck(ctx, dependency.ID)
```

### Maintenance Windows

```go
import "time"

// Schedule maintenance
startTime := time.Now().Add(24 * time.Hour)
endTime := startTime.Add(2 * time.Hour)

client.CreateMaintenance(ctx, &statusincident.CreateMaintenanceRequest{
    Title:       "Database Upgrade",
    Description: "Upgrading PostgreSQL to version 16",
    StartTime:   startTime,
    EndTime:     endTime,
    SystemIDs:   []int{system.ID},
})
```

### SLA Reporting

```go
// Generate monthly SLA report
report, _ := client.GenerateSLAReport(ctx, &statusincident.SLAReportRequest{
    Title:       "February 2026 SLA Report",
    Period:      "monthly",
    GeneratedBy: "automation",
})
```

## Implementing Health Endpoints

For your service to be monitored, implement a health endpoint:

```go
package main

import (
    "encoding/json"
    "net/http"
)

type HealthResponse struct {
    Status  string            `json:"status"`
    Checks  map[string]string `json:"checks,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status: "ok",
        Checks: make(map[string]string),
    }
    healthy := true

    // Check database
    if err := db.Ping(); err != nil {
        response.Checks["database"] = "error: " + err.Error()
        healthy = false
    } else {
        response.Checks["database"] = "ok"
    }

    // Check Redis
    if err := redis.Ping(ctx).Err(); err != nil {
        response.Checks["redis"] = "error: " + err.Error()
        healthy = false
    } else {
        response.Checks["redis"] = "ok"
    }

    w.Header().Set("Content-Type", "application/json")
    if !healthy {
        response.Status = "error"
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/health", healthHandler)
    http.ListenAndServe(":8080", nil)
}
```

## Authentication Options

The client supports three authentication methods:

### API Key (Recommended)

```go
client := statusincident.NewClient("http://localhost:8080",
    statusincident.WithAPIKey("your-api-key"),
)
```

### Basic Auth

```go
client := statusincident.NewClient("http://localhost:8080",
    statusincident.WithBasicAuth("admin", "password"),
)
```

### No Authentication

```go
client := statusincident.NewClient("http://localhost:8080")
```

### Custom HTTP Client

```go
httpClient := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns: 10,
    },
}

client := statusincident.NewClient("http://localhost:8080",
    statusincident.WithHTTPClient(httpClient),
    statusincident.WithAPIKey("your-api-key"),
)
```

## Error Handling

```go
system, err := client.GetSystem(ctx, 999)
if err != nil {
    // Check for specific error types
    if strings.Contains(err.Error(), "status 404") {
        log.Println("System not found")
    } else {
        log.Printf("API error: %v", err)
    }
}
```

## Running the Examples

```bash
# Run the client example
go run main.go client.go

# Run the health endpoint example
go run health_endpoint.go
```
