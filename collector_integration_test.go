//go:build integration

package otel_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	wspotel "github.com/wspulse/metrics-otel"
	wspulse "github.com/wspulse/server"
)

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	c, resp, err := dialer.Dial(url, nil)
	require.NoError(t, err, "Dial failed")
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	return c
}

func collect(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm), "collect metrics")
	return rm
}

func findIntMetric(rm metricdata.ResourceMetrics, name string) int64 {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				if d, ok := m.Data.(metricdata.Sum[int64]); ok {
					var total int64
					for _, dp := range d.DataPoints {
						total += dp.Value
					}
					return total
				}
			}
		}
	}
	return -1
}

// awaitCh waits for a signal on ch or fails the test after timeout.
func awaitCh(t *testing.T, ch <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		require.Fail(t, "timed out waiting for "+label)
	}
}

// waitForMetric polls the ManualReader until the named metric reaches the
// expected value or the timeout expires. This accounts for async metric
// recording in server goroutines (e.g. writePump fires MessageSent after
// the frame is written).
func waitForMetric(t *testing.T, reader *sdkmetric.ManualReader, name string, want int64, timeout time.Duration) int64 {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var got int64
	for time.Now().Before(deadline) {
		rm := collect(t, reader)
		got = findIntMetric(rm, name)
		if got >= want {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	return got
}

// ── Integration tests ────────────────────────────────────────────────────────

func TestIntegration_ConnectionLifecycle(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	collector := wspotel.NewCollector(wspotel.WithMeterProvider(mp))

	connected := make(chan struct{}, 4)
	disconnected := make(chan struct{}, 4)

	srv := wspulse.NewServer(
		func(r *http.Request) (string, string, error) {
			return "test-room", "", nil
		},
		wspulse.WithMetrics(collector),
		wspulse.WithOnConnect(func(_ wspulse.Connection) {
			connected <- struct{}{}
		}),
		wspulse.WithOnDisconnect(func(_ wspulse.Connection, _ error) {
			disconnected <- struct{}{}
		}),
	)
	ts := httptest.NewServer(srv)
	defer func() {
		srv.Close()
		ts.Close()
	}()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Open 2 connections and wait for server to register them.
	c1 := dialWS(t, wsURL)
	c2 := dialWS(t, wsURL)
	awaitCh(t, connected, "connect 1")
	awaitCh(t, connected, "connect 2")

	rm := collect(t, reader)

	assert.Equal(t, int64(2), findIntMetric(rm, "wspulse.connections.opened"), "connections opened")
	assert.Equal(t, int64(2), findIntMetric(rm, "wspulse.connections.active"), "connections active")
	assert.Equal(t, int64(1), findIntMetric(rm, "wspulse.rooms.active"), "rooms active")

	// Close connections and wait for server to process disconnects.
	_ = c1.Close()
	_ = c2.Close()
	awaitCh(t, disconnected, "disconnect 1")
	awaitCh(t, disconnected, "disconnect 2")

	rm = collect(t, reader)

	assert.Equal(t, int64(2), findIntMetric(rm, "wspulse.connections.closed"), "connections closed")
	assert.Equal(t, int64(0), findIntMetric(rm, "wspulse.connections.active"), "connections active after close")
	assert.Equal(t, int64(0), findIntMetric(rm, "wspulse.rooms.active"), "rooms active after close")
}

func TestIntegration_MessageMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	collector := wspotel.NewCollector(wspotel.WithMeterProvider(mp))

	connected := make(chan struct{}, 4)
	var broadcastDone sync.WaitGroup

	var srv wspulse.Server
	srv = wspulse.NewServer(
		func(r *http.Request) (string, string, error) {
			return "test-room", "", nil
		},
		wspulse.WithMetrics(collector),
		wspulse.WithOnConnect(func(_ wspulse.Connection) {
			connected <- struct{}{}
		}),
		wspulse.WithOnMessage(func(conn wspulse.Connection, f wspulse.Frame) {
			defer broadcastDone.Done()
			_ = srv.Broadcast(conn.RoomID(), f)
		}),
	)
	ts := httptest.NewServer(srv)
	defer func() {
		srv.Close()
		ts.Close()
	}()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	c1 := dialWS(t, wsURL)
	defer c1.Close()
	c2 := dialWS(t, wsURL)
	defer c2.Close()
	awaitCh(t, connected, "connect 1")
	awaitCh(t, connected, "connect 2")

	// Send a message — triggers MessageReceived + MessageBroadcast.
	broadcastDone.Add(1)
	err := c1.WriteMessage(websocket.TextMessage, []byte(`{"event":"ping"}`))
	require.NoError(t, err, "write message")

	// Wait for broadcast to complete.
	broadcastDone.Wait()

	// Poll until async metrics reach expected values. MessageSent fires
	// in the server's writePump; MessageReceived and MessageBroadcast fire
	// in the hub/readPump goroutines. All may lag behind broadcastDone.
	assert.Equal(t, int64(2), waitForMetric(t, reader, "wspulse.messages.sent", 2, 3*time.Second), "messages sent")
	assert.Equal(t, int64(1), waitForMetric(t, reader, "wspulse.messages.received", 1, 3*time.Second), "messages received")
	assert.Equal(t, int64(1), waitForMetric(t, reader, "wspulse.messages.broadcast", 1, 3*time.Second), "messages broadcast")
}
