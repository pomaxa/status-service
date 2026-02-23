"""
Status Incident Service - Python Client

A Python client library for integrating with the Status Incident Service API.
Supports system management, incident tracking, webhooks, and health monitoring.

Requirements:
    pip install requests

Usage:
    from status_incident_client import StatusIncidentClient

    client = StatusIncidentClient("http://localhost:8080", api_key="your-api-key")
    systems = client.list_systems()
"""

import requests
from typing import Optional, Dict, List, Any
from dataclasses import dataclass
from datetime import datetime


@dataclass
class System:
    """Represents a monitored system."""
    id: int
    name: str
    description: str
    url: str
    owner: str
    status: str
    sla_target: Optional[float] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


@dataclass
class Dependency:
    """Represents a system dependency (component)."""
    id: int
    system_id: int
    name: str
    description: str
    status: str
    heartbeat_url: Optional[str] = None
    heartbeat_interval: Optional[int] = None


@dataclass
class Incident:
    """Represents an incident."""
    id: int
    title: str
    message: str
    severity: str
    status: str
    created_at: str
    resolved_at: Optional[str] = None


class StatusIncidentClient:
    """Client for the Status Incident Service API."""

    def __init__(
        self,
        base_url: str,
        api_key: Optional[str] = None,
        username: Optional[str] = None,
        password: Optional[str] = None,
        timeout: int = 30
    ):
        """
        Initialize the client.

        Args:
            base_url: Base URL of the Status Incident service (e.g., "http://localhost:8080")
            api_key: API key for authentication (optional)
            username: Username for basic auth (optional)
            password: Password for basic auth (optional)
            timeout: Request timeout in seconds
        """
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.username = username
        self.password = password
        self.timeout = timeout
        self.session = requests.Session()
        self._setup_auth()

    def _setup_auth(self):
        """Configure authentication headers."""
        if self.api_key:
            self.session.headers["X-API-Key"] = self.api_key
        elif self.username and self.password:
            self.session.auth = (self.username, self.password)

    def _request(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict] = None,
        params: Optional[Dict] = None
    ) -> Any:
        """Make an HTTP request to the API."""
        url = f"{self.base_url}/api{endpoint}"
        response = self.session.request(
            method=method,
            url=url,
            json=data,
            params=params,
            timeout=self.timeout
        )
        response.raise_for_status()
        if response.content:
            return response.json()
        return None

    # ==================== Systems API ====================

    def list_systems(self) -> List[Dict]:
        """List all systems."""
        return self._request("GET", "/systems")

    def get_system(self, system_id: int) -> Dict:
        """Get a system by ID."""
        return self._request("GET", f"/systems/{system_id}")

    def create_system(
        self,
        name: str,
        description: str = "",
        url: str = "",
        owner: str = ""
    ) -> Dict:
        """Create a new system."""
        return self._request("POST", "/systems", {
            "name": name,
            "description": description,
            "url": url,
            "owner": owner
        })

    def update_system(
        self,
        system_id: int,
        name: str,
        description: str = "",
        url: str = "",
        owner: str = ""
    ) -> Dict:
        """Update a system."""
        return self._request("PUT", f"/systems/{system_id}", {
            "name": name,
            "description": description,
            "url": url,
            "owner": owner
        })

    def delete_system(self, system_id: int) -> None:
        """Delete a system."""
        self._request("DELETE", f"/systems/{system_id}")

    def update_system_status(
        self,
        system_id: int,
        status: str,
        message: str = ""
    ) -> Dict:
        """
        Update system status.

        Args:
            system_id: System ID
            status: New status ("green", "yellow", or "red")
            message: Optional status message
        """
        return self._request("POST", f"/systems/{system_id}/status", {
            "status": status,
            "message": message
        })

    def get_system_analytics(
        self,
        system_id: int,
        period: str = "24h"
    ) -> Dict:
        """Get system analytics."""
        return self._request("GET", f"/systems/{system_id}/analytics", {"period": period})

    def get_system_sla(
        self,
        system_id: int,
        period: str = "monthly"
    ) -> Dict:
        """Get system SLA status."""
        return self._request("GET", f"/systems/{system_id}/sla", {"period": period})

    # ==================== Dependencies API ====================

    def list_dependencies(self, system_id: int) -> List[Dict]:
        """List all dependencies of a system."""
        return self._request("GET", f"/systems/{system_id}/dependencies")

    def create_dependency(
        self,
        system_id: int,
        name: str,
        description: str = ""
    ) -> Dict:
        """Create a new dependency."""
        return self._request("POST", f"/systems/{system_id}/dependencies", {
            "name": name,
            "description": description
        })

    def update_dependency(
        self,
        dependency_id: int,
        name: str,
        description: str = ""
    ) -> Dict:
        """Update a dependency."""
        return self._request("PUT", f"/dependencies/{dependency_id}", {
            "name": name,
            "description": description
        })

    def delete_dependency(self, dependency_id: int) -> None:
        """Delete a dependency."""
        self._request("DELETE", f"/dependencies/{dependency_id}")

    def update_dependency_status(
        self,
        dependency_id: int,
        status: str,
        message: str = ""
    ) -> Dict:
        """Update dependency status."""
        return self._request("POST", f"/dependencies/{dependency_id}/status", {
            "status": status,
            "message": message
        })

    def configure_heartbeat(
        self,
        dependency_id: int,
        url: str,
        interval: int = 60,
        method: str = "GET",
        headers: Optional[Dict[str, str]] = None,
        expect_status: str = "200",
        expect_body: str = ""
    ) -> Dict:
        """
        Configure automatic health check for a dependency.

        Args:
            dependency_id: Dependency ID
            url: Health check URL
            interval: Check interval in seconds (min: 10)
            method: HTTP method (GET, POST, HEAD)
            headers: Optional HTTP headers
            expect_status: Expected HTTP status codes (comma-separated)
            expect_body: Expected response body regex
        """
        return self._request("POST", f"/dependencies/{dependency_id}/heartbeat", {
            "url": url,
            "interval": interval,
            "method": method,
            "headers": headers or {},
            "expect_status": expect_status,
            "expect_body": expect_body
        })

    def disable_heartbeat(self, dependency_id: int) -> None:
        """Disable heartbeat monitoring for a dependency."""
        self._request("DELETE", f"/dependencies/{dependency_id}/heartbeat")

    def force_check(self, dependency_id: int) -> Dict:
        """Force an immediate health check."""
        return self._request("POST", f"/dependencies/{dependency_id}/check")

    # ==================== Incidents API ====================

    def list_incidents(
        self,
        status: Optional[str] = None,
        severity: Optional[str] = None
    ) -> List[Dict]:
        """List incidents with optional filters."""
        params = {}
        if status:
            params["status"] = status
        if severity:
            params["severity"] = severity
        return self._request("GET", "/incidents", params=params)

    def get_incident(self, incident_id: int) -> Dict:
        """Get an incident by ID."""
        return self._request("GET", f"/incidents/{incident_id}")

    def create_incident(
        self,
        title: str,
        message: str,
        severity: str = "minor",
        system_ids: Optional[List[int]] = None
    ) -> Dict:
        """
        Create a new incident.

        Args:
            title: Incident title
            message: Incident description
            severity: Severity level ("minor", "major", "critical")
            system_ids: List of affected system IDs
        """
        return self._request("POST", "/incidents", {
            "title": title,
            "message": message,
            "severity": severity,
            "system_ids": system_ids or []
        })

    def update_incident(
        self,
        incident_id: int,
        status: Optional[str] = None,
        severity: Optional[str] = None
    ) -> Dict:
        """Update incident status or severity."""
        data = {}
        if status:
            data["status"] = status
        if severity:
            data["severity"] = severity
        return self._request("PUT", f"/incidents/{incident_id}", data)

    def add_incident_update(
        self,
        incident_id: int,
        message: str,
        by: str = ""
    ) -> Dict:
        """Add an update to an incident."""
        return self._request("POST", f"/incidents/{incident_id}/updates", {
            "message": message,
            "by": by
        })

    def resolve_incident(
        self,
        incident_id: int,
        message: str = "",
        postmortem: str = ""
    ) -> Dict:
        """Resolve an incident."""
        return self._request("POST", f"/incidents/{incident_id}/resolve", {
            "message": message,
            "postmortem": postmortem
        })

    # ==================== Webhooks API ====================

    def list_webhooks(self) -> List[Dict]:
        """List all webhooks."""
        return self._request("GET", "/webhooks")

    def create_webhook(
        self,
        name: str,
        url: str,
        webhook_type: str = "generic",
        events: Optional[List[str]] = None,
        system_ids: Optional[List[int]] = None,
        enabled: bool = True
    ) -> Dict:
        """
        Create a webhook.

        Args:
            name: Webhook name
            url: Webhook URL
            webhook_type: Type ("slack", "discord", "telegram", "teams", "generic")
            events: Events to subscribe to
            system_ids: Systems to monitor (empty for all)
            enabled: Whether webhook is enabled
        """
        return self._request("POST", "/webhooks", {
            "name": name,
            "url": url,
            "type": webhook_type,
            "events": events or [
                "system.status_changed",
                "incident.created",
                "incident.resolved"
            ],
            "system_ids": system_ids or [],
            "enabled": enabled
        })

    def delete_webhook(self, webhook_id: int) -> None:
        """Delete a webhook."""
        self._request("DELETE", f"/webhooks/{webhook_id}")

    def test_webhook(self, webhook_id: int) -> Dict:
        """Send a test notification to a webhook."""
        return self._request("POST", f"/webhooks/{webhook_id}/test")

    # ==================== SLA API ====================

    def generate_sla_report(
        self,
        title: str,
        period: str = "monthly",
        generated_by: str = ""
    ) -> Dict:
        """Generate an SLA report."""
        return self._request("POST", "/sla/reports", {
            "title": title,
            "period": period,
            "generated_by": generated_by
        })

    def list_sla_breaches(self, acknowledged: Optional[bool] = None) -> List[Dict]:
        """List SLA breaches."""
        params = {}
        if acknowledged is not None:
            params["acknowledged"] = str(acknowledged).lower()
        return self._request("GET", "/sla/breaches", params=params)

    def acknowledge_breach(self, breach_id: int, by: str = "") -> Dict:
        """Acknowledge an SLA breach."""
        return self._request("POST", f"/sla/breaches/{breach_id}/acknowledge", {
            "by": by
        })

    # ==================== Maintenance API ====================

    def list_maintenances(self) -> List[Dict]:
        """List all maintenance windows."""
        return self._request("GET", "/maintenances")

    def create_maintenance(
        self,
        title: str,
        description: str,
        start_time: str,
        end_time: str,
        system_ids: Optional[List[int]] = None
    ) -> Dict:
        """
        Schedule a maintenance window.

        Args:
            title: Maintenance title
            description: Description
            start_time: Start time (ISO 8601 format)
            end_time: End time (ISO 8601 format)
            system_ids: Affected system IDs
        """
        return self._request("POST", "/maintenances", {
            "title": title,
            "description": description,
            "start_time": start_time,
            "end_time": end_time,
            "system_ids": system_ids or []
        })

    def cancel_maintenance(self, maintenance_id: int) -> None:
        """Cancel a scheduled maintenance."""
        self._request("DELETE", f"/maintenances/{maintenance_id}")

    # ==================== Export/Import API ====================

    def export_data(self) -> Dict:
        """Export all data."""
        return self._request("GET", "/export")

    def import_data(self, data: Dict) -> Dict:
        """Import data from a backup."""
        return self._request("POST", "/import", data)

    # ==================== Analytics API ====================

    def get_analytics(self, period: str = "24h") -> Dict:
        """Get overall analytics."""
        return self._request("GET", "/analytics", {"period": period})

    def get_logs(self, limit: int = 100) -> List[Dict]:
        """Get change logs."""
        return self._request("GET", "/logs", {"limit": limit})


