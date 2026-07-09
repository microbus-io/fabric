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

package httpingress

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
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

	"github.com/golang-jwt/jwt/v5"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/accesstoken/accesstokenapi"
	"github.com/microbus-io/fabric/coreservices/bearertoken/bearertokenapi"
	"github.com/microbus-io/fabric/coreservices/httpingress/middleware"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/fabric/utils"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/sync/singleflight"
)

/*
Service implements the http.ingress.core microservice.

The HTTP ingress microservice relays incoming HTTP requests to the NATS bus.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	httpServers           map[int]*http.Server
	certs                 *httpx.CertStore
	mux                   sync.Mutex
	credentialedOrigins   map[string]bool
	uncredentialedOrigins map[string]bool
	allowedInternalPorts  map[int]bool
	reqMemoryUsed         int64
	secure443             bool
	blockedPaths          map[string]bool
	middleware            *middleware.Chain
	handler               connector.HTTPHandler
	bearerTokenMu         sync.RWMutex
	bearerTokenKeys       map[string]map[string]ed25519.PublicKey
	lastJWKSFetch         map[string]time.Time
	jwksFlight            singleflight.Group
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	err = svc.OnChangedAllowedOrigins(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	err = svc.OnChangedAllowedCredentialedOrigins(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	err = svc.OnChangedAllowedUncredentialedOrigins(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	err = svc.OnChangedPortMappings(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	err = svc.OnChangedAllowedInternalPorts(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	svc.OnChangedBlockedPaths(ctx)

	// Setup the middleware chain
	svc.handler = svc.serveHTTP
	mwHandlers := svc.Middleware().Handlers()
	for h := len(mwHandlers) - 1; h >= 0; h-- {
		svc.handler = mwHandlers[h](svc.handler)
	}
	svc.LogInfo(ctx, "Middleware", "chain", svc.Middleware().String())

	svc.certs = httpx.NewCertStore("", svc)
	svc.certs.Load(ctx)

	err = svc.startHTTPServers(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	svc.Go(ctx, svc.certs.Watch)

	// ASCII art fun
	if svc.Deployment() == connector.LOCAL {
		fmt.Print(connector.Green, `
8"""8"""8 8  8""""8 8"""8  8"""88 8""""8   8   8 8""""8 
8   8   8 8  8    " 8   8  8    8 8    8   8   8 8      
8e  8   8 8e 8e     8eee8e 8    8 8eeee8ee 8e  8 8eeeee 
88  8   8 88 88     88   8 8    8 88     8 88  8     88 
88  8   8 88 88   e 88   8 8    8 88     8 88  8 e   88 
88  8   8 88 88eee8 88   8 8eeee8 88eeeee8 88ee8 8eee88 

`, connector.Reset)
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
	if svc.middleware == nil {
		svc.middleware = svc.defaultMiddleware()
	}
	return svc.middleware
}

/*
OnChangedPorts is called when the Ports config property changes.

Ports is a comma-separated list of HTTP ports on which to listen for requests. A port may be
followed by a "tls" marker, e.g. "80, 443 tls, 8080", to terminate TLS using the SAN-indexed
certificates; a bare port enables TLS only when its legacy httpingress-{port}-cert.pem and -key.pem
files are present. Port 80 is always plaintext.
*/
func (svc *Service) OnChangedPorts(ctx context.Context) (err error) { // MARKER: Ports
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

// portSpec is a single parsed entry of the Ports config.
type portSpec struct {
	port int
	tls  bool
}

// portCertExists reports whether {prefix}{port}-cert.pem and -key.pem both exist in the working
// directory. It backs the bare-port TLS auto-detection in parsePorts.
func portCertExists(port, prefix string) bool {
	if _, err := os.Stat(prefix + port + "-cert.pem"); err != nil {
		return false
	}
	if _, err := os.Stat(prefix + port + "-key.pem"); err != nil {
		return false
	}
	return true
}

// parsePorts parses the Ports config grammar: a comma-separated list where each entry is a port
// optionally followed by a "tls" marker, e.g. "80, 443 tls, 8080". A bare port keeps legacy
// behavior: TLS iff its legacy port-named cert/key files are present. Port 80 is always plaintext
// and an explicit "80 tls" is a startup error. Any other marker is ignored.
func parsePorts(value string) (specs []portSpec, err error) {
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		fields := strings.Fields(entry)
		portInt, convErr := strconv.Atoi(fields[0])
		if convErr != nil || portInt < 1 || portInt > 65535 {
			return nil, errors.New("invalid port '%s'", fields[0])
		}
		markers := map[string]bool{}
		for i := 1; i < len(fields); i++ {
			markers[strings.ToLower(fields[i])] = true
		}
		secure := false
		switch {
		case portInt == 80:
			if markers["tls"] {
				return nil, errors.New("port 80 cannot be TLS")
			}
		case markers["tls"]:
			secure = true
		default:
			// Bare port: TLS iff a port-named cert and key pair is present. Mirrors the
			// httpx.CertStore convention and accepts both the new "{port}-..." form and the
			// legacy "httpingress-{port}-..." form for backward compatibility.
			secure = portCertExists(fields[0], "") || portCertExists(fields[0], "httpingress-")
		}
		specs = append(specs, portSpec{port: portInt, tls: secure})
	}
	return specs, nil
}

