/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/trc"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type contextKeyType struct{}

var (
	contextKey      contextKeyType
	secureEndpoints = map[string]bool{}
	secureMux       sync.Mutex
)

// initTracer initializes an OpenTelemetry tracer
func (c *Connector) initTracer(ctx context.Context) (err error) {
	if c.traceProvider != nil {
		return nil
	}

	// Auto-detect if to use secure gRPC connection
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:4317"
	}
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}
	secureMux.Lock()
	_, ok := secureEndpoints[endpoint]
	if !ok {
		client := http.Client{
			Timeout: 2 * time.Second,
		}
		_, err = client.Get("https://" + endpoint)
		if err != nil && strings.Contains(err.Error(), "first record does not look like a TLS handshake") {
			secureEndpoints[endpoint] = false
		} else {
			secureEndpoints[endpoint] = true
		}
	}
	if !secureEndpoints[endpoint] {
		options = append(options, otlptracegrpc.WithInsecure())
	}
	secureMux.Unlock()

	var bsp sdktrace.SpanProcessor
	switch c.deployment {
	case LOCAL:
		exp, err := otlptracegrpc.New(ctx, options...)
		if err != nil {
			return errors.Trace(err)
		}
		bsp = sdktrace.NewBatchSpanProcessor(exp) // Trace all spans
	case TESTINGAPP:
		bsp = nil // Do not trace when running tests
	default: // PROD, LAB
		exp, err := otlptracegrpc.New(ctx, options...)
		if err != nil {
			return errors.Trace(err)
		}
		c.traceSelector = trc.NewProcessor(exp) // Trace only explicitly selected transactions
		bsp = c.traceSelector
	}
	if bsp != nil {
		c.traceProvider = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithSpanProcessor(bsp),
			sdktrace.WithResource(resource.NewSchemaless(
				attribute.String("service.namespace", c.plane),
				attribute.String("service.name", c.hostName),
				attribute.Int("service.version", c.version),
				attribute.String("service.instance.id", c.id),
			)),
		)
		c.tracer = c.traceProvider.Tracer("")
	}
	return nil
}

// termTracer shuts down the OpenTelemetry tracer
func (c *Connector) termTracer(ctx context.Context) (err error) {
	if c.traceProvider == nil {
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
		ctx, span := c.tracer.Start(ctx, spanName, opts...)
		ctx = context.WithValue(ctx, contextKey, span)
		return ctx, trc.NewSpan(span)
	} else {
		return ctx, trc.NewSpan(nil)
	}
}

// Span returns the tracing span stored in the context.
func (c *Connector) Span(ctx context.Context) trc.Span {
	span := ctx.Value(contextKey)
	if span != nil {
		return trc.NewSpan(span.(trace.Span))
	} else {
		return trc.NewSpan(nil)
	}
}

// ForceTrace forces the trace containing the span to be exported
func (c *Connector) ForceTrace(span trc.Span) {
	if c.traceSelector != nil {
		c.traceSelector.Select(span.TraceID())
		// Broadcast to all microservices to export all spans belonging to this trace
		c.Go(c.lifetimeCtx, func(ctx context.Context) error {
			for r := range c.Publish(c.lifetimeCtx, pub.GET("https://all:888/trace?id="+url.QueryEscape(span.TraceID()))) {
				_, _ = r.Get()
			}
			return nil
		})
	}
}
