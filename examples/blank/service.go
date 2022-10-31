package blank

import (
	"context"
	"net/http"

	"github.com/microbus-io/fabric/examples/blank/intermediate"

	"github.com/microbus-io/fabric/errors"
)

var (
	_ http.Request
)

/*
Service implements the "blank.example" microservice.

This is a blank service.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return // TODO: OnStartup
}

// OnStartup is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return // TODO: OnShutdown
}

// OnChangedMySQL is triggered when the value of the MySQL config property changes.
func (svc *Service) OnChangedMySQL(ctx context.Context) (err error) {
	return // TODO: OnChangedMySQL
}

/*
Multiply two numbers.
*/
func (svc *Service) Multiply(ctx context.Context, x int, y int) (result int, httpStatusCode int, err error) {
	return x * y, 201, nil
}

/*
HelloWorld prints hello world
*/
func (svc *Service) HelloWorld(w http.ResponseWriter, r *http.Request) (err error) {
	_, err = w.Write([]byte("Hello, World!"))
	return errors.Trace(err)
}

/*
MyTickTock runs every minute.
*/
func (svc *Service) MyTickTock(ctx context.Context) (err error) {
	svc.LogInfo(ctx, "Tick Tock")
	return nil
}