// startHTTPServers starts the HTTP servers for each of the designated ports.
func (svc *Service) startHTTPServers(ctx context.Context) (err error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	svc.httpServers = map[int]*http.Server{}

	specs, err := parsePorts(svc.Ports())
	if err != nil {
		svc.LogError(ctx, "Starting HTTP listener", "error", err)
		return errors.Trace(err)
	}

	// secure443 drives the SecureRedirect middleware: redirect :80 -> :443 only when 443 is TLS.
	svc.secure443 = false
	for _, s := range specs {
		if s.port == 443 && s.tls {
			svc.secure443 = true
		}
	}

	for _, s := range specs {
		// https://pkg.go.dev/net/http?utm_source=godoc#Server
		httpServer := &http.Server{
			Addr:              ":" + strconv.Itoa(s.port),
			Handler:           svc,
			ReadHeaderTimeout: svc.ReadHeaderTimeout(),
			ReadTimeout:       svc.ReadTimeout(),
			WriteTimeout:      svc.WriteTimeout(),
			ErrorLog:          newHTTPLogger(svc),
		}
		if s.tls {
			httpServer.TLSConfig = &tls.Config{
				GetCertificate: svc.certs.GetCertificate(s.port),
			}
		}
		svc.httpServers[s.port] = httpServer
		errChan := make(chan error)
		calledChan := make(chan bool)
		if s.tls {
			go func() {
				close(calledChan)
				e := httpServer.ListenAndServeTLS("", "")
				if e != nil {
					errChan <- errors.Trace(e)
				}
			}()
		} else {
			go func() {
				close(calledChan)
				e := httpServer.ListenAndServe()
				if e != nil {
					errChan <- errors.Trace(e)
				}
			}()
		}
		<-calledChan // Goroutine called
		select {
		case err = <-errChan:
			svc.LogError(ctx, "Starting HTTP listener",
				"error", err,
				"port", s.port,
				"secure", s.tls,
			)
			return errors.Trace(err)
		case <-time.After(time.Millisecond * 250):
			svc.LogInfo(ctx, "Started HTTP listener",
				"port", s.port,
				"secure", s.tls,
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
	var span trc.Span
	ctx, span = svc.StartSpan(ctx, ":"+port+r.URL.Path,
		trc.Server(),
		trc.Request(r),
		trc.ClientIP(r.RemoteAddr),
	)
	spanEnded := false
	defer func() {
		if !spanEnded {
			span.End()
		}
	}()
	// Set a frame in the context and the request
	ctx = frame.ContextWithFrameOf(ctx, r)
	r = r.WithContext(ctx)

	ww := httpx.NewResponseRecorder() // This recorder allows modifying the response after it was written
	err := errors.CatchPanic(func() error {
		_ = r.URL.Port() // Validates the port (may panic in malformed requests)
		return svc.handler(ww, r)
	})
	if err != nil {
		// OpenTelemetry: record the error
		span.SetError(err)
		svc.ForceTrace(ctx)
	} else {
		// OpenTelemetry: record the status code
		span.SetOK(ww.StatusCode())
	}
	_ = httpx.Copy(w, ww.Result())

	// Meter
	_ = svc.RecordHistogram(
		ctx,
		"microbus_server_request_duration_seconds",
		time.Since(handlerStartTime).Seconds(),
		"name", "ServeHTTP",
		"port", port,
		"method", r.Method,
		"code", ww.StatusCode(),
		"feature", "ingress",
		"error", func() string {
			if err != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)
	_ = svc.RecordHistogram(
		ctx,
		"microbus_server_response_body_bytes",
		float64(ww.ContentLength()),
		"name", "ServeHTTP",
		"port", port,
		"method", r.Method,
		"code", ww.StatusCode(),
		"feature", "ingress",
		"error", func() string {
			if err != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)
	span.End()
	spanEnded = true
}

// serveHTTP forwards incoming HTTP requests to the appropriate microservice on NATS.
// An incoming request http://localhost:8080/echo.example/echo is forwarded to
// the microservice at https://echo.example/echo .
// serveHTTP implements the http.Handler interface
func (svc *Service) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	// Use the first segment of the path as the hostname to contact
	u, err := resolveInternalURL(r.URL)
	if err != nil {
		// Ignore requests to invalid internal hostnames, such as via https://example.com/%3Fterms=1 or https://example.com/.env
		return errors.Trace(err)
	}
	// Apply the internal-port firewall
	port := 443
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return errors.New("", http.StatusNotFound)
		}
	}
	if !svc.isInternalPortAllowed(port) {
		return errors.New("", http.StatusNotFound)
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

	// Delegate the request over the bus
	internalRes, err := svc.Request(ctx, options...)
	if err != nil {
		return err // No trace
	}
	err = httpx.Copy(w, internalRes)
	return errors.Trace(err)
}

// readRequestBody reads the body of the request into memory, within the memory limit set for the proxy.
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
			return nil, errors.New("insufficient memory", http.StatusRequestEntityTooLarge)
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
			return nil, errors.New("insufficient memory", http.StatusRequestEntityTooLarge)
		}
		result.Write(buf[:n])
	}
	return result.Bytes(), nil
}

