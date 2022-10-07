package connector

import (
	"context"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogDebug logs a message at debug level. The message should be static and concise,
// optional fields can be added.
func (c *Connector) LogDebug(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Debug(msg, fields...)
}

// LogInfo logs a message at info level. The message should be static and concise,
// optional fields can be added.
func (c *Connector) LogInfo(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Info(msg, fields...)
}

// LogWarn logs a message and error at warn level. The message should be static and concise,
// optional fields can be added.
func (c *Connector) LogWarn(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Warn(msg, fields...)
}

// LogError logs a message and error at error level. The message should be static and concise,
// optional fields can be added.
func (c *Connector) LogError(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		return
	}
	c.logger.Error(msg, fields...)
}

// initLogger initializes a logger for the connector.
func (c *Connector) initLogger() (err error) {
	if c.logger != nil {
		return nil
	}

	env := c.Deployment()

	var config zap.Config
	if env == LOCAL {
		config = zap.NewDevelopmentConfig()
	} else if env == LAB {
		config = zap.NewProductionConfig()
		config.Level.SetLevel(zapcore.DebugLevel)
	} else {
		// Default PROD config
		config = zap.NewProductionConfig()
	}

	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)

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

// removeLogger removes the logger from the connector.
func (c *Connector) removeLogger() error {
	if c.logger == nil {
		return nil
	}
	err := c.logger.Sync()
	c.logger = nil
	return errors.Trace(err)
}
