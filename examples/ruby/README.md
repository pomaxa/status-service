# Ruby on Rails Integration Examples

This directory contains Ruby examples for integrating with the Status Incident Service.

## Files

- `status_incident_client.rb` - Full-featured Ruby client for the Status Incident API
- `rails_health_controller.rb` - Rails health check controller implementation
- `rails_integration.rb` - Advanced Rails integration patterns (service class, middleware, notifications)
- `Gemfile` - Ruby dependencies

## Requirements

- Ruby 3.0+
- HTTParty gem

## Installation

```bash
gem install httparty

# Or add to your Gemfile:
gem 'httparty', '~> 0.24'
```

## Quick Start

### Using the Client

```ruby
require_relative 'status_incident_client'

# Initialize the client
client = StatusIncident::Client.new(
  'http://localhost:8080',
  api_key: 'your-api-key'  # Optional
)

# Create a system
system = client.create_system(
  name: 'My Service',
  description: 'Production API service',
  url: 'https://api.example.com',
  owner: 'Backend Team'
)

# Update system status
client.update_system_status(
  system['id'],
  status: 'yellow',
  message: 'High latency detected'
)

# Create an incident
incident = client.create_incident(
  title: 'Database Issues',
  message: 'Connection timeouts occurring',
  severity: 'major',
  system_ids: [system['id']]
)

# Resolve the incident
client.resolve_incident(
  incident['id'],
  message: 'Issue resolved after database restart'
)
```

### Setting Up Webhooks

```ruby
# Configure Slack notifications
client.create_webhook(
  name: 'Slack Alerts',
  url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL',
  type: 'slack',
  events: %w[system.status_changed incident.created incident.resolved]
)

# Configure Discord notifications
client.create_webhook(
  name: 'Discord Alerts',
  url: 'https://discord.com/api/webhooks/YOUR/WEBHOOK',
  type: 'discord',
  events: %w[incident.created incident.resolved]
)
```

### Heartbeat Monitoring

```ruby
# Add a dependency to your system
dependency = client.create_dependency(
  system['id'],
  name: 'PostgreSQL',
  description: 'Primary database'
)

# Configure automatic health checks
client.configure_heartbeat(
  dependency['id'],
  url: 'https://api.example.com/health',
  interval: 60,  # Check every 60 seconds
  expect_status: '200',
  expect_body: '.*ok.*'  # Regex to match response body
)

# Force an immediate check
result = client.force_check(dependency['id'])
```

### SLA Reporting

```ruby
# Generate monthly SLA report
report = client.generate_sla_report(
  title: 'February 2026 SLA Report',
  period: 'monthly',
  generated_by: 'automation'
)

# Check for SLA breaches
breaches = client.list_sla_breaches(acknowledged: false)
breaches.each do |breach|
  puts "Breach: #{breach['title']}"
  
  # Acknowledge the breach
  client.acknowledge_breach(breach['id'], by: 'ops-team')
end
```

## Rails Integration

### Health Controller

Create `app/controllers/health_controller.rb`:

```ruby
class HealthController < ApplicationController
  skip_before_action :authenticate_user!, raise: false

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

    # Redis check
    begin
      Redis.current.ping
      checks[:redis] = 'ok'
    rescue StandardError => e
      checks[:redis] = "error: #{e.message}"
      healthy = false
    end

    render json: {
      status: healthy ? 'ok' : 'error',
      checks: checks
    }, status: healthy ? :ok : :service_unavailable
  end
end
```

Add to `config/routes.rb`:

```ruby
get '/health', to: 'health#check'
```

### Service Class

Create `app/services/status_incident_service.rb`:

```ruby
class StatusIncidentService
  class << self
    def client
      @client ||= StatusIncident::Client.new(
        ENV.fetch('STATUS_INCIDENT_URL'),
        api_key: ENV.fetch('STATUS_INCIDENT_API_KEY', nil)
      )
    end

    def report_degraded(system_id, reason)
      client.update_system_status(system_id, status: 'yellow', message: reason)
    end

    def report_outage(system_id, reason)
      client.update_system_status(system_id, status: 'red', message: reason)
    end

    def report_recovery(system_id)
      client.update_system_status(system_id, status: 'green', message: 'Recovered')
    end
  end
end
```

### Background Job

```ruby
class StatusUpdateJob < ApplicationJob
  queue_as :default

  def perform(system_id, status, message)
    StatusIncidentService.client.update_system_status(
      system_id,
      status: status,
      message: message
    )
  end
end

# Queue status update asynchronously
StatusUpdateJob.perform_later(1, 'yellow', 'High latency')
```

### Rake Tasks

```ruby
# lib/tasks/status_incident.rake
namespace :status_incident do
  desc 'Update system status'
  task :update, %i[status message] => :environment do |_t, args|
    StatusIncidentService.client.update_system_status(
      ENV['SYSTEM_ID'].to_i,
      status: args[:status],
      message: args[:message]
    )
  end
end

# Usage: rails status_incident:update[yellow,"High latency"]
```

## Authentication Options

The client supports three authentication methods:

### API Key (Recommended)

```ruby
client = StatusIncident::Client.new(
  'http://localhost:8080',
  api_key: 'your-api-key'
)
```

### Basic Auth

```ruby
client = StatusIncident::Client.new(
  'http://localhost:8080',
  username: 'admin',
  password: 'password'
)
```

### No Authentication

```ruby
client = StatusIncident::Client.new('http://localhost:8080')
```

## Error Handling

```ruby
begin
  system = client.get_system(999)
rescue StatusIncident::Error => e
  if e.message.include?('status 404')
    puts 'System not found'
  else
    puts "API error: #{e.message}"
  end
end
```

## Environment Variables

For Rails applications, set these environment variables:

```bash
export STATUS_INCIDENT_URL=http://localhost:8080
export STATUS_INCIDENT_API_KEY=your-api-key
export SYSTEM_ID=1
```

## Running the Examples

```bash
# Install dependencies
bundle install

# Run the client example
ruby status_incident_client.rb
```
