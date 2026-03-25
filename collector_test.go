package otel_test

import (
	"context"
	"testing"
	"time"

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
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	return rm
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
	if !found {
		t.Errorf("expected attribute %s=%q, not found in metric %s", key, wantValue, m.Name)
	}
}

// ── Interface compliance ─────────────────────────────────────────────────────

func TestCollector_ImplementsMetricsCollector(t *testing.T) {
	t.Parallel()
	var _ wspulse.MetricsCollector = (*wspotel.Collector)(nil)
}

// ── Option validation ────────────────────────────────────────────────────────

func TestWithMeterProvider_NilPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil MeterProvider")
		}
	}()
	_ = wspotel.WithMeterProvider(nil)
}

// ── Connection lifecycle ─────────────────────────────────────────────────────

func TestConnectionOpened(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.ConnectionOpened("room1", "conn1")
	c.ConnectionOpened("room1", "conn2")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.connections.opened")
	if m == nil {
		t.Fatal("metric wspulse.connections.opened not found")
	}
	if got := sumInt64(m); got != 2 {
		t.Errorf("connections opened: want 2, got %d", got)
	}
	assertAttributeValue(t, m, "room.id", "room1")

	active := findMetric(rm, "wspulse.connections.active")
	if active == nil {
		t.Fatal("metric wspulse.connections.active not found")
	}
	if got := sumInt64(active); got != 2 {
		t.Errorf("connections active: want 2, got %d", got)
	}
}

func TestConnectionClosed(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.ConnectionOpened("room1", "conn1")
	c.ConnectionClosed("room1", "conn1", 5*time.Second, wspulse.DisconnectNormal)

	rm := collectMetrics(t, reader)

	active := findMetric(rm, "wspulse.connections.active")
	if active == nil {
		t.Fatal("metric wspulse.connections.active not found")
	}
	if got := sumInt64(active); got != 0 {
		t.Errorf("connections active: want 0, got %d", got)
	}

	closed := findMetric(rm, "wspulse.connections.closed")
	if closed == nil {
		t.Fatal("metric wspulse.connections.closed not found")
	}
	if got := sumInt64(closed); got != 1 {
		t.Errorf("connections closed: want 1, got %d", got)
	}

	dur := findMetric(rm, "wspulse.connection.duration")
	if dur == nil {
		t.Fatal("metric wspulse.connection.duration not found")
	}

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
			if closed == nil {
				t.Fatal("metric wspulse.connections.closed not found")
			}
			if got := sumInt64(closed); got != 1 {
				t.Errorf("connections closed: want 1, got %d", got)
			}
			assertAttributeValue(t, closed, "disconnect.reason", tt.want)

			dur := findMetric(rm, "wspulse.connection.duration")
			if dur == nil {
				t.Fatal("metric wspulse.connection.duration not found")
			}
			assertAttributeValue(t, dur, "disconnect.reason", tt.want)
		})
	}
}

func TestResumeAttempt(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.ResumeAttempt("room1", "conn1", true)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "wspulse.resume.attempts")
	if m == nil {
		t.Fatal("metric wspulse.resume.attempts not found")
	}
	if got := sumInt64(m); got != 1 {
		t.Errorf("resume attempts: want 1, got %d", got)
	}
}

func TestResumeAttempt_Failure(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.ResumeAttempt("room1", "conn1", false)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "wspulse.resume.attempts")
	if m == nil {
		t.Fatal("metric wspulse.resume.attempts not found")
	}
	if got := sumInt64(m); got != 1 {
		t.Errorf("resume attempts: want 1, got %d", got)
	}
	assertAttributeValue(t, m, "success", "false")
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
	if active == nil {
		t.Fatal("metric wspulse.rooms.active not found")
	}
	if got := sumInt64(active); got != 1 {
		t.Errorf("rooms active: want 1, got %d", got)
	}

	created := findMetric(rm, "wspulse.rooms.created")
	if created == nil {
		t.Fatal("metric wspulse.rooms.created not found")
	}
	if got := sumInt64(created); got != 2 {
		t.Errorf("rooms created: want 2, got %d", got)
	}
}

// ── Throughput ───────────────────────────────────────────────────────────────

func TestMessageReceived(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.MessageReceived("room1", 100)
	c.MessageReceived("room1", 200)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.messages.received")
	if m == nil {
		t.Fatal("metric wspulse.messages.received not found")
	}
	if got := sumInt64(m); got != 2 {
		t.Errorf("messages received: want 2, got %d", got)
	}

	b := findMetric(rm, "wspulse.messages.received.bytes")
	if b == nil {
		t.Fatal("metric wspulse.messages.received.bytes not found")
	}
	if got := sumInt64(b); got != 300 {
		t.Errorf("messages received bytes: want 300, got %d", got)
	}
}

