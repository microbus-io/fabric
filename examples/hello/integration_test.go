/*
Copyright 2023 Microbus LLC and various contributors

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

package hello

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/calculator"
	"github.com/microbus-io/fabric/examples/hello/helloapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *helloapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
	// Include all downstream microservices in the testing app
	// Use .With(options) to initialize with appropriate config values
	App.Include(
		Svc.With(
			Greeting("Ciao"),
			Repeat(5),
		),
		calculator.NewService(),
	)

	err := App.Startup()
	if err != nil {
		return err
	}

	// You may call any of the microservices after the app is started

	return nil
}

// Terminate shuts down the testing app.
func Terminate() error {
	err := App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestHello_Hello(t *testing.T) {
	t.Parallel()
	/*
		Hello(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Hello(ctx, GET()).
		Name("without name arg").
		BodyContains(t, Svc.Greeting()).
		BodyNotContains(t, "Maria").
		Assert(t, func(t *testing.T, res *http.Response, err error) {
			assert.NoError(t, err)
			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, Svc.Repeat(), bytes.Count(body, []byte(Svc.Greeting())))
		})
	Hello(ctx, GET(), QueryArg("name", "Maria")).
		Name("with name arg").
		BodyContains(t, Svc.Greeting()).
		BodyContains(t, "Maria")
}

func TestHello_Echo(t *testing.T) {
	t.Parallel()
	/*
		Echo(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Echo(ctx, POST("PostBody"), Header("Echo123", "EchoEchoEcho"), QueryArg("echo", "123")).
		BodyContains(t, "Echo123: EchoEchoEcho").
		BodyContains(t, "?echo=123").
		BodyContains(t, "PostBody")
}

func TestHello_Ping(t *testing.T) {
	t.Parallel()
	/*
		Ping(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Ping(ctx, GET()).BodyContains(t, Svc.ID()+"."+Svc.HostName())
}

func TestHello_Calculator(t *testing.T) {
	t.Parallel()
	/*
		Calculator(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Calculator(ctx, GET(), Query("x=5&op=*&y=80")).BodyContains(t, "400")
	Calculator(ctx, GET(), Query("x=500&op=/&y=5")).BodyContains(t, "100")
}

func TestHello_BusJPEG(t *testing.T) {
	t.Parallel()
	/*
		BusJPEG(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context(t)
	img, err := Svc.Resources().ReadFile("bus.jpeg")
	assert.NoError(t, err)
	BusJPEG(ctx, GET()).
		StatusOK(t).
		BodyContains(t, img).
		HeaderContains(t, "Content-Type", "image/jpeg")
}

func TestHello_TickTock(t *testing.T) {
	t.Parallel()
	/*
		TickTock(ctx).
			Name(testName).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, err))
	*/
	ctx := Context(t)
	TickTock(ctx).NoError(t)
}
