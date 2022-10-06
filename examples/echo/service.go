package echo

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
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
	s.Subscribe("/echo", s.Echo)
	s.Subscribe("/who", s.Who)
	s.Subscribe("/ping", s.Ping)
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
	msg := fmt.Sprintf(`
Request from instance %s of host %s
Handled by instance %s of host %s

Refresh the page to try again`, frame.Of(r).FromID(), frame.Of(r).FromHost(), s.ID(), s.HostName())
	w.Write([]byte(msg))
	return nil
}

// Ping all microservices and list them
func (s *Service) Ping(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer
	ch := s.Publish(
		r.Context(),
		pub.GET("https://all:888/ping"),
		pub.Multicast(),
	)
	for i := range ch {
		res, err := i.Get()
		if err == nil {
			fromHost := frame.Of(res).FromHost()
			fromID := frame.Of(res).FromID()
			buf.WriteString(fromID)
			buf.WriteString(".")
			buf.WriteString(fromHost)
			buf.WriteString("\r\n")
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(buf.Bytes())
	return nil
}
