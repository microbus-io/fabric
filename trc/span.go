/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Span implements the span interface.
type Span struct {
	internal trace.Span
}

// NewSpan creates a new span.
func NewSpan(ts trace.Span) Span {
	return Span{internal: ts}
}

// End completes the span.
// Updates to the span are not allowed after this method has been called.
func (s Span) End() {
	if s.internal == nil {
		return
	}
	s.internal.End()
}

// SetError sets the status of the span to error.
func (s Span) SetError(err error) {
	if s.internal == nil {
		return
	}
	v := fmt.Sprintf("%+v", err)
	s.internal.RecordError(err, trace.WithAttributes(
		attribute.String("exception.stacktrace", v),
	))
	s.internal.SetStatus(codes.Error, err.Error())
	sc := errors.StatusCode(err)
	s.internal.SetAttributes(attribute.Int("http.status_code", sc))
}

// SetOK sets the status of the span to OK, with the indicated response status code.
func (s Span) SetOK(statusCode int) {
	if s.internal == nil {
		return
	}
	s.internal.SetStatus(codes.Ok, "")
	s.internal.SetAttributes(attribute.Int("http.status_code", statusCode))
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
			group = append(group, slogToTracingAttrs(prefix+f.Key+".", a)...)
		}
		return group
	case slog.KindString:
		return []attribute.KeyValue{
			attribute.String(prefix+f.Key, f.Value.String()),
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

// log records a log event on the span.
func (s Span) log(severity string, msg string, args ...any) {
	if s.internal == nil {
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
	s.internal.AddEvent("log", trace.WithAttributes(attrs...))
}

// LogDebug records a debug log event on the span.
func (s Span) LogDebug(msg string, args ...any) {
	s.log("debug", msg, args...)
}

// LogInfo records an info log event on the span.
func (s Span) LogInfo(msg string, args ...any) {
	s.log("info", msg, args...)
}

// LogWarn records a warning log event on the span.
func (s Span) LogWarn(msg string, args ...any) {
	s.log("warn", msg, args...)
}

// LogError records an error log event on the span.
func (s Span) LogError(msg string, args ...any) {
	s.log("error", msg, args...)
}

/*
SetAttributes tags the span during its creation.
The arguments are expected in the standard slog name=value pairs pattern.

	span.SetAttributes("string", s, "bool", false)
*/
func (s Span) SetAttributes(args ...any) {
	if s.internal == nil {
		return
	}
	for i := 0; i < len(args); i++ {
		var v any
		k, ok := args[i].(string)
		if !ok {
			k = "!BADKEY"
			v = args[i]
		} else {
			if i+1 < len(args) {
				v = args[i+1]
			}
			i++
		}
		switch vv := v.(type) {
		case string:
			s.internal.SetAttributes(attribute.String(k, vv))
		case []string:
			s.internal.SetAttributes(attribute.StringSlice(k, vv))
		case fmt.Stringer:
			s.internal.SetAttributes(attribute.Stringer(k, vv))
		case bool:
			s.internal.SetAttributes(attribute.Bool(k, vv))
		case []bool:
			s.internal.SetAttributes(attribute.BoolSlice(k, vv))
		case int:
			s.internal.SetAttributes(attribute.Int(k, vv))
		case []int:
			s.internal.SetAttributes(attribute.IntSlice(k, vv))
		case int64:
			s.internal.SetAttributes(attribute.Int64(k, vv))
		case []int64:
			s.internal.SetAttributes(attribute.Int64Slice(k, vv))
		case float64:
			s.internal.SetAttributes(attribute.Float64(k, vv))
		case []float64:
			s.internal.SetAttributes(attribute.Float64Slice(k, vv))
		default:
			s.internal.SetAttributes(attribute.String(k, fmt.Sprintf("%v", v)))
		}
	}
}

// SetClientIP tags the span during its creation with the IP address and port number of the client.
func (s Span) SetClientIP(ipPort string) {
	if s.internal == nil {
		return
	}
	p := strings.LastIndex(ipPort, ":")
	b := strings.LastIndex(ipPort, "]") // For IPv6, e.g. [::1]:443
	if p > 0 && p > b {
		portInt, _ := strconv.Atoi(ipPort[p+1:])
		s.internal.SetAttributes(
			attribute.String("client.address", ipPort[:p]),
			attribute.Int("client.port", portInt),
		)
	}
}

// IsEmpty indicates if the span is not initialized.
func (s Span) IsEmpty() bool {
	return s.internal == nil
}

// TraceID is an identifier that groups related spans together.
func (s Span) TraceID() string {
	if s.internal == nil {
		return ""
	}
	return s.internal.SpanContext().TraceID().String()
}
