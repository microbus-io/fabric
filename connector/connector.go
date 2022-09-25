package connector

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/fabric/errors"
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

	onStartup       func(context.Context) error
	onShutdown      func(context.Context) error
	callbackTimeout time.Duration

	natsConn     *nats.Conn
	natsReplySub *nats.Subscription
	subs         []*subscription
	subsLock     sync.Mutex
	started      bool

	reqs         map[string]chan *http.Response
	reqsLock     sync.Mutex
	networkHop   time.Duration
	maxCallDepth int

	configs    map[string]*config
	configLock sync.Mutex
}

// NewConnector constructs a new Connector.
func NewConnector() *Connector {
	c := &Connector{
		id:              strings.ToLower(rand.AlphaNum32(10)),
		reqs:            map[string]chan *http.Response{},
		configs:         map[string]*config{},
		networkHop:      250 * time.Millisecond,
		maxCallDepth:    64,
		callbackTimeout: time.Minute,
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
		return errors.Trace(err)
	}
	if hostName == "all" || strings.HasSuffix(hostName, ".all") {
		// The hostname "all" is reserved to refer to all microservices
		match = false
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

// catchPanic calls the function and returns any panic as a standard error
func catchPanic(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
			err = errors.TraceUp(err, 2)
		}
	}()
	err = f()
	return
}
