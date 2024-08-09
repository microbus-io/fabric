/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connector

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/dlru"
	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/lru"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Ensure interfaces
var (
	_ = service.Service(&Connector{})
)

/*
Connector is the base class of a microservice.
It provides the microservice such functions as connecting to the NATS messaging bus,
communications with other microservices, logging, config, etc.
*/
type Connector struct {
	hostname    string
	id          string
	deployment  string
	description string
	version     int
	locality    string

	onStartup       []service.StartupHandler
	onShutdown      []service.ShutdownHandler
	lifetimeCtx     context.Context
	ctxCancel       context.CancelFunc
	pendingOps      int32
	onStartupCalled bool
	initErr         error
	startupTime     time.Time

	metricsRegistry *prometheus.Registry
	metricsHandler  http.Handler
	metricDefs      map[string]*metric
	metricLock      sync.RWMutex

	traceProvider  *sdktrace.TracerProvider
	tracer         trace.Tracer
	traceProcessor *selectiveProcessor

	natsConn        *nats.Conn
	natsResponseSub *nats.Subscription
	subs            map[string]*sub.Subscription
	subsLock        sync.Mutex
	started         atomic.Bool
	plane           string

	reqs             utils.SyncMap[string, *transferChan]
	networkHop       time.Duration
	maxCallDepth     int
	maxFragmentSize  int64
	multicastChanCap int
	ackTimeout       time.Duration

	requestDefrags  utils.SyncMap[string, *httpx.DefragRequest]
	responseDefrags utils.SyncMap[string, *httpx.DefragResponse]

	knownResponders *lru.Cache[string, map[string]bool]
	postRequestData *lru.Cache[string, string]
	localResponder  *lru.Cache[string, string]

	configs         map[string]*cfg.Config
	configLock      sync.Mutex
	onConfigChanged []service.ConfigChangedHandler

	logger   *slog.Logger
	logDebug bool

	tickers     map[string]*tickerCallback
	tickersLock sync.Mutex

	distribCache *dlru.Cache
	resourcesFS  service.FS
	stringBundle map[string]map[string]string
}

// NewConnector constructs a new Connector.
func NewConnector() *Connector {
	c := &Connector{
		id:               strings.ToLower(rand.AlphaNum32(10)),
		configs:          map[string]*cfg.Config{},
		networkHop:       250 * time.Millisecond,
		ackTimeout:       250 * time.Millisecond,
		maxCallDepth:     64,
		subs:             map[string]*sub.Subscription{},
		tickers:          map[string]*tickerCallback{},
		lifetimeCtx:      context.Background(),
		knownResponders:  lru.NewCache[string, map[string]bool](),
		postRequestData:  lru.NewCache[string, string](),
		localResponder:   lru.NewCache[string, string](),
		multicastChanCap: 32,
		metricDefs:       map[string]*metric{},
	}

	c.SetResFSDir(".")
	c.knownResponders.SetMaxWeight(16 * 1024)
	c.knownResponders.SetMaxAge(24 * time.Hour)
	c.postRequestData.SetMaxWeight(256 * 1024)
	c.postRequestData.SetMaxAge(time.Minute)
	c.localResponder.SetMaxWeight(16 * 1024)
	c.localResponder.SetMaxAge(24 * time.Hour)

	c.newMetricsRegistry()

	return c
}

// New constructs a new Connector with the given hostname.
func New(hostname string) *Connector {
	c := NewConnector()
	c.SetHostname(hostname)
	return c
}

// ID is a unique identifier of a particular instance of the microservice
func (c *Connector) ID() string {
	return c.id
}

// SetHostname sets the hostname of the microservice.
// Hostnames are case-insensitive. Each segment of the hostname may contain letters, numbers, hyphens or underscores only.
// Segments are separated by dots.
// For example, this.is.a.valid.host-name.123.local
func (c *Connector) SetHostname(hostname string) error {
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	hostname = strings.TrimSpace(hostname)
	if err := utils.ValidateHostname(hostname); err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	hn := strings.ToLower(hostname)
	if hn == "all" || strings.HasSuffix(hn, ".all") {
		// The hostname "all" is reserved to refer to all microservices
		return c.captureInitErr(errors.Newf("disallowed hostname '%s'", hostname))
	}
	c.hostname = hostname
	return nil
}

// Hostname returns the hostname of the microservice.
// A microservice is addressable by its hostname.
func (c *Connector) Hostname() string {
	return c.hostname
}

// SetDescription sets a human-friendly description of the microservice.
func (c *Connector) SetDescription(description string) error {
	c.description = description
	return nil
}

