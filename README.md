# Status Incident Service

[![Release](https://img.shields.io/github/v/release/pomaxa/status-service?style=flat)](https://github.com/pomaxa/status-service/releases)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/pomaxa/status-service/actions/workflows/ci.yml/badge.svg)](https://github.com/pomaxa/status-service/actions)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![SQLite](https://img.shields.io/badge/SQLite-3-003B57?style=flat&logo=sqlite&logoColor=white)](https://sqlite.org/)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io-2496ED?style=flat&logo=docker&logoColor=white)](https://ghcr.io/pomaxa/status-service)

Internal service for monitoring system status and tracking incidents.

## Screenshots

### Dashboard
Overview of all systems with real-time status indicators, uptime metrics, and component health.

![Dashboard](screenshots/01-dashboard.png)

### Administration
System management interface with backup/restore, webhook configuration, and system creation.

![Admin](screenshots/02-admin.png)

### Public Status Page
Read-only public page for external stakeholders to view system status.

![Public](screenshots/03-public.png)

### System Details
Detailed view of a system with status controls, analytics, and dependency management.

![System](screenshots/04-system.png)

### SLA Reports
Generate and view SLA compliance reports with target tracking.

![SLA Reports](screenshots/05-sla.png)

### Analytics
Overall performance metrics, per-system analytics, and SLA reference table.

![Analytics](screenshots/06-analytics.png)

### Activity Log
Complete history of all status changes with timestamps and source tracking.

![Logs](screenshots/07-logs.png)

### API Documentation
Interactive Swagger documentation for the REST API.

![API Docs](screenshots/08-api-docs.png)

## Features

- **System Management** - add projects/services with description, URL, and owner
- **Dependencies** - track components of each system (DB, Redis, API, etc.)
- **Traffic Light Status** - green (operational), yellow (degraded), red (outage)
- **Manual Updates** - change status with comments
- **Heartbeat Monitoring** - automatic URL health checks with latency tracking
- **Latency Graphs** - visual latency history and uptime heatmaps
- **Incident Management** - create, track, and resolve incidents with timeline updates
- **Maintenance Windows** - schedule planned downtime excluded from SLA
- **SLA Reports** - generate compliance reports with breach tracking
- **Webhook Notifications** - Slack, Discord, Telegram, Microsoft Teams, generic HTTP
- **Public Status Page** - read-only page for external stakeholders
- **API Keys** - secure API access with scoped permissions
- **Change History** - complete log of all status changes
- **Analytics** - uptime/SLA, incident count, MTTR
- **Export/Import** - backup and restore all data via API
- **Versioned Migrations** - safe database upgrades with automatic backup
- **Smart Auto-refresh** - dashboard updates without interrupting form editing

## Tech Stack

- **Backend:** Go + chi router
- **Database:** SQLite (WAL mode)
- **Frontend:** HTML templates + vanilla JS
- **Architecture:** DDD (Domain-Driven Design)

## Getting Started

### Using Docker (recommended)

```bash
# Pull from GitHub Container Registry
docker pull ghcr.io/pomaxa/status-service:v1.0.0
docker run -p 8080:8080 -v status-data:/app/data ghcr.io/pomaxa/status-service:v1.0.0

# Or use latest
docker pull ghcr.io/pomaxa/status-service:latest
```

### Download Binary

Pre-built Linux binary (amd64) available from [GitHub Releases](https://github.com/pomaxa/status-service/releases).

### Local Build

```bash
# Build
go build -o status-incident .

# Run (port 8080)
./status-incident
```

Service will be available at http://localhost:8080

### API Documentation

Swagger UI is available at http://localhost:8080/swagger/

## Project Structure

```
├── main.go                     # Entry point
├── internal/
│   ├── domain/                 # Business logic (entities, value objects)
│   ├── application/            # Use cases (services)
│   ├── infrastructure/         # SQLite repositories, HTTP checker
│   └── interfaces/             # HTTP handlers, background workers
├── templates/                  # HTML templates
└── static/                     # CSS styles
```

## Web Interface

| Page | URL | Description |
|------|-----|-------------|
| Dashboard | `/` | Overview of all systems |
| System | `/systems/{id}` | System details and dependencies |
| Public | `/status` | Public status page (read-only) |
| SLA | `/sla` | SLA reports and breaches |
| Admin | `/admin` | Manage systems and webhooks |
| Logs | `/logs` | Change history |
| Analytics | `/analytics` | Statistics and SLA |
| Swagger | `/swagger/` | Interactive API documentation |
| Metrics | `/metrics` | Prometheus metrics endpoint |

## REST API

### Systems

```bash
# List systems
GET /api/systems

# Create system
POST /api/systems
{"name": "API", "description": "Main API", "url": "https://api.example.com", "owner": "Backend Team"}

# Get system
GET /api/systems/{id}

# Update system
PUT /api/systems/{id}
{"name": "API", "description": "Updated", "url": "https://api.example.com", "owner": "Backend Team"}

# Delete system
DELETE /api/systems/{id}

# Change status
POST /api/systems/{id}/status
{"status": "yellow", "message": "Degraded performance"}
```

### Dependencies

```bash
# List dependencies
GET /api/systems/{id}/dependencies

# Add dependency
POST /api/systems/{id}/dependencies
{"name": "PostgreSQL", "description": "Main database"}

# Update dependency
PUT /api/dependencies/{id}

# Delete dependency
DELETE /api/dependencies/{id}

# Change dependency status
POST /api/dependencies/{id}/status
{"status": "red", "message": "Connection lost"}

# Configure heartbeat
POST /api/dependencies/{id}/heartbeat
{"url": "https://api.example.com/health", "interval": 60}

# Disable heartbeat
DELETE /api/dependencies/{id}/heartbeat

# Force check
POST /api/dependencies/{id}/check
```

### Analytics

```bash
# Overall analytics
GET /api/analytics?period=24h

# System analytics
GET /api/systems/{id}/analytics?period=7d

# All logs
GET /api/logs?limit=100
```

### Export / Import

```bash
# Export all data (systems, dependencies, logs)
GET /api/export
# Returns JSON file download

# Export only logs
GET /api/export/logs

# Import data from backup
POST /api/import
Content-Type: application/json
# Body: exported JSON data
```

**Export format:**
```json
{
  "exported_at": "2024-01-15T10:30:00Z",
  "version": "1.0",
  "systems": [
    {
      "id": 1,
      "name": "API",
      "description": "Main API",
      "url": "https://api.example.com",
      "owner": "Backend Team",
      "status": "green",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ],
  "dependencies": [
    {
      "id": 1,
      "system_id": 1,
      "name": "PostgreSQL",
      "description": "Main database",
      "status": "green",
      "heartbeat_url": "https://api.example.com/health/db",
      "heartbeat_interval": 60,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "logs": [
    {
      "id": 1,
      "system_id": 1,
      "old_status": "green",
      "new_status": "yellow",
      "message": "High latency detected",
      "source": "manual",
      "created_at": "2024-01-15T09:00:00Z"
    }
  ]
}
```

**Import response:**
```json
{
  "systems_imported": 5,
  "dependencies_imported": 12,
  "logs_imported": 150,
  "errors": []
}
```

## Heartbeat Monitoring

### How It Works

The service performs HTTP GET requests to configured heartbeat URLs every minute.

**Status determination:**
- **GREEN** - HTTP 2xx response (200, 201, 204, etc.)
- **YELLOW** - 1-2 consecutive failures (non-2xx or timeout)
- **RED** - 3+ consecutive failures

### Health Endpoint Examples

Your service should expose a health endpoint that returns appropriate HTTP status codes.

#### GREEN Status (Operational)

```http
GET /health HTTP/1.1
Host: your-service.com

HTTP/1.1 200 OK
Content-Type: application/json

{"status": "ok"}
```

Any 2xx response is considered healthy:
- `200 OK`
- `201 Created`
- `204 No Content`

#### YELLOW Status (Degraded)

Returned after 1-2 consecutive check failures:

```http
GET /health HTTP/1.1
Host: your-service.com

HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{"status": "degraded", "message": "Database connection slow"}
```

Non-2xx responses that trigger degraded status:
- `500 Internal Server Error`
- `502 Bad Gateway`
- `503 Service Unavailable`
- `504 Gateway Timeout`
- Connection timeout
- DNS resolution failure

#### RED Status (Outage)

Returned after 3+ consecutive check failures:

```http
GET /health HTTP/1.1
Host: your-service.com

HTTP/1.1 500 Internal Server Error
Content-Type: application/json

{"status": "error", "message": "Database unreachable"}
```

### Recommended Health Endpoint Implementation

```go
// Go example
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Check your dependencies
    if err := db.Ping(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "error",
            "message": err.Error(),
        })
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "ok",
    })
}
```

```javascript
// Node.js example
app.get('/health', async (req, res) => {
    try {
        await db.query('SELECT 1');
        res.json({ status: 'ok' });
    } catch (err) {
        res.status(503).json({ status: 'error', message: err.message });
    }
});
```

### Request Details

The heartbeat checker sends requests with:
- **Method:** GET
- **Timeout:** 10 seconds
- **User-Agent:** `StatusIncident-HealthChecker/1.0`
- **Redirects:** Follows up to 10 redirects

## Prometheus Metrics

The `/metrics` endpoint exposes metrics in Prometheus format for monitoring and alerting.

### Available Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `status_incident_system_status` | gauge | system_id, system_name | System status (0=green, 1=yellow, 2=red) |
| `status_incident_system_sla_target` | gauge | system_id, system_name | SLA target percentage |
| `status_incident_uptime_24h` | gauge | system_id, system_name | Uptime percentage over last 24h |
| `status_incident_dependency_status` | gauge | system_id, system_name, dependency_id, dependency_name | Dependency status |
| `status_incident_dependency_latency_ms` | gauge | system_id, system_name, dependency_id, dependency_name | Last check latency in ms |
| `status_incident_dependency_consecutive_failures` | gauge | system_id, system_name, dependency_id, dependency_name | Consecutive check failures |
| `status_incident_systems_total` | gauge | - | Total number of systems |
| `status_incident_dependencies_total` | gauge | - | Total number of dependencies |
| `status_incident_incidents_active` | gauge | - | Number of active incidents |
| `status_incident_incidents_total` | gauge | - | Total number of incidents |
| `status_incident_incidents_by_severity` | gauge | severity | Incidents count by severity |
| `status_incident_incidents_by_status` | gauge | status | Incidents count by status |
| `status_incident_maintenances_active` | gauge | - | Active maintenance windows |
| `status_incident_maintenances_scheduled` | gauge | - | Scheduled maintenance windows |
| `status_incident_sla_breaches_unacknowledged` | gauge | - | Unacknowledged SLA breaches |

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'status-incident'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 30s
```

### Example Alerting Rules

```yaml
groups:
  - name: status-incident
    rules:
      - alert: SystemDown
        expr: status_incident_system_status == 2
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "System {{ $labels.system_name }} is down"

      - alert: HighLatency
        expr: status_incident_dependency_latency_ms > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency on {{ $labels.dependency_name }}"

      - alert: SLABreach
        expr: status_incident_sla_breaches_unacknowledged > 0
        labels:
          severity: critical
        annotations:
          summary: "Unacknowledged SLA breach detected"
```

## Documentation

- [Health Check Implementation Guide](docs/HEALTHCHECK_GUIDE.md) - How to implement health endpoints for your services
- [Upgrade Guide](docs/UPGRADE.md) - Safe upgrade process with automatic backups

## Version Info

Check application version:
```bash
./status-incident -version
# Status Incident Service
#   Version:    v1.0.0
#   Commit:     abc123
#   Build time: 2024-01-15_10:30:00
```

## License

MIT
