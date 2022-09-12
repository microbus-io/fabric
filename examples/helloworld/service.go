package helloworld

import (
	"net/http"

	"github.com/microbus-io/fabric/connector"
)

// Service is a hello world microservice
type Service struct {
	*connector.Connector
}

// NewService creates a new hello world microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.NewConnector(),
	}
	s.SetHostName("helloworld.example")
	s.Subscribe(443, "/hello", s.Hello)
	return s
}

// Hello prints a greeting
func (s *Service) Hello(w http.ResponseWriter, r *http.Request) {
	// If a name is provided, add a personal touch
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	// Print the greeting
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Hello, " + name + "!"))
}
