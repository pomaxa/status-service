<?php

/**
 * Laravel Health Endpoint Example
 *
 * This demonstrates how to implement health check endpoints in Laravel
 * that work with the Status Incident Service's heartbeat monitoring.
 *
 * Add these to your routes/api.php or routes/web.php
 */

use Illuminate\Http\JsonResponse;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Redis;
use Illuminate\Support\Facades\Route;

/**
 * Health Check Controller
 *
 * Create this file at: app/Http/Controllers/HealthController.php
 */

/*
namespace App\Http\Controllers;

use Illuminate\Http\JsonResponse;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Redis;

class HealthController extends Controller
{
    public function check(): JsonResponse
    {
        $checks = [];
        $healthy = true;

        // Database check
        try {
            DB::connection()->getPdo();
            DB::select('SELECT 1');
            $checks['database'] = 'ok';
        } catch (\Exception $e) {
            $checks['database'] = 'error: ' . $e->getMessage();
            $healthy = false;
        }

        // Redis check (if using Redis)
        try {
            Redis::ping();
            $checks['redis'] = 'ok';
        } catch (\Exception $e) {
            $checks['redis'] = 'error: ' . $e->getMessage();
            $healthy = false;
        }

        // Queue check (optional)
        // $checks['queue'] = $this->checkQueue();

        $status = $healthy ? 200 : 503;
        
        return response()->json([
            'status' => $healthy ? 'ok' : 'error',
            'checks' => $checks,
        ], $status);
    }

    public function liveness(): JsonResponse
    {
        return response()->json(['status' => 'ok']);
    }

    public function readiness(): JsonResponse
    {
        try {
            DB::connection()->getPdo();
            return response()->json(['status' => 'ok']);
        } catch (\Exception $e) {
            return response()->json([
                'status' => 'error',
                'message' => $e->getMessage(),
            ], 503);
        }
    }
}
*/

/**
 * Routes (add to routes/api.php or routes/web.php)
 */

/*
// Health check routes (no authentication required)
Route::prefix('health')->withoutMiddleware(['auth', 'throttle'])->group(function () {
    Route::get('/', [HealthController::class, 'check']);
    Route::get('/live', [HealthController::class, 'liveness']);
    Route::get('/ready', [HealthController::class, 'readiness']);
});
*/

/**
 * Alternative: Inline route definitions (for simpler setups)
 */

Route::get('/health', function (): JsonResponse {
    $checks = [];
    $healthy = true;

    // Database check
    try {
        DB::connection()->getPdo();
        DB::select('SELECT 1');
        $checks['database'] = 'ok';
    } catch (\Exception $e) {
        $checks['database'] = 'error: ' . $e->getMessage();
        $healthy = false;
    }

    // Redis check
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

Route::get('/health/live', function (): JsonResponse {
    return response()->json(['status' => 'ok']);
})->withoutMiddleware(['auth', 'throttle']);

Route::get('/health/ready', function (): JsonResponse {
    try {
        DB::connection()->getPdo();
        return response()->json(['status' => 'ok']);
    } catch (\Exception $e) {
        return response()->json([
            'status' => 'error',
            'message' => $e->getMessage(),
        ], 503);
    }
})->withoutMiddleware(['auth', 'throttle']);
