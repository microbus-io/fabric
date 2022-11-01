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
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"

	"github.com/microbus-io/fabric/examples/calculator/resources"

	"github.com/microbus-io/fabric/examples/calculator/calculatorapi"
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

	_ calculatorapi.Client
)

// ToDo defines the interface that the microservice must implement.
// The intermediate delegates handling to this interface.
type ToDo interface {
	OnStartup(ctx context.Context) (err error)
	OnShutdown(ctx context.Context) (err error)
	Arithmetic(ctx context.Context, x int, op string, y int) (xEcho int, opEcho string, yEcho int, result int, err error)
	Square(ctx context.Context, x int) (xEcho int, result int, err error)
	Distance(ctx context.Context, p1 calculatorapi.Point, p2 calculatorapi.Point) (d float64, err error)
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
		Connector: connector.New("calculator.example"),
		impl: impl,
	}
	
	svc.SetVersion(version)
	svc.SetDescription(`The Calculator microservice performs simple mathematical operations.`)
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)
	svc.SetOnConfigChanged(svc.doOnConfigChanged)
	svc.Subscribe(`/arithmetic`, svc.doArithmetic)
	svc.Subscribe(`/square`, svc.doSquare)
	svc.Subscribe(`/distance`, svc.doDistance)

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
func (svc *Intermediate) With(initializers ...Initializer) {
	for _, i := range initializers {
		i(svc)
	}
}

// doArithmetic handles marshaling for the "Arithmetic" function.
func (svc *Intermediate) doArithmetic(w http.ResponseWriter, r *http.Request) error {
	i := struct {
		X int `json:"x"`
		Op string `json:"op"`
		Y int `json:"y"`
	}{}
	o := struct {
		XEcho int `json:"xEcho"`
		OpEcho string `json:"opEcho"`
		YEcho int `json:"yEcho"`
		Result int `json:"result"`
	}{}
	err := utils.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.XEcho, o.OpEcho, o.YEcho, o.Result, err = svc.impl.Arithmetic(
		r.Context(),
		i.X,
		i.Op,
		i.Y,
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

// doSquare handles marshaling for the "Square" function.
func (svc *Intermediate) doSquare(w http.ResponseWriter, r *http.Request) error {
	i := struct {
		X int `json:"x"`
	}{}
	o := struct {
		XEcho int `json:"xEcho"`
		Result int `json:"result"`
	}{}
	err := utils.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.XEcho, o.Result, err = svc.impl.Square(
		r.Context(),
		i.X,
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

// doDistance handles marshaling for the "Distance" function.
func (svc *Intermediate) doDistance(w http.ResponseWriter, r *http.Request) error {
	i := struct {
		P1 calculatorapi.Point `json:"p1"`
		P2 calculatorapi.Point `json:"p2"`
	}{}
	o := struct {
		D float64 `json:"d"`
	}{}
	err := utils.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.D, err = svc.impl.Distance(
		r.Context(),
		i.P1,
		i.P2,
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
