<?php

/**
 * Example usage of the Status Incident PHP client.
 *
 * Run with:
 *   php example.php
 */

declare(strict_types=1);

require_once __DIR__ . '/StatusIncidentClient.php';

use StatusIncident\StatusIncidentClient;
use StatusIncident\StatusIncidentException;

// Initialize the client
$client = new StatusIncidentClient(
    baseUrl: 'http://localhost:8080',
    apiKey: 'your-api-key' // Optional: set your API key
);

try {
    // Example: Create a system
    $system = $client->createSystem(
        name: 'Payment Service',
        description: 'Handles all payment processing',
        url: 'https://payments.example.com',
        owner: 'Payments Team'
    );
    echo "Created system: {$system['name']} (ID: {$system['id']})\n";

    // Example: Add a dependency with heartbeat monitoring
    $dependency = $client->createDependency(
        systemId: $system['id'],
        name: 'PostgreSQL',
        description: 'Primary database'
    );
    echo "Created dependency: {$dependency['name']}\n";

    // Configure automatic health checking
    $client->configureHeartbeat(
        dependencyId: $dependency['id'],
        url: 'https://payments.example.com/health',
        interval: 60
    );
    echo "Configured heartbeat monitoring\n";

    // Example: Update system status
    $client->updateSystemStatus(
        systemId: $system['id'],
        status: 'yellow',
        message: 'High latency detected'
    );
    echo "Updated system status to yellow\n";

    // Example: Create an incident
    $incident = $client->createIncident(
        title: 'Database Connection Issues',
        message: 'Experiencing intermittent connection failures to the primary database',
        severity: 'major',
        systemIds: [$system['id']]
    );
    echo "Created incident: {$incident['title']} (ID: {$incident['id']})\n";

    // Add an update to the incident
    $client->addIncidentUpdate(
        incidentId: $incident['id'],
        message: 'Identified root cause: connection pool exhaustion',
        by: 'ops-team'
    );
    echo "Added incident update\n";

    // Resolve the incident
    $client->resolveIncident(
        incidentId: $incident['id'],
        message: 'Increased connection pool size and deployed fix',
        postmortem: 'Connection pool was undersized for traffic load'
    );
    echo "Incident resolved\n";

    // Example: Set up Slack notifications
    $webhook = $client->createWebhook(
        name: 'Slack Alerts',
        url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL',
        type: 'slack',
        events: ['system.status_changed', 'incident.created', 'incident.resolved']
    );
    echo "Created webhook: {$webhook['name']}\n";

    // Example: Schedule maintenance
    $startTime = (new DateTime('+1 day'))->format('c');
    $endTime = (new DateTime('+1 day +2 hours'))->format('c');
    $maintenance = $client->createMaintenance(
        title: 'Database Upgrade',
        description: 'Upgrading PostgreSQL to version 16',
        startTime: $startTime,
        endTime: $endTime,
        systemIds: [$system['id']]
    );
    echo "Scheduled maintenance: {$maintenance['title']}\n";

    // Example: Generate SLA report
    $report = $client->generateSlaReport(
        title: 'Monthly SLA Report - February 2026',
        period: 'monthly',
        generatedBy: 'automation'
    );
    echo "Generated SLA report\n";

    // Example: Get analytics
    $analytics = $client->getAnalytics('24h');
    echo "Analytics: " . json_encode($analytics, JSON_PRETTY_PRINT) . "\n";

    // Example: List all systems
    $systems = $client->listSystems();
    echo "\nAll systems (" . count($systems) . "):\n";
    foreach ($systems as $s) {
        echo "  - {$s['name']} (status: {$s['status']})\n";
    }

    echo "\nAll examples completed successfully!\n";

} catch (StatusIncidentException $e) {
    echo "Error: " . $e->getMessage() . "\n";
    exit(1);
}
