package otel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	wspotel "github.com/wspulse/metrics-otel"
	wspulse "github.com/wspulse/server"
)

func newTestCollector(t *testing.T, opts ...wspotel.Option) (*wspotel.Collector, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	allOpts := append([]wspotel.Option{
		wspotel.WithMeterProvider(mp),
	}, opts...)
	return wspotel.NewCollector(allOpts...), reader
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm), "collect metrics")
	return rm
}

// metricNames returns the names of all metrics in the given ResourceMetrics.
func metricNames(rm metricdata.ResourceMetrics) []string {
	var names []string
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			names = append(names, m.Name)
		}
	}
	return names
}

// findMetric searches for a metric by name across all scopes.
func findMetric(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}

// sumInt64 sums all data points for an Int64 counter or up-down counter.
func sumInt64(m *metricdata.Metrics) int64 {
	switch d := m.Data.(type) {
	case metricdata.Sum[int64]:
		var total int64
		for _, dp := range d.DataPoints {
			total += dp.Value
		}
		return total
	default:
		return 0
	}
}

// histogramSum returns the sum of all data points in a Float64 histogram.
func histogramSum(m *metricdata.Metrics) float64 {
	if d, ok := m.Data.(metricdata.Histogram[float64]); ok {
		var total float64
		for _, dp := range d.DataPoints {
			total += dp.Sum
		}
		return total
	}
	return 0
}

// histogramCount returns the total count across all data points in a Float64 histogram.
func histogramCount(m *metricdata.Metrics) uint64 {
	if d, ok := m.Data.(metricdata.Histogram[float64]); ok {
		var total uint64
		for _, dp := range d.DataPoints {
			total += dp.Count
		}
		return total
	}
	return 0
}

// hasAttribute checks if any data point has a specific attribute key.
func hasAttribute(m *metricdata.Metrics, key string) bool {
	switch d := m.Data.(type) {
	case metricdata.Sum[int64]:
		for _, dp := range d.DataPoints {
			for _, attr := range dp.Attributes.ToSlice() {
				if string(attr.Key) == key {
					return true
				}
			}
		}
	case metricdata.Sum[float64]:
		for _, dp := range d.DataPoints {
			for _, attr := range dp.Attributes.ToSlice() {
				if string(attr.Key) == key {
					return true
				}
			}
		}
	case metricdata.Gauge[float64]:
		for _, dp := range d.DataPoints {
			for _, attr := range dp.Attributes.ToSlice() {
				if string(attr.Key) == key {
					return true
				}
			}
		}
	case metricdata.Histogram[float64]:
		for _, dp := range d.DataPoints {
			for _, attr := range dp.Attributes.ToSlice() {
				if string(attr.Key) == key {
					return true
				}
			}
		}
	}
	return false
}

