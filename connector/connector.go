/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"crypto/ed25519"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/dlru"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/lru"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/transport"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/seamster"
	"golang.org/x/sync/singleflight"

	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
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

	onStartup       service.StartupHandler
	onShutdown      service.ShutdownHandler
	lifetimeCtx     context.Context
	ctxCancel       context.CancelFunc
	pendingOps      atomic.Int32
	onStartupCalled bool
	initErr         error
	startupTime     time.Time

	metricsHandler    http.Handler
	metricLock        sync.RWMutex
	meterProvider     *sdkmetric.MeterProvider
	meter             metric.Meter
	metricInstruments map[string]*metricInstrument
	onObserveMetrics  service.ObserveMetricsHandler
	meterOTLPKey      string

	traceProvider  *sdktrace.TracerProvider
	tracer         trace.Tracer
	traceProcessor *selectiveProcessor
	traceOTLPKey   string

	transportConn transport.Conn
	responseSub   *transport.Subscription
	subs          utils.SyncMap[string, *sub.Subscription]
	phase         atomic.Int32
	plane         string
	controlSubs   bool

	reqs              utils.SyncMap[string, *transferChan]
	networkRoundtrip  time.Duration
	defaultTimeBudget time.Duration
	maxTimeBudget     time.Duration
	maxCallDepth      int
	maxFragmentSize   int64
	multicastChanCap  int
	ackTimeout        time.Duration

	requestDefrags  *defragCache[*httpx.DefragRequest]
	responseDefrags *defragCache[*httpx.DefragResponse]

	knownResponders *lru.Cache[string, map[string]bool]
	postRequestData *lru.Cache[string, string]
	localResponder  *lru.Cache[string, string]

	configs         map[string]*cfg.Config
	configLock      sync.Mutex
	onConfigChanged service.ConfigChangedHandler

	// logger is atomic because request handlers read it concurrently with Shutdown's termLogger, including
	// via in-process short-circuit deliveries from a peer connector that is shutting down at the same time.
	logger      atomic.Pointer[slog.Logger]
	logDebug    bool
	logProvider *sdklog.LoggerProvider
	logOTLPKey  string

	tickers     map[string]*tickerCallback
	tickersLock sync.Mutex

	distribCache *dlru.Cache
	resourcesFS  fs.FS
	stringBundle map[string]map[string]string

	actorKeysLock sync.RWMutex
	actorKeys     map[string]map[string]ed25519.PublicKey
	lastJWKSFetch map[string]time.Time
	jwksFlight    singleflight.Group

	underTest     bool
	underTestName string
	seams         *seamster.Seamster
}

// NewConnector constructs a new Connector.
func NewConnector() *Connector {
	c := &Connector{
		id:                idPrefix + strings.ToLower(utils.RandomIdentifier(10)),
		configs:           map[string]*cfg.Config{},
		networkRoundtrip:  300 * time.Millisecond,
		defaultTimeBudget: 20 * time.Second,
		maxTimeBudget:     15 * time.Minute,
		ackTimeout:        300 * time.Millisecond,
		maxCallDepth:      64,
		tickers:           map[string]*tickerCallback{},
		lifetimeCtx:       context.Background(),
		knownResponders:   lru.New[string, map[string]bool](64<<10, 24*time.Hour), // 64KB
		postRequestData:   lru.New[string, string](256<<10, time.Minute),          // 256KB
		localResponder:    lru.New[string, string](64<<10, 24*time.Hour),          // 64KB
		multicastChanCap:  32,
		metricInstruments: map[string]*metricInstrument{},
		requestDefrags:    newDefragCache[*httpx.DefragRequest](),
		responseDefrags:   newDefragCache[*httpx.DefragResponse](),
		maxFragmentSize:   1 << 20, // 1MB
	}
	c.underTestName, c.underTest = utils.Testing()
	c.seams = seamster.New(c.underTest)
	c.SetResFSDir(".")
	return c
}

// New constructs a new Connector with the given hostname.
func New(hostname string) *Connector {
	c := NewConnector()
	c.SetHostname(hostname)
	return c
}

