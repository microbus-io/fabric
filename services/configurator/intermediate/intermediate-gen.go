// Code generated by Microbus. DO NOT EDIT.

/*
Package intermediate serves as the foundation of the configurator.sys microservice.

The Configurator is a system microservice that centralizes the dissemination of configuration values to other microservices.
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

	"github.com/microbus-io/fabric/services/configurator/resources"
	"github.com/microbus-io/fabric/services/configurator/configuratorapi"
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

	_ configuratorapi.Client
)

// ToDo defines the interface that the microservice must implement.
// The intermediate delegates handling to this interface.
type ToDo interface {
	OnStartup(ctx context.Context) (err error)
	OnShutdown(ctx context.Context) (err error)
	Values(ctx context.Context, names []string) (values map[string]string, err error)
	Refresh(ctx context.Context) (err error)
	Sync(ctx context.Context, timestamp time.Time, values map[string]map[string]string) (err error)
	PeriodicRefresh(ctx context.Context) (err error)
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
		Connector: connector.New("configurator.sys"),
		impl: impl,
	}
	
	svc.SetVersion(version)
	svc.SetDescription(`The Configurator is a system microservice that centralizes the dissemination of configuration values to other microservices.`)
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)
	svc.SetOnConfigChanged(svc.doOnConfigChanged)
	
	// Functions
	svc.Subscribe(`/values`, svc.doValues)
	svc.Subscribe(`/refresh`, svc.doRefresh)
	svc.Subscribe(`/sync`, svc.doSync, sub.NoQueue())
	
	// Tickers
	intervalPeriodicRefresh, _ := time.ParseDuration("20m0s")
	svc.StartTicker("PeriodicRefresh", intervalPeriodicRefresh, svc.impl.PeriodicRefresh)

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

// doValues handles marshaling for the Values function.
func (svc *Intermediate) doValues(w http.ResponseWriter, r *http.Request) error {
	var i configuratorapi.ValuesIn
	var o configuratorapi.ValuesOut
	d := &o.Data
	err := utils.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	d.Values, err = svc.impl.Values(
		r.Context(),
		i.Names,
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

// doRefresh handles marshaling for the Refresh function.
func (svc *Intermediate) doRefresh(w http.ResponseWriter, r *http.Request) error {
	var i configuratorapi.RefreshIn
	var o configuratorapi.RefreshOut
	d := &o.Data
	err := utils.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	err = svc.impl.Refresh(
		r.Context(),
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

// doSync handles marshaling for the Sync function.
func (svc *Intermediate) doSync(w http.ResponseWriter, r *http.Request) error {
	var i configuratorapi.SyncIn
	var o configuratorapi.SyncOut
	d := &o.Data
	err := utils.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	err = svc.impl.Sync(
		r.Context(),
		i.Timestamp,
		i.Values,
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
