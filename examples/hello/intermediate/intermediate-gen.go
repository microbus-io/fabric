// Code generated by Microbus. DO NOT EDIT.

package intermediate

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/codegen/lib"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"

	"github.com/microbus-io/fabric/examples/hello/resources"
)

var (
	_ context.Context
	_ embed.FS
	_ json.Decoder
	_ fmt.Stringer
	_ http.Request
	_ strconv.NumError
	_ time.Duration

	_ cb.Callback
	_ cfg.Config
	_ lib.Nothing
	_ errors.TracedError
	_ sub.Option
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
	TickTock(ctx context.Context) (err error)
}

// Intermediate extends and customized the generic base connector.
// Code-generated microservices extend the intermediate service.
type Intermediate struct {
	*connector.Connector
	impl ToDo
}

// New creates a new intermediate service.
func New(impl ToDo, version int) *Intermediate {
	svc := &Intermediate{
		Connector: connector.New("hello.example"),
		impl: impl,
	}
	
	svc.SetVersion(version)
	svc.SetDescription(`The Hello microservice demonstrates the various capabilities of a microservice.`)
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)
	svc.SetOnConfigChanged(svc.doOnConfigChanged)
	svc.DefineConfig(
		`Greeting`,
		cfg.Description(`Greeting to use.`),
		cfg.DefaultValue(`Hello`),
	)
	svc.DefineConfig(
		`Repeat`,
		cfg.Description(`Repeat indicates how many times to display the greeting.`),
		cfg.Validation(`int [0,100]`),
		cfg.DefaultValue(`1`),
	)
	svc.Subscribe(`/hello`, svc.impl.Hello)
	svc.Subscribe(`/echo`, svc.impl.Echo)
	svc.Subscribe(`/ping`, svc.impl.Ping)
	svc.Subscribe(`/calculator`, svc.impl.Calculator)
	intervalTickTock, _ := time.ParseDuration("10s")
	svc.StartTicker(`TickTock`, intervalTickTock, svc.impl.TickTock)

	return svc
}

// Resources is the in-memory file system of the embedded resources.
func (svc *Intermediate) Resources() embed.FS {
	return resources.FS
}

// doOnConfigChanged is fired when the config of the microservice changed.
func (svc *Intermediate) doOnConfigChanged(ctx context.Context, changed func(string) bool) error {
	return nil
}

/*
Greeting to use.
*/
func (svc *Intermediate) Greeting() (greeting string) {
	_val := svc.Config(`Greeting`)
	return _val
}

/*
Repeat indicates how many times to display the greeting.
*/
func (svc *Intermediate) Repeat() (count int) {
	_val := svc.Config(`Repeat`)
	_i, _ := strconv.ParseInt(_val, 10, 64)
	return int(_i)
}

// Initializer initializes a config property of the microservice.
type Initializer func(svc *Intermediate) error

// With initializes the config properties of the microservice for testings purposes.
func (svc *Intermediate) With(initializers ...Initializer) {
	for _, i := range initializers {
		i(svc)
	}
}

// Greeting initializes the "Greeting" config property of the microservice.
func Greeting(greeting string) Initializer {
	return func(svc *Intermediate) error{
		return svc.InitConfig(`Greeting`, fmt.Sprintf("%v", greeting))
	}
}

// Repeat initializes the "Repeat" config property of the microservice.
func Repeat(count int) Initializer {
	return func(svc *Intermediate) error{
		return svc.InitConfig(`Repeat`, fmt.Sprintf("%v", count))
	}
}