// assertAttributeValue checks that at least one data point in the metric has
// the given attribute key with the expected string value.
func assertAttributeValue(t *testing.T, m *metricdata.Metrics, key, wantValue string) {
	t.Helper()
	found := false
	check := func(attrs attribute.Set) {
		v, ok := attrs.Value(attribute.Key(key))
		if ok && v.Emit() == wantValue {
			found = true
		}
	}
	switch d := m.Data.(type) {
	case metricdata.Sum[int64]:
		for _, dp := range d.DataPoints {
			check(dp.Attributes)
		}
	case metricdata.Sum[float64]:
		for _, dp := range d.DataPoints {
			check(dp.Attributes)
		}
	case metricdata.Gauge[float64]:
		for _, dp := range d.DataPoints {
			check(dp.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, dp := range d.DataPoints {
			check(dp.Attributes)
		}
	case metricdata.Histogram[int64]:
		for _, dp := range d.DataPoints {
			check(dp.Attributes)
		}
	}
	assert.True(t, found, "expected attribute %s=%q, not found in metric %s", key, wantValue, m.Name)
}

// ── Interface compliance ─────────────────────────────────────────────────────

func TestCollector_ImplementsMetricsCollector(t *testing.T) {
	t.Parallel()
	var _ wspulse.MetricsCollector = (*wspotel.Collector)(nil)
}

// ── Option validation ────────────────────────────────────────────────────────

func TestWithMeterProvider_NilPanics(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() {
		_ = wspotel.WithMeterProvider(nil)
	}, "expected panic for nil MeterProvider")
}

// ── Connection lifecycle ─────────────────────────────────────────────────────

func TestConnectionOpened(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.ConnectionOpened("room1", "conn1")
	c.ConnectionOpened("room1", "conn2")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.connections.opened")
	require.NotNil(t, m, "metric wspulse.connections.opened not found")
	assert.Equal(t, int64(2), sumInt64(m), "connections opened")
	assertAttributeValue(t, m, "room.id", "room1")

	active := findMetric(rm, "wspulse.connections.active")
	require.NotNil(t, active, "metric wspulse.connections.active not found")
	assert.Equal(t, int64(2), sumInt64(active), "connections active")
}

func TestConnectionClosed(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.ConnectionOpened("room1", "conn1")
	c.ConnectionClosed("room1", "conn1", 5*time.Second, wspulse.DisconnectNormal)

	rm := collectMetrics(t, reader)

	active := findMetric(rm, "wspulse.connections.active")
	require.NotNil(t, active, "metric wspulse.connections.active not found")
	assert.Equal(t, int64(0), sumInt64(active), "connections active")

	closed := findMetric(rm, "wspulse.connections.closed")
	require.NotNil(t, closed, "metric wspulse.connections.closed not found")
	assert.Equal(t, int64(1), sumInt64(closed), "connections closed")

	dur := findMetric(rm, "wspulse.connection.duration")
	require.NotNil(t, dur, "metric wspulse.connection.duration not found")

	// Verify disconnect.reason attribute value on closed counter and duration.
	assertAttributeValue(t, closed, "disconnect.reason", "normal")
	assertAttributeValue(t, dur, "disconnect.reason", "normal")
}

func TestConnectionClosed_AllReasons(t *testing.T) {
	t.Parallel()

	reasons := []struct {
		reason wspulse.DisconnectReason
		want   string
	}{
		{wspulse.DisconnectNormal, "normal"},
		{wspulse.DisconnectKick, "kick"},
		{wspulse.DisconnectGraceExpired, "grace_expired"},
		{wspulse.DisconnectServerClose, "server_close"},
		{wspulse.DisconnectDuplicate, "duplicate"},
	}

	for _, tt := range reasons {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			c, reader := newTestCollector(t)

			c.ConnectionOpened("room1", "conn1")
			c.ConnectionClosed("room1", "conn1", 2*time.Second, tt.reason)

			rm := collectMetrics(t, reader)

			closed := findMetric(rm, "wspulse.connections.closed")
			require.NotNil(t, closed, "metric wspulse.connections.closed not found")
			assert.Equal(t, int64(1), sumInt64(closed), "connections closed")
			assertAttributeValue(t, closed, "disconnect.reason", tt.want)

			dur := findMetric(rm, "wspulse.connection.duration")
			require.NotNil(t, dur, "metric wspulse.connection.duration not found")
			assertAttributeValue(t, dur, "disconnect.reason", tt.want)
		})
	}
}

func TestResumeAttempt(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.ResumeAttempt("room1", "conn1")

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "wspulse.resume.attempts")
	require.NotNil(t, m, "metric wspulse.resume.attempts not found")
	assert.Equal(t, int64(1), sumInt64(m), "resume attempts")
	// room.id is included by default.
	assertAttributeValue(t, m, "room.id", "room1")
	// success attribute must not exist (regression prevention).
	assert.False(t, hasAttribute(m, "success"), "resume.attempts must not have a success attribute")
}

func TestResumeAttempt_MultipleAttempts(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.ResumeAttempt("room1", "conn1")
	c.ResumeAttempt("room1", "conn2")

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "wspulse.resume.attempts")
	require.NotNil(t, m, "metric wspulse.resume.attempts not found")
	assert.Equal(t, int64(2), sumInt64(m), "resume attempts")
	// room.id is included by default.
	assertAttributeValue(t, m, "room.id", "room1")
	// success attribute must not exist (regression prevention).
	assert.False(t, hasAttribute(m, "success"), "resume.attempts must not have a success attribute")
}

// ── Room lifecycle ───────────────────────────────────────────────────────────

func TestRoomCreatedDestroyed(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.RoomCreated("room1")
	c.RoomCreated("room2")
	c.RoomDestroyed("room1")

	rm := collectMetrics(t, reader)

	active := findMetric(rm, "wspulse.rooms.active")
	require.NotNil(t, active, "metric wspulse.rooms.active not found")
	assert.Equal(t, int64(1), sumInt64(active), "rooms active")

	created := findMetric(rm, "wspulse.rooms.created")
	require.NotNil(t, created, "metric wspulse.rooms.created not found")
	assert.Equal(t, int64(2), sumInt64(created), "rooms created")
}

// ── Throughput ───────────────────────────────────────────────────────────────

func TestMessageReceived(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.MessageReceived("room1", 100)
	c.MessageReceived("room1", 200)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.messages.received")
	require.NotNil(t, m, "metric wspulse.messages.received not found")
	assert.Equal(t, int64(2), sumInt64(m), "messages received")

	b := findMetric(rm, "wspulse.messages.received.bytes")
	require.NotNil(t, b, "metric wspulse.messages.received.bytes not found")
	assert.Equal(t, int64(300), sumInt64(b), "messages received bytes")
}

