package service

import (
	"context"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
)

// Service is the interface that all microservices must support.
// It is implemented by the connector.
type Service interface {
	ID() string
	HostName() string
	SetHostName(hostName string) error
	Deployment() string
	SetDeployment(deployment string) error
	Plane() string
	SetPlane(plane string) error
	Description() string
	SetDescription(description string) error

	Lifer
	Configurable
	Clocker
	Logger
	Publisher
	Subscriber
	Ticker
}

// Lifer is the interface that the connector supports for its lifecycle.
type Lifer interface {
	Startup() error
	Shutdown() error
	IsStarted() bool
	SetOnStartup(handler StartupHandler, options ...cb.Option) error
	SetOnShutdown(handler ShutdownHandler, options ...cb.Option) error
	Lifetime() context.Context
}

// Logger is the interface that the connector supports for logging.
type Logger interface {
	LogDebug(ctx context.Context, msg string, fields ...log.Field)
	LogInfo(ctx context.Context, msg string, fields ...log.Field)
	LogWarn(ctx context.Context, msg string, fields ...log.Field)
	LogError(ctx context.Context, msg string, fields ...log.Field)
}

// Publisher is the interface that the connector supports for making outgoing requests.
type Publisher interface {
	GET(ctx context.Context, url string) (*http.Response, error)
	POST(ctx context.Context, url string, body any) (*http.Response, error)
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
}

// Subscriber is the interface that the connector supports for subscribing to handle incoming requests.
type Subscriber interface {
	Subscribe(path string, handler HTTPHandler, options ...sub.Option) error
	Unsubscribe(path string) error
	UnsubscribeAll() error
}

// Clocker is the interface that the connector supports for mocking time.
type Clocker interface {
	Now() time.Time
	Clock() clock.Clock
	SetClock(newClock clock.Clock) error
}

// Configurable is the interface that the connector supports for configuration.
type Configurable interface {
	DefineConfig(name string, options ...cfg.Option) error
	Config(name string) (value string)
	InitConfig(name string, value string) error
	SetOnConfigChanged(handler ConfigChangedHandler, options ...cb.Option) error
}

// Ticker is the interface that the connector supports for managing tickers.
type Ticker interface {
	StartTicker(name string, interval time.Duration, handler TickerHandler, options ...cb.Option) error
	StopTicker(name string) error
	StopAllTickers() error
}
