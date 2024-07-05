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

package connector

import (
	"strings"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Ensure interface
var _ = sdktrace.Sampler(&muffler{})

// muffler is an OpenTelemetry span sampler that excludes noisy spans.
// It filters out span created for metric collection and for forcing a trace.
type muffler struct {
}

// newMuffler returns a new muffler span sampler.
func newMuffler() *muffler {
	return &muffler{}
}

// ShouldSample returns a SamplingResult based on a decision made from the passed parameters.
func (smp *muffler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	sample := true
	if p.Name == ":888/trace" {
		// Do not sample the force trace broadcasts
		sample = false
	} else {
		// Do not sample Prometheus requests to the metrics core microservice
		// :8080/metrics.core/collect or :8080/metrics.core:443/collect
		if _, path, ok := strings.Cut(p.Name, "/"); ok {
			if path == "metrics.core/collect" || path == "metrics.core:443/collect" {
				sample = false
			}
		}
	}
	return sdktrace.SamplingResult{
		Decision: func() sdktrace.SamplingDecision {
			if sample {
				return sdktrace.RecordAndSample
			} else {
				return sdktrace.Drop
			}
		}(),
		Tracestate: trace.SpanContextFromContext(p.ParentContext).TraceState(),
	}
}

// Description returns information describing the Sampler.
func (smp *muffler) Description() string {
	return "Muffler"
}
