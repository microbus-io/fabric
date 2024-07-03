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

package helloworld

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/service"

	"github.com/microbus-io/fabric/examples/helloworld/helloworldapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ service.Service
	_ *helloworldapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	App.Init(func(svc service.Service) {
		// Initialize all microservices
	})

	// Include all downstream microservices in the testing app
	App.Include(
		Svc.Init(func(svc *Service) {
			// Initialize the microservice under test
		}),
		// downstream.NewService().Init(func(svc *downstream.Service) {}),
	)

	err = App.Startup()
	if err != nil {
		return err
	}
	// All microservices are now running

	return nil
}

// Terminate shuts down the testing app.
func Terminate() (err error) {
	err = App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestHelloworld_HelloWorld(t *testing.T) {
	t.Parallel()
	/*
		ctx := Context()
		HelloWorld_Get(t, ctx, "").BodyContains(value)
		HelloWorld_Post(t, ctx, "", "", body).BodyContains(value)
		httpReq, _ := http.NewRequestWithContext(ctx, method, "?arg=val", body)
		HelloWorld(t, httpReq).BodyContains(value)
	*/

	ctx := Context()
	HelloWorld_Get(t, ctx, "").BodyContains("Hello, World!")
}
