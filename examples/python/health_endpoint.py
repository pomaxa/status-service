"""
Health Endpoint Examples for Python

This file demonstrates how to implement health check endpoints that work with
the Status Incident Service's heartbeat monitoring feature.

Run with:
    pip install flask
    python health_endpoint.py

Or for FastAPI:
    pip install fastapi uvicorn
    uvicorn health_endpoint:fastapi_app --reload
"""

# ==================== Flask Example ====================

from flask import Flask, jsonify

flask_app = Flask(__name__)

# Simulated dependencies (replace with your actual connections)
class MockDB:
    def ping(self):
        return True

class MockRedis:
    def ping(self):
        return True

db = MockDB()
redis_client = MockRedis()


@flask_app.route("/health")
def health():
    """
    Health check endpoint for Status Incident monitoring.
    
    Returns HTTP 200 for healthy, HTTP 503 for unhealthy.
    """
    checks = {}
    healthy = True

    # Check database
    try:
        db.ping()
        checks["database"] = "ok"
    except Exception as e:
        checks["database"] = f"error: {str(e)}"
        healthy = False

    # Check Redis
    try:
        redis_client.ping()
        checks["redis"] = "ok"
    except Exception as e:
        checks["redis"] = f"error: {str(e)}"
        healthy = False

    status_code = 200 if healthy else 503
    return jsonify({
        "status": "ok" if healthy else "error",
        "checks": checks
    }), status_code


@flask_app.route("/health/live")
def liveness():
    """Kubernetes liveness probe - is the application running?"""
    return jsonify({"status": "ok"}), 200


@flask_app.route("/health/ready")
def readiness():
    """Kubernetes readiness probe - is the application ready to serve traffic?"""
    try:
        db.ping()
        return jsonify({"status": "ok"}), 200
    except Exception as e:
        return jsonify({"status": "error", "message": str(e)}), 503


# ==================== FastAPI Example ====================

try:
    from fastapi import FastAPI, Response
    from typing import Dict

    fastapi_app = FastAPI(title="Health Check Example")

    @fastapi_app.get("/health")
    async def fastapi_health(response: Response) -> Dict:
        """
        Health check endpoint for Status Incident monitoring.
        
        Returns HTTP 200 for healthy, HTTP 503 for unhealthy.
        """
        checks = {}
        healthy = True

        # Check database
        try:
            db.ping()
            checks["database"] = "ok"
        except Exception as e:
            checks["database"] = f"error: {str(e)}"
            healthy = False

        # Check Redis
        try:
            redis_client.ping()
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

    @fastapi_app.get("/health/live")
    async def fastapi_liveness() -> Dict:
        """Kubernetes liveness probe."""
        return {"status": "ok"}

    @fastapi_app.get("/health/ready")
    async def fastapi_readiness(response: Response) -> Dict:
        """Kubernetes readiness probe."""
        try:
            db.ping()
            return {"status": "ok"}
        except Exception as e:
            response.status_code = 503
            return {"status": "error", "message": str(e)}

except ImportError:
    fastapi_app = None


# ==================== Django Example (for reference) ====================
"""
# views.py
from django.http import JsonResponse
from django.db import connection

def health(request):
    checks = {}
    healthy = True

    # Check database
    try:
        with connection.cursor() as cursor:
            cursor.execute("SELECT 1")
        checks["database"] = "ok"
    except Exception as e:
        checks["database"] = f"error: {str(e)}"
        healthy = False

    status_code = 200 if healthy else 503
    return JsonResponse({
        "status": "ok" if healthy else "error",
        "checks": checks
    }, status=status_code)

# urls.py
from django.urls import path
from . import views

urlpatterns = [
    path('health/', views.health, name='health'),
]
"""


if __name__ == "__main__":
    print("Starting Flask health endpoint example on http://localhost:5000")
    print("Endpoints:")
    print("  - GET /health       - Full health check")
    print("  - GET /health/live  - Liveness probe")
    print("  - GET /health/ready - Readiness probe")
    flask_app.run(port=5000, debug=True)
