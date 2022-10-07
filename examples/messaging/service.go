package messaging

import (
	"bytes"
	"io"
	"net/http"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
)

// Service is the messaging.example microservice
type Service struct {
	*connector.Connector
}

// NewService creates a new messaging.example microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.NewConnector(),
	}
	s.SetHostName("messaging.example")
	s.Subscribe("/no-queue", s.NoQueue, sub.NoQueue())
	s.Subscribe("/default-queue", s.DefaultQueue)
	s.Subscribe("/home", s.Home)
	return s
}

// NoQueue demonstrates how the NoQueue subscription option is used to create
// a pub/sub communication pattern.
// All instances of this microservice will respond to each request
func (s *Service) NoQueue(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("NoQueue " + s.ID()))
	return nil
}

// DefaultQueue demonstrates how the DefaultQueue subscription option is used to create
// a request/response communication pattern.
// Only one of the instances of this microservice will respond to each request
func (s *Service) DefaultQueue(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("DefaultQueue " + s.ID()))
	return nil
}

// Home demonstrates making requests using multicast pub/sub and request/response patterns
func (s *Service) Home(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer

	// Print the ID of this instance
	// A random instance of this microservice will process this request
	buf.WriteString("Processed by: ")
	buf.WriteString(s.ID())
	buf.WriteString("\r\n\r\n")

	// Make a standard unicast request/response call to the /default-queue endpoint
	// A random instance of this microservice will reply
	res, err := s.Request(r.Context(), pub.GET("https://messaging.example/default-queue"))
	if err != nil {
		return errors.Trace(err)
	}
	buf.WriteString("Request/response (unicast):\r\n")
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Trace(err)
	}
	buf.WriteString("> ")
	buf.Write(b)
	buf.WriteString("\r\n\r\n")

	// Make a multicast pub/sub call to the /no-queue endpoint
	// All instances of this microservice will reply
	ch := s.Publish(r.Context(), pub.GET("https://messaging.example/no-queue"))
	buf.WriteString("Pub/sub (multicast):\r\n")
	lastResponderID := ""
	for i := range ch {
		res, err := i.Get()
		if err != nil {
			return errors.Trace(err)
		}
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return errors.Trace(err)
		}
		buf.WriteString("> ")
		buf.Write(b)
		buf.WriteString("\r\n")

		lastResponderID = frame.Of(res).FromID()
	}
	buf.WriteString("\r\n")

	// Make a direct request to a specific instance
	ch = s.Publish(r.Context(), pub.GET("https://"+lastResponderID+".messaging.example/no-queue"))
	if err != nil {
		return errors.Trace(err)
	}
	buf.WriteString("Direct addressing (unicast):\r\n")
	for i := range ch {
		res, err := i.Get()
		if err != nil {
			return errors.Trace(err)
		}
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return errors.Trace(err)
		}
		buf.WriteString("> ")
		buf.Write(b)
		buf.WriteString("\r\n")
	}
	buf.WriteString("\r\n")

	buf.WriteString("Refresh the page to try again")

	_, err = w.Write(buf.Bytes())
	return errors.Trace(err)
}
