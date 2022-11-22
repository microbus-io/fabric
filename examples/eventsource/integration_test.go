package eventsource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/eventsink"
	"github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *eventsourceapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
	// Include all downstream microservices in the testing app
	// Use .With(...) to initialize with appropriate config values
	App.Include(
		Svc,
		eventsink.NewService(),
	)

	err := App.Startup()
	if err != nil {
		return err
	}

	// You may call any of the microservices after the app is started

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
	t.Parallel()
	/*
		Register(ctx, email).
			Name(testName).
			Expect(t, allowed).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, allowed, err))
	*/
	ctx := Context()
	Register(ctx, "brian@hotmail.com").Name("decline hotmail.com").Expect(t, false)
	Register(ctx, "brian@example.com").Name("accept example.com").Expect(t, true)
	Register(ctx, "brian@example.com").Name("decline dup").Expect(t, false)
}
