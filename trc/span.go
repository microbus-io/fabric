/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package trc

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
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
	Log(severity string, message string, fields ...log.Field)
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

// Log records a log event on the span.
func (s SpanImpl) Log(severity string, msg string, fields ...log.Field) {
	if s.Span == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("severity", severity),
		attribute.String("message", msg),
	}
	for _, f := range fields {
		var attr attribute.KeyValue
		switch f.Type {

		case zapcore.StringType:
			attr = attribute.String(f.Key, f.String)
		case zapcore.StringerType:
			attr = attribute.Stringer(f.Key, f.Interface.(fmt.Stringer))

		case zapcore.BoolType:
			attr = attribute.Bool(f.Key, f.Integer != 0)

		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type,
			zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
			attr = attribute.Int(f.Key, int(f.Integer))

		case zapcore.DurationType:
			attr = attribute.String(f.Key, time.Duration(f.Integer).String())
		case zapcore.TimeType:
			attr = attribute.String(f.Key, time.Unix(0, f.Integer).UTC().Format(time.RFC3339Nano))
		case zapcore.TimeFullType:
			attr = attribute.String(f.Key, f.Interface.(time.Time).Format(time.RFC3339Nano))

		case zapcore.Float64Type:
			attr = attribute.Float64(f.Key, math.Float64frombits(uint64(f.Integer)))
		case zapcore.Float32Type:
			attr = attribute.Float64(f.Key, float64(math.Float32frombits(uint32(f.Integer))))

		case zapcore.ErrorType:
			if f.Key == "error" {
				attr = attribute.String("exception.message", f.Interface.(error).Error())
				err, ok := f.Interface.(*errors.TracedError)
				if ok {
					attrs = append(attrs, attribute.Stringer("exception.stacktrace", err))
				}
			} else {
				attr = attribute.String(f.Key, f.Interface.(error).Error())
			}
		}

		if attr.Key != "" {
			attrs = append(attrs, attr)
		}
	}
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
