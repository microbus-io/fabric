package httpingress

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/connector"
)

// Service is an HTTP ingress microservice
type Service struct {
	*connector.Connector
	httpPort   int
	httpServer *http.Server
}

// NewService creates a new HTTP ingress microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.NewConnector(),
		httpPort:  8080,
	}
	s.SetHostName("http.ingress.sys")
	s.SetOnStartup(s.OnStartup)
	s.SetOnShutdown(s.OnShutdown)

	return s
}

// OnStartup starts the web server
func (s *Service) OnStartup(_ context.Context) error {
	s.httpServer = &http.Server{
		Addr:    ":" + strconv.Itoa(s.httpPort),
		Handler: s,
	}
	s.LogInfo("Starting HTTP listener on port %d", s.httpPort)
	go s.httpServer.ListenAndServe()

	return nil
}

// OnShutdown stops the web server
func (s *Service) OnShutdown(_ context.Context) error {
	if s.httpServer != nil {
		s.LogInfo("Stopping HTTP listener on port %d", s.httpPort)
		err := s.httpServer.Close() // Not a graceful shutdown
		if err != nil {
			return err
		}
	}
	return nil
}

// ServeHTTP forwards incoming HTTP requests to the appropriate microservice on NATS.
// An incoming request http://localhost:8080/echo.example/echo is forwarded to
// the microservice at https://echo.example/echo
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Use the first segment of the URI as the host name to contact
	uri := r.URL.RequestURI()
	internalURL := "https:/" + uri

	// Skip favicon.ico to reduce noise
	if uri == "/favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	s.LogInfo("Request received: %s", internalURL)

	// Prepare the internal request
	internalReq, err := http.NewRequest(r.Method, internalURL, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		s.LogError(err)
		return
	}

	// Copy non-internal headers
	for hdrName, hdrVals := range r.Header {
		if !strings.HasPrefix(hdrName, "Microbus-") {
			for _, val := range hdrVals {
				internalReq.Header.Add(hdrName, val)
			}
		}
	}

	// Make the internal request over NATS
	internalRes, err := s.Request(internalReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		s.LogError(err)
		return
	}

	// Write back non-internal headers
	for hdrName, hdrVals := range internalRes.Header {
		if !strings.HasPrefix(hdrName, "Microbus-") {
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
			w.Write([]byte(err.Error()))
			s.LogError(err)
			return
		}
	}
}
