package connector

import (
	"context"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
)

// SetOnStartup sets a function to be called during the starting up of the microservice
func (c *Connector) SetOnStartup(f func(context.Context) error) {
	c.onStartup = f
}

// SetOnShutdown sets a function to be called during the shutting down of the microservice
func (c *Connector) SetOnShutdown(f func(context.Context) error) {
	c.onShutdown = f
}

// Startup the microservice by connecting to the NATS bus and activating the subscriptions
func (c *Connector) Startup() error {
	ctx := context.Background()

	var err error

	if c.started {
		return errors.New("already started")
	}
	if c.hostName == "" {
		return errors.New("no hostname")
	}

	// Look for configs in the environment or file system
	err = c.loadConfigs()
	if err != nil {
		return errors.Trace(err)
	}

	// Communication plane default
	if c.plane == "" {
		if plane, ok := c.Config("Plane"); ok {
			err := c.SetPlane(plane)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.plane == "" {
			c.plane = "microbus"
		}
	}

	// Load environment deployment config or use default
	if c.deployment == "" {
		if deployment, ok := c.Config("Deployment"); ok {
			err := c.SetDeployment(deployment)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.deployment == "" {
			c.deployment = LOCAL
			if nats, ok := c.Config("NATS"); ok {
				if !strings.Contains(nats, "/127.0.0.1:") &&
					!strings.Contains(nats, "/0.0.0.0:") &&
					!strings.Contains(nats, "/localhost:") {
					c.deployment = PROD
				}
			}
		}
	}

	// Initialize logger
	err = c.initLogger()
	if err != nil {
		return errors.Trace(err)
	}

	c.logConfigs(ctx)

	// Connect to NATS
	err = c.connectToNATS(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	c.started = true

	// Subscribe to the reply subject
	c.natsReplySub, err = c.natsConn.QueueSubscribe(subjectOfReply(c.plane, c.hostName, c.id), c.id, c.onReply)
	if err != nil {
		c.natsConn.Close()
		c.natsConn = nil
		c.started = false
		return errors.Trace(err)
	}

	// Call the callback function, if provided
	if c.onStartup != nil {
		subCtx, cancel := context.WithTimeout(ctx, c.callbackTimeout)
		defer cancel()
		err := catchPanic(func() error {
			return c.onStartup(subCtx)
		})
		if err != nil {
			_ = c.Shutdown()
			return errors.Trace(err)
		}
	}

	// Activate subscriptions
	for _, sub := range c.subs {
		err = c.activateSub(sub)
		if err != nil {
			_ = c.Shutdown()
			return errors.Trace(err)
		}
	}
	time.Sleep(20 * time.Millisecond) // Give time for subscription activation by NATS

	return nil
}

// Shutdown the microservice by deactivating subscriptions and disconnecting from the NATS bus
func (c *Connector) Shutdown() error {
	ctx := context.Background()
	var returnErr error
	if !c.started {
		return errors.New("not started")
	}
	c.started = false

	// Deactivate subscriptions
	for _, sub := range c.subs {
		if sub.NATSSub != nil {
			err := sub.NATSSub.Unsubscribe()
			if err != nil {
				returnErr = errors.Trace(err)
				c.LogError(
					ctx,
					"Failed to deactivate subscription",
					err,
					log.Any("sub", sub),
				)
			}
			sub.NATSSub = nil
		}
	}

	// Call the callback function, if provided
	if c.onShutdown != nil {
		subCtx, cancel := context.WithTimeout(ctx, c.callbackTimeout)
		defer cancel()
		err := catchPanic(func() error {
			return c.onShutdown(subCtx)
		})
		if err != nil {
			returnErr = errors.Trace(err)
			c.LogError(ctx, "Failed onShutdown", err)
		}
	}

	// Unsubscribe from the reply subject
	if c.natsReplySub != nil {
		err := c.natsReplySub.Unsubscribe()
		if err != nil {
			returnErr = errors.Trace(err)
			c.LogError(
				ctx,
				"Failed to unsubscribe from the reply subject",
				err,
			)
		}
		c.natsReplySub = nil
	}

	// Remove logger
	err := c.removeLogger()
	if err != nil {
		c.LogError(
			ctx,
			"Failed to remove logger",
			err,
		)
	}

	// Disconnect from NATS
	if c.natsConn != nil {
		c.natsConn.Close()
		c.natsConn = nil
	}

	return returnErr
}

// IsStarted indicates if the microservice has been successfully started
func (c *Connector) IsStarted() bool {
	return c.started
}
