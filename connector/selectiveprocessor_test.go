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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type exporter struct {
	Callback func(ctx context.Context, spans []sdktrace.ReadOnlySpan) error
}

func (e *exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if e.Callback != nil {
		return e.Callback(ctx, spans)
	}
	return nil
}

func (e *exporter) Shutdown(ctx context.Context) error {
	return nil
}

func TestConnector_TracingExport(t *testing.T) {
	ctx := context.Background()

	countExported := 0
	exportedSpans := map[string]bool{}
	ts := newSelectiveProcessor(&exporter{
		Callback: func(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
			countExported += len(spans)
			for _, span := range spans {
				exportedSpans[span.SpanContext().SpanID().String()] = true
			}
			return nil
		},
	}, 16)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(ts),
	)
	tracer := traceProvider.Tracer("")

	// Nothing traced yet
	assert.Equal(t, 0, int(ts.insertionPoint.Load()))
	assert.Equal(t, 0, countExported)
	assert.Equal(t, 0, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 0, ts.lockCount)

	_, span := tracer.Start(ctx, "1")
	span.End()
	_, span = tracer.Start(ctx, "2")
	span.End()

	subCtx, span := tracer.Start(ctx, "3")
	_, subSpan1 := tracer.Start(subCtx, "3.1")
	assert.Equal(t, span.SpanContext().TraceID(), subSpan1.SpanContext().TraceID())
	subSpan1.End()

	// The spans should be buffered but not yet exported
	assert.Equal(t, 3, int(ts.insertionPoint.Load()))
	assert.Equal(t, 0, countExported)
	assert.Equal(t, 0, ts.lockCount)

	// Select the parent span's trace ID for exporting
	ts.Select(span.SpanContext().TraceID().String())
	ts.ForceFlush(ctx) // Flush the queue
	assert.Equal(t, 1, len(ts.selected1)+len(ts.selected2))

	// The closed subspan should have gotten immediately exported
	assert.Equal(t, 3, int(ts.insertionPoint.Load()))
	assert.Equal(t, 1, countExported)
	assert.True(t, exportedSpans[subSpan1.SpanContext().SpanID().String()])
	assert.Equal(t, 1, ts.lockCount)

	// Add a second subspan
	_, subSpan2 := tracer.Start(subCtx, "3.2")
	assert.Equal(t, span.SpanContext().TraceID(), subSpan2.SpanContext().TraceID())
	subSpan2.End()
	ts.ForceFlush(ctx) // Flush the queue

	// The new subspan should have gotten immediately exported and not buffered
	assert.Equal(t, 3, int(ts.insertionPoint.Load()))
	assert.Equal(t, 2, countExported)
	assert.True(t, exportedSpans[subSpan2.SpanContext().SpanID().String()])
	assert.Equal(t, 2, ts.lockCount)

	span.End()
	ts.ForceFlush(ctx) // Flush the queue

	// The parent span should have gotten immediately exported and not buffered
	assert.Equal(t, 3, int(ts.insertionPoint.Load()))
	assert.Equal(t, 3, countExported)
	assert.True(t, exportedSpans[span.SpanContext().SpanID().String()])
	assert.Equal(t, 3, ts.lockCount)

	// Select the same trace ID a second time
	ts.Select(span.SpanContext().TraceID().String())
	ts.ForceFlush(ctx) // Flush the queue
	assert.Equal(t, 1, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 3, int(ts.insertionPoint.Load()))
	assert.Equal(t, 3, countExported)
	assert.Equal(t, 4, ts.lockCount)
}

func TestConnector_TracingTTLClearMaps(t *testing.T) {
	ts := newSelectiveProcessor(&exporter{}, 16)

	ts.Select("1")
	assert.Equal(t, 1, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 1, ts.lockCount)
	ts.Select("2")
	assert.Equal(t, 2, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 2, ts.lockCount)
	ts.Select("2")
	assert.Equal(t, 2, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 3, ts.lockCount)
	ts.Select("3")
	assert.Equal(t, 3, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 4, ts.lockCount)

	// Selection maps should be cleared after TTL
	ts.clockOffset += time.Second * (maxTTLSeconds + 1)
	ts.Select("4")
	assert.Equal(t, 1, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 5, ts.lockCount)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(ts),
	)
	tracer := traceProvider.Tracer("")

	ctx := context.Background()
	_, span := tracer.Start(ctx, "1")
	span.End()

}

