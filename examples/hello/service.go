/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
	"context"
	"html"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"

	"github.com/microbus-io/fabric/examples/calculator/calculatorapi"
	"github.com/microbus-io/fabric/examples/hello/intermediate"
)

var (
	_ errors.TracedError
	_ http.Request
)

/*
Service implements the "hello.example" microservice.

The Hello microservice demonstrates the various capabilities of a microservice.
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
Hello prints a greeting.
*/
func (svc *Service) Hello(w http.ResponseWriter, r *http.Request) error {
	// If a name is provided, add a personal touch
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	// Prepare the greeting
	greeting := svc.Config("greeting")
	hello := greeting + ", " + name + "!\n"
	repeat, err := strconv.Atoi(svc.Config("repeat"))
	if err != nil {
		return errors.Trace(err)
	}
	hello = strings.Repeat(hello, repeat)

	// Print the greeting
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(hello))
	return nil
}

/*
Echo back the incoming request in wire format.
*/
func (svc *Service) Echo(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer
	err := r.Write(&buf)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(buf.Bytes())
	return nil
}

/*
Ping all microservices and list them.
*/
func (svc *Service) Ping(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer
	ch := svc.Publish(
		r.Context(),
		pub.GET("https://all:888/ping"),
		pub.Multicast(),
	)
	for i := range ch {
		res, err := i.Get()
		if err == nil {
			fromHost := frame.Of(res).FromHost()
			fromID := frame.Of(res).FromID()
			buf.WriteString(fromID)
			buf.WriteString(".")
			buf.WriteString(fromHost)
			buf.WriteString("\r\n")
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(buf.Bytes())
	return nil
}

/*
Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.
*/
func (svc *Service) Calculator(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer
	buf.WriteString(`<html><body><h1>Arithmetic Calculator</h1>`)
	buf.WriteString(`<form method=GET action="calculator"><table>`)

	// X
	x := r.URL.Query().Get("x")
	buf.WriteString(`<tr><td>X</td><td><input name=x type=input value="`)
	buf.WriteString(html.EscapeString(x))
	buf.WriteString(`"></td><tr>`)

	// Op
	op := r.URL.Query().Get("op")
	buf.WriteString(`<tr><td>Op</td><td><select name=op>"`)
	for _, o := range []string{"+", "-", "*", "/"} {
		buf.WriteString(`<option value="`)
		buf.WriteString(o)
		buf.WriteString(`"`)
		if o == op {
			buf.WriteString(` selected`)
		}
		buf.WriteString(`>`)
		buf.WriteString(o)
		buf.WriteString(`</option>`)
	}
	buf.WriteString(`</select></td><tr>`)

	// Y
	y := r.URL.Query().Get("y")
	buf.WriteString(`<tr><td>Y</td><td><input name=y type=input value="`)
	buf.WriteString(html.EscapeString(y))
	buf.WriteString(`"></td><tr>`)

	// Result
	buf.WriteString(`<tr><td>=</td><td>`)
	if x != "" && y != "" && op != "" {
		xx, err := strconv.Atoi(x)
		if err != nil {
			return errors.Trace(err)
		}
		yy, err := strconv.Atoi(y)
		if err != nil {
			return errors.Trace(err)
		}
		// Call the calculator service using its client
		_, _, _, result, err := calculatorapi.NewClient(svc).Arithmetic(r.Context(), xx, op, yy)
		if err != nil {
			buf.WriteString(html.EscapeString(err.Error()))
		} else {
			buf.WriteString(strconv.Itoa(result))
		}
	}
	buf.WriteString(`</td><tr>`)

	buf.WriteString(`</table>`)
	buf.WriteString(`<input type=submit value="Calculate">`)
	buf.WriteString(`</form></body></html>`)

	w.Header().Set("Content-Type", "text/html")
	w.Write(buf.Bytes())
	return nil
}

/*
TickTock is executed every 10 seconds.
*/
func (svc *Service) TickTock(ctx context.Context) error {
	svc.LogInfo(ctx, "Ticktock")
	return nil
}

/*
BusJPEG serves an image from the embedded resources.
*/
func (svc *Service) BusJPEG(w http.ResponseWriter, r *http.Request) (err error) {
	bytes, err := svc.Resources().ReadFile("bus.jpeg")
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(".jpeg"))
	w.Header().Set("Cache-Control", "max-age=3600") // 1 hour
	w.Write(bytes)
	return nil
}
