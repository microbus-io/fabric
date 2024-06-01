/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
		// :8080/metrics.sys/collect or :8080/metrics.sys:443/collect
		slash := strings.Index(p.Name, "/")
		if slash >= 0 {
			path := p.Name[slash+1:]
			if path == "metrics.sys/collect" || path == "metrics.sys:443/collect" {
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