// releaseRequestBody should be called when the request is done to update the used memory counter.
func (svc *Service) releaseRequestBody(body []byte) {
	atomic.AddInt64(&svc.reqMemoryUsed, -int64(len(body)))
}

/*
OnChangedAllowedOrigins is called when the AllowedOrigins config property changes.

AllowedOrigins is DEPRECATED. It has been split into AllowedCredentialedOrigins and
AllowedUncredentialedOrigins so that an origin's access to credentials is always explicit.
Setting this config to any non-empty value causes the microservice to refuse to start,
rather than silently ignore an operator's intended posture.

Deprecated: Use AllowedCredentialedOrigins or AllowedUncredentialedOrigins instead
*/
func (svc *Service) OnChangedAllowedOrigins(ctx context.Context) (err error) { // MARKER: AllowedOrigins
	if strings.TrimSpace(svc.AllowedOrigins()) != "" {
		return errors.New(
			"AllowedOrigins has been split into AllowedCredentialedOrigins and AllowedUncredentialedOrigins (got '%s')",
			svc.AllowedOrigins(),
		)
	}
	return nil
}

/*
OnChangedAllowedCredentialedOrigins is called when the AllowedCredentialedOrigins config property changes.

AllowedCredentialedOrigins is a comma-separated list of CORS origins trusted to make credentialed requests.
A listed origin is reflected in Access-Control-Allow-Origin along with Access-Control-Allow-Credentials: true,
permitting the browser to send and read authenticated (cookie-bearing) requests cross-origin.
The wildcard origin is rejected: combining any-origin with credentials is the classic CORS vulnerability,
and this config exists to make it inexpressible.
When both origin lists are empty (the default), Access-Control-Allow-Origin is pinned to the request's
own scheme://host, which permits only same-origin browser reads.
*/
func (svc *Service) OnChangedAllowedCredentialedOrigins(ctx context.Context) (err error) { // MARKER: AllowedCredentialedOrigins
	newOrigins := map[string]bool{}
	for origin := range strings.SplitSeq(svc.AllowedCredentialedOrigins(), ",") {
		origin = strings.TrimSpace(origin)
		if origin == "*" {
			return errors.New("the wildcard origin cannot be credentialed; list it in AllowedUncredentialedOrigins instead")
		}
		if origin != "" {
			newOrigins[origin] = true
		}
	}
	svc.credentialedOrigins = newOrigins
	return nil
}

