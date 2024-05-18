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
	"net/url"
	"strings"
	"testing"
	"time"

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
		HelloGet(t, ctx, "")
		-or-
		HelloPost(t, ctx, "", "", body)
		-or-
		Hello(t, ctx, httpRequest).
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
	HelloGet(t, ctx, "").
		BodyContains(Svc.Greeting()).
		BodyNotContains("Maria").
		CompletedIn(10 * time.Millisecond).
		Assert(func(t *testing.T, res *http.Response, err error) {
			assert.NoError(t, err)
			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, Svc.Repeat(), bytes.Count(body, []byte(Svc.Greeting())))
		})
	HelloGet(t, ctx, "?name=Maria").
		BodyContains(Svc.Greeting()).
		BodyContains("Maria").
		CompletedIn(10 * time.Millisecond)
}

func TestHello_Echo(t *testing.T) {
	t.Parallel()
	/*
		EchoGet(t, ctx, "")
		-or-
		EchoPost(t, ctx, "", "", body)
		-or-
		Echo(t, ctx, httpRequest).
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
	r, _ := http.NewRequest("POST", "?echo=123", strings.NewReader("PostBody"))
	r.Header.Set("Echo123", "EchoEchoEcho")
	Echo(t, ctx, r).
		BodyContains("Echo123: EchoEchoEcho").
		BodyContains("?echo=123").
		BodyContains("PostBody")
}

func TestHello_Ping(t *testing.T) {
	t.Parallel()
	/*
		PingGet(t, ctx, "")
		-or-
		PingPost(t, ctx, "", "", body)
		-or-
		Ping(t, ctx, httpRequest).
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
	PingGet(t, ctx, "").
		BodyContains(Svc.ID() + "." + Svc.HostName())
}

func TestHello_Calculator(t *testing.T) {
	t.Parallel()
	/*
		CalculatorGet(t, ctx, "")
		-or-
		CalculatorPost(t, ctx, "", "", body)
		-or-
		Calculator(t, ctx, httpRequest).
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
	CalculatorGet(t, ctx, "?x=5&op=*&y=80").
		BodyContains("400")
	CalculatorPost(t, ctx, "", "application/x-www-form-urlencoded", `x=500&op=/&y=5`).
		BodyContains("100")
	CalculatorPost(t, ctx, "", "",
		url.Values{
			"x":  []string{"500"},
			"op": []string{"+"},
			"y":  []string{"580"},
		}).
		BodyContains("1080")
}

func TestHello_BusJPEG(t *testing.T) {
	t.Parallel()
	/*
		BusJPEGAny(t, ctx, httpRequest)
		-or-
		BusJPEG(t, ctx, "").
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
	img, err := Svc.ReadResFile("bus.jpeg")
	assert.NoError(t, err)
	BusJPEG(t, ctx, "").
		StatusOK().
		BodyContains(img).
		HeaderContains("Content-Type", "image/jpeg")
}

func TestHello_TickTock(t *testing.T) {
	t.Parallel()
	/*
		TickTock(t, ctx).
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
		LocalizationGet(t, ctx, "")
		-or-
		LocalizationPost(t, ctx, "", "", body)
		-or-
		Localization(t, ctx, httpRequest).
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
	r, _ := http.NewRequest("GET", "", nil)
	Localization(t, ctx, r).
		StatusOK().
		BodyContains("Hello")

	r.Header.Set("Accept-Language", "en")
	Localization(t, ctx, r).
		StatusOK().
		BodyContains("Hello")

	r.Header.Set("Accept-Language", "en-NZ")
	Localization(t, ctx, r).
		StatusOK().
		BodyContains("Hello")

	r.Header.Set("Accept-Language", "it")
	Localization(t, ctx, r).
		StatusOK().
		BodyContains("Salve")
}

func TestHello_EchoClient(t *testing.T) {
	t.Parallel()
	ctx := Context(t)
	client := helloapi.NewClient(Svc)

	// Nil request
	res, err := client.Echo(ctx, nil)
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "GET /echo "))
		}
	}

	// PATCH request with headers and body
	req, err := http.NewRequest("PATCH", "", strings.NewReader("Sunshine"))
	req.Header.Set("X-Location", "California")
	assert.NoError(t, err)
	res, err = client.Echo(ctx, req)
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "PATCH /echo "))
			assert.Contains(t, string(body), "\r\nX-Location: California")
			assert.Contains(t, string(body), "\r\nSunshine")
		}
	}

	// GET with no URL
	res, err = client.EchoGet(ctx, "")
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "GET /echo "))
		}
	}

	// GET with only query string
	res, err = client.EchoGet(ctx, "?arg=12345")
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "GET /echo?arg=12345 "))
		}
	}

	// GET with relative URL and query string
	res, err = client.EchoGet(ctx, "/echo?arg=12345")
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "GET /echo?arg=12345 "))
		}
	}

	// GET with absolute URL and query string
	res, err = client.EchoGet(ctx, "https://"+HostName+"/echo?arg=12345")
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "GET /echo?arg=12345 "))
		}
	}

	// POST with no URL or content type and form data formDataPayload
	formDataPayload := url.Values{
		"pay":  []string{"11111"},
		"load": []string{"22222"},
	}
	res, err = client.EchoPost(ctx, "", "", formDataPayload)
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "POST /echo "))
			assert.Contains(t, string(body), "\r\nload=22222&pay=11111")
			assert.Contains(t, string(body), "\r\nContent-Type: application/x-www-form-urlencoded")
		}
	}

	// POST with query string
	res, err = client.EchoPost(ctx, "?arg=12345", "", formDataPayload)
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "POST /echo?arg=12345 "))
			assert.Contains(t, string(body), "\r\nload=22222&pay=11111")
			assert.Contains(t, string(body), "\r\nContent-Type: application/x-www-form-urlencoded")
		}
	}

	// POST with content type
	res, err = client.EchoPost(ctx, "", "text/plain", formDataPayload)
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, strings.HasPrefix(string(body), "POST /echo "))
			assert.Contains(t, string(body), "\r\nload=22222&pay=11111")
			assert.Contains(t, string(body), "\r\nContent-Type: text/plain")
		}
	}
}
