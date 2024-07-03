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

package trc

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/frame"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Option is an alias for trace.SpanStartOption which are options used to create tracing spans.
type Option trace.SpanStartOption

// Server indicates that the span represents the operation of handling a request from a client.
func Server() Option {
	return trace.WithSpanKind(trace.SpanKindServer)
}

// Client indicates that the span represents the operation of client making a request to a server.
func Client() Option {
	return trace.WithSpanKind(trace.SpanKindClient)
}

// Producer represents the operation of a producer sending a message to a message broker.
func Producer() Option {
	return trace.WithSpanKind(trace.SpanKindProducer)
}

// Consumer represents the operation of a consumer receiving a message from a message broker.
func Consumer() Option {
	return trace.WithSpanKind(trace.SpanKindConsumer)
}

// Internal indicates that the span represents an internal operation within an application.
func Internal() Option {
	return trace.WithSpanKind(trace.SpanKindInternal)
}

// String tags the span during its creation.
func String(k string, v string) Option {
	return trace.WithAttributes(attribute.String(k, v))
}

// Strings tags the span during its creation.
func Strings(k string, v []string) Option {
	return trace.WithAttributes(attribute.StringSlice(k, v))
}

// Bool tags the span during its creation.
func Bool(k string, v bool) Option {
	return trace.WithAttributes(attribute.Bool(k, v))
}

// Int tags the span during its creation.
func Int(k string, v int) Option {
	return trace.WithAttributes(attribute.Int(k, v))
}

// Float tags the span during its creation.
func Float(k string, v float64) Option {
	return trace.WithAttributes(attribute.Float64(k, v))
}

// Request tags the span during its creation with the request data.
func Request(r *http.Request) Option {
	return trace.WithAttributes(attributesOfRequest(r)...)
}

// attributesOfRequest populates an attribute array from the HTTP request.
func attributesOfRequest(r *http.Request) []attribute.KeyValue {
	// https://opentelemetry.io/docs/specs/semconv/http/http-spans/#http-server
	portInt, _ := strconv.Atoi(r.URL.Port())
	attrs := []attribute.KeyValue{
		attribute.String("http.method", r.Method),
		attribute.String("url.scheme", r.URL.Scheme),
		attribute.String("server.address", r.URL.Hostname()),
		attribute.Int("server.port", portInt),
		attribute.String("url.path", r.URL.Path),
	}
	for k, v := range r.Header {
		if !strings.HasPrefix(k, frame.HeaderPrefix) && k != "Traceparent" && k != "Tracestate" {
			attrs = append(attrs, attribute.StringSlice("http.request.header."+k, v))
		}
	}
	encodedQuery := r.URL.Query().Encode()
	if encodedQuery != "" {
		attrs = append(attrs, attribute.String("url.query", encodedQuery))
	}
	if r.ContentLength > 0 {
		attrs = append(attrs, attribute.Int("http.request.body.size", int(r.ContentLength)))
	}
	return attrs
}

// ClientIP tags the span during its creation with the IP address and port number of the client.
func ClientIP(ip string) Option {
	p := strings.LastIndex(ip, ":")
	if p > 0 {
		portInt, _ := strconv.Atoi(ip[p+1:])
		return trace.WithAttributes(
			attribute.String("client.address", ip[:p]),
			attribute.Int("client.port", portInt),
		)
	}
	return trace.WithAttributes() // No op
}
