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

package eventsink

import (
	"testing"

	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/examples/eventsink/eventsinkapi"
)

var (
	_ *testing.T
	_ testarossa.TestingT
	_ *eventsinkapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Add microservices to the testing app
	err = App.AddAndStartup(
		Svc,
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

func TestEventsink_Registered(t *testing.T) {
	t.Parallel()
	/*
		Registered(t, ctx).
			Expect(emails).
			NoError()
	*/
	ctx := Context()
	registered, err := Registered(t, ctx).Get()
	testarossa.NoError(t, err)
	testarossa.SliceNotContains(t, registered, "jose@example.com")
	testarossa.SliceNotContains(t, registered, "maria@example.com")
	testarossa.SliceNotContains(t, registered, "lee@example.com")
	OnRegistered(t, ctx, "jose@example.com").NoError()
	OnRegistered(t, ctx, "maria@example.com").NoError()
	OnRegistered(t, ctx, "lee@example.com").NoError()
	registered, err = Registered(t, ctx).Get()
	testarossa.NoError(t, err)
	testarossa.SliceContains(t, registered, "jose@example.com")
	testarossa.SliceContains(t, registered, "maria@example.com")
	testarossa.SliceContains(t, registered, "lee@example.com")
}

func TestEventsink_OnAllowRegister(t *testing.T) {
	t.Parallel()
	/*
		OnAllowRegister(ctx, email).
			Expect(allow).
			NoError()
	*/
	ctx := Context()
	OnAllowRegister(t, ctx, "nancy@gmail.com").Expect(false)
	OnAllowRegister(t, ctx, "nancy@hotmail.com").Expect(false)

	OnAllowRegister(t, ctx, "nancy@example.com").Expect(true)
	OnRegistered(t, ctx, "nancy@example.com").NoError()
	OnAllowRegister(t, ctx, "nancy@example.com").Expect(true)
}

func TestEventsink_OnRegistered(t *testing.T) {
	t.Skip() // Tested by TestEventsink_Registered
}