# ==================== Example Usage ====================

if __name__ == "__main__":
    # Initialize client
    client = StatusIncidentClient(
        base_url="http://localhost:8080",
        api_key="your-api-key"  # Optional: set your API key
    )

    # Example: Create a system
    system = client.create_system(
        name="Payment Service",
        description="Handles all payment processing",
        url="https://payments.example.com",
        owner="Payments Team"
    )
    print(f"Created system: {system['name']} (ID: {system['id']})")

    # Example: Add a dependency with heartbeat monitoring
    dependency = client.create_dependency(
        system_id=system["id"],
        name="PostgreSQL",
        description="Primary database"
    )
    print(f"Created dependency: {dependency['name']}")

    # Configure automatic health checking
    client.configure_heartbeat(
        dependency_id=dependency["id"],
        url="https://payments.example.com/health",
        interval=60
    )
    print("Configured heartbeat monitoring")

    # Example: Create an incident
    incident = client.create_incident(
        title="Database Connection Issues",
        message="Experiencing intermittent connection failures to the primary database",
        severity="major",
        system_ids=[system["id"]]
    )
    print(f"Created incident: {incident['title']}")

    # Add an update
    client.add_incident_update(
        incident_id=incident["id"],
        message="Identified root cause: connection pool exhaustion",
        by="ops-team"
    )

    # Resolve the incident
    client.resolve_incident(
        incident_id=incident["id"],
        message="Increased connection pool size and deployed fix",
        postmortem="Connection pool was undersized for traffic load"
    )
    print("Incident resolved")

    # Example: Set up Slack notifications
    webhook = client.create_webhook(
        name="Slack Alerts",
        url="https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
        webhook_type="slack",
        events=["system.status_changed", "incident.created", "incident.resolved"]
    )
    print(f"Created webhook: {webhook['name']}")

    # Example: Generate SLA report
    report = client.generate_sla_report(
        title="Monthly SLA Report - February 2026",
        period="monthly",
        generated_by="automation"
    )
    print(f"Generated SLA report: {report.get('title', 'Report')}")
