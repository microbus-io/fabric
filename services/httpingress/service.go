package httpingress

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
)

// Service is an HTTP ingress microservice
type Service struct {
	*connector.Connector
	httpPort   int
	httpServer *http.Server
	timeBudget time.Duration
}

// NewService creates a new HTTP ingress microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.NewConnector(),
	}
	s.SetHostName("http.ingress.sys")
	s.SetOnStartup(s.OnStartup)
	s.SetOnShutdown(s.OnShutdown)

	return s
}

// OnStartup starts the web server
func (s *Service) OnStartup(ctx context.Context) error {
	// Time budget for requests
	var ok bool
	s.timeBudget, ok = s.ConfigDuration("TimeBudget")
	if !ok {
		s.timeBudget = time.Second * 20
	}

	// Incoming HTTP port
	s.httpPort, ok = s.ConfigInt("Port")
	if !ok {
		s.httpPort = 8080
	}

	// Start HTTP server
	s.httpServer = &http.Server{
		Addr:    ":" + strconv.Itoa(s.httpPort),
		Handler: s,
	}
	s.LogInfo(ctx, "Starting HTTP listener", log.Int("port", s.httpPort))
	go s.httpServer.ListenAndServe()

	return nil
}

// OnShutdown stops the web server
func (s *Service) OnShutdown(ctx context.Context) error {
	// Stop HTTP server
	if s.httpServer != nil {
		s.LogInfo(ctx, "Stopping HTTP listener", log.Int("port", s.httpPort))
		err := s.httpServer.Close() // Not a graceful shutdown
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// ServeHTTP forwards incoming HTTP requests to the appropriate microservice on NATS.
// An incoming request http://localhost:8080/echo.example/echo is forwarded to
// the microservice at https://echo.example/echo .
// ServeHTTP implements the http.Handler interface
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Use the first segment of the URI as the host name to contact
	uri := r.URL.RequestURI()
	internalURL := "https:/" + uri

	// Skip favicon.ico to reduce noise
	if uri == "/favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctx := context.Background()

	s.LogInfo(ctx, "Request received", log.String("url", internalURL))

	// Prepare the internal request options
	defer r.Body.Close()
	options := []pub.Option{
		pub.Method(r.Method),
		pub.URL(internalURL),
		pub.Body(r.Body),
		pub.Unicast(),
	}

	// Add the time budget to the request headers and set it as the context's timeout
	delegateCtx := ctx
	if s.timeBudget > 0 {
		options = append(options, pub.TimeBudget(s.timeBudget))
		var cancel context.CancelFunc
		delegateCtx, cancel = context.WithTimeout(ctx, s.timeBudget)
		defer cancel()
	}

	// Copy non-internal headers
	for hdrName, hdrVals := range r.Header {
		if !strings.HasPrefix(hdrName, frame.HeaderPrefix) {
			for _, val := range hdrVals {
				options = append(options, pub.Header(hdrName, val))
			}
		}
	}

	// Delegate the request over NATS
	internalRes, err := s.Request(delegateCtx, options...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%+v", errors.Trace(err))))
		s.LogError(ctx, "Delegating request", log.Error(err))
		return
	}

	// Write back non-internal headers
	for hdrName, hdrVals := range internalRes.Header {
		if !strings.HasPrefix(hdrName, frame.HeaderPrefix) {
			for _, val := range hdrVals {
				w.Header().Add(hdrName, val)
			}
		}
	}

	// No caching by default
	if internalRes.Header.Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-store")
	}

	// Write back the status code
	w.WriteHeader(internalRes.StatusCode)

	// Write back the body
	if internalRes.Body != nil {
		_, err = io.Copy(w, internalRes.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%+v", errors.Trace(err))))
			s.LogError(ctx, "Copying response body", log.Error(err))
			return
		}
	}
}