func TestMessageBroadcast(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.MessageBroadcast("room1", 50, 10)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.messages.broadcast")
	if m == nil {
		t.Fatal("metric wspulse.messages.broadcast not found")
	}
	if got := sumInt64(m); got != 1 {
		t.Errorf("messages broadcast: want 1, got %d", got)
	}

	f := findMetric(rm, "wspulse.broadcast.fanout")
	if f == nil {
		t.Fatal("metric wspulse.broadcast.fanout not found")
	}
}

func TestMessageSent(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.MessageSent("room1", "conn1", 100)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.messages.sent")
	if m == nil {
		t.Fatal("metric wspulse.messages.sent not found")
	}
	if got := sumInt64(m); got != 1 {
		t.Errorf("messages sent: want 1, got %d", got)
	}
}

func TestFrameDropped(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.FrameDropped("room1", "conn1")
	c.FrameDropped("room1", "conn1")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.frames.dropped")
	if m == nil {
		t.Fatal("metric wspulse.frames.dropped not found")
	}
	if got := sumInt64(m); got != 2 {
		t.Errorf("frames dropped: want 2, got %d", got)
	}
}

func TestSendBufferUtilization(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.SendBufferUtilization("room1", "conn1", 128, 256)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.send_buffer.utilization")
	if m == nil {
		t.Fatal("metric wspulse.send_buffer.utilization not found")
	}
	if got := histogramCount(m); got != 1 {
		t.Errorf("buffer utilization count: want 1, got %d", got)
	}
	if got := histogramSum(m); got != 0.5 {
		t.Errorf("buffer utilization sum: want 0.5, got %v", got)
	}
}

func TestSendBufferUtilization_ZeroCapacity(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.SendBufferUtilization("room1", "conn1", 0, 0)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.send_buffer.utilization")
	if m == nil {
		t.Fatal("metric wspulse.send_buffer.utilization not found")
	}
	if got := histogramCount(m); got != 1 {
		t.Errorf("buffer utilization count: want 1, got %d", got)
	}
	if got := histogramSum(m); got != 0.0 {
		t.Errorf("buffer utilization sum: want 0.0, got %v", got)
	}
}

// ── Heartbeat ────────────────────────────────────────────────────────────────

func TestPongTimeout(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.PongTimeout("room1", "conn1")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "wspulse.pong.timeouts")
	if m == nil {
		t.Fatal("metric wspulse.pong.timeouts not found")
	}
	if got := sumInt64(m); got != 1 {
		t.Errorf("pong timeouts: want 1, got %d", got)
	}
}

func TestWithNamespace_Empty(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t, wspotel.WithNamespace(""))

	c.RoomCreated("room1")

	rm := collectMetrics(t, reader)

	// Empty namespace should fall back to "wspulse".
	m := findMetric(rm, "wspulse.rooms.created")
	if m == nil {
		var names []string
		for _, sm := range rm.ScopeMetrics {
			for _, metric := range sm.Metrics {
				names = append(names, metric.Name)
			}
		}
		t.Fatalf("expected wspulse.rooms.created (empty namespace fallback), got: %v", names)
	}
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
	if m == nil {
		t.Fatal("metric wspulse.connections.opened not found")
	}
	if got := sumInt64(m); got != 1 {
		t.Errorf("connections opened: want 1, got %d", got)
	}
	if hasAttribute(m, "room.id") {
		t.Error("room.id attribute should not exist when WithRoomAttribute(false)")
	}

	// disconnect.reason must be present even without room attribute.
	closed := findMetric(rm, "wspulse.connections.closed")
	if closed == nil {
		t.Fatal("metric wspulse.connections.closed not found")
	}
	if !hasAttribute(closed, "disconnect.reason") {
		t.Error("connections.closed missing disconnect.reason when WithRoomAttribute(false)")
	}
	if hasAttribute(closed, "room.id") {
		t.Error("connections.closed should not have room.id when WithRoomAttribute(false)")
	}
}

// ── WithNamespace ────────────────────────────────────────────────────────────

func TestWithNamespace(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t, wspotel.WithNamespace("myapp"))

	c.RoomCreated("room1")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "myapp.rooms.created")
	if m == nil {
		// List all metric names for debugging.
		var names []string
		for _, sm := range rm.ScopeMetrics {
			for _, metric := range sm.Metrics {
				names = append(names, metric.Name)
			}
		}
		t.Fatalf("expected myapp.rooms.created, got: %v", names)
	}
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
