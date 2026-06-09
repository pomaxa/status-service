# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [1.3.0] - 2026-06-09

### Added
- Integration examples for Python, Go, PHP, and Ruby on Rails
  - API clients covering system management, incidents, webhooks, and SLA
  - Health endpoint implementations for heartbeat monitoring
  - Authentication options (API key, basic auth) and framework-specific patterns (Flask/FastAPI, Laravel, Rails)

### Security
- Replaced insecure base64(username:password) session tokens with secure random tokens, added a session store with expiration and automatic cleanup
- Fixed open-redirect vulnerability in the login redirect parameter
- Added `Secure` (over TLS) and `SameSite=Strict` flags to all cookies
- Fixed API key hashing to use SHA-256; lookup and comparison now operate on hashes instead of plaintext keys
- Added SSRF protection (blocks localhost, loopback, link-local, and RFC1918 addresses) for heartbeat URLs, webhook URLs, and dependency heartbeat configuration
- Added security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Referrer-Policy, Content-Security-Policy, Permissions-Policy)
- Added HTML escaping for error messages on the login page

### Fixed
- Fix dependency creation failing with "NOT NULL constraint failed: dependencies.heartbeat_method"
  - Migration v8 made `heartbeat_method` `NOT NULL DEFAULT 'GET'`, but the repository bound an explicit SQL NULL for an empty method (which bypasses the column default in SQLite), so the Add Dependency flow always failed
  - The repository now defaults an empty method to `GET` on both create and update, covering every write path (create, update, status change, clear-heartbeat, backup import)
  - Removed test workarounds that masked the bug and added regression tests for the create-without-method and clear-heartbeat paths

## [1.2.0] - 2026-02-04

### Added
- Automatic status propagation from dependencies to parent systems
  - When any dependency has issues (yellow/red), the parent system's status is automatically updated to reflect the worst-case status
  - Aggregation strategy: any red → system red, any yellow (no red) → system yellow, all green → system green
  - Triggered after heartbeat checks, manual dependency status updates, and dependency deletion
  - New `SourcePropagation` change source for tracking propagated status changes in logs
  - New `MaxSeverityStatus()` helper function for calculating aggregate status

## [1.1.1] - 2026-01-26

### Fixed
- Fix Overall Performance metrics not correlating with Per-System Analytics
  - Previously, overall uptime/availability was calculated incorrectly when multiple systems had incidents at different times
  - Now overall metrics are computed as the average of per-system metrics, ensuring consistency between Overall Performance and Per-System Analytics views

### Tests
- Added 7 comprehensive correlation tests:
  - Non-overlapping incidents (original bug scenario)
  - Overlapping incidents across systems
  - Single system (overall = per-system)
  - No incidents (100% uptime)
  - No systems edge case
  - Yellow vs Red status (uptime vs availability distinction)
  - Many systems with varied downtime durations

## [1.1.0] - 2026-01-21

### Added
- Advanced health check configuration (custom headers, expected status/body)
- Microsoft Teams webhook support
- Prometheus metrics endpoint with enhanced metrics
- Comprehensive test coverage for domain and application layers

### Fixed
- Allow zero latency in http_checker tests
