package otel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wspotel "github.com/wspulse/metrics-otel"
)

// TestMessageFlow_BroadcastToMultipleRecipients exercises the full message flow:
// receive a message, broadcast it to multiple connections, and verify sent counts.
// This replaces TestIntegration_MessageMetrics.
func TestMessageFlow_BroadcastToMultipleRecipients(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	// Simulate: 1 message received from a client, broadcast to 2 connections.
	c.MessageReceived("test-room", 16)
	c.MessageBroadcast("test-room", 16, 2)
	c.MessageSent("test-room", "conn1", 16)
	c.MessageSent("test-room", "conn2", 16)

	rm := collectMetrics(t, reader)

	received := findMetric(rm, "wspulse.messages.received")
	require.NotNil(t, received, "metric wspulse.messages.received not found")
	assert.Equal(t, int64(1), sumInt64(received), "messages received")

	receivedBytes := findMetric(rm, "wspulse.messages.received.bytes")
	require.NotNil(t, receivedBytes, "metric wspulse.messages.received.bytes not found")
	assert.Equal(t, int64(16), sumInt64(receivedBytes), "messages received bytes")

	broadcast := findMetric(rm, "wspulse.messages.broadcast")
	require.NotNil(t, broadcast, "metric wspulse.messages.broadcast not found")
	assert.Equal(t, int64(1), sumInt64(broadcast), "messages broadcast")

	fanout := findMetric(rm, "wspulse.broadcast.fanout")
	require.NotNil(t, fanout, "metric wspulse.broadcast.fanout not found")
	assert.Equal(t, uint64(1), histogramCount(fanout), "broadcast fanout count")

	sent := findMetric(rm, "wspulse.messages.sent")
	require.NotNil(t, sent, "metric wspulse.messages.sent not found")
	assert.Equal(t, int64(2), sumInt64(sent), "messages sent")
}

// TestMessageFlow_MultipleBroadcasts verifies counters accumulate correctly
// across multiple broadcast cycles.
func TestMessageFlow_MultipleBroadcasts(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	// First message: broadcast to 3 connections.
	c.MessageReceived("room1", 100)
	c.MessageBroadcast("room1", 100, 3)
	c.MessageSent("room1", "conn1", 100)
	c.MessageSent("room1", "conn2", 100)
	c.MessageSent("room1", "conn3", 100)

	// Second message: broadcast to 2 connections.
	c.MessageReceived("room1", 200)
	c.MessageBroadcast("room1", 200, 2)
	c.MessageSent("room1", "conn1", 200)
	c.MessageSent("room1", "conn2", 200)

	rm := collectMetrics(t, reader)

	received := findMetric(rm, "wspulse.messages.received")
	require.NotNil(t, received, "metric wspulse.messages.received not found")
	assert.Equal(t, int64(2), sumInt64(received), "messages received")

	receivedBytes := findMetric(rm, "wspulse.messages.received.bytes")
	require.NotNil(t, receivedBytes, "metric wspulse.messages.received.bytes not found")
	assert.Equal(t, int64(300), sumInt64(receivedBytes), "messages received bytes")

	broadcast := findMetric(rm, "wspulse.messages.broadcast")
	require.NotNil(t, broadcast, "metric wspulse.messages.broadcast not found")
	assert.Equal(t, int64(2), sumInt64(broadcast), "messages broadcast")

	fanout := findMetric(rm, "wspulse.broadcast.fanout")
	require.NotNil(t, fanout, "metric wspulse.broadcast.fanout not found")
	assert.Equal(t, uint64(2), histogramCount(fanout), "broadcast fanout count")

	sent := findMetric(rm, "wspulse.messages.sent")
	require.NotNil(t, sent, "metric wspulse.messages.sent not found")
	assert.Equal(t, int64(5), sumInt64(sent), "messages sent (3+2)")
}

// TestMessageFlow_FrameDropAndBufferUtilization tests backpressure metrics
// during message flow.
func TestMessageFlow_FrameDropAndBufferUtilization(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t)

	// Simulate: message received, broadcast attempted, one connection drops frames.
	c.MessageReceived("room1", 50)
	c.MessageBroadcast("room1", 50, 3)
	c.MessageSent("room1", "conn1", 50)
	c.MessageSent("room1", "conn2", 50)
	c.FrameDropped("room1", "conn3")
	c.SendBufferUtilization("room1", "conn3", 256, 256)

	rm := collectMetrics(t, reader)

	sent := findMetric(rm, "wspulse.messages.sent")
	require.NotNil(t, sent, "metric wspulse.messages.sent not found")
	assert.Equal(t, int64(2), sumInt64(sent), "messages sent")

	dropped := findMetric(rm, "wspulse.frames.dropped")
	require.NotNil(t, dropped, "metric wspulse.frames.dropped not found")
	assert.Equal(t, int64(1), sumInt64(dropped), "frames dropped")

	utilization := findMetric(rm, "wspulse.send_buffer.utilization")
	require.NotNil(t, utilization, "metric wspulse.send_buffer.utilization not found")
	assert.Equal(t, uint64(1), histogramCount(utilization), "buffer utilization count")
	assert.Equal(t, 1.0, histogramSum(utilization), "buffer utilization sum (full)")
}

// TestMessageFlow_WithRoomAttributeDisabled verifies message metrics work
// correctly when room.id is disabled.
func TestMessageFlow_WithRoomAttributeDisabled(t *testing.T) {
	t.Parallel()
	c, reader := newTestCollector(t, wspotel.WithRoomAttribute(false))

	c.MessageReceived("room1", 100)
	c.MessageBroadcast("room1", 100, 2)
	c.MessageSent("room1", "conn1", 100)
	c.MessageSent("room1", "conn2", 100)

	rm := collectMetrics(t, reader)

	received := findMetric(rm, "wspulse.messages.received")
	require.NotNil(t, received, "metric wspulse.messages.received not found")
	assert.Equal(t, int64(1), sumInt64(received), "messages received")
	assert.False(t, hasAttribute(received, "room.id"), "room.id should be absent")

	sent := findMetric(rm, "wspulse.messages.sent")
	require.NotNil(t, sent, "metric wspulse.messages.sent not found")
	assert.Equal(t, int64(2), sumInt64(sent), "messages sent")
	assert.False(t, hasAttribute(sent, "room.id"), "room.id should be absent on sent")
}
