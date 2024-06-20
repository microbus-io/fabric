package helloworld

import (
	"context"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/helloworld/helloworldapi"
	"github.com/microbus-io/fabric/examples/helloworld/intermediate"
)

var (
	_ context.Context
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *helloworldapi.Client
)

/*
Service implements the helloworld.example microservice.

The HelloWorld microservice demonstrates the minimalist classic example.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
HelloWorld prints the classic greeting.
*/
func (svc *Service) HelloWorld(w http.ResponseWriter, r *http.Request) (err error) {
	w.Write([]byte("Hello, World!"))
	return nil
}
