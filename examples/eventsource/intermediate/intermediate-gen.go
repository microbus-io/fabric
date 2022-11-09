// Code generated by Microbus. DO NOT EDIT.

/*
Package intermediate serves as the foundation of the eventsource.example microservice.

The event source microservice fires events that are caught by the event sink microservice.
*/
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
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"

	"github.com/microbus-io/fabric/examples/eventsource/resources"
	"github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
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
	_ errors.TracedError
	_ sub.Option
	_ utils.ResponseRecorder

	_ eventsourceapi.Client
)

// ToDo defines the interface that the microservice must implement.
// The intermediate delegates handling to this interface.
type ToDo interface {
	OnStartup(ctx context.Context) (err error)
	OnShutdown(ctx context.Context) (err error)
	Register(ctx context.Context, email string) (allowed bool, err error)
}

// Intermediate extends and customizes the generic base connector.
// Code generated microservices then extend the intermediate.
type Intermediate struct {
	*connector.Connector
	impl ToDo
}

// New creates a new intermediate service.
func New(impl ToDo, version int) *Intermediate {
	svc := &Intermediate{
		Connector: connector.New("eventsource.example"),
		impl: impl,
	}
	
	svc.SetVersion(version)
	svc.SetDescription(`The event source microservice fires events that are caught by the event sink microservice.`)
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)
	svc.SetOnConfigChanged(svc.doOnConfigChanged)
	
	// Functions
	svc.Subscribe(`:443/register`, svc.doRegister)

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

// Initializer initializes a config property of the microservice.
type Initializer func(svc *Intermediate) error

// With initializes the config properties of the microservice for testings purposes.
func (svc *Intermediate) With(initializers ...Initializer) *Intermediate {
	for _, i := range initializers {
		i(svc)
	}
	return svc
}

// doRegister handles marshaling for the Register function.
func (svc *Intermediate) doRegister(w http.ResponseWriter, r *http.Request) error {
	var i eventsourceapi.RegisterIn
	var o eventsourceapi.RegisterOut
	d := &o.Data
	err := utils.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	d.Allowed, err = svc.impl.Register(
		r.Context(),
		i.Email,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(d)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
