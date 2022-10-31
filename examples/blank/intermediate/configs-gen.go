// Code generated by Microbus. DO NOT EDIT.

package intermediate

import (
	"context"
    "fmt"
    "strconv"
	"time"

	"github.com/microbus-io/fabric/errors"
)

var (
    _ context.Context
	_ fmt.Stringer
	_ strconv.NumError
	_ time.Duration

	_ errors.TracedError
)

// doOnConfigChanged is fired when the config of the microservice changed.
func (svc *Intermediate) doOnConfigChanged(ctx context.Context, changed func(string) bool) error {
	if changed(`MySQL`) {
		err := svc.impl.OnChangedMySQL(ctx)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

/*
MySQL connection string.
*/
func (svc *Intermediate) MySQL() (m string) {
	_val := svc.Config(`MySQL`)
    return _val
}

/*
Alert on error.
*/
func (svc *Intermediate) Alert() (b bool) {
	_val := svc.Config(`Alert`)
    _b, _ := strconv.ParseBool(_val)
    return _b
}

// Initializer initializes a config property of the microservice.
type Initializer func(svc *Intermediate) error

// With initializes the config properties of the microservice for testings purposes.
func (svc *Intermediate) With(initializers ...Initializer) {
	for _, i := range initializers {
		i(svc)
	}
}

// MySQL initializes the "MySQL" config property of the microservice.
func MySQL(m string) Initializer {
	return func(svc *Intermediate) error{
		return svc.InitConfig(`MySQL`, fmt.Sprintf("%v", m))
	}
}

// Alert initializes the "Alert" config property of the microservice.
func Alert(b bool) Initializer {
	return func(svc *Intermediate) error{
		return svc.InitConfig(`Alert`, fmt.Sprintf("%v", b))
	}
}
