package eventsource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
)

var (
	_ testing.T
	_ assert.TestingT
	_ eventsourceapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
    // TODO: Initialize testing app
	
	// Include all downstream microservices in the testing app.
	// Use .With(options) to initialize microservices with appropriate config values.
	// Microservices are detected and added automatically. Comment them out if unneeded.
	App.Include(
		Configurator,
		Svc.With(),
	)

	err := App.Startup()
	if err != nil {
		return err
	}

	// You may call any of the microservices after the app is started,
	// for example to populate data shared among all tests.

	return nil
}

// Terminate shuts down the testing app.
func Terminate() error {
	err := App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestEventsource_Register(t *testing.T) {
	// TODO: Test Register
	t.Parallel()
	// ctx := Context()
	// Register(ctx, email).Expect(t, allowed)
	// Register(ctx, email).NoError(t)
	// Register(ctx, email).Error(t, errContains)
	// Register(ctx, email).Assert(t, func(t *testing.T, allowed bool, err error) {})
}
