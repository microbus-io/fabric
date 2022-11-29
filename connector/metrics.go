package connector

import (
	"sort"

	"github.com/microbus-io/fabric/errors"
	mtr "github.com/microbus-io/fabric/metric"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (c *Connector) newRegistry() error {
	c.registry = prometheus.NewRegistry()
	c.metricsHandler = promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})

	c.DefineHistogram(
		"fabric_handler_duration_seconds",
		"Handler processing duration, in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		[]string{"handler", "port", "method", "code", "error"},
	)
	c.DefineHistogram(
		"fabric_handler_size_bytes",
		"Handler response size, in bytes",
		[]float64{1024, 4 * 1024, 16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024, 4 * 1024 * 1024},
		[]string{"handler", "port", "method", "code", "error"},
	)
	c.DefineHistogram(
		"fabric_request_duration_seconds",
		"Request roundtrip duration, in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		[]string{"method", "host", "port", "path", "error"},
	)
	c.DefineHistogram(
		"fabric_ack_duration_seconds",
		"Ack roundtrip duration, in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		[]string{"method", "host", "port", "path"},
	)
	c.DefineCounter(
		"fabric_log_messages_total",
		"Number of log messages recorded",
		[]string{"message", "severity"},
	)
	c.DefineGauge(
		"fabric_uptime_duration_seconds_total",
		"Duration of time since connector was established, in seconds",
		[]string{},
	)
	c.DefineHistogram(
		"fabric_late_reply_duration_seconds",
		"Late reply roundtrip duration, in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		[]string{"op", "host"},
	)

	return nil
}

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

func (c *Connector) ObserveMetric(name string, val float64, labels ...string) error {
	if c.registry == nil {
		return nil
	}
	m, ok := c.metricDefs[name]
	if !ok {
		return errors.Newf("unknown metric '%s'", name)
	}
	err := m.Observe(val, append([]string{"service", "ver", "id"}, labels...)...)
	return errors.Trace(err, name)
}
