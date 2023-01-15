package mtr

import (
	"github.com/microbus-io/fabric/errors"

	"github.com/prometheus/client_golang/prometheus"
)

// TODO: Prometheus collectors should not be exposed through wrappers

// Metric is an interface that defines operations for the metric collectors.
type Metric interface {
	Observe(val float64, labels ...string) error
	Add(val float64, labels ...string) error
}

// Histogram collects metrics in a histogram.
type Histogram struct {
	HistogramVec *prometheus.HistogramVec
}

// Gauge collects metrics in a gauge.
type Gauge struct {
	GaugeVec *prometheus.GaugeVec
}

// Counter collects metrics in a counter.
type Counter struct {
	CounterVec *prometheus.CounterVec
}

// Observe collects the value in the appropriate bucket of the histogram.
func (h *Histogram) Observe(val float64, labels ...string) error {
	histogram, err := h.HistogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	histogram.Observe(val)
	return nil
}

// Observe sets the current value of the gauge.
func (g *Gauge) Observe(val float64, labels ...string) error {
	gauge, err := g.GaugeVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	gauge.Set(val)
	return nil
}

// Observe observes the current value.
func (c *Counter) Observe(val float64, labels ...string) error {
	return errors.New("counter does not support 'Observe' operation")
}

// Add increments the current value.
func (h *Histogram) Add(val float64, labels ...string) error {
	return errors.New("histogram does not support 'Add' operation")
}

// Add increments the current value of the gauge.
func (g *Gauge) Add(val float64, labels ...string) error {
	gauge, err := g.GaugeVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	gauge.Add(val)
	return nil
}

// Add increments the value of the counter.
// Counters can only increase and the value cannot be negative.
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
