package mtr

import (
	"github.com/microbus-io/fabric/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// Counter is a collector of a counter that can only go up from zero.
type Counter struct {
	*prometheus.CounterVec
	value float64
}

// NewCounter creates a new counter collector.
func NewCounter(name, help string, labels []string) Metric {
	counterVec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	return &Counter{CounterVec: counterVec}
}

// Observe observes the current value.
// Counters can only increase in value.
func (c *Counter) Observe(val float64, labels ...string) error {
	return c.Add(val-c.value, labels...)
}

// Add increments the value of the counter.
// Counters can only increase in value.
func (c *Counter) Add(val float64, labels ...string) error {
	counter, err := c.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	if val < 0 {
		return errors.New("counter can only increase")
	}
	c.value += val
	counter.Add(val)
	return nil
}
