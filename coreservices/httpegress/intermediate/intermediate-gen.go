/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Code generated by Microbus. DO NOT EDIT.

/*
Package intermediate serves as the foundation of the http.egress.sys microservice.

The HTTP egress microservice relays HTTP requests to the internet.
*/
package intermediate

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/openapi"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/sub"

	"gopkg.in/yaml.v3"

	"github.com/microbus-io/fabric/coreservices/httpegress/resources"
	"github.com/microbus-io/fabric/coreservices/httpegress/httpegressapi"
)

var (
	_ context.Context
	_ *embed.FS
	_ *json.Decoder
	_ fmt.Stringer
	_ *http.Request
	_ filepath.WalkFunc
	_ strconv.NumError
	_ strings.Reader
	_ time.Duration
	_ cfg.Option
	_ *errors.TracedError
	_ frame.Frame
	_ *httpx.ResponseRecorder
	_ *log.Field
	_ *openapi.Service
	_ service.Service
	_ sub.Option
	_ yaml.Encoder
	_ httpegressapi.Client
)

// ToDo defines the interface that the microservice must implement.
// The intermediate delegates handling to this interface.
type ToDo interface {
	OnStartup(ctx context.Context) (err error)
	OnShutdown(ctx context.Context) (err error)
	MakeRequest(w http.ResponseWriter, r *http.Request) (err error)
}

// Intermediate extends and customizes the generic base connector.
// Code generated microservices then extend the intermediate.
type Intermediate struct {
	*connector.Connector
	impl ToDo
}

// NewService creates a new intermediate service.
func NewService(impl ToDo, version int) *Intermediate {
	svc := &Intermediate{
		Connector: connector.New("http.egress.sys"),
		impl: impl,
	}
	svc.SetVersion(version)
	svc.SetDescription(`The HTTP egress microservice relays HTTP requests to the internet.`)
	
	// Lifecycle
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)

	// OpenAPI
	svc.Subscribe("GET", `:*/openapi.json`, svc.doOpenAPI)

	// Webs
	svc.Subscribe(`POST`, `:444/make-request`, svc.impl.MakeRequest)

	// Resources file system
	svc.SetResFS(resources.FS)

	return svc
}

// doOpenAPI renders the OpenAPI document of the microservice.
func (svc *Intermediate) doOpenAPI(w http.ResponseWriter, r *http.Request) error {
	oapiSvc := openapi.Service{
		ServiceName: svc.HostName(),
		Description: svc.Description(),
		Version:     svc.Version(),
		Endpoints:   []*openapi.Endpoint{},
		RemoteURI:   frame.Of(r).XForwardedFullURL(),
	}
	if r.URL.Port() == "444" || "444" == "*" {
		oapiSvc.Endpoints = append(oapiSvc.Endpoints, &openapi.Endpoint{
			Type:        `web`,
			Name:        `MakeRequest`,
			Method:      `POST`,
			Path:        `:444/make-request`,
			Summary:     `MakeRequest()`,
			Description: `MakeRequest proxies a request to a URL and returns the HTTP response, respecting the timeout set in the context.
The proxied request is expected to be posted in the body of the request in binary form (RFC7231).`,
			InputArgs: struct {
			}{},
			OutputArgs: struct {
			}{},
		})
	}

	if len(oapiSvc.Endpoints) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.MarshalIndent(&oapiSvc, "", "    ")
	if err != nil {
		return errors.Trace(err)
	}
	_, err = w.Write(b)
	return errors.Trace(err)
}

// doOnConfigChanged is called when the config of the microservice changes.
func (svc *Intermediate) doOnConfigChanged(ctx context.Context, changed func(string) bool) (err error) {
	return nil
}