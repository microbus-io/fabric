package connector

import (
	"context"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogInfo logs a message at info level. The message should be concise and fixed,
// optional fields can be added.
func (c *Connector) LogInfo(ctx context.Context, msg string, fields ...log.Field) {
	if c.logger == nil {
		_ = c.createLogger()
	}
	// TODO: Context is added but make use of this
	c.logger.Info(msg, fields...)
}

// LogError logs a message and error at error level. The message should be concise and fixed,
// optional fields can be added.
func (c *Connector) LogError(ctx context.Context, msg string, err error, fields ...log.Field) {
	if c.logger == nil {
		_ = c.createLogger()
	}
	// TODO: Context is added but make use of this
	fields = append(fields, log.Error(err))
	c.logger.Error(msg, fields...)
}

// createLogger creates a new logger.
func (c *Connector) createLogger() (err error) {
	if c.logger != nil {
		return nil
	}

	// TODO: Remove hardcoded env
	const (
		LOCAL       = "LOCAL"
		DEVELOPMENT = "DEVELOPMENT"
		PRODUCTION  = "PRODUCTION"
	)
	env := LOCAL

	var config zap.Config
	if env == LOCAL || env == DEVELOPMENT {
		config = zap.NewDevelopmentConfig()
		config.Level.SetLevel(zapcore.DebugLevel)
	} else if env == PRODUCTION {
		config = zap.NewProductionConfig()
	} else {
		return errors.New("invalid environment", env)
	}

	c.logger, err = config.Build(zap.AddCallerSkip(1))
	// TODO: Add version
	if c.HostName() != "" {
		c.logger = c.logger.With(
			log.String("serviceName", c.HostName()),
			log.String("serviceID", c.ID()),
		)
	}

	return err
}

// removeLogger removes the logger.
func (c *Connector) removeLogger() error {
	if c.logger == nil {
		return nil
	}
	err := c.logger.Sync()
	c.logger = nil
	return err
}
