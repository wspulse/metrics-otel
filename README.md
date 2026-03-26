# wspulse/metrics-otel

[![CI](https://github.com/wspulse/metrics-otel/actions/workflows/ci.yml/badge.svg)](https://github.com/wspulse/metrics-otel/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/wspulse/metrics-otel.svg)](https://pkg.go.dev/github.com/wspulse/metrics-otel)
[![Go](https://img.shields.io/badge/Go-1.26-blue.svg?logo=go)](https://go.dev)
[![OTel](https://img.shields.io/badge/OpenTelemetry-v1.42.0-blue.svg?logo=opentelemetry)](https://opentelemetry.io)
[![wspulse/server](https://img.shields.io/badge/wspulse%2Fserver-%3E%3D_v0.5.0-blue.svg)](https://github.com/wspulse/server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

OpenTelemetry adapter for [wspulse/server](https://github.com/wspulse/server)'s `MetricsCollector` interface.

---

## Install

```bash
go get github.com/wspulse/metrics-otel
```

---

## Quick Start

```go
import (
    "github.com/wspulse/server"
    wspotel "github.com/wspulse/metrics-otel"
)

// Uses the global MeterProvider (configured elsewhere in your app).
collector := wspotel.NewCollector()

srv := wspulse.NewServer(connect,
    wspulse.WithMetrics(collector),
)
```

Custom MeterProvider:

```go
mp := sdkmetric.NewMeterProvider(
    sdkmetric.WithReader(exporter),
)
collector := wspotel.NewCollector(
    wspotel.WithMeterProvider(mp),
    wspotel.WithRoomAttribute(false), // disable room.id attribute for high-cardinality environments
)
```

---

## Documentation

- [Usage Guide](doc/usage.md) — options, instruments, attributes, histogram buckets
- [Metrics Integration Guide](https://github.com/wspulse/docs/blob/main/guides/metrics.md)

## Related Modules

- [wspulse/server](https://github.com/wspulse/server) — WebSocket server library
- [wspulse/metrics-prometheus](https://github.com/wspulse/metrics-prometheus) — Prometheus adapter
- [wspulse/docs](https://github.com/wspulse/docs) — User-facing documentation
