/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package mtr

import (
	"github.com/microbus-io/fabric/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// Gauge is a collector of a counter that can go up or down.
type Gauge struct {
	*prometheus.GaugeVec
}

// NewGauge creates a new gauge collector.
func NewGauge(name, help string, labels []string) Metric {
	gaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	return &Gauge{GaugeVec: gaugeVec}
}

// Observe sets the current value of the gauge.
func (g *Gauge) Observe(val float64, labels ...string) error {
	gauge, err := g.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	gauge.Set(val)
	return nil
}

// Add increases or decreases the current value of the gauge.
func (g *Gauge) Add(val float64, labels ...string) error {
	gauge, err := g.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	gauge.Add(val)
	return nil
}
