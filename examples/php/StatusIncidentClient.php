<?php

/**
 * Status Incident Service - PHP Client
 *
 * A PHP client library for integrating with the Status Incident Service API.
 * Supports system management, incident tracking, webhooks, and health monitoring.
 *
 * Requirements:
 *   - PHP 8.0+
 *   - curl extension
 *
 * Usage:
 *   $client = new StatusIncidentClient('http://localhost:8080', 'your-api-key');
 *   $systems = $client->listSystems();
 */

declare(strict_types=1);

namespace StatusIncident;

class StatusIncidentException extends \Exception {}

class StatusIncidentClient
{
    private string $baseUrl;
    private ?string $apiKey;
    private ?string $username;
    private ?string $password;
    private int $timeout;

    /**
     * Initialize the client.
     *
     * @param string $baseUrl Base URL of the Status Incident service
     * @param string|null $apiKey API key for authentication (optional)
     * @param string|null $username Username for basic auth (optional)
     * @param string|null $password Password for basic auth (optional)
     * @param int $timeout Request timeout in seconds
     */
    public function __construct(
        string $baseUrl,
        ?string $apiKey = null,
        ?string $username = null,
        ?string $password = null,
        int $timeout = 30
    ) {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->apiKey = $apiKey;
        $this->username = $username;
        $this->password = $password;
        $this->timeout = $timeout;
    }

    /**
     * Make an HTTP request to the API.
     */
    private function request(
        string $method,
        string $endpoint,
        ?array $data = null,
        ?array $params = null
    ): mixed {
        $url = $this->baseUrl . '/api' . $endpoint;
        
        if ($params) {
            $url .= '?' . http_build_query($params);
        }

        $ch = curl_init();
        
        curl_setopt_array($ch, [
            CURLOPT_URL => $url,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => $this->timeout,
            CURLOPT_CUSTOMREQUEST => $method,
        ]);

        $headers = [
            'Content-Type: application/json',
            'Accept: application/json',
        ];

        if ($this->apiKey) {
            $headers[] = 'X-API-Key: ' . $this->apiKey;
        } elseif ($this->username && $this->password) {
            curl_setopt($ch, CURLOPT_USERPWD, $this->username . ':' . $this->password);
        }

        curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);

