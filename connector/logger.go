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
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/microbus-io/fabric/env"
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
	logger := c.logger
	if logger == nil || !c.logDebug {
		return
	}
	span := c.Span(ctx)
	if !span.IsEmpty() {
		traceID := span.TraceID()
		if c.deployment != PROD {
			span.Log("debug", msg, args...)
		}
		args = append(args, "trace", traceID)
	}
	logger.Debug(msg, args...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "DEBUG")
}

/*
LogInfo logs a message at INFO level.
The message should be static and concise. Optional arguments can be added for variable data.
Arguments conform to the standard slog pattern.

Example:

	c.LogInfo(ctx, "File uploaded", "gb", sizeGB)
*/
func (c *Connector) LogInfo(ctx context.Context, msg string, args ...any) {
	logger := c.logger
	if logger == nil {
		return
	}
	span := c.Span(ctx)
	if !span.IsEmpty() {
		traceID := span.TraceID()
		if c.deployment != PROD {
			span.Log("info", msg, args...)
		}
		args = append(args, "trace", traceID)
	}
	logger.Info(msg, args...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "INFO")
}

/*
LogWarn logs a message at WARN level.
The message should be static and concise. Optional arguments can be added for variable data.
Arguments conform to the standard slog pattern.

Example:

	c.LogWarn(ctx, "Dropping job", "job", jobID)
*/
func (c *Connector) LogWarn(ctx context.Context, msg string, args ...any) {
	logger := c.logger
	if logger == nil {
		return
	}
	span := c.Span(ctx)
	if !span.IsEmpty() {
		traceID := span.TraceID()
		if c.deployment != PROD {
			span.Log("warn", msg, args...)
		}
		args = append(args, "trace", traceID)
	}
	logger.Warn(msg, args...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "WARN")

	if c.deployment == LOCAL || c.deployment == TESTING {
		for _, f := range args {
			if err, ok := f.(error); ok {
				sep := strings.Repeat("~", 120)
				fmt.Fprintf(os.Stderr, "%s\n%+v\n%s\n", "\u25bc"+sep+"\u25bc", err, "\u25b2"+sep+"\u25b2")
				break
			}
		}
	}
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
	logger := c.logger
	if logger == nil {
		return
	}
	span := c.Span(ctx)
	if !span.IsEmpty() {
		traceID := span.TraceID()
		if c.deployment != PROD {
			span.Log("error", msg, args...)
		}
		args = append(args, "trace", traceID)
	}
	logger.Error(msg, args...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "ERROR")

	if c.deployment == LOCAL || c.deployment == TESTING {
		for _, f := range args {
			if err, ok := f.(error); ok {
				sep := strings.Repeat("~", 120)
				fmt.Fprintf(os.Stderr, "%s\n%+v\n%s\n", "\u25bc"+sep+"\u25bc", err, "\u25b2"+sep+"\u25b2")
				break
			}
		}
	}
}

// initLogger initializes a logger to match the deployment environment.
func (c *Connector) initLogger() (err error) {
	if c.logger != nil {
		return nil
	}

	if debug := env.Get("MICROBUS_LOG_DEBUG"); debug != "" {
		c.logDebug = true
	}

	env := c.Deployment()

	var handler slog.Handler
	if env == LOCAL || env == TESTING {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelDebug,
		})
	} else if env == LAB {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelDebug,
		})
	} else {
		// Default PROD config
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelInfo,
		})
	}

	c.logger = slog.New(handler).With(
		"host", c.Hostname(),
		"id", c.ID(),
		"ver", c.Version(),
	)
	return nil
}
