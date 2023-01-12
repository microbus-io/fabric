package connector

import (
	"sort"
	"strconv"

	"github.com/microbus-io/fabric/errors"
	mtr "github.com/microbus-io/fabric/mtr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// newRegistry creates a new Prometheus registry, or overwrites the current registry if it already exists.
// Standard metrics common across all microservices are also created and registered here.
func (c *Connector) newRegistry() error {
	c.registry = prometheus.NewRegistry()
	c.metricsHandler = promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})

	c.DefineHistogram(
		"microbus_response_duration_seconds",
		"Handler processing duration, in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		[]string{"handler", "port", "method", "code", "error"},
	)
	c.DefineHistogram(
		"microbus_response_size_bytes",
		"Handler response size, in bytes",
		[]float64{1024, 4 * 1024, 16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024, 4 * 1024 * 1024},
		[]string{"handler", "port", "method", "code", "error"},
	)
	c.DefineHistogram(
		"microbus_request_duration_seconds",
		"Request roundtrip duration, in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		[]string{"method", "host", "port", "path", "error"},
	)
	c.DefineHistogram(
		"microbus_ack_duration_seconds",
		"Ack roundtrip duration, in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		[]string{"method", "host", "port", "path"},
	)
	c.DefineCounter(
		"microbus_log_messages_total",
		"Number of log messages recorded",
		[]string{"message", "severity"},
	)
	c.DefineGauge(
		"microbus_uptime_duration_seconds_total",
		"Duration of time since connector was established, in seconds",
		[]string{},
	)
	c.DefineHistogram(
		"microbus_late_reply_duration_seconds",
		"Late reply roundtrip duration, in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		[]string{"op", "host"},
	)

	return nil
}

// DefineHistogram defines a new histogram metric in Prometheus and stores it in the connector's name -> metric mapping.
func (c *Connector) DefineHistogram(name, help string, buckets []float64, labels []string) error {

	if len(buckets) < 1 {
		return errors.New("empty buckets")
	}

	sort.Float64s(buckets)
	for i := 0; i < len(buckets)-1; i++ {
		if buckets[i+1] <= buckets[i] {
			return errors.New("buckets must be defined in ascending order")
		}
	}

	if c.registry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	if _, ok := c.metricDefs[name]; ok {
		return errors.New("metric already defined")
	}

	histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}, append([]string{"service", "ver", "id"}, labels...))

	m := &mtr.Histogram{
		HistogramVec: histogramVec,
	}

	c.metricDefs[name] = m
	err := c.registry.Register(histogramVec)
	return errors.Trace(err, name)
}

// DefineCounter defines a new counter metric in Prometheus and stores it in the connector's name -> metric mapping.
func (c *Connector) DefineCounter(name, help string, labels []string) error {
	if c.registry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	if _, ok := c.metricDefs[name]; ok {
		return errors.New("metric already defined")
	}

	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, append([]string{"service", "ver", "id"}, labels...))

	m := &mtr.Counter{
		CounterVec: counterVec,
	}

	c.metricDefs[name] = m
	err := c.registry.Register(counterVec)
	return errors.Trace(err, name)
}

// DefineGauge defines a new gauge metric in Prometheus and stores it in the connector's name -> metric mapping.
func (c *Connector) DefineGauge(name, help string, labels []string) error {
	if c.registry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	if _, ok := c.metricDefs[name]; ok {
		return errors.New("metric already defined")
	}

	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, append([]string{"service", "ver", "id"}, labels...))

	m := &mtr.Gauge{
		GaugeVec: gaugeVec,
	}

	c.metricDefs[name] = m
	err := c.registry.Register(gaugeVec)
	return errors.Trace(err, name)
}

// IncrementMetric adds the given value to a counter or gauge metric.
// The name and labels must match a previously defined metric.
// Gauge metrics support subtraction by use of a negative value.
// Counter metrics only allow addition and a negative value will result in an error.
func (c *Connector) IncrementMetric(name string, val float64, labels ...string) error {
	if c.registry == nil {
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
	if c.registry == nil {
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
