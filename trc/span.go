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
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ = Span(SpanImpl{}) // Ensure interface

// Span represents an operation that is being traced.
type Span interface {
	// End completes the span.
	// Updates to the span are not allowed after this method has been called.
	End()
	// SetError sets the status of the span to error.
	SetError(err error)
	// SetOK sets the status of the span to OK, with the indicated response status code.
	SetOK(statusCode int)
	// Log records a log event on the span.
	Log(severity string, message string, args ...any)
	// SetString tags the span during its creation.
	SetString(k string, v string)
	// SetStrings tags the span during its creation.
	SetStrings(k string, v []string)
	// SetBool tags the span during its creation.
	SetBool(k string, v bool)
	// SetInt tags the span during its creation.
	SetInt(k string, v int)
	// SetFloat tags the span during its creation.
	SetFloat(k string, v float64)
	// SetRequest tags the span with the request data.
	// Warning: this has a large memory footprint.
	SetRequest(r *http.Request)
	// SetClientIP tags the span during its creation with the IP address and port number of the client.
	SetClientIP(ip string)
	// IsEmpty indicates if the span is not initialized.
	IsEmpty() bool
	// TraceID is an identifier that groups related spans together.
	TraceID() string
}

// SpanImpl implements the span interface.
type SpanImpl struct {
	trace.Span
}

// NewSpan creates a new span.
func NewSpan(ts trace.Span) Span {
	return SpanImpl{Span: ts}
}

// End completes the span.
// Updates to the span are not allowed after this method has been called.
func (s SpanImpl) End() {
	if s.Span == nil {
		return
	}
	s.Span.End()
}

// SetError sets the status of the span to error.
func (s SpanImpl) SetError(err error) {
	if s.Span == nil {
		return
	}
	v := fmt.Sprintf("%+v", err)
	s.Span.RecordError(err, trace.WithAttributes(
		attribute.String("exception.stacktrace", v),
	))
	s.Span.SetStatus(codes.Error, err.Error())
	sc := errors.StatusCode(err)
	s.Span.SetAttributes(attribute.Int("http.response.status_code", sc))
}

// SetOK sets the status of the span to OK, with the indicated response status code.
func (s SpanImpl) SetOK(statusCode int) {
	if s.Span == nil {
		return
	}
	s.Span.SetStatus(codes.Ok, "")
	if statusCode != http.StatusOK {
		s.Span.SetAttributes(attribute.Int("http.response.status_code", statusCode))
	}
}

// slogToTracingAttrs converts a slog attribute to an OpenTracing set of attribute
func slogToTracingAttrs(prefix string, f slog.Attr) []attribute.KeyValue {
	switch f.Value.Kind() {
	case slog.KindAny:
		return []attribute.KeyValue{
			attribute.String(prefix+f.Key, fmt.Sprintf("%+v", f.Value.Any())),
		}
	case slog.KindBool:
		return []attribute.KeyValue{
			attribute.Bool(prefix+f.Key, f.Value.Bool()),
		}
	case slog.KindDuration:
		return []attribute.KeyValue{
			attribute.String(prefix+f.Key, f.Value.Duration().String()),
		}
	case slog.KindFloat64:
		return []attribute.KeyValue{
			attribute.Float64(prefix+f.Key, f.Value.Float64()),
		}
	case slog.KindGroup:
		var group []attribute.KeyValue
		for _, a := range f.Value.Group() {
			group = append(group, slogToTracingAttrs(f.Key+".", a)...)
		}
		return group
	case slog.KindString:
		return []attribute.KeyValue{
			attribute.String(f.Key, f.Value.String()),
		}
	case slog.KindInt64:
		return []attribute.KeyValue{
			attribute.Int64(prefix+f.Key, f.Value.Int64()),
		}
	case slog.KindLogValuer:
		return slogToTracingAttrs(prefix, slog.Attr{
			Key:   f.Key,
			Value: f.Value.LogValuer().LogValue(),
		})
	case slog.KindTime:
		return []attribute.KeyValue{
			attribute.String(prefix+f.Key, f.Value.Time().Format(time.RFC3339Nano)),
		}
	case slog.KindUint64:
		return []attribute.KeyValue{
			attribute.Int64(prefix+f.Key, int64(f.Value.Uint64())),
		}
	}
	return nil
}

// Log records a log event on the span.
func (s SpanImpl) Log(severity string, msg string, args ...any) {
	if s.Span == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("severity", severity),
		attribute.String("message", msg),
	}
	slogRec := slog.NewRecord(time.Time{}, slog.LevelInfo, msg, 0)
	slogRec.Add(args...)
	slogRec.Attrs(func(f slog.Attr) bool {
		attrs = append(attrs, slogToTracingAttrs("", f)...)
		return true
	})
	s.Span.AddEvent("log", trace.WithAttributes(attrs...))
}

// SetString tags the span during its creation.
func (s SpanImpl) SetString(k string, v string) {
	if s.Span == nil {
		return
	}
	s.Span.SetAttributes(attribute.String(k, v))
}

// SetStrings tags the span during its creation.
func (s SpanImpl) SetStrings(k string, v []string) {
	if s.Span == nil {
		return
	}
	s.Span.SetAttributes(attribute.StringSlice(k, v))
}

// SetBool tags the span during its creation.
func (s SpanImpl) SetBool(k string, v bool) {
	if s.Span == nil {
		return
	}
	s.Span.SetAttributes(attribute.Bool(k, v))
}

// SetInt tags the span during its creation.
func (s SpanImpl) SetInt(k string, v int) {
	if s.Span == nil {
		return
	}
	s.Span.SetAttributes(attribute.Int(k, v))
}

// SetFloat tags the span during its creation.
func (s SpanImpl) SetFloat(k string, v float64) {
	if s.Span == nil {
		return
	}
	s.Span.SetAttributes(attribute.Float64(k, v))
}

// SetRequest tags the span with the request data.
// Warning: this has a large memory footprint.
func (s SpanImpl) SetRequest(r *http.Request) {
	if s.Span == nil {
		return
	}
	s.Span.SetAttributes(attributesOfRequest(r)...)
	s.SetClientIP(r.RemoteAddr)
}

// SetClientIP tags the span during its creation with the IP address and port number of the client.
func (s SpanImpl) SetClientIP(ip string) {
	p := strings.LastIndex(ip, ":")
	if p > 0 {
		portInt, _ := strconv.Atoi(ip[p+1:])
		s.Span.SetAttributes(
			attribute.String("client.address", ip[:p]),
			attribute.Int("client.port", portInt),
		)
	}
}

// IsEmpty indicates if the span is not initialized.
func (s SpanImpl) IsEmpty() bool {
	return s.Span == nil
}

// TraceID is an identifier that groups related spans together.
func (s SpanImpl) TraceID() string {
	if s.Span == nil {
		return ""
	}
	return s.Span.SpanContext().TraceID().String()
}
