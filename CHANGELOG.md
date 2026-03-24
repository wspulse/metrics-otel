# Changelog

## [Unreleased]

### Added

- Initial release: `Collector` implementing `wspulse.MetricsCollector` with OpenTelemetry backend
- `NewCollector(opts ...Option)` constructor
- Options: `WithMeterProvider`, `WithNamespace`, `WithRoomAttribute`
