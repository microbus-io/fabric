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
