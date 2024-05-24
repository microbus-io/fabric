/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package eventsource

import (
	"errors"
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
	App.Include(
		Svc,
		eventsink.NewService(), // Disallows gmail.com and hotmail.com registrations
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

func TestEventsource_Register(t *testing.T) {
	t.Parallel()
	/*
		Register(t, ctx, email).
			Expect(allowed).
			NoError()
	*/
	ctx := Context()
	Register(t, ctx, "brian@hotmail.com").Expect(false) // hotmail.com is disallowed by eventsink service
	Register(t, ctx, "brian@example.com").Expect(true)  // example.com is allowed
	Register(t, ctx, "mandy@example.com").Expect(true)  // example.com is allowed
}

func TestEventsource_OnAllowRegister(t *testing.T) {
	// No parallel: event sinks might clash across tests
	/*
		tc := OnAllowRegister(t).
			Expect(email).
			Return(allow, err)
		...
		tc.Wait()
	*/
	ctx := Context()

	// Sink allows registration
	tc := OnAllowRegister(t).
		Expect("barb@example.com").
		Return(true, nil)
	Register(t, ctx, "barb@example.com").Expect(true)
	tc.Wait()

	// Sink blocks registration
	tc = OnAllowRegister(t).
		Expect("josh@example.com").
		Return(false, nil)
	Register(t, ctx, "josh@example.com").Expect(false)
	tc.Wait()

	// One sink blocks and the other allows
	tc1 := OnAllowRegister(t).
		Expect("josh@example.com").
		Return(true, nil)
	tc2 := OnAllowRegister(t).
		Expect("josh@example.com").
		Return(false, nil)
	Register(t, ctx, "josh@example.com").Expect(false)
	tc1.Wait()
	tc2.Wait()

	// Sink errors out
	tc = OnAllowRegister(t).
		Expect("josh@example.com").
		Return(false, errors.New("oops"))
	Register(t, ctx, "josh@example.com").Error("oops")
	tc.Wait()
}

func TestEventsource_OnRegistered(t *testing.T) {
	// No parallel: event sinks might clash across tests
	/*
		tc := OnRegistered(t).
			Expect(email).
			Return(err)
		...
		tc.Wait()
	*/
	ctx := Context()
	tc := OnRegistered(t).
		Expect("harry@example.com").
		Return(nil)
	Register(t, ctx, "harry@example.com").Expect(true)
	tc.Wait()
}
