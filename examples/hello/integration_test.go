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

package hello

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/examples/calculator"
	"github.com/microbus-io/fabric/examples/hello/helloapi"
	"github.com/microbus-io/fabric/frame"
)

var (
	_ *testing.T
	_ testarossa.TestingT
	_ *helloapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Add microservices to the testing app
	err = App.AddAndStartup(
		Svc.Init(func(svc *Service) {
			// Initialize the microservice
			svc.SetGreeting("Ciao")
			svc.SetRepeat(5)
		}),
		calculator.NewService(),
	)
	if err != nil {
		return err
	}
	return nil
}

// Terminate gets called after the testing app shut down.
func Terminate() (err error) {
	return nil
}

func TestHello_Hello(t *testing.T) {
	t.Parallel()
	/*
		HelloGet(t, ctx, "").
			BodyContains(value).
			NoError()
		HelloPost(t, ctx, "", "", body).
			BodyContains(value).
			NoError()
		Hello(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	Hello_Get(t, ctx, "").
		ContentType("text/plain").
		BodyContains(Svc.Greeting()).
		BodyNotContains("Maria").
		CompletedIn(10 * time.Millisecond).
		Assert(func(t *testing.T, res *http.Response, err error) {
			testarossa.NoError(t, err)
			body, err := io.ReadAll(res.Body)
			testarossa.NoError(t, err)
			testarossa.Equal(t, Svc.Repeat(), bytes.Count(body, []byte(Svc.Greeting())))
		})
	Hello_Get(t, ctx, "?name=Maria").
		ContentType("text/plain").
		BodyContains(Svc.Greeting()).
		BodyContains("Maria").
		CompletedIn(10 * time.Millisecond)
}

func TestHello_Echo(t *testing.T) {
	t.Parallel()
	/*
		EchoGet(t, ctx, "")
			BodyContains(value).
			NoError()
		EchoPost(t, ctx, "", "", body)
			BodyContains(value).
			NoError()
		Echo(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	r, _ := http.NewRequest("POST", "?echo=123", strings.NewReader("PostBody"))
	r.Header.Add("Echo123", "EchoEchoEcho")
	r.Header.Add("Echo123", "WhoaWhoaWhoa")
	Echo(t, r).
		ContentType("text/plain").
		BodyContains("Echo123: EchoEchoEcho").
		BodyContains("Echo123: WhoaWhoaWhoa").
		BodyContains("?echo=123").
		BodyContains("PostBody")
}

func TestHello_Ping(t *testing.T) {
	t.Parallel()
	/*
		PingGet(t, ctx, "")
			BodyContains(value).
			NoError()
		PingPost(t, ctx, "", "", body)
			BodyContains(value).
			NoError()
		Ping(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	Ping_Get(t, ctx, "").
		ContentType("text/plain").
		BodyContains(Svc.ID() + "." + Svc.Hostname())
}

func TestHello_Calculator(t *testing.T) {
	t.Parallel()
	/*
		CalculatorGet(t, ctx, "")
			BodyContains(value).
			NoError()
		CalculatorPost(t, ctx, "", "", body)
			BodyContains(value).
			NoError()
		Calculator(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	Calculator_Post(t, ctx, "", "",
		url.Values{
			"x":  []string{"500"},
			"op": []string{"+"},
			"y":  []string{"580"},
		}).
		ContentType("text/html").
		TagEqual(`TD#result`, "1080").
		TagExists(`TR TD INPUT[name="x"]`).
		TagExists(`TR TD SELECT[name="op"]`).
		TagExists(`TR TD INPUT[name="y"]`)
	Calculator_Get(t, ctx, "?x=5&op=*&y=80").
		ContentType("text/html").
		TagEqual(`TD#result`, "400")
	Calculator_Post(t, ctx, "", "application/x-www-form-urlencoded", `x=500&op=/&y=5`).
		ContentType("text/html").
		TagEqual(`TD#result`, "100")
}

func TestHello_BusPNG(t *testing.T) {
	t.Parallel()
	/*
		BusPNGAny(t, ctx, httpRequest)
			BodyContains(value).
			NoError()
		BusPNG(t, ctx, "").
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	img, err := Svc.ReadResFile("bus.png")
	testarossa.NoError(t, err)
	BusPNG(t, ctx, "").
		StatusOK().
		ContentType("image/png").
		BodyContains(img)
}

func TestHello_TickTock(t *testing.T) {
	t.Parallel()
	/*
		TickTock(t, ctx).
			NoError()
	*/
	ctx := Context()
	TickTock(t, ctx).NoError()
}

