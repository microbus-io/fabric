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
	"net/url"

	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/trc"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// initTracer initializes an OpenTelemetry tracer
func (c *Connector) initTracer(ctx context.Context) (err error) {
	if c.traceProvider != nil {
		// Already initialized
		return nil
	}

	// Use the OTLP HTTP endpoint
	// https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/
	endpoint := env.Get("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if endpoint == "" {
		endpoint = env.Get("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	var sp sdktrace.SpanProcessor
	switch c.deployment {
	case LOCAL, TESTING:
		var exp *otlptrace.Exporter
		if endpoint == "" {
			// Use a nil client rather than return nil to allow for testing of span creation
			exp, err = otlptrace.New(ctx, &nilTraceClient{})
		} else {
			exp, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
		}
		if err != nil {
			return errors.Trace(err)
		}
		sp = sdktrace.NewBatchSpanProcessor(exp)
	case LAB:
		if endpoint == "" {
			return nil // Disables tracing without overhead
		}
		exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
		if err != nil {
			return errors.Trace(err)
		}
		sp = sdktrace.NewBatchSpanProcessor(exp)
	default: // PROD
		if endpoint == "" {
			return nil // Disables tracing without overhead
		}
		// Trace only explicitly selected transactions
		exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
		if err != nil {
			return errors.Trace(err)
		}
		c.traceProcessor = newSelectiveProcessor(exp, 8192) // Approx 10MB per microservice
		sp = c.traceProcessor
	}
	c.traceProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(newMuffler())),
		sdktrace.WithSpanProcessor(sp),
		sdktrace.WithResource(resource.NewSchemaless(
			attribute.String("service.namespace", c.plane),
			attribute.String("service.name", c.hostname),
			attribute.Int("service.version", c.version),
			attribute.String("service.instance.id", c.id),
		)),
	)
	c.tracer = c.traceProvider.Tracer("")
	return nil
}

// termTracer shuts down the OpenTelemetry tracer
func (c *Connector) termTracer(ctx context.Context) (err error) {
	if c.traceProvider == nil {
		// Not initialized
		return nil
	}
	err = c.traceProvider.Shutdown(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	c.traceProvider = nil
	c.tracer = nil
	return nil
}

// StartSpan creates a tracing span and a context containing the newly-created span.
// If the context provided already contains asSpan then the newly-created
// span will be a child of that span, otherwise it will be a root span.
//
// Any Span that is created must also be ended. This is the responsibility of the user.
// Implementations of this API may leak memory or other resources if spans are not ended.
func (c *Connector) StartSpan(ctx context.Context, spanName string, opts ...trc.Option) (context.Context, trc.Span) {
	if c.tracer != nil {
		options := make([]trace.SpanStartOption, len(opts))
		for i := 0; i < len(opts); i++ {
			options[i] = opts[i]
		}
		ctx, span := c.tracer.Start(ctx, spanName, options...)
		return ctx, trc.NewSpan(span)
	} else {
		return ctx, trc.NewSpan(nil)
	}
}

// Span returns the tracing span stored in the context.
func (c *Connector) Span(ctx context.Context) trc.Span {
	span := trace.SpanFromContext(ctx)
	return trc.NewSpan(span)
}

// ForceTrace forces the trace containing the span to be exported
func (c *Connector) ForceTrace(ctx context.Context) {
	if c.traceProcessor != nil {
		traceID := c.Span(ctx).TraceID()
		if traceID != "" {
			if c.traceProcessor.Select(traceID) {
				// Broadcast to all microservices to export all spans belonging to this trace
				c.Go(ctx, func(ctx context.Context) error {
					traceID := c.Span(ctx).TraceID()
					for range c.Publish(ctx, pub.GET("https://all:888/trace?id="+url.QueryEscape(traceID))) {
					}
					return nil
				})
			}
		}
	}
}

type nilTraceClient struct{}

func (nc *nilTraceClient) Start(ctx context.Context) error {
	return nil
}
func (nc *nilTraceClient) Stop(ctx context.Context) error {
	return nil
}
func (nc *nilTraceClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	return nil
}
