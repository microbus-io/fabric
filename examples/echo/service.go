package echo

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
)

// Service is an echo microservice
type Service struct {
	*connector.Connector
}

// NewService creates a new echo microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.NewConnector(),
	}
	s.SetHostName("echo.example")
	s.Subscribe(443, "/echo", s.Echo)
	s.Subscribe(443, "/who", s.Who)
	return s
}

// Echo back the incoming request in wire format
func (s *Service) Echo(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer
	err := r.Write(&buf)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(buf.Bytes())
	return nil
}

// Who prints the identity of this microservice
func (s *Service) Who(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain")
	msg := fmt.Sprintf("Handled by instance %s of host %s\n\nRefresh the page to try again", s.ID(), s.HostName())
	w.Write([]byte(msg))
	return nil
}
