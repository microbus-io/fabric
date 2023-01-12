package mtr

import (
	"github.com/microbus-io/fabric/errors"

	"github.com/prometheus/client_golang/prometheus"
)

// Metric is an interface that defines operations for the various Prometheus metric types.
type Metric interface {
	Observe(val float64, labels ...string) error
	Add(val float64, labels ...string) error
}

// Histogram wraps the Prometheus Histogram metric type.
type Histogram struct {
	HistogramVec *prometheus.HistogramVec
}

// Gauge wraps the Prometheus Gauge metric type.
type Gauge struct {
	GaugeVec *prometheus.GaugeVec
}

// Counter wraps the Prometheus Counter metric type.
type Counter struct {
	CounterVec *prometheus.CounterVec
}

// Observe observes the given value using a Histogram.
func (h *Histogram) Observe(val float64, labels ...string) error {
	histogram, err := h.HistogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	histogram.Observe(val)
	return nil
}

// Observe observes the given value by setting it as a Gauge's value.
func (g *Gauge) Observe(val float64, labels ...string) error {
	gauge, err := g.GaugeVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	gauge.Set(val)
	return nil
}

// Observe observes the given value using a Counter.
func (c *Counter) Observe(val float64, labels ...string) error {
	return errors.New("counter does not support 'Observe' operation")
}

// Add is not supported by the Histogram metric type.
func (h *Histogram) Add(val float64, labels ...string) error {
	return errors.New("histogram does not support 'Add' operation")
}

// Add adds the given value to the Gauge.
func (g *Gauge) Add(val float64, labels ...string) error {
	gauge, err := g.GaugeVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	gauge.Add(val)
	return nil
}

// Add adds the given value to the Counter.
func (c *Counter) Add(val float64, labels ...string) error {
	counter, err := c.CounterVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	if val < 0 {
		return errors.New("value must not be negative")
	}
	counter.Add(val)
	return nil
}
