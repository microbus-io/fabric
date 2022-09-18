package connector

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
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
	c.started = true

	// Look for configs in the environment or file system
	err = c.loadConfigs()
	if err != nil {
		return err
	}
	c.logConfigs()

	// Connect to NATS
	c.natsConn, err = nats.Connect("nats://127.0.0.1:4222")
	if err != nil {
		return err
	}

	// Subscribe to the reply subject
	c.natsReplySub, err = c.natsConn.QueueSubscribe(subjectOfReply(c.hostName, c.id), c.id, c.onReply)
	if err != nil {
		c.natsConn.Close()
		c.natsConn = nil
		return err
	}

	// Call the callback function, if provided
	if c.onStartup != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		err := c.onStartup(ctx)
		if err != nil {
			_ = c.Shutdown()
			return err
		}
	}

	// Activate subscriptions
	for _, sub := range c.subs {
		err = c.activateSub(sub)
		if err != nil {
			_ = c.Shutdown()
			return err
		}
	}
	time.Sleep(20 * time.Millisecond) // Give time for subscription activation by NATS

	return nil
}

// Shutdown the microservice by deactivating subscriptions and disconnecting from the NATS bus
func (c *Connector) Shutdown() error {
	var err error
	if !c.started {
		return errors.New("not started")
	}
	c.started = false

	// Deactivate subscriptions
	for _, sub := range c.subs {
		if sub.natsSubscription != nil {
			err = sub.natsSubscription.Unsubscribe()
			if err != nil {
				c.LogError(err)
			}
			sub.natsSubscription = nil
		}
	}

	// Call the callback function, if provided
	if c.onShutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		err = c.onShutdown(ctx)
		if err != nil {
			c.LogError(err)
		}
	}

	// Unsubscribe from the reply subject
	if c.natsReplySub != nil {
		err = c.natsReplySub.Unsubscribe()
		if err != nil {
			c.LogError(err)
		}
		c.natsReplySub = nil
	}

	// Disconnect from NATS
	if c.natsConn != nil {
		c.natsConn.Close()
		c.natsConn = nil
	}

	return err
}

// IsStarted indicates if the microservice has been successfully started
func (c *Connector) IsStarted() bool {
	return c.started
}