// Init enables a single-statement pattern for initializing the connector.
func (c *Connector) Init(initializer func(c *Connector) (err error)) *Connector {
	c.captureInitErr(initializer(c))
	return c
}

// ID is a unique identifier of a particular instance of the microservice
func (c *Connector) ID() string {
	return c.id
}

// SetHostname sets the hostname of the microservice.
// The hostname must be a canonical Microbus identity per [httpx.ValidateHostname]:
// lowercase letters, digits, dots, and hyphens; no underscores, no "id-" or "loc-"
// first segment, not "all" or "*.all", no leading/trailing whitespace.
// For example, this.is.a.valid.host-name.123.local
func (c *Connector) SetHostname(hostname string) error {
	if !c.isPhase(shutDown) {
		return c.captureInitErr(errors.New("already started"))
	}
	if err := httpx.ValidateHostname(hostname); err != nil {
		return c.captureInitErr(errors.Trace(err))
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
	if !c.isPhase(shutDown, startingUp) {
		return c.captureInitErr(errors.New("already started"))
	}
	if version < 0 {
		return c.captureInitErr(errors.New("negative version '%d'", version))
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
	if !c.isPhase(shutDown, startingUp) {
		return c.captureInitErr(errors.New("already started"))
	}
	deployment = strings.ToUpper(deployment)
	if deployment != "" && deployment != PROD && deployment != LAB && deployment != LOCAL && deployment != TESTING {
		return c.captureInitErr(errors.New("invalid deployment '%s'", deployment))
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
	if !c.isPhase(shutDown, startingUp) {
		return c.captureInitErr(errors.New("already started"))
	}
	if match, _ := regexp.MatchString(`^[0-9a-zA-Z]*$`, plane); !match {
		return c.captureInitErr(errors.New("invalid plane: %s", plane))
	}
	c.plane = plane
	return nil
}

// SetLocality sets the geographic locality of the microservice which is used to optimize routing.
// Localities are hierarchical with the broadest identifier first, separated by hyphens, similar to AWS region/AZ
// identifiers such as "us-west-b-1" or arbitrarily "europe-italy-rome".
// DNS-style dot notation with the most specific identifier first is also accepted: "1.b.west.us" is equivalent
// to "us-west-b-1".
// Localities are case-insensitive. Letters, numbers, hyphens and underscores are allowed.
// The special values "AWS" or "GCP" can be set to determine the locality automatically from the cloud provider's meta-data servers.
func (c *Connector) SetLocality(locality string) error {
	if !c.isPhase(shutDown, startingUp) {
		return c.captureInitErr(errors.New("already started"))
	}
	if strings.Contains(locality, ".") {
		// DNS-style dot notation: reverse segment order so the broadest identifier comes first,
		// then join with hyphens. This is the canonical hyphen-form that gets validated and stored.
		parts := strings.Split(locality, ".")
		for i := range len(parts) / 2 {
			parts[i], parts[len(parts)-1-i] = parts[len(parts)-1-i], parts[i]
		}
		locality = strings.Join(parts, "-")
	}
	if err := httpx.ValidateHostname(locality); err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	c.locality = locality
	return nil
}

// Locality returns the geographic locality of the microservice.
func (c *Connector) Locality() string {
	return c.locality
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

/*
ExternalizeURL converts an internal Microbus URL to an external one using the X-Forwarded-* headers from the context's frame.
The internal Microbus URL is assumed relative to the hostname of this connector.
Relative URLs are returned as is.

Examples:
  - https://my.host/my/path -> https://localhost:8080/my.host/my/path
  - //other.host:123/other/path -> https://localhost:8080/other.host:123/other/path
  - /my/path -> https://localhost:8080/my.host/my/path
  - ../my-other/path -> ../my-other/path
*/
func (c *Connector) ExternalizeURL(ctx context.Context, internalURL string) string {
	xf := frame.Of(ctx).XForwardedBaseURL() + "/"
	if i := strings.Index(internalURL, "://"); i >= 0 {
		return xf + internalURL[i+3:]
	}
	if strings.HasPrefix(internalURL, "//") {
		return xf + internalURL[2:]
	}
	if strings.HasPrefix(internalURL, "/") {
		return xf + c.hostname + internalURL
	}
	return internalURL
}
