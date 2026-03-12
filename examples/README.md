# Integration Examples

This directory contains examples of how to integrate with the Status Incident Service API in various programming languages.

## Available Examples

| Language | Directory | Description |
|----------|-----------|-------------|
| **Python** | [python/](python/) | Full client library, Flask/FastAPI health endpoints |
| **Go** | [go/](go/) | Complete Go client with health endpoint example |
| **PHP** | [php/](php/) | PHP client with Laravel integration examples |
| **Ruby on Rails** | [ruby/](ruby/) | Ruby client with Rails-specific patterns |

## Quick Start

Each example includes:

1. **API Client** - A full-featured client library for the Status Incident API
2. **Health Endpoint** - Implementation of health check endpoints for monitoring
3. **Usage Examples** - Practical examples showing common integration patterns
4. **README** - Language-specific documentation and setup instructions

## Common Integration Patterns

### 1. System Registration

Register your service with Status Incident:

```bash
# Using curl
curl -X POST http://localhost:8080/api/systems \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Service",
    "description": "Production API",
    "url": "https://api.example.com",
    "owner": "Backend Team"
  }'
```

### 2. Health Check Monitoring

Configure automatic health checks for your dependencies:

```bash
# Add a dependency
curl -X POST http://localhost:8080/api/systems/1/dependencies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "PostgreSQL",
    "description": "Primary database"
  }'

# Configure heartbeat monitoring
curl -X POST http://localhost:8080/api/dependencies/1/heartbeat \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://api.example.com/health",
    "interval": 60
  }'
```

### 3. Status Updates

Update system status programmatically:

```bash
curl -X POST http://localhost:8080/api/systems/1/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "yellow",
    "message": "High latency detected"
  }'
```

### 4. Incident Management

Create and manage incidents:

```bash
# Create incident
curl -X POST http://localhost:8080/api/incidents \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Database Connection Issues",
    "message": "Intermittent connection failures",
    "severity": "major",
    "system_ids": [1]
  }'

# Add update
curl -X POST http://localhost:8080/api/incidents/1/updates \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Investigating root cause",
    "by": "ops-team"
  }'

# Resolve incident
curl -X POST http://localhost:8080/api/incidents/1/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Issue resolved",
    "postmortem": "Connection pool was undersized"
  }'
```

### 5. Webhook Notifications

Set up notifications:

```bash
# Slack webhook
curl -X POST http://localhost:8080/api/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Slack Alerts",
    "url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    "type": "slack",
    "events": ["system.status_changed", "incident.created", "incident.resolved"],
    "enabled": true
  }'
```

## Health Endpoint Requirements

For Status Incident to monitor your service, implement a health endpoint that:

1. **Returns HTTP 200** when healthy
2. **Returns HTTP 503** when unhealthy
3. **Responds within 10 seconds**
4. **Returns JSON** (recommended):

```json
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}
```

See individual language examples for implementation details.

## Authentication

The API supports multiple authentication methods:

### API Key (Recommended)

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/systems
```

### Bearer Token

```bash
curl -H "Authorization: Bearer your-api-key" http://localhost:8080/api/systems
```

### Basic Auth

```bash
curl -u "username:password" http://localhost:8080/api/systems
```

## API Documentation

Full API documentation is available at:
- **Swagger UI**: `http://localhost:8080/swagger/`
- **OpenAPI Spec**: `http://localhost:8080/swagger/doc.json`

## Support

For more information, see the main project [README](../README.md) and [Health Check Guide](../docs/HEALTHCHECK_GUIDE.md).
