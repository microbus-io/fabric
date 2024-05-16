/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
	"github.com/microbus-io/fabric/frame"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *helloapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Include all downstream microservices in the testing app
	App.Include(
		Svc.Init(func(svc *Service) {
			// Initialize the microservice
			svc.SetGreeting("Ciao")
			svc.SetRepeat(5)
		}),
		calculator.NewService(),
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

func TestHello_Hello(t *testing.T) {
	t.Parallel()
	/*
		Hello(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Hello(t, ctx, GET()).
		Name("without name arg").
		BodyContains(Svc.Greeting()).
		BodyNotContains("Maria").
		Assert(func(t *testing.T, res *http.Response, err error) {
			assert.NoError(t, err)
			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, Svc.Repeat(), bytes.Count(body, []byte(Svc.Greeting())))
		})
	Hello(t, ctx, GET(), QueryArg("name", "Maria")).
		Name("with name arg").
		BodyContains(Svc.Greeting()).
		BodyContains("Maria")
}

func TestHello_Echo(t *testing.T) {
	t.Parallel()
	/*
		Echo(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Echo(t, ctx, POST("PostBody"), Header("Echo123", "EchoEchoEcho"), QueryArg("echo", "123")).
		BodyContains("Echo123: EchoEchoEcho").
		BodyContains("?echo=123").
		BodyContains("PostBody")
}

func TestHello_Ping(t *testing.T) {
	t.Parallel()
	/*
		Ping(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Ping(t, ctx, GET()).
		BodyContains(Svc.ID() + "." + Svc.HostName())
}

func TestHello_Calculator(t *testing.T) {
	t.Parallel()
	/*
		Calculator(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Calculator(t, ctx, GET(), Query("x=5&op=*&y=80")).BodyContains("400")
	Calculator(t, ctx, GET(), Query("x=500&op=/&y=5")).BodyContains("100")
}

func TestHello_BusJPEG(t *testing.T) {
	t.Parallel()
	/*
		BusJPEG(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	img, err := Svc.ReadResFile("bus.jpeg")
	assert.NoError(t, err)
	BusJPEG(t, ctx, GET()).
		StatusOK().
		BodyContains(img).
		HeaderContains("Content-Type", "image/jpeg")
}

func TestHello_TickTock(t *testing.T) {
	t.Parallel()
	/*
		TickTock(t, ctx).
			Name(testName).
			NoError().
			Error(errContains).
			Assert(func(err))
	*/
	ctx := Context(t)
	TickTock(t, ctx).NoError()
}

func TestHello_Localization(t *testing.T) {
	t.Parallel()
	/*
		Localization(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			ErrorCode(http.StatusOK).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Localization(t, ctx).
		StatusOK().
		BodyContains("Hello")

	frame.Of(ctx).Set("Accept-Language", "en")
	Localization(t, ctx).
		StatusOK().
		BodyContains("Hello")

	frame.Of(ctx).Set("Accept-Language", "en-NZ")
	Localization(t, ctx).
		StatusOK().
		BodyContains("Hello")

	frame.Of(ctx).Set("Accept-Language", "it")
	Localization(t, ctx).
		StatusOK().
		BodyContains("Salve")
}
