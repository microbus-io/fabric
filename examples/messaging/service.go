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

package messaging

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"

	"github.com/microbus-io/fabric/examples/messaging/intermediate"
)

var (
	_ errors.TracedError
	_ http.Request
)

/*
Service implements the messaging.example microservice.

The Messaging microservice demonstrates service-to-service communication patterns.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
Home demonstrates making requests using multicast and unicast request/response patterns.
*/
func (svc *Service) Home(w http.ResponseWriter, r *http.Request) (err error) {
	var buf bytes.Buffer

	// Print the ID of this instance
	// A random instance of this microservice will process this request
	buf.WriteString("Processed by: ")
	buf.WriteString(svc.ID())
	buf.WriteString("\r\n\r\n")

	// Make a standard unicast request to the /default-queue endpoint
	// A random instance of this microservice will respond, effectively load balancing among the instances
	res, err := svc.Request(r.Context(), pub.GET("https://messaging.example/default-queue"))
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
	res, err = svc.Request(r.Context(), pub.GET("https://"+responderID+".messaging.example/default-queue"))
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
	ch := svc.Publish(r.Context(), pub.GET("https://messaging.example/no-queue"))
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
	ch = svc.Publish(r.Context(), pub.GET("https://"+lastResponderID+".messaging.example/no-queue"))
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
	w.Write(buf.Bytes())
	return nil
}

/*
NoQueue demonstrates how the NoQueue subscription option is used to create
a multicast request/response communication pattern.
All instances of this microservice will respond to each request.
*/
func (svc *Service) NoQueue(w http.ResponseWriter, r *http.Request) (err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("NoQueue " + svc.ID()))
	return nil
}

/*
DefaultQueue demonstrates how the DefaultQueue subscription option is used to create
a unicast request/response communication pattern.
Only one of the instances of this microservice will respond to each request.
*/
func (svc *Service) DefaultQueue(w http.ResponseWriter, r *http.Request) (err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("DefaultQueue " + svc.ID()))
	return nil
}

/*
CacheLoad looks up an element in the distributed cache of the microservice.
*/
func (svc *Service) CacheLoad(w http.ResponseWriter, r *http.Request) (err error) {
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	value, ok, err := svc.DistribCache().Load(r.Context(), key)
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
	b.WriteString(svc.ID())

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(b.String()))
	return nil
}

/*
CacheStore stores an element in the distributed cache of the microservice.
*/
func (svc *Service) CacheStore(w http.ResponseWriter, r *http.Request) (err error) {
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	value := r.URL.Query().Get("value")
	if value == "" {
		return errors.New("missing value")
	}
	err = svc.DistribCache().Store(r.Context(), key, []byte(value))
	if err != nil {
		return errors.Trace(err)
	}

	var b strings.Builder
	b.WriteString("key: ")
	b.WriteString(key)
	b.WriteString("\nvalue: ")
	b.WriteString(value)
	b.WriteString("\n\nStored by ")
	b.WriteString(svc.ID())

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(b.String()))
	return nil
}
