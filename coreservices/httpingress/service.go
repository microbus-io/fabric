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

	"github.com/andybalholm/brotli"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/lru"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/fabric/utils"
	"golang.org/x/text/language"

	"go.opentelemetry.io/otel/propagation"

	"github.com/microbus-io/fabric/coreservices/httpingress/intermediate"
	"github.com/microbus-io/fabric/coreservices/metrics/metricsapi"
)

/*
Service implements the http.ingress.core microservice.

The HTTP ingress microservice relays incoming HTTP requests to the NATS bus.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	httpServers     map[int]*http.Server
	mux             sync.Mutex
	allowedOrigins  map[string]bool
	portMappings    map[string]string
	reqMemoryUsed   int64
	secure443       bool
	blockedPaths    map[string]bool
	languageMatcher language.Matcher
	langMatchCache  *lru.Cache[string, string]
	middleware      []Middleware
	handler         connector.HTTPHandler
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	svc.langMatchCache = lru.NewCache[string, string]()
	svc.OnChangedAllowedOrigins(ctx)
	svc.OnChangedPortMappings(ctx)
	svc.OnChangedBlockedPaths(ctx)
	svc.OnChangedServerLanguages(ctx)

	// Setup the middleware chain
	svc.handler = svc.serveHTTP
	for h := len(svc.middleware) - 1; h >= 0; h-- {
		mw := svc.middleware[h]
		svc.handler = func(w http.ResponseWriter, r *http.Request) error {
			return mw.Serve(w, r, svc.serveHTTP) // No trace
		}
	}

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

// AddMiddleware adds a [Middleware] to the chain of middleware.
// Middleware is a processor that can be added to pre- or post-process a request.
// Middlewares are chained together. Each receives the request after it was processed by the preceding (upstream) middleware,
// passing it along to the next (downstream) one. And conversely, each receives the response from the next (downstream) middleware,
// and passes it back to the preceding (upstream) middleware. The request and/or response may be modified in the process.
// A middleware should call the next function in the chain.
func (svc *Service) AddMiddleware(middleware ...Middleware) error {
	if svc.IsStarted() {
		return errors.New("middleware can't be added after starting up")
	}
	svc.middleware = append(svc.middleware, middleware...)
	return nil
}

// AddMiddleware creates a middleware for each [MiddlewareFunc] and adds them to the chain of middleware.
func (svc *Service) AddMiddlewareFunc(handler ...MiddlewareFunc) error {
	var middleware []Middleware
	for _, h := range handler {
		middleware = append(middleware, &simpleMiddleware{
			f: h,
		})
	}
	return svc.AddMiddleware(middleware...)
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

// ServeHTTP forwards incoming HTTP requests to the appropriate microservice on NATS.
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

	// Do not accept internal headers
	for h := range r.Header {
		if strings.HasPrefix(h, frame.HeaderPrefix) {
			r.Header.Del(h)
		}
	}

	ww := httpx.NewResponseRecorder() // This recorder allows modifying the response after it was written
	err := utils.CatchPanic(func() error {
		_ = r.URL.Port() // Validates the port (may panic in malformed requests)
		return svc.handler(ww, r)
	})

	// Do not leak internal headers
	for h := range ww.Header() {
		if strings.HasPrefix(h, frame.HeaderPrefix) {
			ww.Header().Del(h)
		}
	}

	if err != nil {
		var urlStr string
		utils.CatchPanic(func() error {
			urlStr = r.URL.String()
			if len(urlStr) > 2048 {
				urlStr = urlStr[:2048] + "..."
			}
			return nil
		})
		statusCode := errors.StatusCode(err)
		if statusCode <= 0 || statusCode >= 1000 {
			statusCode = http.StatusInternalServerError
		}
		ww.Clear()
		ww.Header().Set("Content-Type", "text/plain")
		ww.WriteHeader(statusCode)
		// Do not leak error details and stack trace
		if svc.Deployment() == connector.LOCAL && httpx.IsLocalhostAddress(r) {
			ww.Write([]byte(fmt.Sprintf("%+v\n\n{%s}", err, span.TraceID())))
		} else {
			ww.Write([]byte(http.StatusText(statusCode) + " {" + span.TraceID() + "}"))
		}
		logFunc := svc.LogError
		if statusCode == http.StatusNotFound {
			logFunc = svc.LogInfo
		} else if statusCode < 500 {
			logFunc = svc.LogWarn
		}
		logFunc(ctx, "Serving",
			"error", err,
			"url", urlStr,
			"status", statusCode,
		)

		// OpenTelemetry: record the error, adding the request attributes
		span.SetRequest(r)
		span.SetError(err)
		svc.ForceTrace(ctx)
	} else {
		// OpenTelemetry: record the status code and content length
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

	// Blocked paths
	if svc.blockedPaths[r.URL.Path] {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	dot := strings.LastIndex(r.URL.Path, ".")
	if dot >= 0 && svc.blockedPaths["*"+r.URL.Path[dot:]] {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	// Detect root path
	origPath := r.URL.Path
	if !strings.HasPrefix(r.URL.Path, "/") {
		r.URL.Path = "/" + r.URL.Path
	}
	if r.URL.Path == "/" {
		r.URL.Path = "/root"
	}

	// CORS
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
	origin := r.Header.Get("Origin")
	if origin != "" {
		// Block disallowed origins
		if !svc.allowedOrigins["*"] && !svc.allowedOrigins[origin] {
			return errors.Newcf(http.StatusForbidden, "disallowed origin '%s", origin)
		}
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		if r.Method == "OPTIONS" {
			// CORS preflight requests are returned empty
			w.WriteHeader(http.StatusNoContent)
			return nil
		}
	}

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
	internalHost := u.Host
	metrics := internalHost == metricsapi.Hostname || strings.HasPrefix(internalHost, metricsapi.Hostname+":")
	if !metrics {
		if internalHost != "favicon.ico" {
			svc.LogInfo(ctx, "Request received",
				"url", internalURL,
			)
		}
		// Automatically redirect HTTP port 80 to HTTPS port 443
		if svc.secure443 && r.TLS == nil && r.URL.Port() == "80" {
			u := *r.URL
			u.Scheme = "https"
			u.Host = strings.TrimSuffix(u.Host, ":80")
			s := u.String()
			http.Redirect(w, r, s, http.StatusTemporaryRedirect)
			return nil
		}
	}

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
		pub.CopyHeaders(r.Header),     // Copy all headers
		pub.ContentLength(len(body)),  // Overwrite the Content-Length header
		pub.Header("Traceparent", ""), // Disallowed header
		pub.Header("Tracestate", ""),  // Disallowed header
	}

	// Add the time budget to the request headers and set it as the context's timeout
	delegateCtx := ctx
	timeBudget := svc.TimeBudget()
	if r.Header.Get("Request-Timeout") != "" {
		headerTimeoutSecs, err := strconv.Atoi(r.Header.Get("Request-Timeout"))
		if err == nil {
			timeBudget = time.Duration(headerTimeoutSecs) * time.Second
		}
	}
	if timeBudget > 0 {
		var cancel context.CancelFunc
		delegateCtx, cancel = context.WithTimeout(ctx, timeBudget)
		defer cancel()
	}

	// Set proxy headers, if there's no upstream proxy
	if r.Header.Get("X-Forwarded-Host") == "" {
		options = append(options, pub.Header("X-Forwarded-Host", r.Host))
		options = append(options, pub.Header("X-Forwarded-For", r.RemoteAddr))
		if r.TLS != nil {
			options = append(options, pub.Header("X-Forwarded-Proto", "https"))
		} else {
			options = append(options, pub.Header("X-Forwarded-Proto", "http"))
		}
		options = append(options, pub.Header("X-Forwarded-Prefix", ""))
	}
	options = append(options, pub.Header("X-Forwarded-Path", origPath))

	// Match request against server languages and override the header to the best match
	bestLang := svc.matchBestLanguage(r)
	if bestLang != "" {
		options = append(options, pub.Header("Accept-Language", bestLang))
	}

	// OpenTelemetry: pass the span in the headers
	carrier := make(propagation.HeaderCarrier)
	propagation.TraceContext{}.Inject(ctx, carrier)
	for k, v := range carrier {
		options = append(options, pub.Header(k, v[0]))
	}

	// Delegate the request over NATS
	internalRes, err := svc.Request(delegateCtx, options...)
	if err != nil {
		return err // No trace
	}

	// Write back headers
	for hdrName, hdrVals := range internalRes.Header {
		for _, val := range hdrVals {
			w.Header().Add(hdrName, val)
		}
	}

	// No caching by default
	if internalRes.Header.Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-store")
	}

	// Compress textual content using gzip or deflate
	var closer io.Closer
	var writer io.Writer
	writer = w
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if acceptEncoding != "" && internalRes.Body != nil {
		contentType := internalRes.Header.Get("Content-Type")
		contentEncoding := internalRes.Header.Get("Content-Encoding")
		if contentEncoding == "" {
			contentEncoding = "identity"
		}
		contentLength, _ := strconv.Atoi(internalRes.Header.Get("Content-Length"))
		if contentLength >= 4*1024 && contentEncoding == "identity" &&
			!strings.HasPrefix(contentType, "image/") &&
			!strings.HasPrefix(contentType, "video/") &&
			!strings.HasPrefix(contentType, "audio/") {
			if strings.Contains(acceptEncoding, "br") {
				w.Header().Del("Content-Length")
				w.Header().Set("Content-Encoding", "br")
				brot := brotli.NewWriter(w)
				writer = brot
				closer = brot
			} else if strings.Contains(acceptEncoding, "deflate") {
				w.Header().Del("Content-Length")
				w.Header().Set("Content-Encoding", "deflate")
				deflater, _ := flate.NewWriter(w, flate.DefaultCompression)
				writer = deflater
				closer = deflater
			} else if strings.Contains(acceptEncoding, "gzip") {
				w.Header().Del("Content-Length")
				w.Header().Set("Content-Encoding", "gzip")
				gzipper := gzip.NewWriter(w)
				writer = gzipper
				closer = gzipper
			}
		}
	}

	// Write back the status code
	if internalRes.StatusCode != 0 {
		w.WriteHeader(internalRes.StatusCode)
	}

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

// matchBestLanguage returns the server language best matching the Accept-Language header of the request.
func (svc *Service) matchBestLanguage(r *http.Request) string {
	if svc.languageMatcher == nil {
		return ""
	}
	hdrVal := r.Header.Get("Accept-Language")
	if cached, ok := svc.langMatchCache.Load(hdrVal); ok {
		return cached
	}
	langs := frame.Of(r).Languages()
	langTags := make([]language.Tag, 0, len(langs))
	for _, l := range langs {
		langTags = append(langTags, language.Make(l))
	}
	bestLang, _, _ := svc.languageMatcher.Match(langTags...)
	bestLangStr := bestLang.String()
	p := strings.Index(bestLangStr, "-u")
	if p > 0 {
		bestLangStr = bestLangStr[:p]
	}
	svc.langMatchCache.Store(hdrVal, bestLangStr)
	return bestLangStr
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

// OnChangedServerLanguages is triggered when the value of the ServerLanguages config property changes.
func (svc *Service) OnChangedServerLanguages(ctx context.Context) (err error) {
	value := svc.ServerLanguages()
	if value == "" {
		svc.languageMatcher = nil
		svc.langMatchCache.Clear()
		return nil
	}
	tags := []language.Tag{}
	for _, lang := range strings.Split(value, ",") {
		lang = strings.TrimSpace(lang)
		if lang == "" {
			continue
		}
		tags = append(tags, language.Make(lang))
	}
	svc.languageMatcher = language.NewMatcher(tags)
	svc.langMatchCache.Clear()
	return nil
}