func TestConnector_TracingTTLNoLock(t *testing.T) {
	ctx := context.Background()

	ts := newSelectiveProcessor(&exporter{}, 16)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(ts),
	)
	tracer := traceProvider.Tracer("")

	// First span should not lock because the selector maps are empty
	_, span := tracer.Start(ctx, "A")
	span.End()

	assert.Equal(t, 0, ts.lockCount)

	// Add a random selection
	ts.Select("123")
	assert.Equal(t, 1, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 1, ts.lockCount)

	// Span should lock because there's a valid selector
	_, span = tracer.Start(ctx, "B")
	span.End()
	assert.Equal(t, 2, ts.lockCount)

	_, span = tracer.Start(ctx, "C")
	span.End()
	assert.Equal(t, 3, ts.lockCount)

	// After TTL passed, the selectors should be ignored so there should be no lock
	ts.clockOffset += time.Second * (maxTTLSeconds + 1)
	_, span = tracer.Start(ctx, "D")
	span.End()
	assert.Equal(t, 3, ts.lockCount)
}

func TestConnector_TracingSelectorCapacityRollover(t *testing.T) {
	ts := newSelectiveProcessor(&exporter{}, 16)

	for i := 0; i < maxSelected/2; i++ {
		ts.Select(strconv.Itoa(i))
	}
	assert.Len(t, ts.selected1, maxSelected/2)

	ts.Select(strconv.Itoa(maxSelected / 2))
	assert.Len(t, ts.selected1, 1)
	assert.Len(t, ts.selected2, maxSelected/2)

	for i := 1; i < maxSelected/2; i++ {
		ts.Select(strconv.Itoa(maxSelected/2 + i))
	}
	assert.Len(t, ts.selected1, maxSelected/2)
	assert.Len(t, ts.selected2, maxSelected/2)

	ts.Select(strconv.Itoa(maxSelected))
	assert.Len(t, ts.selected1, 1)
	assert.Len(t, ts.selected2, maxSelected/2)
}

func TestConnector_TracingBufferCapacityRollover(t *testing.T) {
	ctx := context.Background()

	n := 16
	ts := newSelectiveProcessor(&exporter{}, n)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(ts),
	)
	tracer := traceProvider.Tracer("")

	// Fill in the buffer
	assert.Zero(t, ts.insertionPoint.Load())
	for i := 0; i < n; i++ {
		assert.Equal(t, int32(i), ts.insertionPoint.Load())
		assert.Nil(t, ts.buffer[i].Load())
		_, span := tracer.Start(ctx, "A")
		span.End()
		assert.NotNil(t, ts.buffer[i].Load())
		assert.Equal(t, int32(i+1), ts.insertionPoint.Load())
	}

	// Second pass should overwrite in the buffer
	for i := 0; i < n; i++ {
		if i > 0 {
			assert.Equal(t, int32(i), ts.insertionPoint.Load())
		}
		before := ts.buffer[i].Load()
		assert.NotNil(t, before)
		_, span := tracer.Start(ctx, "A")
		span.End()
		after := ts.buffer[i].Load()
		assert.NotNil(t, after)
		assert.NotEqual(t, before, after)
		assert.Equal(t, int32(i+1), ts.insertionPoint.Load())
	}
}

func BenchmarkConnector_TracingOnEnd(b *testing.B) {
	ctx := context.Background()

	ts := newSelectiveProcessor(&exporter{}, 8192)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(ts),
	)
	tracer := traceProvider.Tracer("")

	arr := make([]trace.Span, b.N)
	for i := 0; i < b.N; i++ {
		_, arr[i] = tracer.Start(ctx, "A")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arr[i].End()
	}
	// N=5078120
	// 301.4 ns/op
	// 400 B/op
	// 2 allocs/op
}

func TestConnector_DuplicateSelect(t *testing.T) {
	ctx := context.Background()

	countExported := 0
	exportedSpans := map[string]bool{}
	ts := newSelectiveProcessor(&exporter{
		Callback: func(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
			countExported += len(spans)
			for _, span := range spans {
				exportedSpans[span.SpanContext().SpanID().String()] = true
			}
			return nil
		},
	}, 16)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(ts),
	)
	tracer := traceProvider.Tracer("")

	// Nothing traced yet
	assert.Equal(t, 0, int(ts.insertionPoint.Load()))
	assert.Equal(t, 0, countExported)
	assert.Equal(t, 0, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 0, ts.lockCount)

	_, span := tracer.Start(ctx, "1")
	span.End()

	ok := ts.Select(span.SpanContext().TraceID().String())
	assert.True(t, ok)
	assert.Equal(t, 1, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 1, ts.lockCount)
	ts.ForceFlush(ctx) // Flush the queue
	assert.Equal(t, 1, countExported)

	ok = ts.Select(span.SpanContext().TraceID().String())
	assert.False(t, ok)
	assert.Equal(t, 1, len(ts.selected1)+len(ts.selected2))
	assert.Equal(t, 2, ts.lockCount)
	ts.ForceFlush(ctx) // Flush the queue
	assert.Equal(t, 1, countExported)
}
