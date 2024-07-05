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

package control

import (
	"context"
	"net/http"

	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/coreservices/control/intermediate"
)

var (
	_ errors.TracedError
	_ http.Request
)

/*
Service implements the control.core microservice.

This microservice is created for the sake of generating the client API for the :888 control subscriptions.
The microservice itself does nothing and should not be included in applications.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return errors.New("unstartable")
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
Ping responds to the message with a pong.
*/
func (svc *Service) Ping(ctx context.Context) (pong int, err error) {
	return 0, nil
}

/*
ConfigRefresh pulls the latest config values from the configurator service.
*/
func (svc *Service) ConfigRefresh(ctx context.Context) (err error) {
	return nil
}

/*
Trace forces exporting the indicated tracing span.
*/
func (svc *Service) Trace(ctx context.Context, id string) (err error) {
	return nil
}
