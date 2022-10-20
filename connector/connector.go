package connector

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frag"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/sub"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
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

	onStartup       *cb.Callback
	onShutdown      *cb.Callback
	lifetimeCtx     context.Context
	ctxCancel       context.CancelFunc
	pendingOps      int32
	onStartupCalled bool

	natsConn        *nats.Conn
	natsResponseSub *nats.Subscription
	subs            map[string]*sub.Subscription
	subsLock        sync.Mutex
	started         bool
	plane           string

	reqs            map[string]chan *http.Response
	reqsLock        sync.Mutex
	networkHop      time.Duration
	maxCallDepth    int
	maxFragmentSize int64

	requestDefrags      map[string]*frag.DefragRequest
	requestDefragsLock  sync.Mutex
	responseDefrags     map[string]*frag.DefragResponse
	responseDefragsLock sync.Mutex

	knownResponders     map[string]map[string]bool
	knownRespondersLock sync.Mutex

	configs    map[string]*cfg.Config
	configLock sync.Mutex

	logger *zap.Logger

	clock       *clock.ClockReference
	clockSet    bool
	tickers     map[string]*tickerCallback
	tickersLock sync.Mutex
}

// NewConnector constructs a new Connector.
func NewConnector() *Connector {
	c := &Connector{
		id:              strings.ToLower(rand.AlphaNum32(10)),
		reqs:            map[string]chan *http.Response{},
		configs:         map[string]*cfg.Config{},
		networkHop:      250 * time.Millisecond,
		maxCallDepth:    64,
		subs:            map[string]*sub.Subscription{},
		knownResponders: map[string]map[string]bool{},
		requestDefrags:  map[string]*frag.DefragRequest{},
		responseDefrags: map[string]*frag.DefragResponse{},
		clock:           clock.NewClockReference(clock.New()),
		tickers:         map[string]*tickerCallback{},
		lifetimeCtx:     context.Background(),
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
		return errors.Newf("invalid host name '%s'", hostName)
	}
	c.hostName = hostName
	return nil
}

// HostName returns the host name of the microservice.
// A microservice is addressable by its host name.
func (c *Connector) HostName() string {
	return c.hostName
}

// Deployment environments
const (
	PROD  string = "PROD"  // PROD for a production environment
	LAB   string = "LAB"   // LAB for all non-production environments such as dev integration, test, staging, etc.
	LOCAL string = "LOCAL" // LOCAL when developing on the local machine or running inside a testing app
)

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
	if deployment != "" && deployment != PROD && deployment != LAB && deployment != LOCAL {
		return errors.Newf("invalid deployment '%s'", deployment)
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

// connectToNATS connects to the NATS cluster based on settings in environment variables
func (c *Connector) connectToNATS() error {
	opts := []nats.Option{}

	// Unique name to identify this connection
	opts = append(opts, nats.Name(c.id+"."+c.hostName))

	// URL
	u := os.Getenv("MICROBUS_NATS")
	if u == "" {
		u = "nats://127.0.0.1:4222"
	}

	// Credentials
	user := os.Getenv("MICROBUS_NATS_USER")
	pw := os.Getenv("MICROBUS_NATS_PASSWORD")
	token := os.Getenv("MICROBUS_NATS_TOKEN")
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
	if exists("ca.pem") {
		opts = append(opts, nats.RootCAs("ca.pem"))
	}
	if exists("cert.pem") && exists("key.pem") {
		opts = append(opts, nats.ClientCert("cert.pem", "key.pem"))
	}

	// Connect
	cn, err := nats.Connect(u, opts...)
	if err != nil {
		return errors.Trace(err, u)
	}

	// Log connection events
	ctx := c.Lifetime()
	c.LogInfo(ctx, "Connected to NATS", log.String("url", cn.ConnectedUrl()))
	cn.SetDisconnectHandler(func(n *nats.Conn) {
		c.LogInfo(ctx, "Disconnected from NATS", log.String("url", cn.ConnectedUrl()))
	})
	cn.SetReconnectHandler(func(n *nats.Conn) {
		c.LogInfo(ctx, "Reconnected to NATS", log.String("url", cn.ConnectedUrl()))
	})

	c.natsConn = cn
	return nil
}
