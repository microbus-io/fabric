package eventsink

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/eventsink/eventsinkapi"
)

var (
	_ testing.T
	_ assert.TestingT
	_ eventsinkapi.Client
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

func TestEventsink_Registered(t *testing.T) {
	// TODO: Test Registered
	t.Parallel()
	// ctx := Context()
	// Registered(ctx).Expect(t, emails)
	// Registered(ctx).NoError(t)
	// Registered(ctx).Error(t, errContains)
	// Registered(ctx).Assert(t, func(t *testing.T, emails []string, err error) {})
}

func TestEventsink_OnAllowRegister(t *testing.T) {
	// TODO: Test OnAllowRegister
	t.Parallel()
	// ctx := Context()
	// OnAllowRegister(ctx, email).Expect(t, allow)
	// OnAllowRegister(ctx, email).NoError(t)
	// OnAllowRegister(ctx, email).Error(t, errContains)
	// OnAllowRegister(ctx, email).Assert(t, func(t *testing.T, allow bool, err error) {})
}

func TestEventsink_OnRegistered(t *testing.T) {
	// TODO: Test OnRegistered
	t.Parallel()
	// ctx := Context()
	// OnRegistered(ctx, email).NoError(t)
	// OnRegistered(ctx, email).Error(t, errContains)
	// OnRegistered(ctx, email).Assert(t, func(t *testing.T, err error) {})
}
