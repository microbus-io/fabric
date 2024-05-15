/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var _ = sdktrace.SpanProcessor(&traceSelector{}) // Ensure interface

const maxBufferedSpans = 8192 // Each span can be about 1KB, so this is about 8GB per microservice
const maxSelected = 8192      // Trace IDs are small so can keep plenty
const maxTTLSeconds = 20

// traceSelector processes spans of traces that have been explicitly selected.
type traceSelector struct {
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

// newTraceSelector creates a new selective trace processor.
func newTraceSelector(downstream sdktrace.SpanExporter) *traceSelector {
	return &traceSelector{
		downstreamProcessor: sdktrace.NewBatchSpanProcessor(downstream),
		buffer:              make([]atomic.Pointer[sdktrace.ReadOnlySpan], maxBufferedSpans),
		selected1:           make(map[string]bool, maxSelected/2),
		selected2:           make(map[string]bool, maxSelected/2),
	}
}

// now returns the current time, offset by the clock offset.
func (e *traceSelector) now() int64 {
	return time.Now().Add(e.clockOffset).Unix()
}

// OnStart is a no op.
func (e *traceSelector) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
}

// OnEnd collects the span in a buffer for a period of time, to be able to process it if selected in the future.
func (e *traceSelector) OnEnd(s sdktrace.ReadOnlySpan) {
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
func (e *traceSelector) Select(traceID string) (ok bool) {
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
func (e *traceSelector) Shutdown(ctx context.Context) error {
	e.downstreamProcessor.Shutdown(ctx)
	return nil
}

// ForceFlush delegates to the downstream processor.
func (e *traceSelector) ForceFlush(ctx context.Context) error {
	e.downstreamProcessor.ForceFlush(ctx)
	return nil
}
