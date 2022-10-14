package connector

import (
	"context"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/utils"
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

	if c.started {
		return errors.New("already started")
	}
	if c.hostName == "" {
		return errors.New("no hostname")
	}

	// Look for configs in the environment or file system
	err := c.loadConfigs()
	if err != nil {
		return errors.Trace(err)
	}

	// Determine the communication plane
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

	// Identify the environment deployment
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

	// Log configs
	c.logConfigs(ctx)

	// Subscribe to :888 control messages
	err = c.subscribeControl()
	if err != nil {
		return errors.Trace(err)
	}

	// Connect to NATS
	err = c.connectToNATS()
	if err != nil {
		return errors.Trace(err)
	}
	c.started = true
	c.maxFragmentSize = c.natsConn.MaxPayload() - 32*1024 // Up to 32K for headers
	if c.maxFragmentSize < 32*1024 {
		c.natsConn.Close()
		c.natsConn = nil
		c.started = false
		return errors.New("message size limit is too restrictive")
	}

	// Subscribe to the response subject
	c.natsResponseSub, err = c.natsConn.QueueSubscribe(subjectOfResponses(c.plane, c.hostName, c.id), c.id, c.onResponse)
	if err != nil {
		c.natsConn.Close()
		c.natsConn = nil
		c.started = false
		return errors.Trace(err)
	}

	// Call the callback function, if provided
	if c.onStartup != nil {
		callbackCtx, cancel := context.WithTimeout(ctx, c.callbackTimeout)
		defer cancel()
		err := utils.CatchPanic(func() error {
			return c.onStartup(callbackCtx)
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
	if !c.started {
		return errors.New("not started")
	}
	c.started = false

	ctx := context.Background()
	var lastErr error

	// Unsubscribe all handlers
	err := c.UnsubscribeAll()
	if err != nil {
		lastErr = errors.Trace(err)
		c.LogError(
			ctx,
			"Deactivating subscriptions",
			log.Error(err),
		)
	}

	// Call the callback function, if provided
	if c.onShutdown != nil {
		callbackCtx, cancel := context.WithTimeout(ctx, c.callbackTimeout)
		defer cancel()
		err := utils.CatchPanic(func() error {
			return c.onShutdown(callbackCtx)
		})
		if err != nil {
			lastErr = errors.Trace(err)
			c.LogError(ctx, "Shutdown callback", log.Error(err))
		}
	}

	// Unsubscribe from the response subject
	if c.natsResponseSub != nil {
		err := c.natsResponseSub.Unsubscribe()
		if err != nil {
			lastErr = errors.Trace(err)
			c.LogError(ctx, "Unsubscribing response sub", log.Error(err))
		}
		c.natsResponseSub = nil
	}

	// Disconnect from NATS
	if c.natsConn != nil {
		c.natsConn.Close()
		c.natsConn = nil
	}

	// Terminate logger
	_ = c.terminateLogger()
	// No point trying to log the error at this point

	return lastErr
}

// IsStarted indicates if the microservice has been successfully started
func (c *Connector) IsStarted() bool {
	return c.started
}
