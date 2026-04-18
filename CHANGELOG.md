# Changelog

## [Unreleased]

## [v0.4.0] - 2026-04-18

### Breaking changes

- Renamed `PongTimeout` method to `HeartbeatFailed` to match the updated
  `MetricsCollector` interface. OTel instrument renamed from
  `wspulse.pong.timeouts` to `wspulse.heartbeat.failures`.

### Changed

- Upgraded `github.com/wspulse/hub` from v0.8.1 to v0.10.0.

---

## [v0.3.0] - 2026-04-04

### Changed

- Replaced integration tests with direct `Collector` component tests for deterministic metric assertions
- Removed network I/O from the test suite
- Adopted `testify` for test assertions
- Removed integration CI job; component tests run in main `check` pipeline

---

## [v0.2.0] - 2026-03-27

### Added

- Initial release: `Collector` implementing `wspulse.MetricsCollector` (server >= v0.6.0) with OpenTelemetry v1.42.0 backend
- `NewCollector(opts ...Option)` constructor
- Options: `WithMeterProvider`, `WithNamespace`, `WithRoomAttribute`
- 16 instruments: counters, up-down counters, and histograms for connection lifecycle, room lifecycle, throughput, and heartbeat
- Attributes: `room.id` (controlled by `WithRoomAttribute`), `disconnect.reason` on `connections.closed` and `connection.duration`
- Explicit histogram bucket boundaries for `connection.duration` (1s-24h), `broadcast.fanout` (1-1000), and `send_buffer.utilization` (0.1-1.0)
- `doc/usage.md` with instruments table, attributes reference, histogram boundaries, and configuration examples
