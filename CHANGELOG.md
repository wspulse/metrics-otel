# Changelog

## [v0.3.0] - 2026-04-04

### Fixed

- Used polling for all async message metrics to prevent flaky reads
- Added timeout to bare channel receives in tests

### Changed

- Migrated integration tests to deterministic component tests — zero network I/O
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
