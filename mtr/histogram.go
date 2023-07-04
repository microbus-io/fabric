/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package mtr

import (
	"github.com/microbus-io/fabric/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// Histogram collects metrics in a histogram.
type Histogram struct {
	*prometheus.HistogramVec
}

// NewHistogram creates a new histogram collector.
func NewHistogram(name, help string, buckets []float64, labels []string) Metric {
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		},
		labels,
	)
	return &Histogram{HistogramVec: histogramVec}
}

// Observe collects the value in the appropriate bucket of the histogram.
func (h *Histogram) Observe(val float64, labels ...string) error {
	histogram, err := h.GetMetricWithLabelValues(labels...)
	if err != nil {
		return errors.Trace(err)
	}
	histogram.Observe(val)
	return nil
}

// Add increments the current value.
func (h *Histogram) Add(val float64, labels ...string) error {
	return errors.New("histogram does not support 'Add' operation")
}
