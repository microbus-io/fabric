package httpingress

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"

	"github.com/microbus-io/fabric/services/httpingress/intermediate"
)

var (
	_ errors.TracedError
	_ http.Request
)

/*
Service implements the "http.ingress.sys" microservice.

The HTTP Ingress microservice relays incoming HTTP requests to the NATS bus.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	httpPort   int
	httpServer *http.Server
	timeBudget time.Duration
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) error {
	// Time budget for requests
	var err error
	svc.timeBudget, err = time.ParseDuration(svc.Config("TimeBudget"))
	if err != nil {
		return errors.Trace(err)
	}

	// Incoming HTTP port
	svc.httpPort, err = strconv.Atoi(svc.Config("Port"))
	if err != nil {
		return errors.Trace(err)
	}

	// Start HTTP server
	svc.httpServer = &http.Server{
		Addr:    ":" + strconv.Itoa(svc.httpPort),
		Handler: svc,
	}
	svc.LogInfo(ctx, "Starting HTTP listener", log.Int("port", svc.httpPort))
	go svc.httpServer.ListenAndServe()

	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	// Stop HTTP server
	if svc.httpServer != nil {
		svc.LogInfo(ctx, "Stopping HTTP listener", log.Int("port", svc.httpPort))
		err := svc.httpServer.Close() // Not a graceful shutdown
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
func (svc *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Use the first segment of the URI as the host name to contact
	uri := r.URL.RequestURI()
	internalURL := "https:/" + uri
	internalHost := strings.Split(uri, "/")[1]

	// Skip favicon.ico to reduce noise
	if uri == "/favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctx := svc.Lifetime()

	svc.LogInfo(ctx, "Request received", log.String("url", internalURL))

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
	if svc.timeBudget > 0 {
		options = append(options, pub.TimeBudget(svc.timeBudget))
		var cancel context.CancelFunc
		delegateCtx, cancel = svc.Clock().WithTimeout(ctx, svc.timeBudget)
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

	// Set proxy headers
	options = append(options, pub.Header("X-Forwarded-Host", r.Host))
	options = append(options, pub.Header("X-Forwarded-For", r.RemoteAddr))
	options = append(options, pub.Header("X-Forwarded-Trimmed-Prefix", "/"+internalHost))

	// Delegate the request over NATS
	internalRes, err := svc.Request(delegateCtx, options...)
	if err != nil {
		err = errors.Trace(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%+v", err)))
		svc.LogError(ctx, "Delegating request", log.Error(err))
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
			err = errors.Trace(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%+v", err)))
			svc.LogError(ctx, "Copying response body", log.Error(err))
			return
		}
	}
}
