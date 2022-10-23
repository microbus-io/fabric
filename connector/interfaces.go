package connector

import (
	"context"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
)

// Service is the interface that all microservices must support.
// It is implemented by the connector
type Service interface {
	Startup() error
	Shutdown() error
	IsStarted() bool

	HostName() string
	SetHostName(hostName string) error
	Deployment() string
	SetDeployment(deployment string) error
	Plane() string
	SetPlane(plane string) error

	Clock() clock.Clock
	Now() time.Time
	SetClock(newClock clock.Clock) error
}

// Logger is the interface that the connector supports for logging
type Logger interface {
	LogDebug(ctx context.Context, msg string, fields ...log.Field)
	LogInfo(ctx context.Context, msg string, fields ...log.Field)
	LogWarn(ctx context.Context, msg string, fields ...log.Field)
	LogError(ctx context.Context, msg string, fields ...log.Field)
}

// Publisher is the interface that the connector supports for making outgoing requests
type Publisher interface {
	GET(ctx context.Context, url string) (*http.Response, error)
	POST(ctx context.Context, url string, body any) (*http.Response, error)
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
}

// Subscriber is the interface that the connector supports for subscribing to handle incoming requests
type Subscriber interface {
	Subscribe(path string, handler HTTPHandler, options ...sub.Option) error
	Unsubscribe(path string) error
	UnsubscribeAll() error
}
