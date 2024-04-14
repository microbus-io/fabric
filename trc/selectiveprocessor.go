/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package trc

import (
	"context"
	"sync"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var _ = sdktrace.SpanProcessor(&SelectiveProcessor{}) // Ensure interface
const maxSpans = 65536
const maxSelected = 1024

// SelectiveProcessor processes spans of traces that have been explicitly selected
type SelectiveProcessor struct {
	downstreamProcessor sdktrace.SpanProcessor
	mux                 sync.Mutex
	spans               []sdktrace.ReadOnlySpan
	insertPoint         int
	selected            map[string]int
	stopped             bool
}

// NewProcessor creates a new selective processor
func NewProcessor(downstream sdktrace.SpanExporter) *SelectiveProcessor {
	return &SelectiveProcessor{
		downstreamProcessor: sdktrace.NewBatchSpanProcessor(downstream),
		spans:               make([]sdktrace.ReadOnlySpan, maxSpans),
		selected:            make(map[string]int, maxSelected),
	}
}

// OnStart is a no op
func (e *SelectiveProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
}

// OnEnd collects the span in a buffer for a period of time, to be able to process it if selected in the future
func (e *SelectiveProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	e.mux.Lock()
	if e.stopped {
		e.mux.Unlock()
		return
	}
	if e.selected[s.SpanContext().TraceID().String()] != 0 {
		// Span had been explicitly selected, so delegate downstream
		e.downstreamProcessor.OnEnd(s)
	} else {
		// Buffer for later
		if e.insertPoint >= len(e.spans) {
			e.insertPoint = 0
		}
		e.spans[e.insertPoint] = s
		e.insertPoint++
	}
	e.mux.Unlock()
}

// Select pushes all buffered matching the identified trace to the downstream processor as well as future spans
func (e *SelectiveProcessor) Select(traceID string) {
	e.mux.Lock()
	if e.stopped || e.selected[traceID] != 0 {
		e.mux.Unlock()
		return
	}
	// Push all buffered spans that match the trace ID
	for i := range e.spans {
		if e.spans[i] == nil {
			continue
		}
		if traceID == e.spans[i].SpanContext().TraceID().String() {
			e.downstreamProcessor.OnEnd(e.spans[i])
			e.spans[i] = nil
		}
	}
	// Mark future spans with same trace ID to be pushed immediately
	n := len(e.selected)
	e.selected[traceID] = n + 1
	// Compact the map if it grew too big
	if n > maxSelected {
		newSelected := make(map[string]int, maxSelected)
		for k, v := range e.selected {
			if v > n/2 {
				newSelected[k] = v - n/2
			}
		}
		e.selected = newSelected
	}
	e.mux.Unlock()
}

// Shutdown prevents further spans from being processed
func (e *SelectiveProcessor) Shutdown(ctx context.Context) error {
	e.mux.Lock()
	e.stopped = true
	e.spans = make([]sdktrace.ReadOnlySpan, maxSpans)
	e.selected = make(map[string]int)
	e.mux.Unlock()
	e.downstreamProcessor.Shutdown(ctx)
	return nil
}

// ForceFlush delegates to the downstream processor
func (e *SelectiveProcessor) ForceFlush(ctx context.Context) error {
	e.downstreamProcessor.ForceFlush(ctx)
	return nil
}