func TestHello_Localization(t *testing.T) {
	t.Parallel()
	/*
		LocalizationGet(t, ctx, "")
			BodyContains(value).
			NoError()
		LocalizationPost(t, ctx, "", "", body)
			BodyContains(value).
			NoError()
		Localization(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	r, _ := http.NewRequest("GET", "", nil)
	frm := frame.Of(r)

	Localization(t, r).
		StatusOK().
		BodyContains("Hello")

	frm.SetLanguages("en")
	Localization(t, r).
		StatusOK().
		BodyContains("Hello")

	frm.SetLanguages("en-NZ")
	Localization(t, r).
		StatusOK().
		BodyContains("Hello")

	frm.SetLanguages("it")
	Localization(t, r).
		StatusOK().
		BodyContains("Salve")
}

func TestHello_EchoClient(t *testing.T) {
	t.Parallel()
	ctx := Context()
	client := helloapi.NewClient(Svc)

	// Nil request
	res, err := client.Echo(nil)
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "GET /echo "))
		}
	}

	// PATCH request with headers and body
	req, err := http.NewRequest("PATCH", "", strings.NewReader("Sunshine"))
	req.Header.Set("X-Location", "California")
	testarossa.NoError(t, err)
	res, err = client.Echo(req)
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "PATCH /echo "))
			testarossa.Contains(t, string(body), "\r\nX-Location: California")
			testarossa.Contains(t, string(body), "\r\nSunshine")
		}
	}

	// GET with no URL
	res, err = client.Echo_Get(ctx, "")
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "GET /echo "))
		}
	}

	// GET with only query string
	res, err = client.Echo_Get(ctx, "?arg=12345")
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "GET /echo?arg=12345 "))
		}
	}

	// GET with relative URL and query string
	res, err = client.Echo_Get(ctx, "/echo?arg=12345")
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "GET /echo?arg=12345 "))
		}
	}

	// GET with absolute URL and query string
	res, err = client.Echo_Get(ctx, "https://"+Hostname+"/echo?arg=12345")
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "GET /echo?arg=12345 "))
		}
	}

	// POST with no URL or content type and form data formDataPayload
	formDataPayload := url.Values{
		"pay":  []string{"11111"},
		"load": []string{"22222"},
	}
	res, err = client.Echo_Post(ctx, "", "", formDataPayload)
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "POST /echo "))
			testarossa.Contains(t, string(body), "\r\nload=22222&pay=11111")
			testarossa.Contains(t, string(body), "\r\nContent-Type: application/x-www-form-urlencoded")
		}
	}

	// POST with query string
	res, err = client.Echo_Post(ctx, "?arg=12345", "", formDataPayload)
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "POST /echo?arg=12345 "))
			testarossa.Contains(t, string(body), "\r\nload=22222&pay=11111")
			testarossa.Contains(t, string(body), "\r\nContent-Type: application/x-www-form-urlencoded")
		}
	}

	// POST with content type
	res, err = client.Echo_Post(ctx, "", "text/plain", formDataPayload)
	if testarossa.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, strings.HasPrefix(string(body), "POST /echo "))
			testarossa.Contains(t, string(body), "\r\nload=22222&pay=11111")
			testarossa.Contains(t, string(body), "\r\nContent-Type: text/plain")
		}
	}
}

func TestHello_Root(t *testing.T) {
	t.Parallel()
	/*
		ctx := Context()
		httpReq, _ := http.NewRequestWithContext(ctx, method, "?arg=val", body)
		Root_Get(t, ctx, "").BodyContains(value)
		Root_Post(t, ctx, "", "", body).BodyContains(value)
		Root(t, httpRequest).BodyContains(value)
	*/

	ctx := Context()
	Root_Get(t, ctx, "").NoError()
}
