# Usage Guide

## Install

```bash
go get github.com/wspulse/metrics-otel
```

## Quick Start

```go
import (
    wspulse "github.com/wspulse/hub"
    wspotel "github.com/wspulse/metrics-otel"
)

// Uses the global MeterProvider (configured elsewhere in your app).
collector := wspotel.NewCollector()

hub := wspulse.NewHub(connect,
    wspulse.WithMetrics(collector),
)
```

With a custom MeterProvider:

```go
mp := sdkmetric.NewMeterProvider(
    sdkmetric.WithReader(exporter),
)
collector := wspotel.NewCollector(
    wspotel.WithMeterProvider(mp),
)
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithMeterProvider(mp)` | `otel.GetMeterProvider()` | Sets the OTel MeterProvider used to create instruments. Panics if nil. |
| `WithNamespace(ns)` | `"wspulse"` | Sets the meter name and instrument name prefix. Empty string is ignored. |
| `WithRoomAttribute(bool)` | `true` | Controls whether `room.id` is included as an attribute. Set to `false` in high-cardinality environments. |

### High-Cardinality Environments

If your application creates many unique rooms (e.g., one per livestream or per user session), the `room.id` attribute can produce excessive cardinality. Disable it:

```go
collector := wspotel.NewCollector(
    wspotel.WithMeterProvider(mp),
    wspotel.WithRoomAttribute(false),
)
```

## Instruments

### Connection Lifecycle

| Instrument | Type | Unit | Description |
|------------|------|------|-------------|
| `{ns}.connections.opened` | Counter (Int64) | | Total connections opened |
| `{ns}.connections.closed` | Counter (Int64) | | Total connections closed |
| `{ns}.connections.active` | UpDownCounter (Int64) | | Currently active connections |
| `{ns}.connection.duration` | Histogram (Float64) | `s` | Duration of connections in seconds |
| `{ns}.resume.attempts` | Counter (Int64) | | Total session resume attempts |

### Room Lifecycle

| Instrument | Type | Unit | Description |
|------------|------|------|-------------|
| `{ns}.rooms.active` | UpDownCounter (Int64) | | Currently active rooms |
| `{ns}.rooms.created` | Counter (Int64) | | Total rooms created |
| `{ns}.rooms.destroyed` | Counter (Int64) | | Total rooms destroyed |

### Throughput

| Instrument | Type | Unit | Description |
|------------|------|------|-------------|
| `{ns}.messages.received` | Counter (Int64) | | Total messages received |
| `{ns}.messages.received.bytes` | Counter (Int64) | `By` | Total bytes of messages received |
| `{ns}.messages.broadcast` | Counter (Int64) | | Total messages broadcast |
| `{ns}.broadcast.fanout` | Histogram (Int64) | | Recipients per broadcast |
| `{ns}.messages.sent` | Counter (Int64) | | Total messages sent to connections |
| `{ns}.frames.dropped` | Counter (Int64) | | Total frames dropped due to backpressure |
| `{ns}.send_buffer.utilization` | Histogram (Float64) | | Send buffer utilization ratio (0.0-1.0) |

### Heartbeat

| Instrument | Type | Unit | Description |
|------------|------|------|-------------|
| `{ns}.heartbeat.failures` | Counter (Int64) | | Total heartbeat failures |

`{ns}` defaults to `wspulse`. Change it with `WithNamespace`.

## Attributes

| Attribute | Type | Applied To | Description |
|-----------|------|------------|-------------|
| `room.id` | string | All instruments (when enabled) | Room identifier. Controlled by `WithRoomAttribute`. |
| `disconnect.reason` | string | `connections.closed`, `connection.duration` | Disconnect cause: `normal`, `kick`, `grace_expired`, `hub_close`, `duplicate` |

## Histogram Bucket Boundaries

Default buckets are tuned for WebSocket workloads. Override them with [OTel Views](https://opentelemetry.io/docs/languages/go/instrumentation/#views) on your MeterProvider.

| Histogram | Default Boundaries |
|-----------|--------------------|
| `connection.duration` | 1, 5, 15, 30, 60, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400 (seconds) |
| `broadcast.fanout` | 1, 2, 5, 10, 25, 50, 100, 250, 500, 1000 (recipients) |
| `send_buffer.utilization` | 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99, 1.0 (ratio) |

Example: override `connection.duration` buckets (the instrument name must match the configured namespace, default `wspulse`):

```go
view := sdkmetric.NewView(
    sdkmetric.Instrument{Name: "wspulse.connection.duration"},
    sdkmetric.Stream{
        Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
            Boundaries: []float64{1, 10, 60, 600, 3600},
        },
    },
)
mp := sdkmetric.NewMeterProvider(
    sdkmetric.WithReader(exporter),
    sdkmetric.WithView(view),
)
```
