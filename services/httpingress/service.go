package httpingress

import (
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
	svc.stopHTTPServers(ctx)
	err = svc.startHTTPServers(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// stopHTTPServers stop the running HTTP servers.
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
			ReadHeaderTimeout: time.Minute,
			ReadTimeout:       time.Minute * 5, // Enough for 750MB at 20Mbps
			WriteTimeout:      time.Minute * 5,
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
	err := svc.serveHTTP(w, r)
	if err != nil {
		uri := r.URL.RequestURI()
		statusCode := errors.Convert(err).StatusCode
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				statusCode = http.StatusRequestEntityTooLarge
			}
		}
		w.WriteHeader(statusCode)
		if svc.Deployment() != connector.PROD {
			w.Write([]byte(fmt.Sprintf("%+v", err)))
		} else {
			w.Write([]byte(http.StatusText(statusCode)))
		}
		svc.LogError(ctx, "Serving", log.Error(err), log.String("uri", uri))
	}
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
			err := errors.New("no root")
			errors.Convert(err).StatusCode = http.StatusNotFound
			return errors.Trace(err)
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
			err := errors.New("disallowed origin", origin)
			errors.Convert(err).StatusCode = http.StatusForbidden
			return errors.Trace(err)
		}
	}

	// Use the first segment of the URI as the host name to contact
	u := resolveInternalURL(r.URL, svc.portMappings)
	internalURL := u.String()
	internalHost := strings.Split(uri, "/")[1]

	svc.LogInfo(ctx, "Request received", log.String("url", internalURL))

	// Prepare the internal request options
	r.Body = http.MaxBytesReader(w, r.Body, int64(svc.MaxBodySize()))
	defer r.Body.Close()
	options := []pub.Option{
		pub.Method(r.Method),
		pub.URL(internalURL),
		pub.Body(r.Body),
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
