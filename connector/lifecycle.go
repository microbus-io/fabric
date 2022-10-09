package connector

import (
	"context"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
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
	c.logConfigs()

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

	// Deployment default
	if c.deployment == "" {
		if deployment, ok := c.Config("Deployment"); ok {
			err := c.SetDeployment(deployment)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.deployment == "" {
			c.deployment = "LOCAL"
			if nats, ok := c.Config("NATS"); ok {
				if !strings.Contains(nats, "/127.0.0.1:") &&
					!strings.Contains(nats, "/0.0.0.0:") &&
					!strings.Contains(nats, "/localhost:") {
					c.deployment = "PROD"
				}
			}
		}
	}

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
		ctx, cancel := context.WithTimeout(context.Background(), c.callbackTimeout)
		defer cancel()
		err := catchPanic(func() error {
			return c.onStartup(ctx)
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
	var lastErr error
	if !c.started {
		return errors.New("not started")
	}
	c.started = false

	// Unsubscribe all handlers
	err := c.UnsubscribeAll()
	if err != nil {
		lastErr = errors.Trace(err)
		c.LogError(err)
	}

	// Call the callback function, if provided
	if c.onShutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), c.callbackTimeout)
		defer cancel()
		err := catchPanic(func() error {
			return c.onShutdown(ctx)
		})
		if err != nil {
			lastErr = errors.Trace(err)
			c.LogError(err)
		}
	}

	// Unsubscribe from the response subject
	if c.natsResponseSub != nil {
		err := c.natsResponseSub.Unsubscribe()
		if err != nil {
			lastErr = errors.Trace(err)
			c.LogError(err)
		}
		c.natsResponseSub = nil
	}

	// Disconnect from NATS
	if c.natsConn != nil {
		c.natsConn.Close()
		c.natsConn = nil
	}

	return lastErr
}

// IsStarted indicates if the microservice has been successfully started
func (c *Connector) IsStarted() bool {
	return c.started
}
