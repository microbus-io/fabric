// Code generated by Microbus. DO NOT EDIT.

/*
Package intermediate serves as the foundation of the eventsink.example microservice.

The event sink microservice handles events that are fired by the event source microservice.
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

	"github.com/microbus-io/fabric/examples/eventsink/resources"
	"github.com/microbus-io/fabric/examples/eventsink/eventsinkapi"
	
	eventsourceapi1 "github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
	eventsourceapi2 "github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
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

	_ eventsinkapi.Client
)

// ToDo defines the interface that the microservice must implement.
// The intermediate delegates handling to this interface.
type ToDo interface {
	OnStartup(ctx context.Context) (err error)
	OnShutdown(ctx context.Context) (err error)
	Registered(ctx context.Context) (emails []string, err error)
	OnAllowRegister(ctx context.Context, email string) (allow bool, err error)
	OnRegistered(ctx context.Context, email string) (err error)
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
		Connector: connector.New("eventsink.example"),
		impl: impl,
	}
	
	svc.SetVersion(version)
	svc.SetDescription(`The event sink microservice handles events that are fired by the event source microservice.`)
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)
	svc.SetOnConfigChanged(svc.doOnConfigChanged)
	
	// Functions
	svc.Subscribe(`:443/registered`, svc.doRegistered)
	
	// Sinks
	eventsourceapi1.NewHook(svc).OnAllowRegister(svc.impl.OnAllowRegister)
	eventsourceapi2.NewHook(svc).OnRegistered(svc.impl.OnRegistered)

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

// doRegistered handles marshaling for the Registered function.
func (svc *Intermediate) doRegistered(w http.ResponseWriter, r *http.Request) error {
	var i eventsinkapi.RegisteredIn
	var o eventsinkapi.RegisteredOut
	err := utils.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Emails, err = svc.impl.Registered(
		r.Context(),
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