        if ($data !== null) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
        }

        $response = curl_exec($ch);
        $statusCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $error = curl_error($ch);
        
        curl_close($ch);

        if ($error) {
            throw new StatusIncidentException("Request failed: $error");
        }

        if ($statusCode >= 400) {
            throw new StatusIncidentException("API error (status $statusCode): $response");
        }

        if ($response) {
            return json_decode($response, true);
        }

        return null;
    }

    // ==================== Systems API ====================

    /**
     * List all systems.
     */
    public function listSystems(): array
    {
        return $this->request('GET', '/systems') ?? [];
    }

    /**
     * Get a system by ID.
     */
    public function getSystem(int $systemId): array
    {
        return $this->request('GET', "/systems/{$systemId}");
    }

    /**
     * Create a new system.
     */
    public function createSystem(
        string $name,
        string $description = '',
        string $url = '',
        string $owner = ''
    ): array {
        return $this->request('POST', '/systems', [
            'name' => $name,
            'description' => $description,
            'url' => $url,
            'owner' => $owner,
        ]);
    }

    /**
     * Update a system.
     */
    public function updateSystem(
        int $systemId,
        string $name,
        string $description = '',
        string $url = '',
        string $owner = ''
    ): array {
        return $this->request('PUT', "/systems/{$systemId}", [
            'name' => $name,
            'description' => $description,
            'url' => $url,
            'owner' => $owner,
        ]);
    }

    /**
     * Delete a system.
     */
    public function deleteSystem(int $systemId): void
    {
        $this->request('DELETE', "/systems/{$systemId}");
    }

    /**
     * Update system status.
     *
     * @param int $systemId System ID
     * @param string $status New status ("green", "yellow", or "red")
     * @param string $message Optional status message
     */
    public function updateSystemStatus(
        int $systemId,
        string $status,
        string $message = ''
    ): array {
        return $this->request('POST', "/systems/{$systemId}/status", [
            'status' => $status,
            'message' => $message,
        ]);
    }

    /**
     * Get system analytics.
     */
    public function getSystemAnalytics(int $systemId, string $period = '24h'): array
    {
        return $this->request('GET', "/systems/{$systemId}/analytics", null, ['period' => $period]);
    }

    /**
     * Get system SLA status.
     */
    public function getSystemSla(int $systemId, string $period = 'monthly'): array
    {
        return $this->request('GET', "/systems/{$systemId}/sla", null, ['period' => $period]);
    }

    // ==================== Dependencies API ====================

    /**
     * List all dependencies of a system.
     */
    public function listDependencies(int $systemId): array
    {
        return $this->request('GET', "/systems/{$systemId}/dependencies") ?? [];
    }

    /**
     * Create a new dependency.
     */
    public function createDependency(
        int $systemId,
        string $name,
        string $description = ''
    ): array {
        return $this->request('POST', "/systems/{$systemId}/dependencies", [
            'name' => $name,
            'description' => $description,
        ]);
    }

    /**
     * Update a dependency.
     */
    public function updateDependency(
        int $dependencyId,
        string $name,
        string $description = ''
    ): array {
        return $this->request('PUT', "/dependencies/{$dependencyId}", [
            'name' => $name,
            'description' => $description,
        ]);
    }

    /**
     * Delete a dependency.
     */
    public function deleteDependency(int $dependencyId): void
    {
        $this->request('DELETE', "/dependencies/{$dependencyId}");
    }

    /**
     * Update dependency status.
     */
    public function updateDependencyStatus(
        int $dependencyId,
        string $status,
        string $message = ''
    ): array {
        return $this->request('POST', "/dependencies/{$dependencyId}/status", [
            'status' => $status,
            'message' => $message,
        ]);
    }

    /**
     * Configure automatic health check for a dependency.
     *
     * @param int $dependencyId Dependency ID
     * @param string $url Health check URL
     * @param int $interval Check interval in seconds (min: 10)
     * @param string $method HTTP method (GET, POST, HEAD)
     * @param array $headers Optional HTTP headers
     * @param string $expectStatus Expected HTTP status codes (comma-separated)
     * @param string $expectBody Expected response body regex
     */
    public function configureHeartbeat(
        int $dependencyId,
        string $url,
        int $interval = 60,
        string $method = 'GET',
        array $headers = [],
        string $expectStatus = '200',
        string $expectBody = ''
    ): array {
        return $this->request('POST', "/dependencies/{$dependencyId}/heartbeat", [
            'url' => $url,
            'interval' => $interval,
            'method' => $method,
            'headers' => $headers,
            'expect_status' => $expectStatus,
            'expect_body' => $expectBody,
        ]);
    }

    /**
     * Disable heartbeat monitoring for a dependency.
     */
    public function disableHeartbeat(int $dependencyId): void
    {
        $this->request('DELETE', "/dependencies/{$dependencyId}/heartbeat");
    }

    /**
     * Force an immediate health check.
     */
    public function forceCheck(int $dependencyId): array
    {
        return $this->request('POST', "/dependencies/{$dependencyId}/check");
    }

    // ==================== Incidents API ====================

    /**
     * List incidents with optional filters.
     */
    public function listIncidents(?string $status = null, ?string $severity = null): array
    {
        $params = [];
        if ($status) {
            $params['status'] = $status;
        }
        if ($severity) {
            $params['severity'] = $severity;
        }
        return $this->request('GET', '/incidents', null, $params ?: null) ?? [];
    }

    /**
     * Get an incident by ID.
     */
    public function getIncident(int $incidentId): array
    {
        return $this->request('GET', "/incidents/{$incidentId}");
    }

    /**
     * Create a new incident.
     *
     * @param string $title Incident title
     * @param string $message Incident description
     * @param string $severity Severity level ("minor", "major", "critical")
     * @param array $systemIds List of affected system IDs
     */
    public function createIncident(
        string $title,
        string $message,
        string $severity = 'minor',
        array $systemIds = []
    ): array {
        return $this->request('POST', '/incidents', [
            'title' => $title,
            'message' => $message,
            'severity' => $severity,
            'system_ids' => $systemIds,
        ]);
    }

    /**
     * Update incident status or severity.
     */
    public function updateIncident(
        int $incidentId,
        ?string $status = null,
        ?string $severity = null
    ): array {
        $data = [];
        if ($status) {
            $data['status'] = $status;
        }
        if ($severity) {
            $data['severity'] = $severity;
        }
        return $this->request('PUT', "/incidents/{$incidentId}", $data);
    }

    /**
     * Add an update to an incident.
     */
    public function addIncidentUpdate(
        int $incidentId,
        string $message,
        string $by = ''
    ): array {
        return $this->request('POST', "/incidents/{$incidentId}/updates", [
            'message' => $message,
            'by' => $by,
        ]);
    }

    /**
     * Resolve an incident.
     */
    public function resolveIncident(
        int $incidentId,
        string $message = '',
        string $postmortem = ''
    ): array {
        return $this->request('POST', "/incidents/{$incidentId}/resolve", [
            'message' => $message,
            'postmortem' => $postmortem,
        ]);
    }

    // ==================== Webhooks API ====================

    /**
     * List all webhooks.
     */
    public function listWebhooks(): array
    {
        return $this->request('GET', '/webhooks') ?? [];
    }

    /**
     * Create a webhook.
     *
     * @param string $name Webhook name
     * @param string $url Webhook URL
     * @param string $type Type ("slack", "discord", "telegram", "teams", "generic")
     * @param array $events Events to subscribe to
     * @param array $systemIds Systems to monitor (empty for all)
     * @param bool $enabled Whether webhook is enabled
     */
    public function createWebhook(
        string $name,
        string $url,
        string $type = 'generic',
        array $events = [],
        array $systemIds = [],
        bool $enabled = true
    ): array {
        return $this->request('POST', '/webhooks', [
            'name' => $name,
            'url' => $url,
            'type' => $type,
            'events' => $events ?: [
                'system.status_changed',
                'incident.created',
                'incident.resolved',
            ],
            'system_ids' => $systemIds,
            'enabled' => $enabled,
        ]);
    }

    /**
     * Delete a webhook.
     */
    public function deleteWebhook(int $webhookId): void
    {
        $this->request('DELETE', "/webhooks/{$webhookId}");
    }

    /**
     * Send a test notification to a webhook.
     */
    public function testWebhook(int $webhookId): array
    {
        return $this->request('POST', "/webhooks/{$webhookId}/test");
    }

    // ==================== SLA API ====================

    /**
     * Generate an SLA report.
     */
    public function generateSlaReport(
        string $title,
        string $period = 'monthly',
        string $generatedBy = ''
    ): array {
        return $this->request('POST', '/sla/reports', [
            'title' => $title,
            'period' => $period,
            'generated_by' => $generatedBy,
        ]);
    }

    /**
     * List SLA breaches.
     */
    public function listSlaBreaches(?bool $acknowledged = null): array
    {
        $params = [];
        if ($acknowledged !== null) {
            $params['acknowledged'] = $acknowledged ? 'true' : 'false';
        }
        return $this->request('GET', '/sla/breaches', null, $params ?: null) ?? [];
    }

    /**
     * Acknowledge an SLA breach.
     */
    public function acknowledgeBreach(int $breachId, string $by = ''): array
    {
        return $this->request('POST', "/sla/breaches/{$breachId}/acknowledge", [
            'by' => $by,
        ]);
    }

    // ==================== Maintenance API ====================

    /**
     * List all maintenance windows.
     */
    public function listMaintenances(): array
    {
        return $this->request('GET', '/maintenances') ?? [];
    }

    /**
     * Schedule a maintenance window.
     *
     * @param string $title Maintenance title
     * @param string $description Description
     * @param string $startTime Start time (ISO 8601 format)
     * @param string $endTime End time (ISO 8601 format)
     * @param array $systemIds Affected system IDs
     */
    public function createMaintenance(
        string $title,
        string $description,
        string $startTime,
        string $endTime,
        array $systemIds = []
    ): array {
        return $this->request('POST', '/maintenances', [
            'title' => $title,
            'description' => $description,
            'start_time' => $startTime,
            'end_time' => $endTime,
            'system_ids' => $systemIds,
        ]);
    }

    /**
     * Cancel a scheduled maintenance.
     */
    public function cancelMaintenance(int $maintenanceId): void
    {
        $this->request('DELETE', "/maintenances/{$maintenanceId}");
    }

    // ==================== Export/Import API ====================

    /**
     * Export all data.
     */
    public function exportData(): array
    {
        return $this->request('GET', '/export');
    }

    /**
     * Import data from a backup.
     */
    public function importData(array $data): array
    {
        return $this->request('POST', '/import', $data);
    }

    // ==================== Analytics API ====================

    /**
     * Get overall analytics.
     */
    public function getAnalytics(string $period = '24h'): array
    {
        return $this->request('GET', '/analytics', null, ['period' => $period]);
    }

    /**
     * Get change logs.
     */
    public function getLogs(int $limit = 100): array
    {
        return $this->request('GET', '/logs', null, ['limit' => $limit]) ?? [];
    }
}
