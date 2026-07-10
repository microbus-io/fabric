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
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/mem"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// logActorKeyType is a private context-key type so only this package can read or write the
// log-actor cache.
type logActorKeyType struct{}

var logActorKey = logActorKeyType{}

// logActorClaims returns a context carrying the verified actor claims for use as log attributes.
// It is intended to be called by the connector immediately after verifyToken succeeds.
func logActorClaims(ctx context.Context, claims jwt.MapClaims) context.Context {
	if ctx == nil || claims == nil {
		return ctx
	}
	return context.WithValue(ctx, logActorKey, claims)
}

// appendActorClaims appends iss/sub/tenant log attributes from the cached actor claims, if present.
// Sub is potentially PII (often an email) - operators must apply log retention and access
// controls accordingly.
func appendActorClaims(args []any, ctx context.Context) []any {
	if ctx == nil {
		return args
	}
	claims, _ := ctx.Value(logActorKey).(jwt.MapClaims)
	if claims == nil {
		return args
	}
	if iss, ok := claims["iss"].(string); ok && iss != "" {
		args = append(args, "iss", iss)
	}
	if sub, ok := claims["sub"].(string); ok && sub != "" {
		args = append(args, "sub", sub)
	}
	if tenant, ok := claims["tenant"]; ok {
		args = append(args, "tenant", tenant)
	} else if tid, ok := claims["tid"]; ok {
		args = append(args, "tenant", tid)
	}
	if roles, ok := claims["roles"]; ok && roles != nil {
		args = append(args, "roles", roles)
	}
	return args
}

const (
	Gray    = "\033[38;2;128;128;128m" // #808080
	Magenta = "\033[38;2;197;134;192m" // #c586c0
	Yellow  = "\033[38;2;215;186;125m" // #d7ba7d
	Orange  = "\033[38;2;206;145;120m" // #ce9178
	Green   = "\033[38;2;106;153;85m"  // #6A9955
	Cyan    = "\033[38;2;156;220;254m" // #9cdcfe
	Red     = "\033[38;2;244;71;71m"   // #f44747
	White   = "\033[38;2;212;212;212m" // #d4d4d4
	Blue    = "\033[38;2;86;159;214m"  // #569cd6
	Reset   = "\033[0m"
)

/*
LogDebug logs a message at DEBUG level.
DEBUG level messages are ignored in PROD environments or if the MICROBUS_LOG_DEBUG environment variable is not set.
The message should be static and concise. Optional arguments can be added for variable data.
Arguments conform to the standard slog pattern.

Example:

	c.LogDebug(ctx, "Tight loop", "index", i)
*/
func (c *Connector) LogDebug(ctx context.Context, msg string, args ...any) {
	logger := c.logger.Load()
	if logger == nil {
		return
	}
	logger.DebugContext(ctx, msg, args...)
}

/*
LogInfo logs a message at INFO level.
The message should be static and concise. Optional arguments can be added for variable data.
Arguments conform to the standard slog pattern.

Example:

	c.LogInfo(ctx, "File uploaded", "gb", sizeGB)
*/
func (c *Connector) LogInfo(ctx context.Context, msg string, args ...any) {
	logger := c.logger.Load()
	if logger == nil {
		return
	}
	logger.InfoContext(ctx, msg, args...)
}

/*
LogWarn logs a message at WARN level.
The message should be static and concise. Optional arguments can be added for variable data.
Arguments conform to the standard slog pattern.

Example:

	c.LogWarn(ctx, "Dropping job", "job", jobID)
*/
func (c *Connector) LogWarn(ctx context.Context, msg string, args ...any) {
	logger := c.logger.Load()
	if logger == nil {
		return
	}
	logger.WarnContext(ctx, msg, args...)
}

/*
LogError logs a message at ERROR level.
The message should be static and concise. Optional arguments can be added for variable data.
Arguments conform to the standard slog pattern.
When logging an error object, name it "error".

Example:

	c.LogError(ctx, "Opening file", "error", err, "file", fileName)
*/
func (c *Connector) LogError(ctx context.Context, msg string, args ...any) {
	logger := c.logger.Load()
	if logger == nil {
		return
	}
	logger.ErrorContext(ctx, msg, args...)
}

