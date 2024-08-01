/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connector

import (
	"sort"
	"strconv"

	"github.com/microbus-io/fabric/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metric holds the Prometheus collector of the metric.
type metric struct {
	Counter   *prometheus.CounterVec
	Gauge     *prometheus.GaugeVec
	Histogram *prometheus.HistogramVec
}

// newMetricsRegistry creates a new Prometheus registry, or overwrites the current registry if it already exists.
// Standard metrics common across all microservices are also created and registered here.
func (c *Connector) newMetricsRegistry() (err error) {
	c.metricsRegistry = prometheus.NewRegistry()
	c.metricsHandler = promhttp.HandlerFor(c.metricsRegistry, promhttp.HandlerOpts{})
	c.DefineHistogram(
		"microbus_callback_duration_seconds",
		"Handler processing duration, in seconds",
		[]float64{.005, .010, .025, .050, .100, .250, .500, 1, 5, 15, 30, 60},
		[]string{"handler", "error"},
	)
	c.DefineHistogram(
		"microbus_response_duration_seconds",
		"Response processing duration, in seconds",
		[]float64{.005, .010, .025, .050, .100, .250, .500, 1, 5, 15, 30, 60},
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
		[]float64{.005, .010, .025, .050, .100, .250, .500, 1},
		[]string{"host"},
	)
	c.DefineCounter(
		"microbus_log_messages_total",
		"Number of log messages recorded",
		[]string{"message", "severity"},
	)
	c.DefineGauge(
		"microbus_uptime_duration_seconds_total",
		"Duration since connector was established, in seconds",
		[]string{},
	)
	c.DefineGauge(
		"microbus_cache_hits_total",
		"Number of distributed cache hits to load operations on the local shard",
		[]string{},
	)
	c.DefineGauge(
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
// Histograms can only be observed.
func (c *Connector) DefineHistogram(name, help string, buckets []float64, labels []string) (err error) {
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
	vec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		},
		append([]string{"service", "ver", "id"}, labels...),
	)
	c.metricDefs[name] = &metric{Histogram: vec}
	err = c.metricsRegistry.Register(vec)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	return nil
}

// DefineCounter defines a new counter metric.
// Counters can only be incremented.
func (c *Connector) DefineCounter(name, help string, labels []string) (err error) {
	if c.metricsRegistry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	if _, ok := c.metricDefs[name]; ok {
		return c.captureInitErr(errors.Newf("metric '%s' already defined", name))
	}
	vec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		append([]string{"service", "ver", "id"}, labels...),
	)
	c.metricDefs[name] = &metric{Counter: vec}
	err = c.metricsRegistry.Register(vec)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	return nil
}

// DefineGauge defines a new gauge metric.
// Gauges can be observed or incremented.
func (c *Connector) DefineGauge(name, help string, labels []string) (err error) {
	if c.metricsRegistry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	if _, ok := c.metricDefs[name]; ok {
		return c.captureInitErr(errors.Newf("metric '%s' already defined", name))
	}
	vec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		append([]string{"service", "ver", "id"}, labels...),
	)
	c.metricDefs[name] = &metric{Gauge: vec}
	err = c.metricsRegistry.Register(vec)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	return nil
}

// IncrementMetric adds the given value to a counter or gauge metric.
// The name and labels must match a previously defined metric.
// Gauge metrics support subtraction by use of a negative value.
// Counter metrics only allow addition and a negative value will result in an error.
func (c *Connector) IncrementMetric(name string, val float64, labels ...string) (err error) {
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
	if m.Counter != nil {
		if val < 0 {
			return errors.Newf("counter metric '%s' can only increase", name)
		}
		counter, err := m.Counter.GetMetricWithLabelValues(append([]string{c.Hostname(), strconv.Itoa(c.Version()), c.ID()}, labels...)...)
		if err != nil {
			return errors.Trace(err)
		}
		counter.Add(val)
	} else if m.Gauge != nil {
		gauge, err := m.Gauge.GetMetricWithLabelValues(append([]string{c.Hostname(), strconv.Itoa(c.Version()), c.ID()}, labels...)...)
		if err != nil {
			return errors.Trace(err)
		}
		gauge.Add(val)
	} else {
		return errors.Newf("metric '%s' cannot be incremented", name)
	}
	return nil
}

// ObserveMetric observes the given value using a histogram or summary, or sets it as a gauge's value.
// The name and labels must match a previously defined metric.
func (c *Connector) ObserveMetric(name string, val float64, labels ...string) (err error) {
	if c.metricsRegistry == nil {
		return nil
	}
	c.metricLock.RLock()
	defer c.metricLock.RUnlock()
	m, ok := c.metricDefs[name]
	if !ok {
		return errors.Newf("unknown metric '%s'", name)
	}
	if m.Gauge != nil {
		gauge, err := m.Gauge.GetMetricWithLabelValues(append([]string{c.Hostname(), strconv.Itoa(c.Version()), c.ID()}, labels...)...)
		if err != nil {
			return errors.Trace(err)
		}
		gauge.Set(val)
	} else if m.Histogram != nil {
		histogram, err := m.Histogram.GetMetricWithLabelValues(append([]string{c.Hostname(), strconv.Itoa(c.Version()), c.ID()}, labels...)...)
		if err != nil {
			return errors.Trace(err)
		}
		histogram.Observe(val)
	} else {
		return errors.Newf("metric '%s' cannot be observed", name)
	}
	return nil
}