/*
OnChangedAllowedUncredentialedOrigins is called when the AllowedUncredentialedOrigins config property changes.

AllowedUncredentialedOrigins is a comma-separated list of CORS origins allowed to make uncredentialed
requests, or * to allow all origins. The browser blocks credentials for these origins, which is the
correct semantics for a public API. An origin listed in AllowedCredentialedOrigins takes precedence.
When both origin lists are empty (the default), Access-Control-Allow-Origin is pinned to the request's
own scheme://host, which permits only same-origin browser reads.
*/
func (svc *Service) OnChangedAllowedUncredentialedOrigins(ctx context.Context) (err error) { // MARKER: AllowedUncredentialedOrigins
	newOrigins := map[string]bool{}
	for origin := range strings.SplitSeq(svc.AllowedUncredentialedOrigins(), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			newOrigins[origin] = true
		}
	}
	svc.uncredentialedOrigins = newOrigins
	return nil
}

/*
OnChangedPortMappings is called when the PortMappings config property changes.

PortMappings is REMOVED. The x:y->z port-rewrite model has been replaced by AllowedInternalPorts
(internal-port allowlist, no rewrite). Setting this config to any non-empty value causes the
microservice to refuse to start, rather than silently ignore an operator's intended posture.
*/
func (svc *Service) OnChangedPortMappings(ctx context.Context) (err error) { // MARKER: PortMappings
	if strings.TrimSpace(svc.PortMappings()) != "" {
		return errors.New(
			"PortMappings has been removed; use AllowedInternalPorts (got '%s')",
			svc.PortMappings(),
		)
	}
	return nil
}

/*
OnChangedAllowedInternalPorts is called when the AllowedInternalPorts config property changes.

AllowedInternalPorts is the operator-tunable allowlist of internal destination ports the
ingress is willing to forward to, in addition to the implicitly-allowed :443. Entries are
comma-separated and may be a single port or an inclusive range "N-M", e.g. "1234, 10000-11000".
All entries must satisfy 1024 <= port <= 65535; the microservice refuses to start otherwise.
Ports :666 and :888 are hard-blocked in every deployment mode and cannot be allowlisted. In LOCAL
deployment this config is ignored and every port except :666 and :888 is reachable.
*/
func (svc *Service) OnChangedAllowedInternalPorts(ctx context.Context) (err error) { // MARKER: AllowedInternalPorts
	set, err := parseAllowedInternalPorts(svc.AllowedInternalPorts())
	if err != nil {
		return errors.Trace(err)
	}
	svc.allowedInternalPorts = set
	return nil
}

