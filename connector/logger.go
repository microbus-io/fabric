package connector

import (
	"context"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogDebug logs a message at DEBUG level.
// The message should be static and concise, optional fields can be added
func (c *Connector) LogDebug(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Debug(msg, fields...)
}

// LogInfo logs a message at INFO level.
// The message should be static and concise, optional fields can be added
func (c *Connector) LogInfo(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Info(msg, fields...)
}

// LogWarn logs a message at WARN level.
// The message should be static and concise, optional fields can be added
func (c *Connector) LogWarn(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Warn(msg, fields...)
}

// LogError logs a message at ERROR level.
// The message should be static and concise, optional fields can be added.
// To log an error object use the log.Error field
func (c *Connector) LogError(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Error(msg, fields...)
}

// initLogger initializes a logger to match the deployment environment
func (c *Connector) initLogger() (err error) {
	if c.logger != nil {
		return nil
	}

	env := c.Deployment()

	var config zap.Config
	if env == LOCAL {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
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