// discardLogger is returned by Logger before Startup, so the accessor never returns nil and pre-startup logging is a
// no-op, matching the convenience LogDebug/LogInfo/LogWarn/LogError methods.
var discardLogger = slog.New(slog.DiscardHandler)

// Logger returns the structured logger of the microservice. Before Startup it returns a logger that discards all
// records, matching the no-op behavior of LogDebug/LogInfo/LogWarn/LogError. Logging through the logger's
// context-aware methods (DebugContext, InfoContext, WarnContext, ErrorContext) enriches each record with trace
// correlation and actor identity drawn from the context; the non-context methods (Debug, Info, ...) log without
// that enrichment. The convenience LogDebug/LogInfo/LogWarn/LogError methods delegate to the context-aware methods.
//
// Fetch the logger at point of use rather than caching it across Startup: the discard logger is replaced by the
// real one during Startup, so a reference held from before Startup keeps discarding.
func (c *Connector) Logger() *slog.Logger {
	logger := c.logger.Load()
	if logger == nil {
		return discardLogger
	}
	return logger
}

// initLogger initializes a logger to match the deployment environment, fanning out to an OTLP logs exporter when one
// is configured via the environment.
func (c *Connector) initLogger(ctx context.Context) (err error) {
	if c.logger.Load() != nil {
		return nil
	}

	if v := env.Get("MICROBUS_LOG_DEBUG"); v == "1" || strings.EqualFold(v, "true") {
		c.logDebug = true
	}

	var console slog.Handler
	switch c.Deployment() {
	case LOCAL:
		console = &colorfulLogHandler{}
	case TESTING:
		console = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelDebug,
		})
	case LAB:
		console = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelDebug,
		})
	default:
		// Default PROD config
		console = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelInfo,
		})
	}

	otelLeg, err := c.initLogExporter(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	c.logger.Store(slog.New(&logHandler{c: c, console: console, otel: otelLeg}).With(
		"plane", c.Plane(),
		"service", c.Hostname(),
		"ver", c.Version(),
		"id", c.ID(),
		"deployment", c.Deployment(),
	))
	return nil
}

