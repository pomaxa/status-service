# Health Check Implementation Guide

This guide describes how to implement health check endpoints for integration with Status Incident monitoring service.

## Quick Start

Create a `/health` endpoint that returns:
- **HTTP 200** when service is healthy
- **HTTP 503** when service is unhealthy

```go
// Minimal example (Go)
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
})
```

## Status Mapping

| HTTP Code | Status Incident | Meaning |
|-----------|-----------------|---------|
| 2xx (200, 201, 204) | GREEN | Operational |
| 1-2 failures | YELLOW | Degraded (automatic) |
| 3+ failures | RED | Outage (automatic) |

## Endpoint Requirements

### URL
- Path: `/health`, `/healthz`, `/health/live`, or custom
- Method: `GET`
- No authentication required (or use separate internal endpoint)

### Response
- Content-Type: `application/json` (recommended)
- Body: JSON with status field

```json
{"status": "ok"}
```

### Timeout
- Response must complete within **10 seconds**
- Longer responses are treated as failures

## Implementation Examples

### Go (net/http)

```go
package main

import (
    "encoding/json"
    "net/http"
)

type HealthResponse struct {
    Status  string            `json:"status"`
    Checks  map[string]string `json:"checks,omitempty"`
    Message string            `json:"message,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    health := HealthResponse{
        Status: "ok",
        Checks: make(map[string]string),
    }

    // Check database
    if err := db.Ping(); err != nil {
        health.Status = "error"
        health.Checks["database"] = "failed"
        health.Message = err.Error()
        w.WriteHeader(http.StatusServiceUnavailable)
    } else {
        health.Checks["database"] = "ok"
    }

    // Check Redis
    if err := redis.Ping(ctx).Err(); err != nil {
        health.Status = "error"
        health.Checks["redis"] = "failed"
        w.WriteHeader(http.StatusServiceUnavailable)
    } else {
        health.Checks["redis"] = "ok"
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}

func main() {
    http.HandleFunc("/health", healthHandler)
    http.ListenAndServe(":8080", nil)
}
```

### Go (chi router)

```go
r := chi.NewRouter()

r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
    checks := map[string]string{}
    healthy := true

    // Database check
    if err := db.PingContext(r.Context()); err != nil {
        checks["db"] = "error: " + err.Error()
        healthy = false
    } else {
        checks["db"] = "ok"
    }

    w.Header().Set("Content-Type", "application/json")
    if !healthy {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": map[bool]string{true: "ok", false: "error"}[healthy],
        "checks": checks,
    })
})
```

### Node.js (Express)

```javascript
const express = require('express');
const app = express();

app.get('/health', async (req, res) => {
    const checks = {};
    let healthy = true;

    // Database check
    try {
        await db.query('SELECT 1');
        checks.database = 'ok';
    } catch (err) {
        checks.database = `error: ${err.message}`;
        healthy = false;
    }

    // Redis check
    try {
        await redis.ping();
        checks.redis = 'ok';
    } catch (err) {
        checks.redis = `error: ${err.message}`;
        healthy = false;
    }

    res.status(healthy ? 200 : 503).json({
        status: healthy ? 'ok' : 'error',
        checks
    });
});

app.listen(3000);
```

### Python (FastAPI)

```python
from fastapi import FastAPI, Response
from typing import Dict

app = FastAPI()

@app.get("/health")
async def health_check(response: Response) -> Dict:
    checks = {}
    healthy = True

    # Database check
    try:
        await database.execute("SELECT 1")
        checks["database"] = "ok"
    except Exception as e:
        checks["database"] = f"error: {str(e)}"
        healthy = False

    # Redis check
    try:
        await redis.ping()
        checks["redis"] = "ok"
    except Exception as e:
        checks["redis"] = f"error: {str(e)}"
        healthy = False

    if not healthy:
        response.status_code = 503

    return {
        "status": "ok" if healthy else "error",
        "checks": checks
    }
```

### Python (Flask)

```python
from flask import Flask, jsonify

app = Flask(__name__)

@app.route('/health')
def health():
    checks = {}
    healthy = True

    # Database check
    try:
        db.session.execute('SELECT 1')
        checks['database'] = 'ok'
    except Exception as e:
        checks['database'] = f'error: {str(e)}'
        healthy = False

    status_code = 200 if healthy else 503
    return jsonify({
        'status': 'ok' if healthy else 'error',
        'checks': checks
    }), status_code

if __name__ == '__main__':
    app.run(port=5000)
```

### Rust (Actix-web)

```rust
use actix_web::{web, App, HttpResponse, HttpServer};
use serde::Serialize;

#[derive(Serialize)]
struct HealthResponse {
    status: String,
    checks: std::collections::HashMap<String, String>,
}

async fn health(db: web::Data<Pool>) -> HttpResponse {
    let mut checks = std::collections::HashMap::new();
    let mut healthy = true;

    // Database check
    match db.get_ref().get().await {
        Ok(conn) => {
            checks.insert("database".to_string(), "ok".to_string());
        }
        Err(e) => {
            checks.insert("database".to_string(), format!("error: {}", e));
            healthy = false;
        }
    }

    let response = HealthResponse {
        status: if healthy { "ok" } else { "error" }.to_string(),
        checks,
    };

    if healthy {
        HttpResponse::Ok().json(response)
    } else {
        HttpResponse::ServiceUnavailable().json(response)
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    HttpServer::new(|| {
        App::new()
            .route("/health", web::get().to(health))
    })
    .bind("127.0.0.1:8080")?
    .run()
    .await
}
```

## What to Check

### Essential Checks
- **Database connection** - can execute simple query
- **Cache connection** - Redis/Memcached ping
- **File system** - can write to data directory

### Optional Checks
- External API dependencies (with timeout)
- Queue connection (RabbitMQ, Kafka)
- Memory/disk thresholds

### Avoid in Health Checks
- Long-running queries
- Full database scans
- External API calls without short timeout
- Authentication/authorization logic

## Best Practices

1. **Keep it fast** - health check should complete in <1 second
2. **Check critical dependencies only** - DB, cache, essential services
3. **Use timeouts** - don't let one slow check block the response
4. **No side effects** - health check should be read-only
5. **Separate endpoints** - use `/health/live` for liveness, `/health/ready` for readiness

## Response Examples

### Healthy Service
```json
{
    "status": "ok",
    "checks": {
        "database": "ok",
        "redis": "ok"
    }
}
```

### Degraded Service
```json
{
    "status": "degraded",
    "checks": {
        "database": "ok",
        "redis": "error: connection refused"
    },
    "message": "Non-critical dependency unavailable"
}
```

### Unhealthy Service
```json
{
    "status": "error",
    "checks": {
        "database": "error: connection timeout",
        "redis": "ok"
    },
    "message": "Critical dependency unavailable"
}
```

## Kubernetes Integration

If using Kubernetes, configure probes:

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 30
    readinessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
```

## Registering with Status Incident

After implementing the health endpoint:

1. Go to Status Incident → System → Dependencies
2. Click "Heartbeat" on the dependency
3. Enter the health check URL (e.g., `http://your-service:8080/health`)
4. Set interval (default: 60 seconds, minimum: 10 seconds)
5. Save

The service will be automatically monitored and status updated based on health check results.
