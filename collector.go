// Package otel provides an OpenTelemetry adapter for wspulse/hub's
// MetricsCollector interface. It translates server lifecycle events into
// OTel instruments (counters, up-down counters, histograms).
package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	wspulse "github.com/wspulse/hub"
)

// Collector implements wspulse.MetricsCollector using OpenTelemetry metrics.
// All methods are safe for concurrent use.
type Collector struct {
	cfg *collectorConfig

	// Connection lifecycle
	connectionsOpened  metric.Int64Counter
	connectionsClosed  metric.Int64Counter
	connectionsActive  metric.Int64UpDownCounter
	connectionDuration metric.Float64Histogram
	resumeAttempts     metric.Int64Counter

	// Room
	roomsActive    metric.Int64UpDownCounter
	roomsCreated   metric.Int64Counter
	roomsDestroyed metric.Int64Counter

	// Throughput
	messagesReceived      metric.Int64Counter
	messagesReceivedBytes metric.Int64Counter
	messagesBroadcast     metric.Int64Counter
	broadcastFanout       metric.Int64Histogram
	messagesSent          metric.Int64Counter
	framesDropped         metric.Int64Counter
	sendBufferUtilization metric.Float64Histogram

	// Heartbeat
	pongTimeouts metric.Int64Counter
}

// compile-time check: Collector must satisfy wspulse.MetricsCollector.
var _ wspulse.MetricsCollector = (*Collector)(nil)

// NewCollector creates a Collector and initializes all OTel instruments.
// Panics if instrument creation fails.
func NewCollector(opts ...Option) *Collector {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	meter := cfg.meterProvider.Meter(cfg.namespace)
	must := func(err error) {
		if err != nil {
			panic(fmt.Sprintf("wspulse: failed to create instrument: %v", err))
		}
	}

	c := &Collector{cfg: cfg}
	var err error

	// Connection lifecycle
	c.connectionsOpened, err = meter.Int64Counter(cfg.namespace+".connections.opened",
		metric.WithDescription("Total number of connections opened."))
	must(err)
	c.connectionsClosed, err = meter.Int64Counter(cfg.namespace+".connections.closed",
		metric.WithDescription("Total number of connections closed."))
	must(err)
	c.connectionsActive, err = meter.Int64UpDownCounter(cfg.namespace+".connections.active",
		metric.WithDescription("Number of currently active connections."))
	must(err)
	c.connectionDuration, err = meter.Float64Histogram(cfg.namespace+".connection.duration",
		metric.WithDescription("Duration of connections in seconds."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(1, 5, 15, 30, 60, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400))
	must(err)
	c.resumeAttempts, err = meter.Int64Counter(cfg.namespace+".resume.attempts",
		metric.WithDescription("Total number of session resume attempts."))
	must(err)

	// Room
	c.roomsActive, err = meter.Int64UpDownCounter(cfg.namespace+".rooms.active",
		metric.WithDescription("Number of currently active rooms."))
	must(err)
	c.roomsCreated, err = meter.Int64Counter(cfg.namespace+".rooms.created",
		metric.WithDescription("Total number of rooms created."))
	must(err)
	c.roomsDestroyed, err = meter.Int64Counter(cfg.namespace+".rooms.destroyed",
		metric.WithDescription("Total number of rooms destroyed."))
	must(err)

	// Throughput
	c.messagesReceived, err = meter.Int64Counter(cfg.namespace+".messages.received",
		metric.WithDescription("Total number of messages received."))
	must(err)
	c.messagesReceivedBytes, err = meter.Int64Counter(cfg.namespace+".messages.received.bytes",
		metric.WithDescription("Total bytes of messages received."),
		metric.WithUnit("By"))
	must(err)
	c.messagesBroadcast, err = meter.Int64Counter(cfg.namespace+".messages.broadcast",
		metric.WithDescription("Total number of messages broadcast."))
	must(err)
	c.broadcastFanout, err = meter.Int64Histogram(cfg.namespace+".broadcast.fanout",
		metric.WithDescription("Number of recipients per broadcast."),
		metric.WithExplicitBucketBoundaries(1, 2, 5, 10, 25, 50, 100, 250, 500, 1000))
	must(err)
	c.messagesSent, err = meter.Int64Counter(cfg.namespace+".messages.sent",
		metric.WithDescription("Total number of messages sent to connections."))
	must(err)
	c.framesDropped, err = meter.Int64Counter(cfg.namespace+".frames.dropped",
		metric.WithDescription("Total number of frames dropped due to backpressure."))
	must(err)
	c.sendBufferUtilization, err = meter.Float64Histogram(cfg.namespace+".send_buffer.utilization",
		metric.WithDescription("Send buffer utilization ratio (used/capacity)."),
		metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99, 1.0))
	must(err)

	// Heartbeat
	c.pongTimeouts, err = meter.Int64Counter(cfg.namespace+".pong.timeouts",
		metric.WithDescription("Total number of pong timeouts."))
	must(err)

	return c
}

