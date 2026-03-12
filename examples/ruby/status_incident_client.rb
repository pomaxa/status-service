# frozen_string_literal: true

# Status Incident Service - Ruby Client
#
# A Ruby client library for integrating with the Status Incident Service API.
# Supports system management, incident tracking, webhooks, and health monitoring.
#
# Requirements:
#   gem install httparty
#
# Usage:
#   require_relative 'status_incident_client'
#
#   client = StatusIncident::Client.new('http://localhost:8080', api_key: 'your-api-key')
#   systems = client.list_systems

require 'httparty'
require 'json'

module StatusIncident
  class Error < StandardError; end

  # Main client class for the Status Incident API.
  class Client
    include HTTParty

    # Initialize the client.
    #
    # @param base_url [String] Base URL of the Status Incident service
    # @param api_key [String, nil] API key for authentication (optional)
    # @param username [String, nil] Username for basic auth (optional)
    # @param password [String, nil] Password for basic auth (optional)
    # @param timeout [Integer] Request timeout in seconds
    def initialize(base_url, api_key: nil, username: nil, password: nil, timeout: 30)
      @base_url = base_url.chomp('/')
      @api_key = api_key
      @username = username
      @password = password
      @timeout = timeout
    end

    # ==================== Systems API ====================

    # List all systems.
    def list_systems
      get('/systems')
    end

    # Get a system by ID.
    def get_system(system_id)
      get("/systems/#{system_id}")
    end

    # Create a new system.
    #
    # @param name [String] System name
    # @param description [String] System description
    # @param url [String] System URL
    # @param owner [String] System owner
    def create_system(name:, description: '', url: '', owner: '')
      post('/systems', {
        name: name,
        description: description,
        url: url,
        owner: owner
      })
    end

    # Update a system.
    def update_system(system_id, name:, description: '', url: '', owner: '')
      put("/systems/#{system_id}", {
        name: name,
        description: description,
        url: url,
        owner: owner
      })
    end

    # Delete a system.
    def delete_system(system_id)
      delete("/systems/#{system_id}")
    end

    # Update system status.
    #
    # @param system_id [Integer] System ID
    # @param status [String] New status ("green", "yellow", or "red")
    # @param message [String] Optional status message
    def update_system_status(system_id, status:, message: '')
      post("/systems/#{system_id}/status", {
        status: status,
        message: message
      })
    end

    # Get system analytics.
    def get_system_analytics(system_id, period: '24h')
      get("/systems/#{system_id}/analytics", period: period)
    end

    # Get system SLA status.
    def get_system_sla(system_id, period: 'monthly')
      get("/systems/#{system_id}/sla", period: period)
    end

    # ==================== Dependencies API ====================

    # List all dependencies of a system.
    def list_dependencies(system_id)
      get("/systems/#{system_id}/dependencies")
    end

    # Create a new dependency.
    def create_dependency(system_id, name:, description: '')
      post("/systems/#{system_id}/dependencies", {
        name: name,
        description: description
      })
    end

    # Update a dependency.
    def update_dependency(dependency_id, name:, description: '')
      put("/dependencies/#{dependency_id}", {
        name: name,
        description: description
      })
    end

    # Delete a dependency.
    def delete_dependency(dependency_id)
      delete("/dependencies/#{dependency_id}")
    end

    # Update dependency status.
    def update_dependency_status(dependency_id, status:, message: '')
      post("/dependencies/#{dependency_id}/status", {
        status: status,
        message: message
      })
    end

    # Configure automatic health check for a dependency.
    #
    # @param dependency_id [Integer] Dependency ID
    # @param url [String] Health check URL
    # @param interval [Integer] Check interval in seconds (min: 10)
    # @param method [String] HTTP method (GET, POST, HEAD)
    # @param headers [Hash] Optional HTTP headers
    # @param expect_status [String] Expected HTTP status codes (comma-separated)
    # @param expect_body [String] Expected response body regex
    def configure_heartbeat(dependency_id, url:, interval: 60, method: 'GET',
                            headers: {}, expect_status: '200', expect_body: '')
      post("/dependencies/#{dependency_id}/heartbeat", {
        url: url,
        interval: interval,
        method: method,
        headers: headers,
        expect_status: expect_status,
        expect_body: expect_body
      })
    end

    # Disable heartbeat monitoring for a dependency.
    def disable_heartbeat(dependency_id)
      delete("/dependencies/#{dependency_id}/heartbeat")
    end

    # Force an immediate health check.
    def force_check(dependency_id)
      post("/dependencies/#{dependency_id}/check")
    end

    # ==================== Incidents API ====================

    # List incidents with optional filters.
    def list_incidents(status: nil, severity: nil)
      params = {}
      params[:status] = status if status
      params[:severity] = severity if severity
      get('/incidents', params)
    end

    # Get an incident by ID.
    def get_incident(incident_id)
      get("/incidents/#{incident_id}")
    end

    # Create a new incident.
    #
    # @param title [String] Incident title
    # @param message [String] Incident description
    # @param severity [String] Severity level ("minor", "major", "critical")
    # @param system_ids [Array<Integer>] List of affected system IDs
    def create_incident(title:, message:, severity: 'minor', system_ids: [])
      post('/incidents', {
        title: title,
        message: message,
        severity: severity,
        system_ids: system_ids
      })
    end

    # Update incident status or severity.
    def update_incident(incident_id, status: nil, severity: nil)
      data = {}
      data[:status] = status if status
      data[:severity] = severity if severity
      put("/incidents/#{incident_id}", data)
    end

    # Add an update to an incident.
    def add_incident_update(incident_id, message:, by: '')
      post("/incidents/#{incident_id}/updates", {
        message: message,
        by: by
      })
    end

    # Resolve an incident.
    def resolve_incident(incident_id, message: '', postmortem: '')
      post("/incidents/#{incident_id}/resolve", {
        message: message,
        postmortem: postmortem
      })
    end

    # ==================== Webhooks API ====================

    # List all webhooks.
    def list_webhooks
      get('/webhooks')
    end

    # Create a webhook.
    #
    # @param name [String] Webhook name
    # @param url [String] Webhook URL
    # @param type [String] Type ("slack", "discord", "telegram", "teams", "generic")
    # @param events [Array<String>] Events to subscribe to
    # @param system_ids [Array<Integer>] Systems to monitor (empty for all)
    # @param enabled [Boolean] Whether webhook is enabled
    def create_webhook(name:, url:, type: 'generic', events: nil, system_ids: [], enabled: true)
      events ||= %w[system.status_changed incident.created incident.resolved]
      post('/webhooks', {
        name: name,
        url: url,
        type: type,
        events: events,
        system_ids: system_ids,
        enabled: enabled
      })
    end

    # Delete a webhook.
    def delete_webhook(webhook_id)
      delete("/webhooks/#{webhook_id}")
    end

    # Send a test notification to a webhook.
    def test_webhook(webhook_id)
      post("/webhooks/#{webhook_id}/test")
    end

    # ==================== SLA API ====================

    # Generate an SLA report.
    def generate_sla_report(title:, period: 'monthly', generated_by: '')
      post('/sla/reports', {
        title: title,
        period: period,
        generated_by: generated_by
      })
    end

    # List SLA breaches.
    def list_sla_breaches(acknowledged: nil)
      params = {}
      params[:acknowledged] = acknowledged.to_s unless acknowledged.nil?
      get('/sla/breaches', params)
    end

    # Acknowledge an SLA breach.
    def acknowledge_breach(breach_id, by: '')
      post("/sla/breaches/#{breach_id}/acknowledge", { by: by })
    end

    # ==================== Maintenance API ====================

    # List all maintenance windows.
    def list_maintenances
      get('/maintenances')
    end

    # Schedule a maintenance window.
    #
    # @param title [String] Maintenance title
    # @param description [String] Description
    # @param start_time [String, Time] Start time (ISO 8601 format)
    # @param end_time [String, Time] End time (ISO 8601 format)
    # @param system_ids [Array<Integer>] Affected system IDs
    def create_maintenance(title:, description:, start_time:, end_time:, system_ids: [])
      post('/maintenances', {
        title: title,
        description: description,
        start_time: start_time.respond_to?(:iso8601) ? start_time.iso8601 : start_time,
        end_time: end_time.respond_to?(:iso8601) ? end_time.iso8601 : end_time,
        system_ids: system_ids
      })
    end

    # Cancel a scheduled maintenance.
    def cancel_maintenance(maintenance_id)
      delete("/maintenances/#{maintenance_id}")
    end

    # ==================== Export/Import API ====================

    # Export all data.
    def export_data
      get('/export')
    end

    # Import data from a backup.
    def import_data(data)
      post('/import', data)
    end

    # ==================== Analytics API ====================

    # Get overall analytics.
    def get_analytics(period: '24h')
      get('/analytics', period: period)
    end

    # Get change logs.
    def get_logs(limit: 100)
      get('/logs', limit: limit)
    end

    private

    def headers
      h = {
        'Content-Type' => 'application/json',
        'Accept' => 'application/json'
      }
      h['X-API-Key'] = @api_key if @api_key
      h
    end

    def auth
      return nil unless @username && @password

      { username: @username, password: @password }
    end

    def request_options
      opts = {
        headers: headers,
        timeout: @timeout
      }
      opts[:basic_auth] = auth if auth
      opts
    end

    def get(endpoint, params = {})
      url = "#{@base_url}/api#{endpoint}"
      response = HTTParty.get(url, request_options.merge(query: params))
      handle_response(response)
    end

    def post(endpoint, body = nil)
      url = "#{@base_url}/api#{endpoint}"
      opts = request_options
      opts[:body] = body.to_json if body
      response = HTTParty.post(url, opts)
      handle_response(response)
    end

    def put(endpoint, body)
      url = "#{@base_url}/api#{endpoint}"
      opts = request_options
      opts[:body] = body.to_json
      response = HTTParty.put(url, opts)
      handle_response(response)
    end

    def delete(endpoint)
      url = "#{@base_url}/api#{endpoint}"
      response = HTTParty.delete(url, request_options)
      handle_response(response)
    end

    def handle_response(response)
      return response.parsed_response if response.success?

      raise Error, "API error (status #{response.code}): #{response.body}"
    end
  end
