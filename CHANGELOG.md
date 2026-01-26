# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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
