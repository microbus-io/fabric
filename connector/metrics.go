/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"sort"
	"strconv"

	"github.com/microbus-io/fabric/errors"
	mtr "github.com/microbus-io/fabric/mtr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// newMetricsRegistry creates a new Prometheus registry, or overwrites the current registry if it already exists.
// Standard metrics common across all microservices are also created and registered here.
func (c *Connector) newMetricsRegistry() error {
	c.metricsRegistry = prometheus.NewRegistry()
	c.metricsHandler = promhttp.HandlerFor(c.metricsRegistry, promhttp.HandlerOpts{})
	c.DefineHistogram(
		"microbus_callback_duration_seconds",
		"Handler processing duration, in seconds",
		[]float64{.010, .050, .100, .500, 1, 5, 10, 60},
		[]string{"handler", "error"},
	)
	c.DefineHistogram(
		"microbus_response_duration_seconds",
		"Response processing duration, in seconds",
		[]float64{.010, .050, .100, .500, 1, 5, 10, 60},
		[]string{"handler", "port", "method", "code", "error"},
	)
	c.DefineHistogram(
		"microbus_response_size_bytes",
		"Response size, in bytes",
		[]float64{16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024, 4 * 1024 * 1024},
		[]string{"handler", "port", "method", "code", "error"},
	)
	c.DefineCounter(
		"microbus_request_count_total",
		"Number of outgoing requests",
		[]string{"method", "host", "port", "code", "error"},
	)
	c.DefineHistogram(
		"microbus_ack_duration_seconds",
		"Ack roundtrip duration, in seconds",
		[]float64{.010, .050, .100, .500, 1},
		[]string{"host"},
	)
	c.DefineCounter(
		"microbus_log_messages_total",
		"Number of log messages recorded",
		[]string{"message", "severity"},
	)
	c.DefineCounter(
		"microbus_uptime_duration_seconds_total",
		"Duration since connector was established, in seconds",
		[]string{},
	)
	c.DefineCounter(
		"microbus_cache_hits_total",
		"Number of distributed cache hits to load operations on the local shard",
		[]string{},
	)
	c.DefineCounter(
		"microbus_cache_misses_total",
		"Number of distributed cache misses to load operations on the local shard",
		[]string{},
	)
	c.DefineGauge(
		"microbus_cache_weight_total",
		"Total weight of elements in the local shard of the distributed cache",
		[]string{},
	)
	c.DefineGauge(
		"microbus_cache_len_total",
		"Total number of elements in the local shard of the distributed cache",
		[]string{},
	)
	return nil
}

// DefineHistogram defines a new histogram metric.
// The metric can later be observed.
func (c *Connector) DefineHistogram(name, help string, buckets []float64, labels []string) error {
	if len(buckets) < 1 {
		return c.captureInitErr(errors.New("empty buckets"))
	}
	sort.Float64s(buckets)
	for i := 0; i < len(buckets)-1; i++ {
		if buckets[i+1] <= buckets[i] {
			return c.captureInitErr(errors.New("buckets must be defined in ascending order"))
		}
	}

	if c.metricsRegistry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	if _, ok := c.metricDefs[name]; ok {
		return c.captureInitErr(errors.Newf("metric '%s' already defined", name))
	}
	m := mtr.NewHistogram(name, help, buckets, append([]string{"service", "ver", "id"}, labels...))
	c.metricDefs[name] = m
	err := c.metricsRegistry.Register(m.(prometheus.Collector))
	if err != nil {
		return c.captureInitErr(errors.Trace(err, name))
	}
	return nil
}

// DefineCounter defines a new counter metric.
// The metric can later be incremented.
func (c *Connector) DefineCounter(name, help string, labels []string) error {
	if c.metricsRegistry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	if _, ok := c.metricDefs[name]; ok {
		return c.captureInitErr(errors.Newf("metric '%s' already defined", name))
	}
	m := mtr.NewCounter(name, help, append([]string{"service", "ver", "id"}, labels...))
	c.metricDefs[name] = m
	err := c.metricsRegistry.Register(m.(prometheus.Collector))
	if err != nil {
		return c.captureInitErr(errors.Trace(err, name))
	}
	return nil
}

// DefineGauge defines a new gauge metric.
// The metric can later be observed or incremented.
func (c *Connector) DefineGauge(name, help string, labels []string) error {
	if c.metricsRegistry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	if _, ok := c.metricDefs[name]; ok {
		return c.captureInitErr(errors.Newf("metric '%s' already defined", name))
	}
	m := mtr.NewGauge(name, help, append([]string{"service", "ver", "id"}, labels...))
	c.metricDefs[name] = m
	err := c.metricsRegistry.Register(m.(prometheus.Collector))
	if err != nil {
		return c.captureInitErr(errors.Trace(err, name))
	}
	return nil
}

// IncrementMetric adds the given value to a counter or gauge metric.
// The name and labels must match a previously defined metric.
// Gauge metrics support subtraction by use of a negative value.
// Counter metrics only allow addition and a negative value will result in an error.
func (c *Connector) IncrementMetric(name string, val float64, labels ...string) error {
	if c.metricsRegistry == nil {
		return nil
	}
	if val == 0 {
		return nil
	}
	c.metricLock.RLock()
	defer c.metricLock.RUnlock()
	m, ok := c.metricDefs[name]
	if !ok {
		return errors.Newf("unknown metric '%s'", name)
	}
	err := m.Add(val, append([]string{c.HostName(), strconv.Itoa(c.Version()), c.ID()}, labels...)...)
	return errors.Trace(err, name)
}

// ObserveMetric observes the given value using a histogram or summary, or sets it as a gauge's value.
// The name and labels must match a previously defined metric.
func (c *Connector) ObserveMetric(name string, val float64, labels ...string) error {
	if c.metricsRegistry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	m, ok := c.metricDefs[name]
	if !ok {
		return errors.Newf("unknown metric '%s'", name)
	}
	err := m.Observe(val, append([]string{c.HostName(), strconv.Itoa(c.Version()), c.ID()}, labels...)...)
	return errors.Trace(err, name)
}
