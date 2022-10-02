package connector

import (
	"context"
	"os"
	"regexp"
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
		envar := os.Getenv("MICROBUS_PLANE")
		if match, _ := regexp.MatchString(`^[0-9a-zA-Z]*$`, envar); !match {
			return errors.New("invalid plane specified by MICROBUS_PLANE envar: %s", envar)
		}
		c.plane = envar
		if c.plane == "" {
			c.plane = "microbus"
		}
	}

	// Connect to NATS
	err = c.connectToNATS()
	if err != nil {
		return errors.Trace(err)
	}
	c.started = true

	// Deployment default
	if c.deployment == "" {
		envar := strings.ToUpper(os.Getenv("MICROBUS_DEPLOYMENT"))
		if envar == "PROD" || envar == "LAB" || envar == "LOCAL" {
			c.deployment = envar
		}
	}
	if c.deployment == "" {
		u := c.natsConn.ConnectedUrl()
		if strings.Contains(u, "/127.0.0.1:") ||
			strings.Contains(u, "/0.0.0.0:") ||
			strings.Contains(u, "/localhost:") {
			c.deployment = "LOCAL"
		} else {
			c.deployment = "PROD"
		}
	}

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
				returnErr = err
				c.LogError(err)
			}
			sub.NATSSub = nil
		}
	}

	// Call the callback function, if provided
	if c.onShutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), c.callbackTimeout)
		defer cancel()
		err := catchPanic(func() error {
			return c.onShutdown(ctx)
		})
		if err != nil {
			returnErr = err
			c.LogError(err)
		}
	}

	// Unsubscribe from the reply subject
	if c.natsReplySub != nil {
		err := c.natsReplySub.Unsubscribe()
		if err != nil {
			returnErr = err
			c.LogError(err)
		}
		c.natsReplySub = nil
	}

	// Disconnect from NATS
	if c.natsConn != nil {
		c.natsConn.Close()
		c.natsConn = nil
	}

	return errors.Trace(returnErr)
}

// IsStarted indicates if the microservice has been successfully started
func (c *Connector) IsStarted() bool {
	return c.started
}
