# Changelog

## [Unreleased]

### Added

- Initial release: `Collector` implementing `wspulse.MetricsCollector` (server >= v0.5.0) with OpenTelemetry v1.42.0 backend
- `NewCollector(opts ...Option)` constructor
- Options: `WithMeterProvider`, `WithNamespace`, `WithRoomAttribute`
- 16 instruments: counters, up-down counters, and histograms for connection lifecycle, room lifecycle, throughput, and heartbeat
- Attributes: `room.id` (controlled by `WithRoomAttribute`), `disconnect.reason` on `connections.closed` and `connection.duration`, `success` on `resume.attempts`
- Explicit histogram bucket boundaries for `connection.duration` (1s-24h), `broadcast.fanout` (1-1000), and `send_buffer.utilization` (0.1-1.0)
- `doc/usage.md` with instruments table, attributes reference, histogram boundaries, and configuration examples
