package messaging

import (
	"bytes"
	"io"
	"net/http"
	"strings"

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
		Connector: connector.New("messaging.example"),
	}
	s.SetDescription("The Messaging microservice demonstrates service-to-service communication patterns.")
	s.Subscribe("/no-queue", s.NoQueue, sub.NoQueue())
	s.Subscribe("/default-queue", s.DefaultQueue)
	s.Subscribe("/home", s.Home)
	s.Subscribe("/cache/load", s.CacheLoad)
	s.Subscribe("/cache/store", s.CacheStore)
	return s
}

// NoQueue demonstrates how the NoQueue subscription option is used to create
// a multicast request/response communication pattern.
// All instances of this microservice will respond to each request
func (s *Service) NoQueue(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("NoQueue " + s.ID()))
	return nil
}

// DefaultQueue demonstrates how the DefaultQueue subscription option is used to create
// a unicast request/response communication pattern.
// Only one of the instances of this microservice will respond to each request
func (s *Service) DefaultQueue(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("DefaultQueue " + s.ID()))
	return nil
}

// Home demonstrates making requests using multicast and unicast request/response patterns
func (s *Service) Home(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer

	// Print the ID of this instance
	// A random instance of this microservice will process this request
	buf.WriteString("Processed by: ")
	buf.WriteString(s.ID())
	buf.WriteString("\r\n\r\n")

	// Make a standard unicast request to the /default-queue endpoint
	// A random instance of this microservice will respond, effectively load balancing among the instances
	res, err := s.Request(r.Context(), pub.GET("https://messaging.example/default-queue"))
	if err != nil {
		return errors.Trace(err)
	}
	responderID := frame.Of(res).FromID()
	buf.WriteString("Unicast\r\n")
	buf.WriteString("GET https://messaging.example/default-queue\r\n")
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Trace(err)
	}
	buf.WriteString("> ")
	buf.Write(b)
	buf.WriteString("\r\n\r\n")

	// Make a direct addressing unicast request to the /default-queue endpoint
	// The specific instance will always respond, circumventing load balancing
	res, err = s.Request(r.Context(), pub.GET("https://"+responderID+".messaging.example/default-queue"))
	if err != nil {
		return errors.Trace(err)
	}
	buf.WriteString("Direct addressing unicast\r\n")
	buf.WriteString("GET https://" + responderID + ".messaging.example/default-queue\r\n")
	b, err = io.ReadAll(res.Body)
	if err != nil {
		return errors.Trace(err)
	}
	buf.WriteString("> ")
	buf.Write(b)
	buf.WriteString("\r\n\r\n")

	// Make a multicast request call to the /no-queue endpoint
	// All instances of this microservice will respond
	ch := s.Publish(r.Context(), pub.GET("https://messaging.example/no-queue"))
	buf.WriteString("Multicast\r\n")
	buf.WriteString("GET https://messaging.example/no-queue\r\n")
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

	// Make a direct addressing request to the /no-queue endpoint
	// Only the specific instance will respond
	ch = s.Publish(r.Context(), pub.GET("https://"+lastResponderID+".messaging.example/no-queue"))
	if err != nil {
		return errors.Trace(err)
	}
	buf.WriteString("Direct addressing multicast\r\n")
	buf.WriteString("GET https://" + lastResponderID + ".messaging.example/no-queue\r\n")
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

	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write(buf.Bytes())
	return errors.Trace(err)
}

// CacheStore stores an element in the distributed cache of the microservice.
func (s *Service) CacheStore(w http.ResponseWriter, r *http.Request) error {
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	value := r.URL.Query().Get("value")
	if value == "" {
		return errors.New("missing value")
	}
	err := s.DistribCache().Store(r.Context(), key, []byte(value))
	if err != nil {
		return errors.Trace(err)
	}

	var b strings.Builder
	b.WriteString("key: ")
	b.WriteString(key)
	b.WriteString("\nvalue: ")
	b.WriteString(value)
	b.WriteString("\n\nStored by ")
	b.WriteString(s.ID())

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(b.String()))
	return nil
}

// CacheLoad looks up an element in the distributed cache of the microservice.
func (s *Service) CacheLoad(w http.ResponseWriter, r *http.Request) error {
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	value, ok, err := s.DistribCache().Load(r.Context(), key)
	if err != nil {
		return errors.Trace(err)
	}

	var b strings.Builder
	b.WriteString("key: ")
	b.WriteString(key)
	if ok {
		b.WriteString("\nfound: yes")
		b.WriteString("\nvalue: ")
		b.Write(value)
	} else {
		b.WriteString("\nfound: no")
	}
	b.WriteString("\n\nLoaded by ")
	b.WriteString(s.ID())

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(b.String()))
	return nil
}
