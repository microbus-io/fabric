package connector

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

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

	onStartup  func(context.Context) error
	onShutdown func(context.Context) error

	natsConn     *nats.Conn
	natsReplySub *nats.Subscription
	subs         []*subscription
	subsLock     sync.Mutex
	started      bool

	reqs     map[string]chan *http.Response
	reqsLock sync.Mutex

	configs map[string]string
}

// NewConnector constructs a new Connector.
func NewConnector() *Connector {
	c := &Connector{
		id:      strings.ToLower(rand.AlphaNum32(8)),
		reqs:    map[string]chan *http.Response{},
		configs: map[string]string{},
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
	hostName = strings.TrimSpace(strings.ToLower(hostName))
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
