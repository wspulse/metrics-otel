package otel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wspotel "github.com/wspulse/metrics-otel"
	wspulse "github.com/wspulse/server"
)

// TestConnectionLifecycle_FullCycle exercises the complete connection lifecycle
// by calling Collector methods directly: open two connections in the same room,
// verify active counts, then close both and verify final state including room
// teardown. This replaces TestIntegration_ConnectionLifecycle.
func TestConnectionLifecycle_FullCycle(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	// Room is created when the first connection joins.
	c.RoomCreated("test-room")
	c.ConnectionOpened("test-room", "conn1")
	c.ConnectionOpened("test-room", "conn2")

	rm := collectMetrics(t, reader)

	opened := findMetric(rm, "wspulse.connections.opened")
	require.NotNil(t, opened, "metric wspulse.connections.opened not found")
	assert.Equal(t, int64(2), sumInt64(opened), "connections opened")

	active := findMetric(rm, "wspulse.connections.active")
	require.NotNil(t, active, "metric wspulse.connections.active not found")
	assert.Equal(t, int64(2), sumInt64(active), "connections active")

	roomsActive := findMetric(rm, "wspulse.rooms.active")
	require.NotNil(t, roomsActive, "metric wspulse.rooms.active not found")
	assert.Equal(t, int64(1), sumInt64(roomsActive), "rooms active")

	// Close both connections. Room is destroyed when the last connection leaves.
	c.ConnectionClosed("test-room", "conn1", 10*time.Second, wspulse.DisconnectNormal)
	c.ConnectionClosed("test-room", "conn2", 15*time.Second, wspulse.DisconnectNormal)
	c.RoomDestroyed("test-room")

	rm = collectMetrics(t, reader)

	closed := findMetric(rm, "wspulse.connections.closed")
	require.NotNil(t, closed, "metric wspulse.connections.closed not found")
	assert.Equal(t, int64(2), sumInt64(closed), "connections closed")

	active = findMetric(rm, "wspulse.connections.active")
	require.NotNil(t, active, "metric wspulse.connections.active not found")
	assert.Equal(t, int64(0), sumInt64(active), "connections active after close")

	roomsActive = findMetric(rm, "wspulse.rooms.active")
	require.NotNil(t, roomsActive, "metric wspulse.rooms.active not found")
	assert.Equal(t, int64(0), sumInt64(roomsActive), "rooms active after close")

	// Verify connection duration was recorded for both.
	dur := findMetric(rm, "wspulse.connection.duration")
	require.NotNil(t, dur, "metric wspulse.connection.duration not found")
	assert.Equal(t, uint64(2), histogramCount(dur), "connection duration count")
	assert.Equal(t, 25.0, histogramSum(dur), "connection duration sum (10+15)")
}

// TestConnectionLifecycle_MultiRoom verifies metrics are correctly attributed
// when connections span multiple rooms.
func TestConnectionLifecycle_MultiRoom(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.RoomCreated("room-a")
	c.RoomCreated("room-b")
	c.ConnectionOpened("room-a", "conn1")
	c.ConnectionOpened("room-b", "conn2")

	rm := collectMetrics(t, reader)

	opened := findMetric(rm, "wspulse.connections.opened")
	require.NotNil(t, opened, "metric wspulse.connections.opened not found")
	assert.Equal(t, int64(2), sumInt64(opened), "connections opened across rooms")

	roomsActive := findMetric(rm, "wspulse.rooms.active")
	require.NotNil(t, roomsActive, "metric wspulse.rooms.active not found")
	assert.Equal(t, int64(2), sumInt64(roomsActive), "rooms active")

	assertAttributeValue(t, opened, "room.id", "room-a")
	assertAttributeValue(t, opened, "room.id", "room-b")

	// Close one room entirely.
	c.ConnectionClosed("room-a", "conn1", 5*time.Second, wspulse.DisconnectNormal)
	c.RoomDestroyed("room-a")

	rm = collectMetrics(t, reader)

	roomsActive = findMetric(rm, "wspulse.rooms.active")
	require.NotNil(t, roomsActive, "metric wspulse.rooms.active not found")
	assert.Equal(t, int64(1), sumInt64(roomsActive), "rooms active after one destroyed")
}

// TestConnectionLifecycle_DisconnectReasons verifies that the full lifecycle
// correctly records disconnect.reason attributes for different close causes.
func TestConnectionLifecycle_DisconnectReasons(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	c.RoomCreated("room1")
	c.ConnectionOpened("room1", "conn1")
	c.ConnectionOpened("room1", "conn2")
	c.ConnectionOpened("room1", "conn3")

	c.ConnectionClosed("room1", "conn1", 3*time.Second, wspulse.DisconnectNormal)
	c.ConnectionClosed("room1", "conn2", 1*time.Second, wspulse.DisconnectKick)
	c.ConnectionClosed("room1", "conn3", 2*time.Second, wspulse.DisconnectGraceExpired)
	c.RoomDestroyed("room1")

	rm := collectMetrics(t, reader)

	closed := findMetric(rm, "wspulse.connections.closed")
	require.NotNil(t, closed, "metric wspulse.connections.closed not found")
	assert.Equal(t, int64(3), sumInt64(closed), "connections closed")

	assertAttributeValue(t, closed, "disconnect.reason", "normal")
	assertAttributeValue(t, closed, "disconnect.reason", "kick")
	assertAttributeValue(t, closed, "disconnect.reason", "grace_expired")

	dur := findMetric(rm, "wspulse.connection.duration")
	require.NotNil(t, dur, "metric wspulse.connection.duration not found")
	assert.Equal(t, uint64(3), histogramCount(dur), "connection duration count")

	active := findMetric(rm, "wspulse.connections.active")
	require.NotNil(t, active, "metric wspulse.connections.active not found")
	assert.Equal(t, int64(0), sumInt64(active), "connections active after all closed")
}

// TestConnectionLifecycle_WithRoomAttributeDisabled verifies the full lifecycle
// works correctly when room.id attributes are disabled.
func TestConnectionLifecycle_WithRoomAttributeDisabled(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t, wspotel.WithRoomAttribute(false))

	c.RoomCreated("room1")
	c.ConnectionOpened("room1", "conn1")
	c.ConnectionOpened("room1", "conn2")
	c.ConnectionClosed("room1", "conn1", 5*time.Second, wspulse.DisconnectNormal)
	c.ConnectionClosed("room1", "conn2", 3*time.Second, wspulse.DisconnectKick)
	c.RoomDestroyed("room1")

	rm := collectMetrics(t, reader)

	opened := findMetric(rm, "wspulse.connections.opened")
	require.NotNil(t, opened, "metric wspulse.connections.opened not found")
	assert.Equal(t, int64(2), sumInt64(opened), "connections opened")
	assert.False(t, hasAttribute(opened, "room.id"), "room.id should be absent")

	closed := findMetric(rm, "wspulse.connections.closed")
	require.NotNil(t, closed, "metric wspulse.connections.closed not found")
	assert.True(t, hasAttribute(closed, "disconnect.reason"), "disconnect.reason should be present")
	assert.False(t, hasAttribute(closed, "room.id"), "room.id should be absent on closed")

	active := findMetric(rm, "wspulse.connections.active")
	require.NotNil(t, active, "metric wspulse.connections.active not found")
	assert.Equal(t, int64(0), sumInt64(active), "connections active after close")

	roomsActive := findMetric(rm, "wspulse.rooms.active")
	require.NotNil(t, roomsActive, "metric wspulse.rooms.active not found")
	assert.Equal(t, int64(0), sumInt64(roomsActive), "rooms active after destroy")
}