// Description returns the human-friendly description of the microservice.
func (c *Connector) Description() string {
	return c.description
}

// SetVersion sets the sequential version number of the microservice.
func (c *Connector) SetVersion(version int) error {
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	if version < 0 {
		return c.captureInitErr(errors.Newf("negative version '%d'", version))
	}
	c.version = version
	return nil
}

// Version is the sequential version number of the microservice.
func (c *Connector) Version() int {
	return c.version
}

// Deployment environments
const (
	PROD    string = "PROD"    // PROD for a production environment
	LAB     string = "LAB"     // LAB for all non-production environments such as dev integration, test, staging, etc.
	LOCAL   string = "LOCAL"   // LOCAL when developing on the local machine
	TESTING string = "TESTING" // TESTING when running inside a testing app
)

// Deployment indicates what deployment environment the microservice is running in:
// PROD for a production environment;
// LAB for all non-production environments such as dev integration, test, staging, etc.;
// LOCAL when developing on the local machine;
// TESTING when running inside a testing app.
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
// LOCAL when developing on the local machine;
// TESTING when running inside a testing app.
func (c *Connector) SetDeployment(deployment string) error {
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	deployment = strings.ToUpper(deployment)
	if deployment != "" && deployment != PROD && deployment != LAB && deployment != LOCAL && deployment != TESTING {
		return c.captureInitErr(errors.Newf("invalid deployment '%s'", deployment))
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
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	if match, _ := regexp.MatchString(`^[0-9a-zA-Z]*$`, plane); !match {
		return c.captureInitErr(errors.Newf("invalid plane: %s", plane))
	}
	c.plane = plane
	return nil
}

// SetLocality sets the geographic locality of the microservice which is used to optimize routing.
// Localities are hierarchical with the more specific identifiers first, separated by dots.
// It can be set to correlate to AWS regions such as az1.dc2.west.us, or arbitrarily to rome.italy.europe for example.
// Localities are case-insensitive. Each segment of the hostname may contain letters, numbers, hyphens or underscores only.
func (c *Connector) SetLocality(locality string) error {
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	locality = strings.TrimSpace(locality)
	if err := utils.ValidateHostname(locality); err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	c.locality = strings.ToLower(locality)
	return nil
}

// Locality returns the geographic locality of the microservice.
func (c *Connector) Locality() string {
	return c.locality
}

// connectToNATS connects to the NATS cluster based on settings in environment variables
func (c *Connector) connectToNATS(ctx context.Context) error {
	opts := []nats.Option{}

	// Unique name to identify this connection
	opts = append(opts, nats.Name(c.id+"."+c.hostname))

	// URL
	u := env.Get("MICROBUS_NATS")
	if u == "" {
		u = "nats://127.0.0.1:4222"
	}

	// Credentials
	user := env.Get("MICROBUS_NATS_USER")
	pw := env.Get("MICROBUS_NATS_PASSWORD")
	token := env.Get("MICROBUS_NATS_TOKEN")
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
		return errors.Trace(err)
	}

	// Log connection events
	natsURL := cn.ConnectedUrl()
	natsServerID := cn.ConnectedServerId()
	c.LogInfo(ctx, "Connected to NATS",
		"url", natsURL,
		"server", natsServerID,
	)
	cn.SetDisconnectErrHandler(func(cn *nats.Conn, err error) {
		c.LogInfo(c.lifetimeCtx, "Disconnected from NATS",
			"url", natsURL,
			"server", natsServerID,
		)
	})
	cn.SetReconnectHandler(func(cn *nats.Conn) {
		natsURL = cn.ConnectedUrl()
		natsServerID = cn.ConnectedServerId()
		c.LogInfo(c.lifetimeCtx, "Reconnected to NATS",
			"url", natsURL,
			"server", natsServerID,
		)
	})

	c.natsConn = cn
	return nil
}

// DistribCache is a cache that stores data among all peers of the microservice.
// By default the cache is limited to 32MB per peer and a 1 hour TTL.
//
// Operating on a distributed cache is slower than on a local cache because
// it involves network communication among peers.
// However, the memory capacity of a distributed cache scales linearly with the number of peers
// and its content is often able to survive a restart of the microservice.
//
// Cache elements can get evicted for various reason and without warning.
// Cache only that which you can afford to lose and reconstruct.
// Do not use the cache to share state among peers.
// The cache is subject to race conditions in rare situations.
func (c *Connector) DistribCache() *dlru.Cache {
	return c.distribCache
}
