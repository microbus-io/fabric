package dlru

import (
	"context"
	"net/http"

	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
)

// Service is an interface abstraction of a microservice used by the distributed cache.
// The connector implements this interface.
type Service interface {
	ID() string
	HostName() string
	Subscribe(path string, handler sub.HTTPHandler, options ...sub.Option) error
	Unsubscribe(path string) error
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	LogInfo(ctx context.Context, msg string, fields ...log.Field)
	LogDebug(ctx context.Context, msg string, fields ...log.Field)
}
