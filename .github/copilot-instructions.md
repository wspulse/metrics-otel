# Copilot Instructions — wspulse/metrics-otel

## Project Overview

wspulse/metrics-otel is an **OpenTelemetry adapter** for wspulse/server's `MetricsCollector` interface. It translates server lifecycle events into OTel instruments (counters, up-down counters, histograms, gauges). Module path: `github.com/wspulse/metrics-otel`. Package name: `otel`.

## Architecture

- **`collector.go`** — `Collector` struct implementing `wspulse.MetricsCollector`. Creates all OTel instruments on construction. Each interface method records to the corresponding instrument.
- **`options.go`** — `Option` functional options: `WithMeterProvider`, `WithNamespace`, `WithRoomAttribute`.
- **`collector_test.go`** — Unit tests using `sdkmetric.NewManualReader()` to verify instrument recordings.

## Dependencies

- `github.com/wspulse/server` — source of `MetricsCollector` interface
- `go.opentelemetry.io/otel/metric` — OTel metrics API
- `go.opentelemetry.io/otel/sdk/metric` — OTel metrics SDK (test only)

## Development Workflow

```bash
make fmt        # format source files
make check      # fmt + lint + test (pre-commit gate)
make test       # unit tests with race detector
make test-cover # tests with coverage report
make bench      # benchmarks
make tidy       # go mod tidy
```

## Conventions

- **Go style**: same as wspulse/server — `gofmt`/`goimports`, GoDoc on all public symbols.
- **Naming**: interface names use full words. Package name is `otel`.
- **Instrument naming**: dot-separated (`wspulse.connections.opened`), follows [OTel Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/).
- **Attribute naming**: dot-separated keys (`room.id`, not `room_id`).
- **Error format**: `fmt.Errorf("wspulse: <context>: %w", err)`.
- **Markdown**: no emojis in documentation files.
- **Git**: commit messages follow [commit-message-instructions.md](instructions/commit-message-instructions.md). Branch strategy: `feat/`, `fix/`, `chore/`. Never push directly to `main`.
- **File encoding**: all files must be UTF-8 without BOM. Do not use any other encoding.

## Critical Rules

1. **Read before write** — read the target file before editing.
2. **STOP — test first, fix second** — when a bug is discovered or reported, do NOT touch production code until a failing test exists. Follow this exact sequence: (1) write a failing test, (2) confirm it fails, (3) fix the code, (4) confirm it passes, (5) run `make check`.
3. **`make check` gates every commit** — fmt + lint + test must pass.
4. **Minimal changes** — one concern per edit.
5. **No breaking changes without version bump** — exported symbols are a public contract.
6. **Thread safety** — all `Collector` methods are called concurrently from server goroutines. OTel instruments are safe for concurrent use, but verify any custom state is properly synchronized.
7. **Accuracy** — verify instrument names, types, and attribute sets against the plan in the workspace `doc/local/plan/metrics-otel.md`.
8. **Documentation sync** — when changing public API or options, update `docs/reference/` and `docs/guides/metrics.md` in the docs repo.
