# Changelog

## [Unreleased]

### Changed

- **BREAKING:** Upgrade `wspulse/server` dependency from v0.4.0 to v0.5.0
- `ConnectionClosed` now accepts a `DisconnectReason` parameter (matching server v0.5.0 interface)
- `connections.closed` and `connection.duration` now include a `disconnect.reason` attribute (`normal`, `kick`, `grace_expired`, `server_close`, `duplicate`)

### Added

- Initial release: `Collector` implementing `wspulse.MetricsCollector` with OpenTelemetry backend
- `NewCollector(opts ...Option)` constructor
- Options: `WithMeterProvider`, `WithNamespace`, `WithRoomAttribute`
