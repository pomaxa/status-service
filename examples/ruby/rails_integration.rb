# frozen_string_literal: true

# Ruby on Rails Integration Example
#
# This demonstrates how to integrate Status Incident Service with a Rails application.
# It includes a service class for API interactions and examples of usage patterns.

# ==================== Configuration ====================
#
# Add to config/initializers/status_incident.rb:
#
# StatusIncident.configure do |config|
#   config.base_url = ENV.fetch('STATUS_INCIDENT_URL', 'http://localhost:8080')
#   config.api_key = ENV.fetch('STATUS_INCIDENT_API_KEY', nil)
# end

# ==================== Service Class ====================
#
# Create at: app/services/status_incident_service.rb

require_relative 'status_incident_client'

class StatusIncidentService
  class << self
    def client
      @client ||= StatusIncident::Client.new(
        ENV.fetch('STATUS_INCIDENT_URL', 'http://localhost:8080'),
        api_key: ENV.fetch('STATUS_INCIDENT_API_KEY', nil)
      )
    end

    # Update system status based on application state
    def update_status(system_id, status:, message: nil)
      client.update_system_status(system_id, status: status, message: message || '')
      Rails.logger.info "[StatusIncident] Updated system #{system_id} to #{status}"
    rescue StandardError => e
      Rails.logger.error "[StatusIncident] Failed to update status: #{e.message}"
      raise unless Rails.env.production?
    end

    # Create an incident
    def create_incident(title:, message:, severity: 'minor', system_ids: [])
      incident = client.create_incident(
        title: title,
        message: message,
        severity: severity,
        system_ids: system_ids
      )
      Rails.logger.info "[StatusIncident] Created incident: #{incident['id']}"
      incident
    rescue StandardError => e
      Rails.logger.error "[StatusIncident] Failed to create incident: #{e.message}"
      raise unless Rails.env.production?
    end

    # Resolve an incident
    def resolve_incident(incident_id, message: '', postmortem: '')
      client.resolve_incident(incident_id, message: message, postmortem: postmortem)
      Rails.logger.info "[StatusIncident] Resolved incident: #{incident_id}"
    rescue StandardError => e
      Rails.logger.error "[StatusIncident] Failed to resolve incident: #{e.message}"
      raise unless Rails.env.production?
    end

    # Report a degraded state
    def report_degraded(system_id, reason)
      update_status(system_id, status: 'yellow', message: reason)
    end

    # Report an outage
    def report_outage(system_id, reason)
      update_status(system_id, status: 'red', message: reason)
    end

    # Report recovery
    def report_recovery(system_id, message = 'System recovered')
      update_status(system_id, status: 'green', message: message)
    end
  end
end

# ==================== Usage in Controllers ====================
#
# class PaymentsController < ApplicationController
#   rescue_from StandardError, with: :handle_error
#
#   def create
#     @payment = Payment.create!(payment_params)
#     render json: @payment
#   end
#
#   private
#
#   def handle_error(error)
#     # Report degraded performance on errors
#     StatusIncidentService.report_degraded(
#       ENV['SYSTEM_ID'],
#       "Payment processing error: #{error.class.name}"
#     )
#     raise error
#   end
# end

# ==================== Background Job Integration ====================
#
# class StatusUpdateJob < ApplicationJob
#   queue_as :default
#
#   def perform(system_id, status, message)
#     StatusIncidentService.update_status(system_id, status: status, message: message)
#   end
# end
#
# # Queue status update asynchronously
# StatusUpdateJob.perform_later(1, 'yellow', 'High latency detected')

# ==================== Rack Middleware for Automatic Monitoring ====================
#
# Create at: lib/middleware/status_incident_middleware.rb

class StatusIncidentMiddleware
  FAILURE_THRESHOLD = 10
  RECOVERY_THRESHOLD = 5

  def initialize(app, system_id:)
    @app = app
    @system_id = system_id
    @failure_count = 0
    @success_count = 0
    @current_status = 'green'
    @mutex = Mutex.new
  end

  def call(env)
    response = @app.call(env)
    status = response[0]

    @mutex.synchronize do
      if status >= 500
        handle_failure
      else
        handle_success
      end
    end

    response
  rescue StandardError => e
    @mutex.synchronize { handle_failure }
    raise e
  end

  private

  def handle_failure
    @failure_count += 1
    @success_count = 0

    if @failure_count >= FAILURE_THRESHOLD && @current_status != 'red'
      update_status('red', "#{@failure_count} consecutive failures")
    elsif @failure_count >= FAILURE_THRESHOLD / 2 && @current_status == 'green'
      update_status('yellow', "#{@failure_count} failures detected")
    end
  end

  def handle_success
    @success_count += 1

    if @success_count >= RECOVERY_THRESHOLD && @current_status != 'green'
      @failure_count = 0
      update_status('green', 'Service recovered')
    end
  end

  def update_status(status, message)
    @current_status = status
    Thread.new do
      StatusIncidentService.update_status(@system_id, status: status, message: message)
    end
  end
end

# Add to config/application.rb:
# config.middleware.use StatusIncidentMiddleware, system_id: ENV['SYSTEM_ID'].to_i

# ==================== ActiveSupport Notifications Integration ====================
#
# Create at: config/initializers/status_incident_subscriber.rb

# Subscribe to slow database queries
ActiveSupport::Notifications.subscribe('sql.active_record') do |*args|
  event = ActiveSupport::Notifications::Event.new(*args)
  
  # Report if query takes longer than 5 seconds
  if event.duration > 5000
    StatusIncidentService.report_degraded(
      ENV['SYSTEM_ID'].to_i,
      "Slow database query: #{event.duration.round}ms"
    )
  end
end

# Subscribe to controller errors
ActiveSupport::Notifications.subscribe('process_action.action_controller') do |*args|
  event = ActiveSupport::Notifications::Event.new(*args)
  
  if event.payload[:exception]
    exception_class, exception_message = event.payload[:exception]
    StatusIncidentService.create_incident(
      title: "Application Error: #{exception_class}",
      message: exception_message,
      severity: 'major',
      system_ids: [ENV['SYSTEM_ID'].to_i]
    )
  end
end

# ==================== Rake Tasks ====================
#
# Create at: lib/tasks/status_incident.rake

namespace :status_incident do
  desc 'Update system status'
  task :update_status, %i[status message] => :environment do |_t, args|
    system_id = ENV.fetch('SYSTEM_ID')
    StatusIncidentService.update_status(
      system_id.to_i,
      status: args[:status],
      message: args[:message]
    )
    puts "Updated system #{system_id} to #{args[:status]}"
  end

  desc 'Create an incident'
  task :create_incident, %i[title message severity] => :environment do |_t, args|
    system_id = ENV.fetch('SYSTEM_ID')
    incident = StatusIncidentService.create_incident(
      title: args[:title],
      message: args[:message],
      severity: args[:severity] || 'minor',
      system_ids: [system_id.to_i]
    )
    puts "Created incident: #{incident['id']}"
  end

  desc 'Report system healthy'
  task healthy: :environment do
    system_id = ENV.fetch('SYSTEM_ID')
    StatusIncidentService.report_recovery(system_id.to_i)
    puts "System #{system_id} marked as healthy"
  end
end

# Usage:
# rails status_incident:update_status[yellow,"High latency detected"]
# rails status_incident:create_incident["Database Issues","Connection timeouts",major]
# rails status_incident:healthy
