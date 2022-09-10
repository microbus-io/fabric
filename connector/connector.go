package connector

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/microbus-io/fabric/rand"
	"github.com/nats-io/nats.go"
)

/*
Connector is the base class of a microservice.
It provides the microservice such functions as connecting to the NATS messaging bus,
communications with other microservices, logging, config, etc.
*/
type Connector struct {
	hostName string
	id       string

	onStartUp  func(context.Context) error
	onShutDown func(context.Context) error

	natsConn *nats.Conn
}

// NewConnector constructs a new Connector.
func NewConnector() *Connector {
	c := &Connector{
		id: strings.ToLower(rand.AlphaNum32(8)),
	}
	return c
}

// ID is a unique identifier of a particular instance of the microservice
func (c *Connector) ID() string {
	return c.id
}

// SetHostName sets the host name of the microservice.
// Host names are case-insensitive. Each segment of the host name may contain letters and numbers only.
// Segments are seprated by dots.
// For example, this.is.a.valid.hostname.123.local
func (c *Connector) SetHostName(hostName string) error {
	hostName = strings.ToLower(hostName)
	match, err := regexp.MatchString(`^[a-z0-9]+(\.[a-z0-9]+)*$`, hostName)
	if err != nil {
		return err
	}
	if !match {
		return fmt.Errorf("invalid host name: %s", hostName)
	}
	c.hostName = hostName
	return nil
}

// HostName returns the host name of the microservice.
// A microservice is addressable by its host name.
func (c *Connector) HostName() string {
	return c.hostName
}

// OnStartUp sets a function to be called during the starting up of the microservice
func (c *Connector) OnStartUp(f func(context.Context) error) {
	c.onStartUp = f
}

// OnShutDown sets a function to be called during the shutting down of the microservice
func (c *Connector) OnShutDown(f func(context.Context) error) {
	c.onShutDown = f
}

// StartUp the microservice by connecting to the NATS bus and activating the subscriptions
func (c *Connector) StartUp() error {
	// Connect to NATS
	var err error
	c.natsConn, err = nats.Connect("nats://127.0.0.1:4222")
	if err != nil {
		return err
	}

	// TODO: Subscribe to reply channel

	// Call the callback function, if provided
	if c.onStartUp != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		err := c.onStartUp(ctx)
		if err != nil {
			return err
		}
	}

	// TODO: Activate subscriptions

	return nil
}

// ShutDown the microservice by deactivating subscriptions and disconnecting from the NATS bus
func (c *Connector) ShutDown() error {
	// TODO: Deactivate subscriptions

	// Call the callback function, if provided
	if c.onShutDown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		err := c.onShutDown(ctx)
		if err != nil {
			return err
		}
	}

	// TODO: Deactivate reply channel

	// TODO: Disconnect from NATS
	c.natsConn.Close()

	return nil
}

// Log a message to standard output
func (c *Connector) Log(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
