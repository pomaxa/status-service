# Python Integration Examples

This directory contains Python examples for integrating with the Status Incident Service.

## Files

- `status_incident_client.py` - Full-featured Python client for the Status Incident API
- `health_endpoint.py` - Health check endpoint implementations (Flask, FastAPI)
- `requirements.txt` - Python dependencies

## Installation

```bash
pip install -r requirements.txt
```

## Quick Start

### Using the Client

```python
from status_incident_client import StatusIncidentClient

# Initialize the client
client = StatusIncidentClient(
    base_url="http://localhost:8080",
    api_key="your-api-key"  # Optional
)

# Create a system
system = client.create_system(
    name="My Service",
    description="Production API service",
    url="https://api.example.com",
    owner="Backend Team"
)

# Update system status
client.update_system_status(
    system_id=system["id"],
    status="yellow",
    message="High latency detected"
)

# Create an incident
incident = client.create_incident(
    title="Database Issues",
    message="Connection timeouts occurring",
    severity="major",
    system_ids=[system["id"]]
)

# Resolve the incident
client.resolve_incident(
    incident_id=incident["id"],
    message="Issue resolved after database restart"
)
```

### Setting Up Webhooks

```python
# Configure Slack notifications
client.create_webhook(
    name="Slack Alerts",
    url="https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    webhook_type="slack",
    events=["system.status_changed", "incident.created", "incident.resolved"]
)

# Configure Discord notifications
client.create_webhook(
    name="Discord Alerts",
    url="https://discord.com/api/webhooks/YOUR/WEBHOOK",
    webhook_type="discord",
    events=["incident.created", "incident.resolved"]
)
```

### Heartbeat Monitoring

```python
# Add a dependency to your system
dependency = client.create_dependency(
    system_id=system["id"],
    name="PostgreSQL",
    description="Primary database"
)

# Configure automatic health checks
client.configure_heartbeat(
    dependency_id=dependency["id"],
    url="https://api.example.com/health",
    interval=60,  # Check every 60 seconds
    expect_status="200",
    expect_body=".*ok.*"  # Regex to match response body
)

# Force an immediate check
result = client.force_check(dependency["id"])
```

### SLA Reporting

```python
# Generate monthly SLA report
report = client.generate_sla_report(
    title="February 2026 SLA Report",
    period="monthly",
    generated_by="automation"
)

# Check for SLA breaches
breaches = client.list_sla_breaches(acknowledged=False)
for breach in breaches:
    print(f"Breach: {breach['title']}")
    
    # Acknowledge the breach
    client.acknowledge_breach(
        breach_id=breach["id"],
        by="ops-team"
    )
```

## Implementing Health Endpoints

For your service to be monitored, implement a health endpoint:

### Flask

```python
from flask import Flask, jsonify

app = Flask(__name__)

@app.route("/health")
def health():
    checks = {}
    healthy = True

    # Check your dependencies
    try:
        db.ping()
        checks["database"] = "ok"
    except Exception as e:
        checks["database"] = f"error: {e}"
        healthy = False

    return jsonify({
        "status": "ok" if healthy else "error",
        "checks": checks
    }), 200 if healthy else 503
```

### FastAPI

```python
from fastapi import FastAPI, Response

app = FastAPI()

@app.get("/health")
async def health(response: Response):
    checks = {}
    healthy = True

    try:
        await db.ping()
        checks["database"] = "ok"
    except Exception as e:
        checks["database"] = f"error: {e}"
        healthy = False

    if not healthy:
        response.status_code = 503

    return {"status": "ok" if healthy else "error", "checks": checks}
```

## Authentication Options

The client supports three authentication methods:

### API Key (Recommended)

```python
client = StatusIncidentClient(
    base_url="http://localhost:8080",
    api_key="your-api-key"
)
```

### Basic Auth

```python
client = StatusIncidentClient(
    base_url="http://localhost:8080",
    username="admin",
    password="password"
)
```

### No Authentication

```python
client = StatusIncidentClient(
    base_url="http://localhost:8080"
)
```

## Error Handling

```python
from requests.exceptions import HTTPError

try:
    system = client.get_system(999)
except HTTPError as e:
    if e.response.status_code == 404:
        print("System not found")
    else:
        print(f"API error: {e}")
```

## Running the Examples

```bash
# Run the client example
python status_incident_client.py

# Run the health endpoint example (Flask)
python health_endpoint.py
```
