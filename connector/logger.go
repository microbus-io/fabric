/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

/*
LogDebug logs a message at DEBUG level.
DEBUG level messages are ignored in PROD environments or if the MICROBUS_LOG_DEBUG environment variable is not set.
The message should be static and concise. Optional fields can be added for variable data.

Example:

	c.LogDebug(ctx, "Tight loop", log.String("index", i))
*/
func (c *Connector) LogDebug(ctx context.Context, msg string, fields ...log.Field) {
	logger := c.logger
	if logger == nil || !c.logDebug {
		return
	}
	span := c.Span(ctx)
	if !span.IsEmpty() {
		traceID := span.TraceID()
		span.Log("debug", msg, fields...)
		fields = append(fields, log.String("trace", traceID))
	}
	logger.Debug(msg, fields...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "DEBUG")
}

/*
LogInfo logs a message at INFO level.
The message should be static and concise. Optional fields can be added for variable data.

Example:

	c.LogInfo(ctx, "File uploaded", log.String("gb", sizeGB))
*/
func (c *Connector) LogInfo(ctx context.Context, msg string, fields ...log.Field) {
	logger := c.logger
	if logger == nil {
		return
	}
	span := c.Span(ctx)
	if !span.IsEmpty() {
		traceID := span.TraceID()
		span.Log("info", msg, fields...)
		fields = append(fields, log.String("trace", traceID))
	}
	logger.Info(msg, fields...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "INFO")
}

/*
LogWarn logs a message at WARN level.
The message should be static and concise. Optional fields can be added for variable data.

Example:

	c.LogWarn(ctx, "Dropping job", log.String("job", jobID))
*/
func (c *Connector) LogWarn(ctx context.Context, msg string, fields ...log.Field) {
	logger := c.logger
	if logger == nil {
		return
	}
	span := c.Span(ctx)
	if !span.IsEmpty() {
		traceID := span.TraceID()
		span.Log("warn", msg, fields...)
		fields = append(fields, log.String("trace", traceID))
	}
	logger.Warn(msg, fields...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "WARN")

	if c.deployment == LOCAL || c.deployment == TESTINGAPP {
		for _, f := range fields {
			if f.Key == "error" {
				if err, ok := f.Interface.(error); ok {
					sep := strings.Repeat("~", 120)
					fmt.Fprintf(os.Stderr, "%s\n%+v\n%s\n", "\u25bc"+sep+"\u25bc", err, "\u25b2"+sep+"\u25b2")
					break
				}
			}
		}
	}
}

/*
LogError logs a message at ERROR level.
The message should be static and concise. Optional fields can be added for variable data.
To log an error object use the log.Error field.

Example:

	c.LogError(ctx, "Opening file", log.Error(err), log.String("file", fileName))
*/
func (c *Connector) LogError(ctx context.Context, msg string, fields ...log.Field) {
	logger := c.logger
	if logger == nil {
		return
	}
	span := c.Span(ctx)
	if !span.IsEmpty() {
		traceID := span.TraceID()
		span.Log("error", msg, fields...)
		fields = append(fields, log.String("trace", traceID))
	}
	logger.Error(msg, fields...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "ERROR")

	if c.deployment == LOCAL || c.deployment == TESTINGAPP {
		for _, f := range fields {
			if f.Key == "error" {
				if err, ok := f.Interface.(error); ok {
					sep := strings.Repeat("~", 120)
					fmt.Fprintf(os.Stderr, "%s\n%+v\n%s\n", "\u25bc"+sep+"\u25bc", err, "\u25b2"+sep+"\u25b2")
					break
				}
			}
		}
	}
}

// initLogger initializes a logger to match the deployment environment
func (c *Connector) initLogger() (err error) {
	if c.logger != nil {
		return nil
	}

	if debug := os.Getenv("MICROBUS_LOG_DEBUG"); debug != "" {
		c.logDebug = true
	}

	env := c.Deployment()

	var config zap.Config
	if env == LOCAL || env == TESTINGAPP {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
		config.DisableStacktrace = true
		// config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else if env == LAB {
		config = zap.NewProductionConfig()
		config.Level.SetLevel(zapcore.DebugLevel)
		config.DisableStacktrace = true
	} else {
		// Default PROD config
		config = zap.NewProductionConfig()
		config.DisableStacktrace = true
	}

	c.logger, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return errors.Trace(err)
	}
	c.logger = c.logger.With(
		log.String("host", c.HostName()),
		log.String("id", c.ID()),
		log.Int("ver", c.Version()),
	)
	return nil
}

// terminateLogger flushes and terminates the logger
func (c *Connector) terminateLogger() error {
	logger := c.logger
	if logger == nil {
		return nil
	}
	c.logger = nil
	err := logger.Sync()
	return errors.Trace(err)
}
