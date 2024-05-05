/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpegress

import (
	"bufio"
	"context"
	"net/http"
	"time"

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
Service implements the http.egress.sys microservice.

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
MakeRequest makes a request to a URL given the raw HTTP request and returns the HTTP response, respecting the timeout set in the context.
*/
func (svc *Service) MakeRequest(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()
	req, err := http.ReadRequest(bufio.NewReader(r.Body))
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
	_, span := svc.StartSpan(ctx, req.URL.Hostname(), trc.Client())
	span.SetRequest(req)
	defer span.End()
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		span.SetError(err)
		return errors.Trace(err)
	}
	err = httpx.Copy(w, resp)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
