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
