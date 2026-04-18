# Changelog

## [Unreleased]

## [v0.5.0] - 2026-04-18

### Breaking changes

- Renamed `PongTimeout` method to `HeartbeatFailed` to match the updated
  `MetricsCollector` interface. OTel instrument renamed from
  `wspulse.pong.timeouts` to `wspulse.heartbeat.failures`.

### Changed

- Upgraded `github.com/wspulse/hub` from v0.8.1 to v0.10.0.

### Chore

- Added `pr-to-main-gate` CI workflow.

---

## [v0.4.0] - 2026-04-09

### Breaking changes

- Migrated import path from `github.com/wspulse/server` to
  `github.com/wspulse/hub` following the upstream rename.

### Changed

- Upgraded `github.com/wspulse/hub` from v0.6.0 to v0.8.1.

---

## [v0.3.0] - 2026-04-04

### Changed

- Replaced integration tests with direct `Collector` component tests for deterministic metric assertions
- Removed network I/O from the test suite
- Adopted `testify` for test assertions
- Removed integration CI job; component tests run in main `check` pipeline

---

## [v0.2.0] - 2026-03-27

### Breaking changes

- Removed `success` attribute from `ResumeAttempt` to match the updated
  `MetricsCollector` interface.

### Changed

- Upgraded `github.com/wspulse/server` to v0.6.0.

---

## [v0.1.0] - 2026-03-26

### Added

- Initial release: `Collector` implementing `wspulse.MetricsCollector` (server >= v0.5.0) with OpenTelemetry v1.42.0 backend
- `NewCollector(opts ...Option)` constructor
- Options: `WithMeterProvider`, `WithNamespace`, `WithRoomAttribute`
- 16 instruments: counters, up-down counters, and histograms for connection lifecycle, room lifecycle, throughput, and heartbeat
- Attributes: `room.id` (controlled by `WithRoomAttribute`), `disconnect.reason` on `connections.closed` and `connection.duration`
- Explicit histogram bucket boundaries for `connection.duration` (1s-24h), `broadcast.fanout` (1-1000), and `send_buffer.utilization` (0.1-1.0)
- `doc/usage.md` with instruments table, attributes reference, histogram boundaries, and configuration examples

[Unreleased]: https://github.com/wspulse/metrics-otel/compare/v0.5.0...HEAD
[v0.5.0]: https://github.com/wspulse/metrics-otel/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/wspulse/metrics-otel/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/wspulse/metrics-otel/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/wspulse/metrics-otel/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/wspulse/metrics-otel/releases/tag/v0.1.0
