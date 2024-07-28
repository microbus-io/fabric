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
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
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
	s.internal.SetAttributes(attribute.Int("http.response.status_code", sc))
}

// SetOK sets the status of the span to OK, with the indicated response status code.
func (s Span) SetOK(statusCode int) {
	if s.internal == nil {
		return
	}
	s.internal.SetStatus(codes.Ok, "")
	if statusCode != http.StatusOK {
		s.internal.SetAttributes(attribute.Int("http.response.status_code", statusCode))
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
func (s Span) Log(severity string, msg string, args ...any) {
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

// SetString tags the span during its creation.
func (s Span) SetString(k string, v string) {
	if s.internal == nil {
		return
	}
	s.internal.SetAttributes(attribute.String(k, v))
}

// SetStrings tags the span during its creation.
func (s Span) SetStrings(k string, v []string) {
	if s.internal == nil {
		return
	}
	s.internal.SetAttributes(attribute.StringSlice(k, v))
}

// SetBool tags the span during its creation.
func (s Span) SetBool(k string, v bool) {
	if s.internal == nil {
		return
	}
	s.internal.SetAttributes(attribute.Bool(k, v))
}

// SetInt tags the span during its creation.
func (s Span) SetInt(k string, v int) {
	if s.internal == nil {
		return
	}
	s.internal.SetAttributes(attribute.Int(k, v))
}

// SetFloat tags the span during its creation.
func (s Span) SetFloat(k string, v float64) {
	if s.internal == nil {
		return
	}
	s.internal.SetAttributes(attribute.Float64(k, v))
}

// SetRequest tags the span with the request data.
// Warning: this has a large memory footprint.
func (s Span) SetRequest(r *http.Request) {
	if s.internal == nil {
		return
	}
	s.internal.SetAttributes(attributesOfRequest(r)...)
	s.SetClientIP(r.RemoteAddr)
}

// SetClientIP tags the span during its creation with the IP address and port number of the client.
func (s Span) SetClientIP(ip string) {
	p := strings.LastIndex(ip, ":")
	if p > 0 {
		portInt, _ := strconv.Atoi(ip[p+1:])
		s.internal.SetAttributes(
			attribute.String("client.address", ip[:p]),
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

// Attributes returns the attributes set on the span.
func (s Span) Attributes() map[string]string {
	m := map[string]string{}
	attributes := reflect.ValueOf(s.internal).Elem().FieldByName("attributes")
	for i := 0; i < attributes.Len(); i++ {
		k := attributes.Index(i).FieldByName("Key").String()
		v := attributes.Index(i).FieldByName("Value").FieldByName("stringly").String()
		if v == "" {
			i := attributes.Index(i).FieldByName("Value").FieldByName("numeric").Uint()
			if i != 0 {
				v = strconv.Itoa(int(i))
			}
		}
		if v == "" {
			slice := attributes.Index(i).FieldByName("Value").FieldByName("slice").Elem()
			if slice.Len() == 1 {
				v = slice.Index(0).String()
			}
		}
		m[k] = v
	}
	return m
}

// Status returns the status of the span: 0=unset; 1=error; 2=OK.
func (s Span) Status() int {
	status := reflect.ValueOf(s.internal).Elem().FieldByName("status")
	return int(status.FieldByName("Code").Uint())
}
