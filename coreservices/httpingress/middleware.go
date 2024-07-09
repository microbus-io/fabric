package httpingress

import (
	"net/http"

	"github.com/microbus-io/fabric/connector"
)

// MiddlewareFunc is a processor that can be added to pre- or post-process a request.
// It should call the next function in the chain.
type MiddlewareFunc func(w http.ResponseWriter, r *http.Request, next connector.HTTPHandler) (err error)

// Middleware is a processor that can be added to pre- or post-process a request.
// It should call the next function in the chain.
type Middleware interface {
	Serve(w http.ResponseWriter, r *http.Request, next connector.HTTPHandler) (err error)
}

// simpleMiddleware converts a function to a middleware interface.
type simpleMiddleware struct {
	f MiddlewareFunc
}

// Serve pre- or post-processes a request.
func (fmw *simpleMiddleware) Serve(w http.ResponseWriter, r *http.Request, next connector.HTTPHandler) (err error) {
	return fmw.f(w, r, next)
}
