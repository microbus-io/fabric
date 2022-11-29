package mtr

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

type Metric interface {
	Observe(val float64, labels ...string) error
	Add(val float64, labels ...string) error
}

type Histogram struct {
	HistogramVec *prometheus.HistogramVec
}

type Gauge struct {
	GaugeVec *prometheus.GaugeVec
}

type Counter struct {
	CounterVec *prometheus.CounterVec
}

func (h *Histogram) Observe(val float64, labels ...string) error {
	histogram, err := h.HistogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return err
	}
	histogram.Observe(val)
	return nil
}

func (g *Gauge) Observe(val float64, labels ...string) error {
	gauge, err := g.GaugeVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return err
	}
	gauge.Set(val)
	return nil
}

func (c *Counter) Observe(val float64, labels ...string) error {
	return errors.New("unsupported operation")
}

func (h *Histogram) Add(val float64, labels ...string) error {
	return errors.New("unsupported operation")
}

func (g *Gauge) Add(val float64, labels ...string) error {
	gauge, err := g.GaugeVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return err
	}
	gauge.Add(val)
	return nil
}

func (c *Counter) Add(val float64, labels ...string) error {
	counter, err := c.CounterVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return err
	}
	if val < 0 {
		return errors.New("value must not be negative")
	}
	counter.Add(val)
	return nil
}