end

# ==================== Example Usage ====================

if __FILE__ == $PROGRAM_NAME
  client = StatusIncident::Client.new(
    'http://localhost:8080',
    api_key: 'your-api-key'
  )

  # Example: Create a system
  system = client.create_system(
    name: 'Payment Service',
    description: 'Handles all payment processing',
    url: 'https://payments.example.com',
    owner: 'Payments Team'
  )
  puts "Created system: #{system['name']} (ID: #{system['id']})"

  # Example: Add a dependency with heartbeat monitoring
  dependency = client.create_dependency(
    system['id'],
    name: 'PostgreSQL',
    description: 'Primary database'
  )
  puts "Created dependency: #{dependency['name']}"

  # Configure automatic health checking
  client.configure_heartbeat(
    dependency['id'],
    url: 'https://payments.example.com/health',
    interval: 60
  )
  puts 'Configured heartbeat monitoring'

  # Example: Create an incident
  incident = client.create_incident(
    title: 'Database Connection Issues',
    message: 'Experiencing intermittent connection failures to the primary database',
    severity: 'major',
    system_ids: [system['id']]
  )
  puts "Created incident: #{incident['title']}"

  # Add an update
  client.add_incident_update(
    incident['id'],
    message: 'Identified root cause: connection pool exhaustion',
    by: 'ops-team'
  )

  # Resolve the incident
  client.resolve_incident(
    incident['id'],
    message: 'Increased connection pool size and deployed fix',
    postmortem: 'Connection pool was undersized for traffic load'
  )
  puts 'Incident resolved'

  # Example: Set up Slack notifications
  webhook = client.create_webhook(
    name: 'Slack Alerts',
    url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL',
    type: 'slack',
    events: %w[system.status_changed incident.created incident.resolved]
  )
  puts "Created webhook: #{webhook['name']}"
end
