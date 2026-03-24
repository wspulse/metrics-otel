package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Option configures a Collector.
type Option func(*collectorConfig)

type collectorConfig struct {
	meterProvider metric.MeterProvider
	namespace     string
	roomAttribute bool
}

func defaultConfig() *collectorConfig {
	return &collectorConfig{
		meterProvider: otel.GetMeterProvider(),
		namespace:     "wspulse",
		roomAttribute: true,
	}
}

// WithMeterProvider sets the OTel MeterProvider used to create instruments.
// Defaults to otel.GetMeterProvider() (the global provider).
// Panics if mp is nil.
func WithMeterProvider(mp metric.MeterProvider) Option {
	if mp == nil {
		panic("wspulse/metrics-otel: WithMeterProvider: provider must not be nil")
	}
	return func(c *collectorConfig) { c.meterProvider = mp }
}

// WithNamespace sets the meter name used when creating the OTel Meter.
// Defaults to "wspulse". Instruments are named "{namespace}.connections.opened", etc.
func WithNamespace(ns string) Option {
	return func(c *collectorConfig) { c.namespace = ns }
}

// WithRoomAttribute controls whether room.id is included as an attribute on
// instruments. Defaults to true. Set to false in high-cardinality environments
// (e.g. one room per livestream) to avoid excessive attribute combinations.
func WithRoomAttribute(enabled bool) Option {
	return func(c *collectorConfig) { c.roomAttribute = enabled }
}
