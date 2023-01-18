/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
	"compress/flate"
	"compress/gzip"
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

	httpServers    map[int]*http.Server
	mux            sync.Mutex
	allowedOrigins map[string]bool
	portMappings   map[string]string
	reqMemoryUsed  int64
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	svc.OnChangedAllowedOrigins(ctx)
	svc.OnChangedPortMappings(ctx)
	err = svc.startHTTPServers(ctx)
	if err != nil {
		return errors.Trace(err)
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
			svc.LogError(ctx, "Stopping HTTP listener", log.Int("port", httpPort), log.Error(lastErr))
		} else {
			svc.LogInfo(ctx, "Stopped HTTP listener", log.Int("port", httpPort))
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
			svc.LogError(ctx, "Starting HTTP listener", log.Int("port", portInt), log.Error(err))
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
		}
		svc.httpServers[portInt] = httpServer
		if secure {
			go httpServer.ListenAndServeTLS(certFile, keyFile)
		} else {
			go httpServer.ListenAndServe()
		}
		svc.LogInfo(ctx, "Started HTTP listener", log.Int("port", portInt), log.Bool("secure", secure))
	}
	return nil
}

// ServeHTTP forwards incoming HTTP requests to the appropriate microservice on NATS.
// An incoming request http://localhost:8080/echo.example/echo is forwarded to
// the microservice at https://echo.example/echo .
// ServeHTTP implements the http.Handler interface
func (svc *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := svc.Lifetime()
	handlerStartTime := time.Now()
	pt := &PassThrough{W: w}
	err := svc.serveHTTP(pt, r)
	if err != nil {
		uri := r.URL.RequestURI()
		statusCode := errors.Convert(err).StatusCode
		if statusCode <= 0 || statusCode >= 1000 {
			statusCode = http.StatusInternalServerError
		}
		w.WriteHeader(statusCode)
		if svc.Deployment() != connector.PROD {
			w.Write([]byte(fmt.Sprintf("%+v", err)))
		} else {
			w.Write([]byte(http.StatusText(statusCode)))
		}
		svc.LogError(ctx, "Serving", log.Error(err), log.String("uri", uri))
	}

	// Meter
	_ = svc.ObserveMetric(
		"microbus_response_duration_seconds",
		time.Since(handlerStartTime).Seconds(),
		"ServeHTTP",
		r.URL.Port(),
		r.Method,
		strconv.Itoa(pt.SC),
		func() string {
			if err != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)
	_ = svc.ObserveMetric(
		"microbus_response_size_bytes",
		float64(pt.N),
		"ServeHTTP",
		r.URL.Port(),
		r.Method,
		strconv.Itoa(pt.SC),
		func() string {
			if err != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)
}

// ServeHTTP forwards incoming HTTP requests to the appropriate microservice on NATS.
// An incoming request http://localhost:8080/echo.example/echo is forwarded to
// the microservice at https://echo.example/echo .
// ServeHTTP implements the http.Handler interface
func (svc *Service) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	ctx := svc.Lifetime()

	// Fill in the gaps
	r.URL.Host = r.Host
	if !strings.Contains(r.Host, ":") {
		if r.TLS != nil {
			r.URL.Host += ":443"
		} else {
			r.URL.Host += ":80"
		}
	}

	// Skip favicon.ico to reduce noise
	uri := r.URL.RequestURI()
	if uri == "/favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	// Handle the root path
	if uri == "/" {
		redirectTo := svc.RedirectRoot()
		if redirectTo == "" {
			return errors.Newc(http.StatusNotFound, "no root")
		}
		redirectTo = strings.TrimPrefix(redirectTo, "https:/")
		redirectTo = strings.TrimPrefix(redirectTo, "http:/")
		prefix := r.Header.Get("X-Forwarded-Prefix")
		http.Redirect(w, r, prefix+redirectTo, http.StatusFound)
		return nil
	}

	// Block disallowed origins
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
	origin := r.Header.Get("Origin")
	if origin != "" {
		if !svc.allowedOrigins["*"] && !svc.allowedOrigins[origin] {
			return errors.Newc(http.StatusForbidden, "disallowed origin", origin)
		}
	}

	// Use the first segment of the URI as the host name to contact
	u := resolveInternalURL(r.URL, svc.portMappings)
	internalURL := u.String()
	internalHost := strings.Split(uri, "/")[1]

	svc.LogInfo(ctx, "Request received", log.String("url", internalURL))

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
	}

	// Add the time budget to the request headers and set it as the context's timeout
	delegateCtx := ctx
	timeBudget := svc.TimeBudget()
	if timeBudget > 0 {
		options = append(options, pub.TimeBudget(timeBudget))
		var cancel context.CancelFunc
		delegateCtx, cancel = svc.Clock().WithTimeout(ctx, timeBudget)
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

	// Overwrite the Content-Length header
	options = append(options, pub.ContentLength(len(body)))

	// Set proxy headers, if there's no upstream proxy
	if r.Header.Get("X-Forwarded-Host") == "" {
		options = append(options, pub.Header("X-Forwarded-Host", r.Host))
		options = append(options, pub.Header("X-Forwarded-For", r.RemoteAddr))
		if r.TLS != nil {
			options = append(options, pub.Header("X-Forwarded-Proto", "https"))
		} else {
			options = append(options, pub.Header("X-Forwarded-Proto", "http"))
		}
	}
	prefix := r.Header.Get("X-Forwarded-Prefix")
	options = append(options, pub.Header("X-Forwarded-Prefix", prefix+"/"+internalHost))

	// Authentication token
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(authz, "Bearer ") {
		options = append(options, pub.Header(frame.HeaderAuthToken, strings.TrimPrefix(authz, "Bearer")))
	} else {
		for _, cookie := range r.Cookies() {
			if strings.ToLower(cookie.Name) == "authtoken" || strings.ToLower(cookie.Name) == "token" {
				options = append(options, pub.Header(frame.HeaderAuthToken, cookie.Value))
				break
			}
		}
	}

	// Delegate the request over NATS
	internalRes, err := svc.Request(delegateCtx, options...)
	if err != nil {
		return errors.Trace(err)
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

	// CORS headers
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "*")
	}

	// Compress textual content using gzip or deflate
	var writer io.Writer
	writer = w
	var closer io.Closer
	if internalRes.Body != nil {
		contentType := internalRes.Header.Get("Content-Type")
		contentEncoding := internalRes.Header.Get("Content-Encoding")
		if (strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "application/json")) &&
			(contentEncoding == "" || contentEncoding == "identity") {
			acceptEncoding := r.Header.Get("Accept-Encoding")
			if strings.Contains(acceptEncoding, "gzip") {
				w.Header().Del("Content-Length")
				w.Header().Set("Content-Encoding", "gzip")
				gzipper := gzip.NewWriter(w)
				writer = gzipper
				closer = gzipper
			} else if strings.Contains(acceptEncoding, "deflate") {
				w.Header().Del("Content-Length")
				w.Header().Set("Content-Encoding", "deflate")
				deflater, _ := flate.NewWriter(w, flate.DefaultCompression)
				writer = deflater
				closer = deflater
			}
		}
	}

	// Write back the status code
	w.WriteHeader(internalRes.StatusCode)

	// Write back the body
	if internalRes.Body != nil {
		_, err = io.Copy(writer, internalRes.Body)
		if closer != nil {
			closer.Close()
		}
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
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
		q := strings.Index(m, "->")
		if q < 0 {
			svc.LogWarn(ctx, "Invalid port mapping", log.String("mapping", m))
			continue
		}
		newMappings[m[:q]] = m[q+2:]
	}
	svc.portMappings = newMappings
	return nil
}

// resolveInternalURL resolves the NATS URL from the external URL.
func resolveInternalURL(externalURL *url.URL, portMappings map[string]string) (natsURL *url.URL) {
	externalPort := externalURL.Port()
	if externalPort == "" {
		externalPort = "443"
	}
	u, _ := url.Parse("https:/" + externalURL.RequestURI()) // First part of the URL is the internal host
	internalPort := u.Port()
	if internalPort == "" {
		internalPort = "443"
	}
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
	return u
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
