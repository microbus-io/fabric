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

// Code generated by Microbus. DO NOT EDIT.

/*
Package intermediate serves as the foundation of the hello.example microservice.

The Hello microservice demonstrates the various capabilities of a microservice.
*/
package intermediate

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/shardedsql"
	"github.com/microbus-io/fabric/sub"

	"github.com/microbus-io/fabric/examples/hello/resources"
	"github.com/microbus-io/fabric/examples/hello/helloapi"
)

var (
	_ context.Context
	_ *embed.FS
	_ *json.Decoder
	_ fmt.Stringer
	_ *http.Request
	_ filepath.WalkFunc
	_ strconv.NumError
	_ strings.Reader
	_ time.Duration
	_ cb.Option
	_ cfg.Option
	_ *errors.TracedError
	_ *httpx.ResponseRecorder
	_ *log.Field
	_ *shardedsql.DB
	_ sub.Option
	_ helloapi.Client
)

// ToDo defines the interface that the microservice must implement.
// The intermediate delegates handling to this interface.
type ToDo interface {
	OnStartup(ctx context.Context) (err error)
	OnShutdown(ctx context.Context) (err error)
	Hello(w http.ResponseWriter, r *http.Request) (err error)
	Echo(w http.ResponseWriter, r *http.Request) (err error)
	Ping(w http.ResponseWriter, r *http.Request) (err error)
	Calculator(w http.ResponseWriter, r *http.Request) (err error)
	BusJPEG(w http.ResponseWriter, r *http.Request) (err error)
	TickTock(ctx context.Context) (err error)
}

// Intermediate extends and customizes the generic base connector.
// Code generated microservices then extend the intermediate.
type Intermediate struct {
	*connector.Connector
	impl ToDo
}

// NewService creates a new intermediate service.
func NewService(impl ToDo, version int) *Intermediate {
	svc := &Intermediate{
		Connector: connector.New("hello.example"),
		impl: impl,
	}
	svc.SetVersion(version)
	svc.SetDescription(`The Hello microservice demonstrates the various capabilities of a microservice.`)

	// Lifecycle
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)

	// Configs
	svc.SetOnConfigChanged(svc.doOnConfigChanged)
	svc.DefineConfig(
		"Greeting",
		cfg.Description(`Greeting to use.`),
		cfg.DefaultValue(`Hello`),
	)
	svc.DefineConfig(
		"Repeat",
		cfg.Description(`Repeat indicates how many times to display the greeting.`),
		cfg.Validation(`int [0,100]`),
		cfg.DefaultValue(`1`),
	)
	
	// Webs
	svc.Subscribe(`:443/hello`, svc.impl.Hello)
	svc.Subscribe(`:443/echo`, svc.impl.Echo)
	svc.Subscribe(`:443/ping`, svc.impl.Ping)
	svc.Subscribe(`:443/calculator`, svc.impl.Calculator)
	svc.Subscribe(`:443/bus.jpeg`, svc.impl.BusJPEG)
	
	// Tickers
	intervalTickTock, _ := time.ParseDuration("10s")
	svc.StartTicker("TickTock", intervalTickTock, svc.impl.TickTock)

	return svc
}

// Resources is the in-memory file system of the embedded resources.
func (svc *Intermediate) Resources() embed.FS {
	return resources.FS
}

// doOnConfigChanged is called when the config of the microservice changes.
func (svc *Intermediate) doOnConfigChanged(ctx context.Context, changed func(string) bool) (err error) {
	return nil
}

/*
Greeting to use.
*/
func (svc *Intermediate) Greeting() (greeting string) {
	_val := svc.Config("Greeting")
	return _val
}

/*
Greeting to use.
*/
func Greeting(greeting string) (func(connector.Service) error) {
	return func(svc connector.Service) error {
		return svc.SetConfig("Greeting", fmt.Sprintf("%v", greeting))
	}
}

/*
Repeat indicates how many times to display the greeting.
*/
func (svc *Intermediate) Repeat() (count int) {
	_val := svc.Config("Repeat")
	_i, _ := strconv.ParseInt(_val, 10, 64)
	return int(_i)
}

/*
Repeat indicates how many times to display the greeting.
*/
func Repeat(count int) (func(connector.Service) error) {
	return func(svc connector.Service) error {
		return svc.SetConfig("Repeat", fmt.Sprintf("%v", count))
	}
}
