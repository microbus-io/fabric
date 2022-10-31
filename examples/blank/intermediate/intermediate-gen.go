// Code generated by Microbus. DO NOT EDIT.

package intermediate

import (
	"embed"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/blank/resources"
)

var (
	_ embed.FS
	_ time.Duration

	_ cb.Callback
	_ cfg.Config
	_ errors.TracedError
)

// Intermediate extends and customized the generic base connector.
// Code-generated microservices extend the intermediate service.
type Intermediate struct {
	*connector.Connector
    impl ToDo
}

// New creates a new intermediate service.
func New(impl ToDo) *Intermediate {
	svc := &Intermediate{
        Connector: connector.New("blank.example"),
        impl: impl,
    }

    svc.SetDescription(`This is a blank service.`)
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)
	svc.SetOnConfigChanged(svc.doOnConfigChanged)
	svc.DefineConfig(
		`MySQL`,
		cfg.Description(`MySQL connection string.`),
		cfg.Validation(`str`),
		cfg.Secret(),
	)
	svc.DefineConfig(
		`Alert`,
		cfg.Description(`Alert on error.`),
		cfg.Validation(`bool`),
		cfg.DefaultValue(`false`),
	)
	svc.Subscribe(`/multiply`, svc.doMultiply)
	svc.Subscribe(`/helloworld`, svc.impl.HelloWorld)
	intervalMyTickTock, _ := time.ParseDuration("1m0s")
	timeBudgetMyTickTock, _ := time.ParseDuration("10s")
	svc.StartTicker(`MyTickTock`, intervalMyTickTock, svc.impl.MyTickTock, cb.TimeBudget(timeBudgetMyTickTock))

	return svc
}

// Resources is the in-memory file system of the embedded resources.
func (svc *Intermediate) Resources() embed.FS {
	return resources.FS
}