func TestMessageBroadcast(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.MessageBroadcast("room1", 50, 10)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.messages.broadcast")
	require.NotNil(t, m, "metric wspulse.messages.broadcast not found")
	assert.Equal(t, int64(1), sumInt64(m), "messages broadcast")

	f := findMetric(rm, "wspulse.broadcast.fanout")
	require.NotNil(t, f, "metric wspulse.broadcast.fanout not found")
}

func TestMessageSent(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.MessageSent("room1", "conn1", 100)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.messages.sent")
	require.NotNil(t, m, "metric wspulse.messages.sent not found")
	assert.Equal(t, int64(1), sumInt64(m), "messages sent")
}

func TestFrameDropped(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.FrameDropped("room1", "conn1")
	c.FrameDropped("room1", "conn1")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.frames.dropped")
	require.NotNil(t, m, "metric wspulse.frames.dropped not found")
	assert.Equal(t, int64(2), sumInt64(m), "frames dropped")
}

func TestSendBufferUtilization(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.SendBufferUtilization("room1", "conn1", 128, 256)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.send_buffer.utilization")
	require.NotNil(t, m, "metric wspulse.send_buffer.utilization not found")
	assert.Equal(t, uint64(1), histogramCount(m), "buffer utilization count")
	assert.Equal(t, 0.5, histogramSum(m), "buffer utilization sum")
}

func TestSendBufferUtilization_ZeroCapacity(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.SendBufferUtilization("room1", "conn1", 0, 0)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.send_buffer.utilization")
	require.NotNil(t, m, "metric wspulse.send_buffer.utilization not found")
	assert.Equal(t, uint64(1), histogramCount(m), "buffer utilization count")
	assert.Equal(t, 0.0, histogramSum(m), "buffer utilization sum")
}

// ── Heartbeat ────────────────────────────────────────────────────────────────

func TestPongTimeout(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.PongTimeout("room1", "conn1")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.pong.timeouts")
	require.NotNil(t, m, "metric wspulse.pong.timeouts not found")
	assert.Equal(t, int64(1), sumInt64(m), "pong timeouts")
}

func TestWithNamespace_Empty(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t, wspotel.WithNamespace(""))

	c.RoomCreated("room1")

	rm := collectMetrics(t, reader)

	// Empty namespace is a no-op — default "wspulse" is used.
	m := findMetric(rm, "wspulse.rooms.created")
	require.NotNilf(t, m, "expected wspulse.rooms.created (empty namespace ignored), collected: %v", metricNames(rm))
}

// ── WithRoomAttribute(false) ─────────────────────────────────────────────────

func TestWithRoomAttribute_False(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t, wspotel.WithRoomAttribute(false))

	c.ConnectionOpened("room1", "conn1")
	c.ConnectionClosed("room1", "conn1", 2*time.Second, wspulse.DisconnectKick)
	c.MessageReceived("room1", 100)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.connections.opened")
	require.NotNil(t, m, "metric wspulse.connections.opened not found")
	assert.Equal(t, int64(1), sumInt64(m), "connections opened")
	assert.False(t, hasAttribute(m, "room.id"), "room.id attribute should not exist when WithRoomAttribute(false)")

	// disconnect.reason must be present even without room attribute.
	closed := findMetric(rm, "wspulse.connections.closed")
	require.NotNil(t, closed, "metric wspulse.connections.closed not found")
	assert.True(t, hasAttribute(closed, "disconnect.reason"), "connections.closed missing disconnect.reason when WithRoomAttribute(false)")
	assert.False(t, hasAttribute(closed, "room.id"), "connections.closed should not have room.id when WithRoomAttribute(false)")
}

// ── WithNamespace ────────────────────────────────────────────────────────────

func TestWithNamespace(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t, wspotel.WithNamespace("myapp"))

	c.RoomCreated("room1")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "myapp.rooms.created")
	require.NotNilf(t, m, "expected myapp.rooms.created, collected: %v", metricNames(rm))
}

// ── Benchmarks ──────────────────────────────────────────────────────────────

func newBenchCollector(b *testing.B, opts ...wspotel.Option) *wspotel.Collector {
	b.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	allOpts := append([]wspotel.Option{
		wspotel.WithMeterProvider(mp),
	}, opts...)
	return wspotel.NewCollector(allOpts...)
}

func BenchmarkConnectionOpened(b *testing.B) {
	c := newBenchCollector(b)
	b.ResetTimer()
	for b.Loop() {
		c.ConnectionOpened("room1", "conn1")
	}
}

func BenchmarkMessageReceived(b *testing.B) {
	c := newBenchCollector(b)
	b.ResetTimer()
	for b.Loop() {
		c.MessageReceived("room1", 256)
	}
}

func BenchmarkConnectionOpened_NoRoomAttr(b *testing.B) {
	c := newBenchCollector(b, wspotel.WithRoomAttribute(false))
	b.ResetTimer()
	for b.Loop() {
		c.ConnectionOpened("room1", "conn1")
	}
}
