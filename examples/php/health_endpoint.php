<?php

/**
 * Health Endpoint Example for PHP
 *
 * This demonstrates how to implement health check endpoints that work with
 * the Status Incident Service's heartbeat monitoring feature.
 *
 * For standalone PHP:
 *   php -S localhost:8080 health_endpoint.php
 *
 * Then configure Status Incident to monitor: http://localhost:8080/health
 */

declare(strict_types=1);

/**
 * Health check response builder.
 */
class HealthChecker
{
    private ?PDO $db = null;
    private ?Redis $redis = null;

    public function __construct(?PDO $db = null, ?Redis $redis = null)
    {
        $this->db = $db;
        $this->redis = $redis;
    }

    /**
     * Perform all health checks.
     */
    public function check(): array
    {
        $checks = [];
        $healthy = true;

        // Database check
        if ($this->db !== null) {
            try {
                $this->db->query('SELECT 1');
                $checks['database'] = 'ok';
            } catch (PDOException $e) {
                $checks['database'] = 'error: ' . $e->getMessage();
                $healthy = false;
            }
        } else {
            $checks['database'] = 'ok (mock)';
        }

        // Redis check
        if ($this->redis !== null) {
            try {
                $this->redis->ping();
                $checks['redis'] = 'ok';
            } catch (RedisException $e) {
                $checks['redis'] = 'error: ' . $e->getMessage();
                $healthy = false;
            }
        } else {
            $checks['redis'] = 'ok (mock)';
        }

        return [
            'status' => $healthy ? 'ok' : 'error',
            'checks' => $checks,
            'healthy' => $healthy,
        ];
    }
}

/**
 * Simple router for the health endpoints.
 */
function handleRequest(): void
{
    $uri = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
    
    // Initialize health checker (replace with your actual connections)
    $checker = new HealthChecker();

    header('Content-Type: application/json');

    switch ($uri) {
        case '/health':
            $result = $checker->check();
            if (!$result['healthy']) {
                http_response_code(503);
            }
            unset($result['healthy']);
            echo json_encode($result);
            break;

        case '/health/live':
            // Kubernetes liveness probe - is the application running?
            echo json_encode(['status' => 'ok']);
            break;

        case '/health/ready':
            // Kubernetes readiness probe - is the application ready?
            $result = $checker->check();
            if (!$result['healthy']) {
                http_response_code(503);
            }
            echo json_encode(['status' => $result['status']]);
            break;

        default:
            http_response_code(404);
            echo json_encode(['error' => 'Not found']);
    }
}

// Run the router
handleRequest();
