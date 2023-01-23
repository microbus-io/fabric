/*
Copyright 2023 Microbus LLC and various contributors

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
	ctx := Context(t)
	Register(ctx, "brian@hotmail.com").Name("decline hotmail.com").Expect(t, false)
	Register(ctx, "brian@example.com").Name("accept example.com").Expect(t, true)
	Register(ctx, "brian@example.com").Name("decline dup").Expect(t, false)
}

func TestEventsource_OnAllowRegister(t *testing.T) {
	t.Parallel()
	/*
		OnAllowRegister().
			Name(testName).
			Return(allow, err).
			Expect(t, email).
			Assert(t, func(t, ctx, email))
	*/
	ctx := Context(t)
	OnAllowRegister().
		Return(true, nil).
		Expect(t, "barb@example.com")
	Register(ctx, "barb@example.com").Expect(t, true)
	OnAllowRegister().
		Return(false, nil).
		Expect(t, "josh@example.com")
	Register(ctx, "josh@example.com").Expect(t, false)
}

func TestEventsource_OnRegistered(t *testing.T) {
	t.Parallel()
	/*
		OnRegistered().
			Name(testName).
			Return(err).
			Expect(t, email).
			Assert(t, func(t, ctx, email))
	*/
	ctx := Context(t)
	OnRegistered().
		Expect(t, "harry@example.com")
	Register(ctx, "harry@example.com").Expect(t, true)
}
