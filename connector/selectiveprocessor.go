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
	"context"
	"sync"
	"sync/atomic"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Ensure interface
var _ = sdktrace.SpanProcessor(&selectiveProcessor{})

const maxSelected = 8192 // Trace IDs are small so can keep plenty
const maxTTLSeconds = 20

// selectiveProcessor is a selective span processor, which is in essence a tail sampler.
// This processor buffers spans rather than export them immediately upon ending.
// When a trace ID is explicitly selected, all buffered spans and future spans matching the selected trace ID get exported.
// The buffer has a limited capacity and spans may therefore be dropped in high volume situations.
// https://opentelemetry.io/docs/concepts/sampling/
type selectiveProcessor struct {
	downstreamProcessor sdktrace.SpanProcessor
	buffer              []atomic.Pointer[sdktrace.ReadOnlySpan]
	insertionPoint      atomic.Int32

	mux          sync.Mutex
	selected1    map[string]bool
	selected2    map[string]bool
	lastSelected atomic.Int64

	// For testing
	clockOffset time.Duration
	lockCount   int
}

// newSelectiveProcessor creates a new selective span processor.
// The capacity determines the number of spans to buffer. Each span can be approx 1KB to buffer.
func newSelectiveProcessor(exporter sdktrace.SpanExporter, capacity int) *selectiveProcessor {
	return &selectiveProcessor{
		downstreamProcessor: sdktrace.NewBatchSpanProcessor(exporter),
		buffer:              make([]atomic.Pointer[sdktrace.ReadOnlySpan], capacity),
		selected1:           make(map[string]bool, maxSelected/2),
		selected2:           make(map[string]bool, maxSelected/2),
	}
}

// now returns the current time, offset by the clock offset.
func (e *selectiveProcessor) now() int64 {
	return time.Now().Add(e.clockOffset).Unix()
}

// OnStart is a no op.
func (e *selectiveProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
}

// OnEnd collects the span in a buffer for a period of time, to be able to process it if selected in the future.
func (e *selectiveProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	now := e.now()
	if e.lastSelected.Load() >= now-maxTTLSeconds {
		traceID := s.SpanContext().TraceID().String()
		e.mux.Lock()
		e.lockCount++
		selected := e.selected1[traceID] || e.selected2[traceID]
		e.mux.Unlock()
		if selected {
			// Span had been explicitly selected, so delegate downstream
			e.downstreamProcessor.OnEnd(s)
			return
		}
	}
	// Buffer for later
	ip := int(e.insertionPoint.Add(1)) - 1
	n := len(e.buffer)
	if ip >= n {
		ip -= n
		e.insertionPoint.Add(-int32(n))
	}
	e.buffer[ip].Store(&s)
}

// Select pushes all buffered matching the identified trace to the downstream processor as well as future spans.
// A false return value indicates that the trace had already been selected.
func (e *selectiveProcessor) Select(traceID string) (ok bool) {
	// Insert into the selected maps so future spans with this trace ID are delegated on receipt
	now := e.now()
	lastSelected := e.lastSelected.Swap(now)
	e.mux.Lock()
	e.lockCount++
	selectedAlready := e.selected1[traceID] || e.selected2[traceID]
	if !selectedAlready {
		if lastSelected < now-maxTTLSeconds {
			// Last selection was too long ago, so can clear both maps
			e.selected1 = make(map[string]bool, maxSelected/2)
			e.selected2 = make(map[string]bool, maxSelected/2)
		}
		if len(e.selected1) >= maxSelected/2 {
			// 1 is full, so shift it to 2 and clear it
			e.selected2 = e.selected1
			e.selected1 = make(map[string]bool, maxSelected/2)
		}
		e.selected1[traceID] = true
	}
	e.mux.Unlock()
	if selectedAlready {
		return false
	}

	// Push all buffered spans that match the trace ID
	for i := range e.buffer {
		s := e.buffer[i].Load()
		if s == nil {
			continue
		}
		if traceID == (*s).SpanContext().TraceID().String() {
			e.buffer[i].Store(nil) // Avoid pushing twice
			e.downstreamProcessor.OnEnd((*s))
		}
	}
	return true
}

// Shutdown prevents further spans from being processed.
func (e *selectiveProcessor) Shutdown(ctx context.Context) error {
	e.downstreamProcessor.Shutdown(ctx)
	return nil
}

// ForceFlush delegates to the downstream processor.
func (e *selectiveProcessor) ForceFlush(ctx context.Context) error {
	e.downstreamProcessor.ForceFlush(ctx)
	return nil
}