// parseAllowedInternalPorts parses comma-separated ports and inclusive ranges "N-M". Every port
// must satisfy 1024 <= port <= 65535.
func parseAllowedInternalPorts(value string) (map[int]bool, error) {
	set := map[int]bool{}
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		start, end, isRange := strings.Cut(entry, "-")
		startPort, err := strconv.Atoi(strings.TrimSpace(start))
		if err != nil {
			return nil, errors.New("invalid port '%s' in AllowedInternalPorts", entry)
		}
		endPort := startPort
		if isRange {
			endPort, err = strconv.Atoi(strings.TrimSpace(end))
			if err != nil {
				return nil, errors.New("invalid port range '%s' in AllowedInternalPorts", entry)
			}
			if endPort < startPort {
				return nil, errors.New("inverted port range '%s' in AllowedInternalPorts", entry)
			}
		}
		if startPort < 1024 || endPort > 65535 {
			return nil, errors.New("port '%s' out of range [1024,65535] in AllowedInternalPorts", entry)
		}
		for p := startPort; p <= endPort; p++ {
			set[p] = true
		}
	}
	return set, nil
}

// isInternalPortAllowed reports whether the firewall permits forwarding to port.
func (svc *Service) isInternalPortAllowed(port int) bool {
	if port <= 0 || port > 65535 {
		return false
	}
	if port == 666 || port == 888 {
		return false
	}
	if svc.Deployment() == connector.LOCAL {
		return true
	}
	if port == 443 {
		return true
	}
	return svc.allowedInternalPorts[port]
}

// resolveInternalURL extracts the internal NATS URL from the external URL.
func resolveInternalURL(externalURL *url.URL) (natsURL *url.URL, err error) {
	externalURI := externalURL.RequestURI()
	if !strings.HasPrefix(externalURI, "/") {
		externalURI = "/" + externalURI
	}
	// NATS subject caps at 1024 so reject outright
	if len(externalURI) > 1024 {
		return nil, errors.New("", http.StatusRequestURITooLong)
	}
	internalURL, err := httpx.ParseURL("https:/" + externalURI) // First part of the URL is the internal host
	if err != nil {
		return nil, errors.New("", http.StatusNotFound)
	}
	internalURL.Host = strings.ToLower(internalURL.Host)
	internalURL.Host = strings.TrimSuffix(internalURL.Host, ":443")
	return internalURL, nil
}

/*
OnChangedReadTimeout is called when the ReadTimeout config property changes.

ReadTimeout specifies the timeout for fully reading a request.
*/
func (svc *Service) OnChangedReadTimeout(ctx context.Context) (err error) { // MARKER: ReadTimeout
	return svc.restartHTTPServers(ctx)
}

/*
OnChangedWriteTimeout is called when the WriteTimeout config property changes.

WriteTimeout specifies the timeout for fully writing the response to a request.
*/
func (svc *Service) OnChangedWriteTimeout(ctx context.Context) (err error) { // MARKER: WriteTimeout
	return svc.restartHTTPServers(ctx)
}

/*
OnChangedReadHeaderTimeout is called when the ReadHeaderTimeout config property changes.

ReadHeaderTimeout specifies the timeout for fully reading the header of a request.
*/
func (svc *Service) OnChangedReadHeaderTimeout(ctx context.Context) (err error) { // MARKER: ReadHeaderTimeout
	return svc.restartHTTPServers(ctx)
}

