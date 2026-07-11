/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/exampleservices/eventsink/eventsinkapi"
	"github.com/microbus-io/fabric/exampleservices/eventsource/eventsourceapi"
)

var (
	_ context.Context
	_ *testing.T
	_ jwt.MapClaims
	_ application.Application
	_ connector.Connector
	_ frame.Frame
	_ pub.Option
	_ testarossa.Asserter
	_ eventsinkapi.Client
)

// MARKER: Registered

// MARKER: OnAllowRegister

// MARKER: OnRegistered

func TestEventsink_Registered(t *testing.T) { // MARKER: Registered
	t.Parallel()
	ctx := t.Context()

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	client := eventsinkapi.NewClient(tester)
	eventsourceTrigger := eventsourceapi.NewMulticastTrigger(tester)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	t.Run("initially_empty", func(t *testing.T) {
		assert := testarossa.For(t)
		// With plane-specific storage, each test has its own isolated data
		registered, err := client.Registered(ctx)
		assert.Expect(
			registered, []string{},
			err, nil,
		)
	})

	t.Run("after_registration", func(t *testing.T) {
		assert := testarossa.For(t)

		// Register multiple users
		emails := []string{"jose@example.com", "maria@example.com", "lee@example.com"}
		for _, email := range emails {
			for i := range eventsourceTrigger.OnRegistered(ctx, email) {
				err := i.Get()
				assert.NoError(err)
			}
		}

		// Verify all users are registered
		registered, err := client.Registered(ctx)
		assert.Expect(err, nil)
		for _, email := range emails {
			assert.Contains(registered, email)
		}
		assert.Equal(len(registered), 3)
	})

	t.Run("case_insensitive", func(t *testing.T) {
		assert := testarossa.For(t)

		// Register with mixed case
		for i := range eventsourceTrigger.OnRegistered(ctx, "ALEX@Example.COM") {
			err := i.Get()
			assert.NoError(err)
		}

		// Should be stored as lowercase
		registered, err := client.Registered(ctx)
		assert.Expect(err, nil)
		assert.Contains(registered, "alex@example.com")
	})

	t.Run("duplicate_registration", func(t *testing.T) {
		assert := testarossa.For(t)

		// Get current count
		before, err := client.Registered(ctx)
		assert.NoError(err)
		countBefore := len(before)

		// Register same email twice
		email := "duplicate@example.com"
		for i := range eventsourceTrigger.OnRegistered(ctx, email) {
			err := i.Get()
			assert.NoError(err)
		}
		for i := range eventsourceTrigger.OnRegistered(ctx, email) {
			err := i.Get()
			assert.NoError(err)
		}

		// Both registrations succeed (service doesn't prevent duplicates)
		after, err := client.Registered(ctx)
		assert.NoError(err)
		assert.Equal(len(after), countBefore+2)
	})
}

func TestEventsink_OnAllowRegister(t *testing.T) { // MARKER: OnAllowRegister
	t.Parallel()
	ctx := t.Context()

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	client := eventsinkapi.NewClient(tester)
	eventsourceTrigger := eventsourceapi.NewMulticastTrigger(tester)
	_ = client

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	t.Run("blocked_registrations", func(t *testing.T) {
		assert := testarossa.For(t)

		testCases := []string{
			"user@gmail.com",
			"user@subdomain.gmail.com",
			"user@hotmail.com",
			"user@subdomain.hotmail.com",
		}

		for _, tc := range testCases {
			for i := range eventsourceTrigger.OnAllowRegister(ctx, tc) {
				if frame.Of(i.HTTPResponse).FromHost() == svc.Hostname() {
					allow, err := i.Get()
					assert.Expect(allow, false, err, nil)
				}
			}
		}
	})

	t.Run("allowed_registrations", func(t *testing.T) {
		assert := testarossa.For(t)

		testCases := []string{
			"nancy@example.com",
			"user@company.com",
			"admin@test.org",
			"dev@custom.io",
			"dev@subdomain.custom.io",
		}

		for _, tc := range testCases {
			for i := range eventsourceTrigger.OnAllowRegister(ctx, tc) {
				if frame.Of(i.HTTPResponse).FromHost() == svc.Hostname() {
					allow, err := i.Get()
					assert.Expect(allow, true, err, nil)
				}
			}
		}
	})

	t.Run("case_insensitive_blocking", func(t *testing.T) {
		assert := testarossa.For(t)

		// Gmail with various cases should still be blocked
		testCases := []string{
			"User@GMAIL.com",
			"user@Gmail.COM",
			"USER@GMAIL.COM",
		}

		for _, tc := range testCases {
			for i := range eventsourceTrigger.OnAllowRegister(ctx, tc) {
				if frame.Of(i.HTTPResponse).FromHost() == svc.Hostname() {
					allow, err := i.Get()
					assert.Expect(allow, false, err, nil)
				}
			}
		}
	})

	t.Run("invalid_email", func(t *testing.T) {
		assert := testarossa.For(t)

		testCases := []string{
			"@gmail.com",
			"",
			"gmail.com",
			"peter!example.com",
		}

		for _, tc := range testCases {
			for i := range eventsourceTrigger.OnAllowRegister(ctx, tc) {
				if frame.Of(i.HTTPResponse).FromHost() == svc.Hostname() {
					allow, err := i.Get()
					assert.Expect(allow, false, err, nil)
				}
			}
		}
	})
}

func TestEventsink_OnRegistered(t *testing.T) { // MARKER: OnRegistered
	t.Skip() // Tested by TestEventsink_Registered
}

func TestEventSink_Registered(t *testing.T) { // MARKER: Registered
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := eventsinkapi.NewClient(tester)
	_ = client

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			emails, err := client.Registered(ctx)
			assert.Expect(
				emails, expectedEmails,
				err, nil,
			)
		})
	*/
}

func TestEventSink_OnAllowRegister(t *testing.T) { // MARKER: OnAllowRegister
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	trigger := eventsourceapi.NewMulticastTrigger(tester)
	_ = trigger

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			for e := range trigger.OnAllowRegister(ctx, email) {
				allow, err := e.Get()
				if frame.Of(e.HTTPResponse).FromHost() == svc.Hostname() {
					assert.Expect(
						allow, expectedAllow,
						err, nil,
					)
				}
			}
		})
	*/
}

func TestEventSink_OnRegistered(t *testing.T) { // MARKER: OnRegistered
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	trigger := eventsourceapi.NewMulticastTrigger(tester)
	_ = trigger

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			for e := range trigger.OnRegistered(ctx, email) {
				err := e.Get()
				if frame.Of(e.HTTPResponse).FromHost() == svc.Hostname() {
					assert.NoError(err)
				}
			}
		})
	*/
}
