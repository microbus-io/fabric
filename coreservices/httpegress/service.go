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

package httpegress

import (
	"bufio"
	"context"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/trc"

	"github.com/microbus-io/fabric/coreservices/httpegress/httpegressapi"
	"github.com/microbus-io/fabric/coreservices/httpegress/intermediate"
)

var (
	_ context.Context
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *httpegressapi.Client
)

/*
Service implements the http.egress.core microservice.

The HTTP egress microservice relays HTTP requests to the internet.
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
MakeRequest proxies a request to a URL and returns the HTTP response, respecting the timeout set in the context.
The proxied request is expected to be posted in the body of the request in binary form (RFC7231).
*/
func (svc *Service) MakeRequest(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()
	req, err := http.ReadRequest(bufio.NewReaderSize(r.Body, 64))
	if err != nil {
		return errors.Trace(err)
	}
	if req.URL.Port() == "" {
		if req.URL.Scheme == "https" {
			req.URL.Host += ":443"
		} else if req.URL.Scheme == "http" {
			req.URL.Host += ":80"
		}
	}
	req.RequestURI = "" // Avoid "http: Request.RequestURI can't be set in client requests"

	// OpenTelemetry: create a child span
	spanOptions := []trc.Option{
		trc.Client(),
		// Do not record the request attributes yet because they take a lot of memory, they will be added if there's an error
	}
	if svc.Deployment() == connector.LOCAL {
		// Add the request attributes in LOCAL deployment to facilitate debugging
		spanOptions = append(spanOptions, trc.Request(r))
	}
	_, span := svc.StartSpan(ctx, req.URL.Hostname(), spanOptions...)
	defer span.End()

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// OpenTelemetry: record the error, adding the request attributes
		span.SetRequest(req)
		span.SetError(err)
		svc.ForceTrace(ctx)
		return errors.Trace(err)
	}
	err = httpx.Copy(w, resp)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
