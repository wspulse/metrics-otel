//go:build integration

package otel_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	wspotel "github.com/wspulse/metrics-otel"
	wspulse "github.com/wspulse/server"
)

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	c, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	return c
}

func collect(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
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

// ── Integration tests ────────────────────────────────────────────────────────

func TestIntegration_ConnectionLifecycle(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	collector := wspotel.NewCollector(wspotel.WithMeterProvider(mp))

	srv := wspulse.NewServer(
		func(r *http.Request) (string, string, error) {
			return "test-room", "", nil
		},
		wspulse.WithMetrics(collector),
	)
	ts := httptest.NewServer(srv)
	defer func() {
		srv.Close()
		ts.Close()
	}()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Open 2 connections.
	c1 := dialWS(t, wsURL)
	c2 := dialWS(t, wsURL)
	time.Sleep(100 * time.Millisecond)

	rm := collect(t, reader)

	if got := findIntMetric(rm, "wspulse.connections.opened"); got != 2 {
		t.Errorf("connections opened: want 2, got %d", got)
	}
	if got := findIntMetric(rm, "wspulse.connections.active"); got != 2 {
		t.Errorf("connections active: want 2, got %d", got)
	}
	if got := findIntMetric(rm, "wspulse.rooms.active"); got != 1 {
		t.Errorf("rooms active: want 1, got %d", got)
	}

	// Close connections.
	_ = c1.Close()
	_ = c2.Close()
	time.Sleep(200 * time.Millisecond)

	rm = collect(t, reader)

	if got := findIntMetric(rm, "wspulse.connections.closed"); got != 2 {
		t.Errorf("connections closed: want 2, got %d", got)
	}
	if got := findIntMetric(rm, "wspulse.connections.active"); got != 0 {
		t.Errorf("connections active after close: want 0, got %d", got)
	}
	if got := findIntMetric(rm, "wspulse.rooms.active"); got != 0 {
		t.Errorf("rooms active after close: want 0, got %d", got)
	}
}

func TestIntegration_MessageMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	collector := wspotel.NewCollector(wspotel.WithMeterProvider(mp))

	var srv wspulse.Server
	srv = wspulse.NewServer(
		func(r *http.Request) (string, string, error) {
			return "test-room", "", nil
		},
		wspulse.WithMetrics(collector),
		wspulse.WithOnMessage(func(conn wspulse.Connection, f wspulse.Frame) {
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
	time.Sleep(100 * time.Millisecond)

	// Send a message — triggers MessageReceived + MessageBroadcast.
	err := c1.WriteMessage(websocket.TextMessage, []byte(`{"event":"ping"}`))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	rm := collect(t, reader)

	if got := findIntMetric(rm, "wspulse.messages.received"); got != 1 {
		t.Errorf("messages received: want 1, got %d", got)
	}
	if got := findIntMetric(rm, "wspulse.messages.broadcast"); got != 1 {
		t.Errorf("messages broadcast: want 1, got %d", got)
	}
	if got := findIntMetric(rm, "wspulse.messages.sent"); got != 2 {
		t.Errorf("messages sent: want 2, got %d", got)
	}
}
