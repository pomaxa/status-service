# frozen_string_literal: true

# Health Check Controller for Ruby on Rails
#
# This demonstrates how to implement health check endpoints in Rails
# that work with the Status Incident Service's heartbeat monitoring.
#
# Installation:
#   1. Create this file at: app/controllers/health_controller.rb
#   2. Add routes to config/routes.rb (see bottom of file)
#   3. Configure Status Incident to monitor: http://your-app.com/health
#
# The health endpoint should be accessible without authentication.

class HealthController < ApplicationController
  # Skip authentication for health checks
  skip_before_action :authenticate_user!, raise: false
  skip_before_action :verify_authenticity_token, raise: false

  # GET /health
  # Full health check for Status Incident monitoring.
  # Returns HTTP 200 for healthy, HTTP 503 for unhealthy.
  def check
    checks = {}
    healthy = true

    # Database check
    begin
      ActiveRecord::Base.connection.execute('SELECT 1')
      checks[:database] = 'ok'
    rescue StandardError => e
      checks[:database] = "error: #{e.message}"
      healthy = false
    end

    # Redis check (if using Redis)
    if defined?(Redis) && Redis.current
      begin
        Redis.current.ping
        checks[:redis] = 'ok'
      rescue StandardError => e
        checks[:redis] = "error: #{e.message}"
        healthy = false
      end
    end

    # Sidekiq check (if using Sidekiq)
    if defined?(Sidekiq)
      begin
        Sidekiq.redis(&:ping)
        checks[:sidekiq] = 'ok'
      rescue StandardError => e
        checks[:sidekiq] = "error: #{e.message}"
        healthy = false
      end
    end

    # Cache check (if using Rails cache with Redis/Memcached)
    begin
      Rails.cache.write('health_check', 'ok', expires_in: 1.minute)
      if Rails.cache.read('health_check') == 'ok'
        checks[:cache] = 'ok'
      else
        checks[:cache] = 'error: cache read failed'
        healthy = false
      end
    rescue StandardError => e
      checks[:cache] = "error: #{e.message}"
      healthy = false
    end

    status_code = healthy ? :ok : :service_unavailable

    render json: {
      status: healthy ? 'ok' : 'error',
      checks: checks
    }, status: status_code
  end

  # GET /health/live
  # Kubernetes liveness probe - is the application running?
  def liveness
    render json: { status: 'ok' }
  end

  # GET /health/ready
  # Kubernetes readiness probe - is the application ready to serve traffic?
  def readiness
    ActiveRecord::Base.connection.execute('SELECT 1')
    render json: { status: 'ok' }
  rescue StandardError => e
    render json: {
      status: 'error',
      message: e.message
    }, status: :service_unavailable
  end
end

# ==================== Routes Configuration ====================
#
# Add to config/routes.rb:
#
# Rails.application.routes.draw do
#   # Health check routes (no authentication)
#   get '/health', to: 'health#check'
#   get '/health/live', to: 'health#liveness'
#   get '/health/ready', to: 'health#readiness'
#
#   # Or use a scope:
#   # scope '/health' do
#   #   get '/', to: 'health#check'
#   #   get '/live', to: 'health#liveness'
#   #   get '/ready', to: 'health#readiness'
#   # end
# end
