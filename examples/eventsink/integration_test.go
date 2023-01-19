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
	// Use .With(...) to initialize with appropriate config values
	App.Include(
		Svc,
		// downstream.NewService().With(),
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

func TestEventsink_Registered(t *testing.T) {
	t.Parallel()
	/*
		Registered(ctx).
			Name(testName).
			Expect(t, emails).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, emails, err))
	*/
	ctx := Context(t)
	registered, err := Registered(ctx).Get()
	assert.NoError(t, err)
	assert.NotContains(t, registered, "jose@example.com")
	assert.NotContains(t, registered, "maria@example.com")
	assert.NotContains(t, registered, "lee@example.com")
	OnRegistered(ctx, "jose@example.com").NoError(t)
	OnRegistered(ctx, "maria@example.com").NoError(t)
	OnRegistered(ctx, "lee@example.com").NoError(t)
	registered, err = Registered(ctx).Get()
	assert.NoError(t, err)
	assert.Contains(t, registered, "jose@example.com")
	assert.Contains(t, registered, "maria@example.com")
	assert.Contains(t, registered, "lee@example.com")
}

func TestEventsink_OnAllowRegister(t *testing.T) {
	t.Parallel()
	/*
		OnAllowRegister(ctx, email).
			Name(testName).
			Expect(t, allow).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, allow, err))
	*/
	ctx := Context(t)
	OnAllowRegister(ctx, "nancy@gmail.com").Name("disallow gmail.com").Expect(t, false)
	OnAllowRegister(ctx, "nancy@hotmail.com").Name("disallow hotmail.com").Expect(t, false)

	OnAllowRegister(ctx, "nancy@example.com").Name("allow hotmail.com").Expect(t, true)
	OnRegistered(ctx, "nancy@example.com").NoError(t)
	OnAllowRegister(ctx, "nancy@example.com").Name("disallow dup").Expect(t, false)
}

func TestEventsink_OnRegistered(t *testing.T) {
	t.Skip() // Tested by TestEventsink_Registered
}
