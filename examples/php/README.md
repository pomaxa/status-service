# PHP Integration Examples

This directory contains PHP examples for integrating with the Status Incident Service.

## Files

- `StatusIncidentClient.php` - Full-featured PHP client for the Status Incident API
- `example.php` - Example usage of the client
- `health_endpoint.php` - Standalone health check endpoint
- `laravel_health.php` - Laravel-specific health endpoint implementation

## Requirements

- PHP 8.0+
- curl extension

## Quick Start

### Using the Client

```php
<?php

require_once 'StatusIncidentClient.php';

use StatusIncident\StatusIncidentClient;

// Initialize the client
$client = new StatusIncidentClient(
    baseUrl: 'http://localhost:8080',
    apiKey: 'your-api-key'  // Optional
);

// Create a system
$system = $client->createSystem(
    name: 'My Service',
    description: 'Production API service',
    url: 'https://api.example.com',
    owner: 'Backend Team'
);

// Update system status
$client->updateSystemStatus(
    systemId: $system['id'],
    status: 'yellow',
    message: 'High latency detected'
);

// Create an incident
$incident = $client->createIncident(
    title: 'Database Issues',
    message: 'Connection timeouts occurring',
    severity: 'major',
    systemIds: [$system['id']]
);

// Resolve the incident
$client->resolveIncident(
    incidentId: $incident['id'],
    message: 'Issue resolved after database restart'
);
```

### Setting Up Webhooks

```php
// Configure Slack notifications
$client->createWebhook(
    name: 'Slack Alerts',
    url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL',
    type: 'slack',
    events: ['system.status_changed', 'incident.created', 'incident.resolved']
);

// Configure Discord notifications
$client->createWebhook(
    name: 'Discord Alerts',
    url: 'https://discord.com/api/webhooks/YOUR/WEBHOOK',
    type: 'discord',
    events: ['incident.created', 'incident.resolved']
);
```

### Heartbeat Monitoring

```php
// Add a dependency to your system
$dependency = $client->createDependency(
    systemId: $system['id'],
    name: 'PostgreSQL',
    description: 'Primary database'
);

// Configure automatic health checks
$client->configureHeartbeat(
    dependencyId: $dependency['id'],
    url: 'https://api.example.com/health',
    interval: 60,  // Check every 60 seconds
    expectStatus: '200',
    expectBody: '.*ok.*'  // Regex to match response body
);

// Force an immediate check
$result = $client->forceCheck($dependency['id']);
```

### SLA Reporting

```php
// Generate monthly SLA report
$report = $client->generateSlaReport(
    title: 'February 2026 SLA Report',
    period: 'monthly',
    generatedBy: 'automation'
);

// Check for SLA breaches
$breaches = $client->listSlaBreaches(acknowledged: false);
foreach ($breaches as $breach) {
    echo "Breach: {$breach['title']}\n";
    
    // Acknowledge the breach
    $client->acknowledgeBreach(
        breachId: $breach['id'],
        by: 'ops-team'
    );
}
```

## Implementing Health Endpoints

For your service to be monitored, implement a health endpoint.

### Standalone PHP

```php
<?php

header('Content-Type: application/json');

$checks = [];
$healthy = true;

// Check database
try {
    $pdo = new PDO($dsn, $user, $pass);
    $pdo->query('SELECT 1');
    $checks['database'] = 'ok';
} catch (PDOException $e) {
    $checks['database'] = 'error: ' . $e->getMessage();
    $healthy = false;
}

if (!$healthy) {
    http_response_code(503);
}

echo json_encode([
    'status' => $healthy ? 'ok' : 'error',
    'checks' => $checks,
]);
```

### Laravel

```php
// routes/api.php
Route::get('/health', function () {
    $checks = [];
    $healthy = true;

    try {
        DB::select('SELECT 1');
        $checks['database'] = 'ok';
    } catch (\Exception $e) {
        $checks['database'] = 'error: ' . $e->getMessage();
        $healthy = false;
    }

    try {
        Redis::ping();
        $checks['redis'] = 'ok';
    } catch (\Exception $e) {
        $checks['redis'] = 'error: ' . $e->getMessage();
        $healthy = false;
    }

    return response()->json([
        'status' => $healthy ? 'ok' : 'error',
        'checks' => $checks,
    ], $healthy ? 200 : 503);
})->withoutMiddleware(['auth', 'throttle']);
```

### Symfony

```php
// src/Controller/HealthController.php
namespace App\Controller;

use Doctrine\DBAL\Connection;
use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\HttpFoundation\JsonResponse;
use Symfony\Component\Routing\Annotation\Route;

class HealthController extends AbstractController
{
    #[Route('/health', methods: ['GET'])]
    public function check(Connection $connection): JsonResponse
    {
        $checks = [];
        $healthy = true;

        try {
            $connection->executeQuery('SELECT 1');
            $checks['database'] = 'ok';
        } catch (\Exception $e) {
            $checks['database'] = 'error: ' . $e->getMessage();
            $healthy = false;
        }

        return new JsonResponse([
            'status' => $healthy ? 'ok' : 'error',
            'checks' => $checks,
        ], $healthy ? 200 : 503);
    }
}
```

## Authentication Options

The client supports three authentication methods:

### API Key (Recommended)

```php
$client = new StatusIncidentClient(
    baseUrl: 'http://localhost:8080',
    apiKey: 'your-api-key'
);
```

### Basic Auth

```php
$client = new StatusIncidentClient(
    baseUrl: 'http://localhost:8080',
    username: 'admin',
    password: 'password'
);
```

### No Authentication

```php
$client = new StatusIncidentClient(
    baseUrl: 'http://localhost:8080'
);
```

## Error Handling

```php
use StatusIncident\StatusIncidentException;

try {
    $system = $client->getSystem(999);
} catch (StatusIncidentException $e) {
    if (str_contains($e->getMessage(), 'status 404')) {
        echo "System not found\n";
    } else {
        echo "API error: " . $e->getMessage() . "\n";
    }
}
```

## Running the Examples

```bash
# Run the client example
php example.php

# Run the health endpoint example
php -S localhost:8080 health_endpoint.php
```