// roomAttrs returns the OTel attributes for room-scoped instruments.
func (c *Collector) roomAttrs(roomID string) metric.MeasurementOption {
	if !c.cfg.roomAttribute {
		return metric.WithAttributes()
	}
	return metric.WithAttributes(attribute.String("room.id", roomID))
}

// roomReasonAttrs returns room attributes combined with an extra attribute
// (typically disconnect.reason). Used by ConnectionClosed.
func (c *Collector) roomReasonAttrs(roomID string, extra attribute.KeyValue) metric.MeasurementOption {
	if !c.cfg.roomAttribute {
		return metric.WithAttributes(extra)
	}
	return metric.WithAttributes(attribute.String("room.id", roomID), extra)
}

// ConnectionOpened increments the connections opened counter and active up-down counter.
func (c *Collector) ConnectionOpened(roomID, _ string) {
	attrs := c.roomAttrs(roomID)
	c.connectionsOpened.Add(context.Background(), 1, attrs)
	c.connectionsActive.Add(context.Background(), 1, attrs)
}

// ConnectionClosed increments the connections closed counter, decrements the
// active up-down counter, and records the connection duration. The disconnect.reason
// attribute distinguishes disconnect causes (normal, kick, grace_expired, etc.).
func (c *Collector) ConnectionClosed(roomID, _ string, duration time.Duration, reason wspulse.DisconnectReason) {
	reasonAttr := attribute.String("disconnect.reason", string(reason))
	activeAttrs := c.roomAttrs(roomID)
	closedAttrs := c.roomReasonAttrs(roomID, reasonAttr)
	c.connectionsClosed.Add(context.Background(), 1, closedAttrs)
	c.connectionsActive.Add(context.Background(), -1, activeAttrs)
	c.connectionDuration.Record(context.Background(), duration.Seconds(), closedAttrs)
}

// ResumeAttempt increments the resume attempts counter.
func (c *Collector) ResumeAttempt(roomID, _ string) {
	c.resumeAttempts.Add(context.Background(), 1, c.roomAttrs(roomID))
}

// RoomCreated increments the rooms created counter and active up-down counter.
func (c *Collector) RoomCreated(roomID string) {
	attrs := c.roomAttrs(roomID)
	c.roomsCreated.Add(context.Background(), 1, attrs)
	c.roomsActive.Add(context.Background(), 1, attrs)
}

// RoomDestroyed increments the rooms destroyed counter and decrements the active up-down counter.
func (c *Collector) RoomDestroyed(roomID string) {
	attrs := c.roomAttrs(roomID)
	c.roomsDestroyed.Add(context.Background(), 1, attrs)
	c.roomsActive.Add(context.Background(), -1, attrs)
}

// MessageReceived increments the messages received counter and bytes counter.
func (c *Collector) MessageReceived(roomID string, sizeBytes int) {
	attrs := c.roomAttrs(roomID)
	c.messagesReceived.Add(context.Background(), 1, attrs)
	c.messagesReceivedBytes.Add(context.Background(), int64(sizeBytes), attrs)
}

// MessageBroadcast increments the messages broadcast counter and records fanout.
func (c *Collector) MessageBroadcast(roomID string, _ int, fanOut int) {
	attrs := c.roomAttrs(roomID)
	c.messagesBroadcast.Add(context.Background(), 1, attrs)
	c.broadcastFanout.Record(context.Background(), int64(fanOut), attrs)
}

// MessageSent increments the messages sent counter.
func (c *Collector) MessageSent(roomID, _ string, _ int) {
	c.messagesSent.Add(context.Background(), 1, c.roomAttrs(roomID))
}

// FrameDropped increments the frames dropped counter.
func (c *Collector) FrameDropped(roomID, _ string) {
	c.framesDropped.Add(context.Background(), 1, c.roomAttrs(roomID))
}

// SendBufferUtilization records the send buffer utilization ratio.
func (c *Collector) SendBufferUtilization(roomID, _ string, used, capacity int) {
	ratio := 0.0
	if capacity > 0 {
		ratio = float64(used) / float64(capacity)
	}
	c.sendBufferUtilization.Record(context.Background(), ratio, c.roomAttrs(roomID))
}

// PongTimeout increments the pong timeouts counter.
func (c *Collector) PongTimeout(roomID, _ string) {
	c.pongTimeouts.Add(context.Background(), 1, c.roomAttrs(roomID))
}