// initLogExporter builds an OTLP logs handler over the shared connection when a logs endpoint is configured,
// returning nil when logs export is disabled. The handler bridges slog records to the OpenTelemetry logs pipeline.
func (c *Connector) initLogExporter(ctx context.Context) (slog.Handler, error) {
	endpoint := resolveOTLPEndpoint("LOGS")
	if endpoint == "" {
		return nil, nil
	}
	conn, key, err := acquireOTLPConn("LOGS", endpoint)
	if err != nil {
		return nil, errors.Trace(err)
	}
	c.logOTLPKey = key
	var exp sdklog.Exporter
	if conn.protocol() == "grpc" {
		exp, err = otlploggrpc.New(ctx,
			otlploggrpc.WithGRPCConn(conn.grpc),
			otlploggrpc.WithRetry(otlploggrpc.RetryConfig{Enabled: false}),
		)
	} else {
		exp, err = otlploghttp.New(ctx,
			otlploghttp.WithEndpointURL(endpoint),
			otlploghttp.WithHTTPClient(conn.http),
			otlploghttp.WithRetry(otlploghttp.RetryConfig{Enabled: false}),
		)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	c.logProvider = sdklog.NewLoggerProvider(
		sdklog.WithResource(c.otelResource()),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
	)
	return otelslog.NewHandler("microbus", otelslog.WithLoggerProvider(c.logProvider)), nil
}

// termLogger flushes and shuts down the OTLP logs provider and releases its shared connection. The logger is reset so
// a subsequent Startup rebuilds the chain. It is the first OpenTelemetry provider torn down on shutdown - the reverse
// of initialization order, and after the final log entry - since logging feeds the trace and metric providers.
func (c *Connector) termLogger(ctx context.Context) (err error) {
	if c.logProvider != nil {
		err = c.logProvider.Shutdown(ctx)
		c.logProvider = nil
	}
	releaseOTLPConn(c.logOTLPKey)
	c.logOTLPKey = ""
	c.logger.Store(nil)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
logHandler is the connector's root slog handler. It applies context-derived enrichment - stamping the trace ID and
actor claims onto each record, mirroring the record onto the active span as an event, counting the message, and
dumping errors to stderr in developer deployments - and then fans the record out to the deployment's terminal
(console/JSON) handler and, when configured, the OTLP logs handler.

Enrichment happens here, in the handler, rather than in the LogXXX methods, so that records logged through the slog
logger returned by Logger() are enriched identically. The non-context logger methods carry a background context with
no span or actor, so they pass through un-enriched, which is the intended behavior.

The trace ID is added as a string attribute only on the console leg, where it is the sole correlation handle. The
OTLP leg omits it because the otelslog bridge reads the span context and stamps native TraceId/SpanId on the log
record, which backends use to correlate logs with traces.
*/
type logHandler struct {
	c       *Connector
	console slog.Handler // Deployment terminal handler: colorful (LOCAL), text (TESTING), or JSON (LAB/PROD)
	otel    slog.Handler // OTLP logs bridge, or nil when logs export is disabled
}

// Enabled gates DEBUG records on the MICROBUS_LOG_DEBUG flag and otherwise defers to whichever leg accepts the level.
func (h *logHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level < slog.LevelInfo && !h.c.logDebug {
		return false
	}
	if h.console.Enabled(ctx, level) {
		return true
	}
	return h.otel != nil && h.otel.Enabled(ctx, level)
}

func (h *logHandler) Handle(ctx context.Context, rec slog.Record) error {
	span := h.c.Span(ctx)
	if !span.IsEmpty() && h.c.deployment != PROD {
		switch {
		case rec.Level >= slog.LevelError:
			span.LogError(rec.Message, recordArgs(rec)...)
		case rec.Level >= slog.LevelWarn:
			span.LogWarn(rec.Message, recordArgs(rec)...)
		case rec.Level >= slog.LevelInfo:
			span.LogInfo(rec.Message, recordArgs(rec)...)
		default:
			span.LogDebug(rec.Message, recordArgs(rec)...)
		}
	}
	_ = h.c.IncrementCounter(ctx, "microbus_log_messages", 1,
		"message", rec.Message,
		"severity", rec.Level.String(),
	)
	if (h.c.deployment == LOCAL || h.c.deployment == TESTING) && rec.Level >= slog.LevelWarn {
		dumpRecordError(h.c.deployment, rec)
	}

	actorArgs := appendActorClaims(nil, ctx)

	// OTLP leg: native trace correlation comes from the context via the bridge, so no trace string attribute.
	var firstErr error
	if h.otel != nil && h.otel.Enabled(ctx, rec.Level) {
		otelRec := rec.Clone()
		otelRec.Add(actorArgs...)
		firstErr = h.otel.Handle(ctx, otelRec)
	}
	// Console leg: the trace ID is the only correlation handle in the human/structured output.
	if h.console.Enabled(ctx, rec.Level) {
		consoleRec := rec
		if h.otel != nil {
			consoleRec = rec.Clone()
		}
		if !span.IsEmpty() {
			consoleRec.AddAttrs(slog.String("trace", span.TraceID()))
		}
		consoleRec.Add(actorArgs...)
		if e := h.console.Handle(ctx, consoleRec); firstErr == nil {
			firstErr = e
		}
	}
	return firstErr
}

func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := &logHandler{c: h.c, console: h.console.WithAttrs(attrs)}
	if h.otel != nil {
		clone.otel = h.otel.WithAttrs(attrs)
	}
	return clone
}

func (h *logHandler) WithGroup(name string) slog.Handler {
	clone := &logHandler{c: h.c, console: h.console.WithGroup(name)}
	if h.otel != nil {
		clone.otel = h.otel.WithGroup(name)
	}
	return clone
}

var _ slog.Handler = &logHandler{}

// recordArgs flattens a record's attributes into the slog name=value pattern for mirroring onto a span event.
func recordArgs(rec slog.Record) []any {
	args := make([]any, 0, rec.NumAttrs()*2)
	rec.Attrs(func(a slog.Attr) bool {
		args = append(args, a.Key, a.Value.Any())
		return true
	})
	return args
}

// dumpRecordError prints the first error attribute of a record to stderr, framed by separators, for visibility in
// developer deployments. The frame is colored red in LOCAL.
func dumpRecordError(deployment string, rec slog.Record) {
	rec.Attrs(func(a slog.Attr) bool {
		err, ok := a.Value.Any().(error)
		if !ok {
			return true
		}
		color := ""
		reset := ""
		if deployment == LOCAL {
			color = Red
			reset = Reset
		}
		sep := strings.Repeat("~", 120)
		fmt.Fprintf(os.Stderr, "%s%s\n%+v\n%s%s\n", color, "\u25bc"+sep+"\u25bc", err, "\u25b2"+sep+"\u25b2", reset)
		return false
	})
}

// colorfulLogHandler renders colorized, human-readable log lines, used as the terminal handler in the LOCAL deployment.
type colorfulLogHandler struct {
	attrs []slog.Attr
}

func (h *colorfulLogHandler) Enabled(ctx context.Context, _ slog.Level) bool {
	return true
}

func (h *colorfulLogHandler) Handle(ctx context.Context, rec slog.Record) error {
	block := mem.Alloc(512)
	defer mem.Free(block)
	w := bytes.NewBuffer(block)

	w.WriteString(Gray)

	// Timestamp
	if !rec.Time.IsZero() {
		w.WriteString(rec.Time.Format("15:04:05.000 "))
	}

	// Level
	switch rec.Level {
	case slog.LevelDebug:
		w.WriteString(White)
		w.WriteString("DBUG")
	case slog.LevelInfo:
		w.WriteString(White)
		w.WriteString("INFO")
	case slog.LevelWarn:
		w.WriteString(Yellow)
		w.WriteString("WARN")
	case slog.LevelError:
		w.WriteString(Red)
		w.WriteString("ERR!")
	default:
		w.WriteString(White)
		lvl := rec.Level.String()
		if len(lvl) > 4 {
			lvl = lvl[:4]
		}
		fmt.Fprintf(w, "%-4s", lvl)
	}

	// Service
	service := ""
	for _, a := range h.attrs {
		if a.Key == "service" {
			service = a.Value.String()
			break
		}
	}
	const maxlen = 20
	if len([]rune(service)) > maxlen {
		service = string([]rune(service)[:maxlen-1]) + "\u2026"
	}
	w.WriteString(Magenta)
	fmt.Fprintf(w, " %-*s", maxlen, service)

	// Message
	w.WriteString(Cyan)
	fmt.Fprintf(w, " %-30s ", rec.Message)

	// Attributes
	allZeros := func(s string) bool {
		for _, r := range s {
			if r != '0' {
				return false
			}
		}
		return true
	}
	rec.Attrs(func(a slog.Attr) bool {
		if a.Key == "trace" && allZeros(a.Value.String()) {
			return true
		}
		w.WriteString(" ")
		w.WriteString(Blue)
		w.WriteString(a.Key)
		w.WriteString(Gray)
		w.WriteString("=")
		orig := a.Value.String()
		quoted := strconv.Quote(orig)
		quoted = quoted[1 : len(quoted)-1]
		if quoted != orig {
			w.WriteString(Gray)
			w.WriteString(`"`)
			w.WriteString(White)
			w.WriteString(quoted)
			w.WriteString(Gray)
			w.WriteString(`"`)
		} else {
			w.WriteString(White)
			w.WriteString(orig)
		}
		return true
	})

	w.WriteString(Reset)
	w.WriteString("\n")

	os.Stderr.Write(w.Bytes())
	return nil
}

func (h *colorfulLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &colorfulLogHandler{
		attrs: append(h.attrs, attrs...),
	}
}

func (h *colorfulLogHandler) WithGroup(name string) slog.Handler {
	return h
}

var _ slog.Handler = &colorfulLogHandler{}
