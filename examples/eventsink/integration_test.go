/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package eventsink

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/eventsink/eventsinkapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *eventsinkapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
	// Include all downstream microservices in the testing app
	App.Include(
		Svc,
		// downstream.NewService().With(),
	)

	err := App.Startup()
	if err != nil {
		return err
	}
	// All microservices are now running

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
	t.Parallel()
	/*
		Registered(t, ctx).
			Expect(emails).
			NoError()
	*/
	ctx := Context(t)
	registered, err := Registered(t, ctx).Get()
	assert.NoError(t, err)
	assert.NotContains(t, registered, "jose@example.com")
	assert.NotContains(t, registered, "maria@example.com")
	assert.NotContains(t, registered, "lee@example.com")
	OnRegistered(t, ctx, "jose@example.com").NoError()
	OnRegistered(t, ctx, "maria@example.com").NoError()
	OnRegistered(t, ctx, "lee@example.com").NoError()
	registered, err = Registered(t, ctx).Get()
	assert.NoError(t, err)
	assert.Contains(t, registered, "jose@example.com")
	assert.Contains(t, registered, "maria@example.com")
	assert.Contains(t, registered, "lee@example.com")
}

func TestEventsink_OnAllowRegister(t *testing.T) {
	t.Parallel()
	/*
		OnAllowRegister(ctx, email).
			Expect(allow).
			NoError()
	*/
	ctx := Context(t)
	OnAllowRegister(t, ctx, "nancy@gmail.com").Expect(false)
	OnAllowRegister(t, ctx, "nancy@hotmail.com").Expect(false)

	OnAllowRegister(t, ctx, "nancy@example.com").Expect(true)
	OnRegistered(t, ctx, "nancy@example.com").NoError()
	OnAllowRegister(t, ctx, "nancy@example.com").Expect(true)
}

func TestEventsink_OnRegistered(t *testing.T) {
	t.Skip() // Tested by TestEventsink_Registered
}
