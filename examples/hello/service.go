package hello

import (
	"bytes"
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
)

// Service is the hello.example microservice
type Service struct {
	*connector.Connector
}

// NewService creates a new hello.example microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.NewConnector(),
	}
	s.SetHostName("hello.example")
	s.Subscribe("/hello", s.Hello)
	s.Subscribe("/echo", s.Echo)
	s.Subscribe("/ping", s.Ping)
	s.Subscribe("/calculator", s.Calculator)
	return s
}

// Hello prints a greeting
func (s *Service) Hello(w http.ResponseWriter, r *http.Request) error {
	// If a name is provided, add a personal touch
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	// Prepare the greeting
	greeting, ok := s.Config("greeting")
	if !ok {
		greeting = "Hello"
	}
	hello := greeting + ", " + name + "!\n"
	repeat, ok := s.ConfigInt("repeat")
	if !ok {
		repeat = 1
	}
	hello = strings.Repeat(hello, repeat)

	// Print the greeting
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte(hello))
	return errors.Trace(err)
}

// Echo back the incoming request in wire format
func (s *Service) Echo(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer
	err := r.Write(&buf)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write(buf.Bytes())
	return errors.Trace(err)
}

// Ping all microservices and list them
func (s *Service) Ping(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer
	ch := s.Publish(
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
	_, err := w.Write(buf.Bytes())
	return errors.Trace(err)
}

// Calculator renders a UI for a calculator.
// The calculation operation is delegated to another microservice in order to demonstrate
// calls from one microservice to another.
func (s *Service) Calculator(w http.ResponseWriter, r *http.Request) error {
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
		res, err := s.Request(
			r.Context(),
			pub.GET("https://calculator.example/arithmetic?x="+url.QueryEscape(x)+"&op="+url.QueryEscape(op)+"&y="+url.QueryEscape(y)),
		)
		if err != nil {
			buf.WriteString(html.EscapeString(err.Error()))
		} else {
			var result struct {
				Result int64 `json:"result"`
			}
			err = json.NewDecoder(res.Body).Decode(&result)
			if err != nil {
				return errors.Trace(err)
			}
			buf.WriteString(strconv.FormatInt(result.Result, 10))
		}
	}
	buf.WriteString(`</td><tr>`)

	buf.WriteString(`</table>`)
	buf.WriteString(`<input type=submit value="Calculate">`)
	buf.WriteString(`</form></body></html>`)

	w.Header().Set("Content-Type", "text/html")
	_, err := w.Write(buf.Bytes())
	return errors.Trace(err)
}