/*
OnChangedBlockedPaths is called when the BlockedPaths config property changes.

A newline-separated list of paths or extensions to block with a 404.
Paths should not include any arguments and are matched exactly.
Extensions are specified with "*.ext" and are matched against the extension of the path only.
*/
func (svc *Service) OnChangedBlockedPaths(ctx context.Context) (err error) { // MARKER: BlockedPaths
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

// lookupBearerTokenKey returns the cached Ed25519 public key for the given issuer host and kid.
func (svc *Service) lookupBearerTokenKey(host, kid string) (ed25519.PublicKey, bool) {
	svc.bearerTokenMu.RLock()
	defer svc.bearerTokenMu.RUnlock()
	key, ok := svc.bearerTokenKeys[host][kid]
	return key, ok
}

// fetchBearerTokenKeys fetches JWKS from the given host and updates the key cache.
// It is a no-op when the last fetch for the host was within 1s.
// Concurrent calls for the same host share a single fetch.
func (svc *Service) fetchBearerTokenKeys(ctx context.Context, host string) error {
	_, err, _ := svc.jwksFlight.Do(host, func() (any, error) {
		svc.bearerTokenMu.Lock()
		if svc.lastJWKSFetch == nil {
			svc.lastJWKSFetch = map[string]time.Time{}
		}
		if last, ok := svc.lastJWKSFetch[host]; ok && time.Since(last) < time.Second {
			svc.bearerTokenMu.Unlock()
			return nil, nil
		}
		svc.bearerTokenMu.Unlock()

		jwks, err := bearertokenapi.NewClient(svc).ForHost(host).JWKS(ctx)
		if err != nil {
			return nil, errors.Trace(err)
		}
		// Build the issuer's fresh key set, then swap it in wholesale so a successful fetch is
		// authoritative: a kid the issuer has rotated out of its JWKS is evicted, not retained.
		fresh := make(map[string]ed25519.PublicKey, len(jwks))
		for _, jwk := range jwks {
			pubBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
			if err != nil {
				continue
			}
			fresh[jwk.KID] = ed25519.PublicKey(pubBytes)
		}

		svc.bearerTokenMu.Lock()
		defer svc.bearerTokenMu.Unlock()
		svc.lastJWKSFetch[host] = time.Now()
		if svc.bearerTokenKeys == nil {
			svc.bearerTokenKeys = make(map[string]map[string]ed25519.PublicKey)
		}
		svc.bearerTokenKeys[host] = fresh
		return nil, nil
	})
	return errors.Trace(err)
}

// exchangeToken validates the external bearer token and returns a corresponding internal access token.
func (svc *Service) exchangeToken(ctx context.Context, bearerToken string) (accessToken string, err error) {
	if !utils.LooksLikeJWT(bearerToken) {
		return "", nil
	}

	// Parse the external bearer JWT (unverified) to extract issuer and kid
	parsedToken, _ := jwt.Parse(bearerToken, nil)
	if parsedToken == nil {
		return "", nil
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil
	}

	// Extract the issuer hostname and pin it to the framework's bearer token service.
	// External IDPs cannot drop tokens directly through the ingress - they must wrap
	// through bearer.token.core to obtain a framework-issued token.
	issStr, _ := claims["iss"].(string)
	_, issuerHost, ok := strings.Cut(issStr, "://")
	if !ok {
		issuerHost = issStr
	}
	if issuerHost != bearertokenapi.Hostname {
		return "", nil
	}

	// Extract kid from JWT header
	kid, _ := parsedToken.Header["kid"].(string)
	if kid == "" {
		return "", nil
	}

	// Look up the public key, refresh cache if needed
	key, found := svc.lookupBearerTokenKey(issuerHost, kid)
	if !found {
		err = svc.fetchBearerTokenKeys(ctx, issuerHost)
		if err != nil {
			if errors.StatusCode(err) == http.StatusNotFound {
				return "", nil
			}
			return "", errors.Trace(err)
		}
		key, found = svc.lookupBearerTokenKey(issuerHost, kid)
		if !found {
			return "", nil
		}
	}

	// Verify the JWT signature
	verified, err := jwt.Parse(bearerToken, func(t *jwt.Token) (any, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{"EdDSA"}))
	if err != nil {
		return "", nil
	}
	verifiedClaims, ok := verified.Claims.(jwt.MapClaims)
	if !ok || verifiedClaims == nil {
		return "", nil
	}

	// Mint an internal access token JWT with the validated claims
	accessToken, err = accesstokenapi.NewClient(svc).Mint(ctx, map[string]any(verifiedClaims))
	if err != nil {
		return "", errors.Trace(err)
	}
	return accessToken, nil
}
