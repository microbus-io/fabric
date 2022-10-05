package connector

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/sub"
	"github.com/nats-io/nats.go"
)

/*
Connector is the base class of a microservice.
It provides the microservice such functions as connecting to the NATS messaging bus,
communications with other microservices, logging, config, etc.
*/
type Connector struct {
	hostName   string
	id         string
	deployment string

	onStartup       func(context.Context) error
	onShutdown      func(context.Context) error
	callbackTimeout time.Duration

	natsConn     *nats.Conn
	natsReplySub *nats.Subscription
	subs         map[string]*sub.Subscription
	subsLock     sync.Mutex
	started      bool
	plane        string

	reqs              map[string]chan *http.Response
	reqsLock          sync.Mutex
	networkHop        time.Duration
	maxCallDepth      int
	defaultTimeBudget time.Duration

	knownResponders     map[string]map[string]bool
	knownRespondersLock sync.Mutex

	configs    map[string]*config
	configLock sync.Mutex
}

// NewConnector constructs a new Connector.
func NewConnector() *Connector {
	c := &Connector{
		id:                strings.ToLower(rand.AlphaNum32(10)),
		reqs:              map[string]chan *http.Response{},
		configs:           map[string]*config{},
		networkHop:        250 * time.Millisecond,
		maxCallDepth:      64,
		callbackTimeout:   time.Minute,
		defaultTimeBudget: 20 * time.Second,
		subs:              map[string]*sub.Subscription{},
		knownResponders:   map[string]map[string]bool{},
	}
	return c
}

// ID is a unique identifier of a particular instance of the microservice
func (c *Connector) ID() string {
	return c.id
}

// SetHostName sets the host name of the microservice.
// Host names are case-insensitive. Each segment of the host name may contain letters and numbers only.
// Segments are separated by dots.
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
		return errors.Newf("invalid host name: %s", hostName)
	}
	c.hostName = hostName
	return nil
}

// HostName returns the host name of the microservice.
// A microservice is addressable by its host name.
func (c *Connector) HostName() string {
	return c.hostName
}

// Deployment indicates what deployment environment the microservice is running in:
// PROD for a production environment;
// LAB for all non-production environments such as dev integration, test, staging, etc.;
// LOCAL when developing on the local machine or running unit tests
func (c *Connector) Deployment() string {
	return c.deployment
}

// SetDeployment sets what deployment environment the microservice is running in.
// Explicitly setting a deployment will override any value specified by the Deployment config property.
// Setting an empty value will clear this override.
//
// Valid values are:
// PROD for a production environment;
// LAB for all non-production environments such as dev integration, test, staging, etc.;
// LOCAL when developing on the local machine or running unit tests
func (c *Connector) SetDeployment(deployment string) error {
	if c.started {
		return errors.New("already started")
	}
	deployment = strings.ToUpper(deployment)
	if deployment != "" && deployment != "PROD" && deployment != "LAB" && deployment != "LOCAL" {
		return errors.Newf("invalid deployment: %s", deployment)
	}
	c.deployment = deployment
	return nil
}

// Plane is a unique prefix set for all communications sent or received by this microservice.
// It is used to isolate communication among a group of microservices over a NATS cluster
// that is shared with other microservices.
// If not explicitly set, the value is pulled from the Plane config, or the default "microbus" is used
func (c *Connector) Plane() string {
	return c.plane
}

// SetPlane sets a unique prefix for all communications sent or received by this microservice.
// A plane is used to isolate communication among a group of microservices over a NATS cluster
// that is shared with other microservices.
// Explicitly setting a plane overrides any value specified by the Plane config.
// The plane can only contain alphanumeric case-sensitive characters.
// Setting an empty value will clear this override
func (c *Connector) SetPlane(plane string) error {
	if c.started {
		return errors.New("already started")
	}
	if match, _ := regexp.MatchString(`^[0-9a-zA-Z]*$`, plane); !match {
		return errors.New("invalid plane: %s", plane)
	}
	c.plane = plane
	return nil
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

// connectToNATS connects to the NATS cluster based on settings in environment variables
func (c *Connector) connectToNATS() error {
	opts := []nats.Option{}

	// Unique name to identify this connection
	opts = append(opts, nats.Name(c.id+"."+c.hostName))

	// URL
	u, _ := c.Config("NATS")
	if u == "" {
		u = "nats://127.0.0.1:4222"
	}

	// Credentials
	user, _ := c.Config("NATSUser")
	pw, _ := c.Config("NATSPassword")
	token, _ := c.Config("NATSToken")
	if user != "" && pw != "" {
		opts = append(opts, nats.UserInfo(user, pw))
	}
	if token != "" {
		opts = append(opts, nats.Token(token))
	}

	// Root CA and client certs
	exists := func(fileName string) bool {
		_, err := os.Stat(fileName)
		return err == nil
	}
	hostSegments := strings.Split(c.hostName, ".")
	var foundCA, foundCertKey bool
	for i := 0; i <= len(hostSegments); i++ {
		host := strings.Join(hostSegments[i:], ".")
		if host == "" {
			host = "all"
		}
		host += "-"
		if !foundCA && exists(host+"ca.pem") {
			opts = append(opts, nats.RootCAs(host+"ca.pem"))
			foundCA = true
		}
		if !foundCertKey && exists(host+"cert.pem") && exists(host+"key.pem") {
			opts = append(opts, nats.ClientCert(host+"cert.pem", host+"key.pem"))
			foundCertKey = true
		}
	}

	// Connect
	cn, err := nats.Connect(u, opts...)
	if err != nil {
		return errors.Trace(err, u)
	}

	// Log connection events
	c.LogInfo("Connected to NATS at %s", cn.ConnectedUrl())
	cn.SetDisconnectHandler(func(n *nats.Conn) {
		c.LogInfo("Disconnected from NATS at %s", cn.ConnectedUrl())
	})
	cn.SetReconnectHandler(func(n *nats.Conn) {
		c.LogInfo("Reconnected to NATS at %s", cn.ConnectedUrl())
	})

	c.natsConn = cn
	return nil
}
