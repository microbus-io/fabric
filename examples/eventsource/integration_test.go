/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package eventsource

import (
	"errors"
	"testing"

	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/examples/eventsink"
	"github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
)

var (
	_ *testing.T
	_ testarossa.TestingT
	_ *eventsourceapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Add microservices to the testing app
	err = App.AddAndStartup(
		Svc,
		eventsink.NewService(), // Disallows gmail.com and hotmail.com registrations
	)
	if err != nil {
		return err
	}
	return nil
}

// Terminate gets called after the testing app shut down.
func Terminate() (err error) {
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
