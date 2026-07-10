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

package connector

import (
	stderrors "errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/testarossa"
)

func TestConnector_Log(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()
	stderror := stderrors.New("error")

	con := New("log.connector")
	assert.False(con.IsStarted())

	// No-op when logger is nil, no logs to observe
	assert.Nil(con.logger.Load())
	con.LogDebug(ctx, "This is a log debug message", "someStr", "some string")
	con.LogInfo(ctx, "This is a log info message", "someStr", "some string")
	con.LogWarn(ctx, "This is a log warn message", "error", stderror, "someStr", "some string")
	con.LogError(ctx, "This is a log error message", "error", stderror, "someStr", "some string")

	// Start service to initialize logger
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Logger initialized, it can now be observed
	assert.NotNil(con.logger.Load())

	// Observe the logs to assert expected values. The capture logger composes the same logHandler the connector
	// uses, so debug gating and enrichment are exercised; only the terminal handler is swapped for an in-memory one.
	var buf strings.Builder
	con.logger.Store(slog.New(&logHandler{c: con, console: slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})}))

	// LogDebug is gated by c.logDebug, which Startup sets from MICROBUS_LOG_DEBUG.
	// Drive it explicitly so the test is independent of the ambient env var and
	// verifies both gating directions on every run.
	con.logDebug = false
	con.LogDebug(ctx, "This is a log debug message", "someStr", "some string")
	con.LogInfo(ctx, "This is a log info message", "someStr", "some string")
	con.LogWarn(ctx, "This is a log warn message", "error", stderror, "someStr", "some string")
	con.LogError(ctx, "This is a log error message", "error", stderror, "someStr", "some string")

	bufStr := buf.String()
	assert.Contains(bufStr, `level=INFO msg="This is a log info message"`)
	assert.Contains(bufStr, `level=WARN msg="This is a log warn message"`)
	assert.Contains(bufStr, `level=ERROR msg="This is a log error message"`)
	assert.NotContains(bufStr, `level=DEBUG msg="This is a log debug message"`)

	// With debug logging enabled, LogDebug emits.
	con.logDebug = true
	con.LogDebug(ctx, "This is a log debug message", "someStr", "some string")
	assert.Contains(buf.String(), `level=DEBUG msg="This is a log debug message"`)
}

// TestConnector_OTLPLogsProvider verifies that a configured logs endpoint builds the OTLP logs provider over the
// shared connection, and that shutting down releases it.
func TestConnector_OTLPLogsProvider(t *testing.T) {
	// No parallel - sets environment and mutates the shared connection registry
	assert := testarossa.For(t)
	ctx := t.Context()

	env.Push("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	env.Push("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://127.0.0.1:14319") // Valid format, never dialed
	defer env.Pop("OTEL_EXPORTER_OTLP_PROTOCOL")
	defer env.Pop("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")

	con := New("otlp.logs.connector")
	err := con.Startup(ctx)
	assert.NoError(err)
	assert.NotNil(con.logProvider)
	key := con.logOTLPKey
	assert.True(key != "", "logs connection key should be set")

	err = con.Shutdown(ctx)
	assert.NoError(err)
	assert.True(con.logProvider == nil, "log provider should be cleared on shutdown")
	assert.Equal(-1, connRefs(key), "shared connection should be released on shutdown")
}

// TestConnector_OTLPLogsUnreachable verifies that a connector starts up and shuts down promptly even when the
// configured OTLP logs collector is unreachable - the logs signal honors the same best-effort export contract as
// traces and metrics.
func TestConnector_OTLPLogsUnreachable(t *testing.T) {
	// No parallel - sets environment
	ctx := t.Context()
	assert := testarossa.For(t)

	// 127.0.0.1:1 refuses connections immediately on loopback - simulates a down collector without a connect wait.
	env.Push("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	env.Push("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://127.0.0.1:1")
	defer env.Pop("OTEL_EXPORTER_OTLP_PROTOCOL")
	defer env.Pop("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")

	con := New("otlp.logs.unreachable.connector")

	startupStart := time.Now()
	err := con.Startup(ctx)
	startupElapsed := time.Since(startupStart)
	if assert.NoError(err) {
		assert.True(startupElapsed < 5*time.Second, "startup should not block on unreachable OTel collector, took %s", startupElapsed)
	}

	shutdownStart := time.Now()
	err = con.Shutdown(ctx)
	shutdownElapsed := time.Since(shutdownStart)
	if assert.NoError(err) {
		assert.True(shutdownElapsed < 20*time.Second, "shutdown should not block on unreachable OTel collector, took %s", shutdownElapsed)
	}
}

// TestConnector_LogTraceRouting verifies that, with both a console and an OTLP leg, the trace ID is added as a string
// attribute only on the console leg - the OTLP leg relies on the bridge to stamp native trace context - while both
// legs receive the record.
func TestConnector_LogTraceRouting(t *testing.T) {
	// No parallel - sets environment
	assert := testarossa.For(t)
	ctx := t.Context()

	env.Push("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "nil") // nil trace client, so spans are created
	defer env.Pop("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")

	con := New("log.trace.routing.connector")
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	con.logDebug = true
	var console, otel strings.Builder
	con.logger.Store(slog.New(&logHandler{
		c:       con,
		console: slog.NewTextHandler(&console, &slog.HandlerOptions{Level: slog.LevelDebug}),
		otel:    slog.NewTextHandler(&otel, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}))

	spanCtx, span := con.StartSpan(ctx, "test")
	con.LogInfo(spanCtx, "with span", "someStr", "some string")
	span.End()

	assert.Contains(console.String(), "with span")
	assert.Contains(console.String(), "trace=")
	assert.Contains(otel.String(), "with span")
	assert.NotContains(otel.String(), "trace=")
}
