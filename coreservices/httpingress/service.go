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

package httpingress

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/trc"

	"go.opentelemetry.io/otel/propagation"

	"github.com/microbus-io/fabric/coreservices/httpingress/intermediate"
	"github.com/microbus-io/fabric/coreservices/httpingress/middleware"
)

/*
Service implements the http.ingress.core microservice.

The HTTP ingress microservice relays incoming HTTP requests to the NATS bus.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	httpServers    map[int]*http.Server
	mux            sync.Mutex
	allowedOrigins map[string]bool
	portMappings   map[string]string
	reqMemoryUsed  int64
	secure443      bool
	blockedPaths   map[string]bool
	middleware     *middleware.Chain
	handler        connector.HTTPHandler
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	svc.OnChangedAllowedOrigins(ctx)
	svc.OnChangedPortMappings(ctx)
	svc.OnChangedBlockedPaths(ctx)

	// Setup the middleware chain
	svc.handler = svc.serveHTTP
	mwHandlers := svc.Middleware().Handlers()
	for h := len(mwHandlers) - 1; h >= 0; h-- {
		svc.handler = mwHandlers[h](svc.handler)
	}
	svc.LogInfo(ctx, "Middleware", "chain", svc.Middleware().String())

	err = svc.startHTTPServers(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	// ASCII art fun
	if svc.Deployment() == connector.LOCAL {
		fmt.Print(`
8"""8"""8 8  8""""8 8"""8  8"""88 8""""8   8   8 8""""8 
8   8   8 8  8    " 8   8  8    8 8    8   8   8 8      
8e  8   8 8e 8e     8eee8e 8    8 8eeee8ee 8e  8 8eeeee 
88  8   8 88 88     88   8 8    8 88     8 88  8     88 
88  8   8 88 88   e 88   8 8    8 88     8 88  8 e   88 
88  8   8 88 88eee8 88   8 8eeee8 88eeeee8 88ee8 8eee88 

`)
	}

	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	err = svc.stopHTTPServers(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Middleware returns the middleware chain set for the ingress proxy.
// The chain is initialized to a default that can be customized.
// Changing the middleware after the server starts has no effect.
func (svc *Service) Middleware() *middleware.Chain {
	// Default middleware
	if svc.middleware == nil {
		m := &middleware.Chain{}

		// Warning: renaming or removing middleware is a breaking change because the names are used as location markers
		m.Append("ErrorPrinter", middleware.ErrorPrinter())
		m.Append("BlockedPaths", middleware.BlockedPaths(func(path string) bool {
			if svc.blockedPaths[path] {
				return true
			}
			dot := strings.LastIndex(path, ".")
			if dot >= 0 && svc.blockedPaths["*"+path[dot:]] {
				return true
			}
			return false
		}))
		m.Append("Logger", middleware.Logger(svc))
		m.Append("Enter", middleware.NoOp()) // Marker
		m.Append("SecureRedirect", middleware.SecureRedirect(func() bool {
			return svc.secure443
		}))
		m.Append("CORS", middleware.Cors(func(origin string) bool {
			return svc.allowedOrigins["*"] || svc.allowedOrigins[origin]
		}))
		m.Append("XForward", middleware.XForwarded())
		m.Append("InternalHeaders", middleware.InternalHeaders())
		m.Append("RootPath", middleware.RewriteRootPath("/root"))
		m.Append("Timeout", middleware.RequestTimeout(func() time.Duration {
			return svc.TimeBudget()
		}))
		m.Append("Ready", middleware.NoOp()) // Marker
		m.Append("CacheControl", middleware.CacheControl("no-store"))
		m.Append("Compress", middleware.Compress())
		m.Append("DefaultFavIcon", middleware.DefaultFavIcon())

		svc.middleware = m
	}
	return svc.middleware
}

// OnChangedPorts is triggered when the value of the Ports config property changes.
func (svc *Service) OnChangedPorts(ctx context.Context) (err error) {
	return svc.restartHTTPServers(ctx)
}

// restartHTTPServers stops and then restarts the HTTP servers.
func (svc *Service) restartHTTPServers(ctx context.Context) (err error) {
	svc.stopHTTPServers(ctx)
	err = svc.startHTTPServers(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// stopHTTPServers stops the running HTTP servers.
func (svc *Service) stopHTTPServers(ctx context.Context) (err error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	var lastErr error
	for httpPort, httpServer := range svc.httpServers {
		err = httpServer.Close() // Not a graceful shutdown
		if err != nil {
			lastErr = errors.Trace(err)
			svc.LogError(ctx, "Stopping HTTP listener",
				"port", httpPort,
				"error", lastErr,
			)
		} else {
			svc.LogInfo(ctx, "Stopped HTTP listener",
				"port", httpPort,
			)
		}
	}
	svc.httpServers = map[int]*http.Server{}
	return lastErr
}

// startHTTPServers starts the HTTP servers for each of the designated ports.
func (svc *Service) startHTTPServers(ctx context.Context) (err error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	svc.httpServers = map[int]*http.Server{}
	ports := strings.Split(svc.Ports(), ",")
	for _, port := range ports {
		port = strings.TrimSpace(port)
		if port == "" {
			continue
		}
		portInt, err := strconv.Atoi(port)
		if err != nil || (portInt < 1 || portInt > 65535) {
			err = errors.Newf("invalid port '%s'", port)
			svc.LogError(ctx, "Starting HTTP listener",
				"port", portInt,
				"error", err,
			)
			return errors.Trace(err)
		}

		// Look for TLS certs
		certFile := "httpingress-" + port + "-cert.pem"
		keyFile := "httpingress-" + port + "-key.pem"
		secure := true
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			secure = false
		}
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			secure = false
		}

		// https://pkg.go.dev/net/http?utm_source=godoc#Server
		httpServer := &http.Server{
			Addr:              ":" + port,
			Handler:           svc,
			ReadHeaderTimeout: svc.ReadHeaderTimeout(),
			ReadTimeout:       svc.ReadTimeout(),
			WriteTimeout:      svc.WriteTimeout(),
			ErrorLog:          newHTTPLogger(svc),
		}
		svc.httpServers[portInt] = httpServer
		errChan := make(chan error)
		calledChan := make(chan bool)
		if secure {
			if portInt == 443 {
				svc.secure443 = true
			}
			go func() {
				close(calledChan)
				err = httpServer.ListenAndServeTLS(certFile, keyFile)
				if err != nil {
					errChan <- errors.Trace(err)
				}
			}()
		} else {
			go func() {
				close(calledChan)
				err = httpServer.ListenAndServe()
				if err != nil {
					errChan <- errors.Trace(err)
				}
			}()
		}
		<-calledChan // Goroutine called
		select {
		case err = <-errChan:
			svc.LogError(ctx, "Starting HTTP listener",
				"error", err,
				"port", portInt,
				"secure", secure,
			)
			return errors.Trace(err)
		case <-time.After(time.Millisecond * 250):
			svc.LogInfo(ctx, "Started HTTP listener",
				"port", portInt,
				"secure", secure,
			)
		}
	}
	return nil
}

// ServeHTTP forwards incoming HTTP requests to the appropriate microservice.
// An incoming request http://localhost:8080/echo.example/echo is forwarded to
// the microservice at https://echo.example/echo .
// ServeHTTP implements the http.Handler interface
func (svc *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := svc.Lifetime()
	handlerStartTime := time.Now()

	// Fill in the gaps
	port := ""
	if p := strings.LastIndex(r.Host, ":"); p >= 0 {
		port = r.Host[p+1:]
	} else if r.TLS != nil {
		r.Host += ":443"
		port = "443"
	} else {
		r.Host += ":80"
		port = "80"
	}
	r.URL.Host = r.Host
	if r.TLS != nil {
		r.URL.Scheme = "https"
	} else {
		r.URL.Scheme = "http"
	}
	if !strings.HasPrefix(r.URL.Path, "/") {
		r.URL.Path = "/" + r.URL.Path
	}

	// OpenTelemetry: create the root span
	spanOptions := []trc.Option{
		trc.Server(),
		// Do not record the request attributes yet because they take a lot of memory, they will be added if there's an error
	}
	if svc.Deployment() == connector.LOCAL {
		// Add the request attributes in LOCAL deployment to facilitate debugging
		spanOptions = append(spanOptions, trc.Request(r))
	}
	var span trc.Span
	ctx, span = svc.StartSpan(ctx, ":"+port+r.URL.Path, spanOptions...)
	defer span.End()
	r = r.WithContext(ctx)

	ww := httpx.NewResponseRecorder() // This recorder allows modifying the response after it was written
	err := errors.CatchPanic(func() error {
		_ = r.URL.Port() // Validates the port (may panic in malformed requests)
		return svc.handler(ww, r)
	})
	if err != nil {
		// OpenTelemetry: record the error, adding the request attributes
		span.SetRequest(r)
		span.SetError(err)
		svc.ForceTrace(ctx)
	} else {
		// OpenTelemetry: record the status code
		span.SetOK(ww.StatusCode())
	}
	_ = httpx.Copy(w, ww.Result())

	// Meter
	_ = svc.ObserveMetric(
		"microbus_response_duration_seconds",
		time.Since(handlerStartTime).Seconds(),
		r.Host+"/",
		port,
		r.Method,
		strconv.Itoa(ww.StatusCode()),
		func() string {
			if err != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)
	_ = svc.ObserveMetric(
		"microbus_response_size_bytes",
		float64(ww.ContentLength()),
		r.Host+"/",
		port,
		r.Method,
		strconv.Itoa(ww.StatusCode()),
		func() string {
			if err != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)
}

// serveHTTP forwards incoming HTTP requests to the appropriate microservice on NATS.
// An incoming request http://localhost:8080/echo.example/echo is forwarded to
// the microservice at https://echo.example/echo .
// serveHTTP implements the http.Handler interface
func (svc *Service) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	// Use the first segment of the URI as the hostname to contact
	u, err := resolveInternalURL(r.URL, svc.portMappings)
	if err != nil {
		// Ignore requests to invalid internal hostnames, such as via https://example.com/%3Fterms=1 or https://example.com/.env
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	// Disallow requests to internal port 888
	if u.Port() == "888" {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	internalURL := u.String()

	// Read the body fully
	body, err := svc.readRequestBody(r)
	if err != nil {
		return errors.Trace(err)
	}
	defer svc.releaseRequestBody(body)

	// Prepare the internal request options
	options := []pub.Option{
		pub.Method(r.Method),
		pub.URL(internalURL),
		pub.Body(body),
		pub.Unicast(),
		pub.CopyHeaders(r.Header),    // Copy all headers
		pub.ContentLength(len(body)), // Overwrite the Content-Length header
	}

	// OpenTelemetry: pass the span in the headers
	carrier := make(propagation.HeaderCarrier)
	propagation.TraceContext{}.Inject(ctx, carrier)
	for k, v := range carrier {
		options = append(options, pub.Header(k, v[0]))
	}

	// Delegate the request over NATS
	internalRes, err := svc.Request(ctx, options...)
	if err != nil {
		return err // No trace
	}
	err = httpx.Copy(w, internalRes)
	return errors.Trace(err)
}

// readRequestBody reads the body of the request into memory, within the memory limit
// set for the proxy.
func (svc *Service) readRequestBody(r *http.Request) (body []byte, err error) {
	if r.Body == nil || r.ContentLength == 0 {
		return []byte{}, nil
	}
	defer r.Body.Close()
	limit := int64(svc.RequestMemoryLimit()) * 1024 * 1024
	limit /= 2 // Because body is duplicated when creating the NATS request
	if r.ContentLength > 0 {
		used := atomic.LoadInt64(&svc.reqMemoryUsed)
		if used+r.ContentLength > limit {
			return nil, errors.Newc(http.StatusRequestEntityTooLarge, "insufficient memory")
		}
	}
	bufSize := r.ContentLength
	if bufSize < 0 || bufSize > 64*1024 {
		// Max 64KB
		bufSize = 64 * 1024
	}
	var result bytes.Buffer
	buf := make([]byte, bufSize)
	nn := 0
	done := false
	for !done {
		n, err := io.ReadFull(r.Body, buf)
		if err == io.EOF {
			break
		}
		if err == io.ErrUnexpectedEOF {
			err = nil
			done = true
		}
		if err != nil {
			atomic.AddInt64(&svc.reqMemoryUsed, -int64(nn))
			return nil, errors.Trace(err)
		}
		nn += n
		used := atomic.AddInt64(&svc.reqMemoryUsed, int64(n))
		if used > limit {
			atomic.AddInt64(&svc.reqMemoryUsed, -int64(nn))
			return nil, errors.Newc(http.StatusRequestEntityTooLarge, "insufficient memory")
		}
		result.Write(buf[:n])
	}
	return result.Bytes(), nil
}

// releaseRequestBody should be called when the request is done to update the used memory counter.
func (svc *Service) releaseRequestBody(body []byte) {
	atomic.AddInt64(&svc.reqMemoryUsed, -int64(len(body)))
}

// OnChangedAllowedOrigins is triggered when the value of the AllowedOrigins config property changes.
func (svc *Service) OnChangedAllowedOrigins(ctx context.Context) (err error) {
	value := svc.AllowedOrigins()
	newOrigins := map[string]bool{}
	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			newOrigins[origin] = true
		}
	}
	svc.allowedOrigins = newOrigins
	return nil
}

// OnChangedPortMappings is triggered when the value of the PortMappings config property changes.
func (svc *Service) OnChangedPortMappings(ctx context.Context) (err error) {
	value := svc.PortMappings() // e.g. "8080:*->*, 443:*->443, 80:*->443"
	newMappings := map[string]string{}
	for _, m := range strings.Split(value, ",") {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		external, internal, ok := strings.Cut(m, "->")
		if !ok {
			svc.LogWarn(ctx, "Invalid port mapping",
				"mapping", m,
			)
			continue
		}
		newMappings[external] = internal
	}
	svc.portMappings = newMappings
	return nil
}

// resolveInternalURL resolves the NATS URL from the external URL.
func resolveInternalURL(externalURL *url.URL, portMappings map[string]string) (natsURL *url.URL, err error) {
	externalPort := externalURL.Port()
	if externalPort == "" {
		externalPort = "443"
	}
	u, err := httpx.ParseURL("https:/" + externalURL.RequestURI()) // First part of the URL is the internal host
	if err != nil {
		return nil, errors.Trace(err)
	}
	internalPort := u.Port()
	mappedInternalPort := internalPort
	mappingKeys := []string{
		externalPort + ":" + internalPort,
		"*:" + internalPort,
		externalPort + ":*",
		"*:*",
	}
	for _, mappingKey := range mappingKeys {
		if portMappings[mappingKey] != "" && portMappings[mappingKey] != "*" {
			mappedInternalPort = portMappings[mappingKey]
			break
		}
	}
	if mappedInternalPort != internalPort {
		p := strings.Index(u.Host, ":")
		if p < 0 {
			p = len(u.Host)
		}
		u.Host = u.Host[:p] + ":" + mappedInternalPort
	}
	u.Host = strings.TrimSuffix(u.Host, ":443")
	return u, nil
}

// OnChangedReadTimeout is triggered when the value of the ReadTimeout config property changes.
func (svc *Service) OnChangedReadTimeout(ctx context.Context) (err error) {
	return svc.restartHTTPServers(ctx)
}

// OnChangedWriteTimeout is triggered when the value of the WriteTimeout config property changes.
func (svc *Service) OnChangedWriteTimeout(ctx context.Context) (err error) {
	return svc.restartHTTPServers(ctx)
}

// OnChangedReadHeaderTimeout is triggered when the value of the ReadHeaderTimeout config property changes.
func (svc *Service) OnChangedReadHeaderTimeout(ctx context.Context) (err error) {
	return svc.restartHTTPServers(ctx)
}

// OnChangedBlockedPaths is triggered when the value of the BlockPaths config property changes.
func (svc *Service) OnChangedBlockedPaths(ctx context.Context) (err error) {
	value := svc.BlockedPaths()
	newPaths := map[string]bool{}
	for _, path := range strings.Split(value, "\n") {
		path = strings.TrimSpace(path)
		if path != "" {
			newPaths[path] = true
		}
	}
	svc.blockedPaths = newPaths
	return nil
}
