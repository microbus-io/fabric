/*
Copyright 2023 Microbus LLC and various contributors

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
	if c.logger == nil || !c.logDebug {
		return
	}
	c.logger.Debug(msg, fields...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "DEBUG")
}

/*
LogInfo logs a message at INFO level.
The message should be static and concise. Optional fields can be added for variable data.

Example:

	c.LogInfo(ctx, "File uploaded", log.String("gb", sizeGB))
*/
func (c *Connector) LogInfo(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Info(msg, fields...)
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "INFO")
}

/*
LogWarn logs a message at WARN level.
The message should be static and concise. Optional fields can be added for variable data.

Example:

	c.LogWarn(ctx, "Dropping job", log.String("job", jobID))
*/
func (c *Connector) LogWarn(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Warn(msg, fields...)

	if c.deployment == LOCAL || c.deployment == TESTINGAPP {
		for _, f := range fields {
			if f.Type == zapcore.ErrorType && f.Key == "error" {
				sep := strings.Repeat("~", 120)
				fmt.Fprintf(os.Stderr, "%s\n%+v\n%s\n", "\u25bc"+sep+"\u25bc", f.Interface, "\u25b2"+sep+"\u25b2")
				break
			}
		}
	}
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "WARN")
}

/*
LogError logs a message at ERROR level.
The message should be static and concise. Optional fields can be added for variable data.
To log an error object use the log.Error field.

Example:

	c.LogError(ctx, "Opening file", log.Error(err), log.String("file", fileName))
*/
func (c *Connector) LogError(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Error(msg, fields...)

	if c.deployment == LOCAL || c.deployment == TESTINGAPP {
		for _, f := range fields {
			if f.Type == zapcore.ErrorType && f.Key == "error" {
				sep := strings.Repeat("~", 120)
				fmt.Fprintf(os.Stderr, "%s\n%+v\n%s\n", "\u25bc"+sep+"\u25bc", f.Interface, "\u25b2"+sep+"\u25b2")
				break
			}
		}
	}
	_ = c.IncrementMetric("microbus_log_messages_total", 1, msg, "ERROR")
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
		// config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else if env == LAB {
		config = zap.NewProductionConfig()
		config.Level.SetLevel(zapcore.DebugLevel)
	} else {
		// Default PROD config
		config = zap.NewProductionConfig()
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
	if c.logger == nil {
		return nil
	}
	err := c.logger.Sync()
	c.logger = nil
	return errors.Trace(err)
}
