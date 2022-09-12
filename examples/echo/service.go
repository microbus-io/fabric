package echo

import (
	"bytes"
	"net/http"

	"github.com/microbus-io/fabric/connector"
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
	return s
}

// Echo back the incoming request in wire format
func (s *Service) Echo(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	err := r.Write(&buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(buf.Bytes())
}
