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

package embedder

import (
	"slices"
	"testing"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/exampleservices/embedder/embedderapi"
	"github.com/microbus-io/testarossa"
)

// TestEmbedder_Mock exercises the Service-interface mocking pattern. The end-to-end Python
// path is covered by github.com/microbus-io/pyvenv's own integration tests; we don't repeat
// that here.

// MARKER: Embed

// MARKER: Similarity

// TestEmbedder_PythonSubscriptionsManual guards the manual-subscription wiring. The Python-backed
// endpoints must be registered with sub.Manual() and sub.Tag("python") so they stay off the bus
// until the venv liveness callback activates the python-tagged group; the untagged web endpoints are
// not gated. The test fails if the Manual/Tag wiring is ever dropped (as a codegen regression once did).
func TestEmbedder_PythonSubscriptionsManual(t *testing.T) {
	t.Parallel()
	tt := testarossa.For(t)

	svc := NewService()
	app := application.New()
	app.Add(svc)
	app.RunInTest(t)

	subs := map[string]connector.SubscriptionInfo{}
	for _, s := range svc.Subscriptions() {
		subs[s.Name] = s
	}

	for _, name := range []string{"Embed", "Similarity"} {
		s, ok := subs[name]
		if tt.True(ok, "subscription %s not found", name) {
			tt.True(s.Manual, "%s must be a manual subscription", name)
			tt.True(slices.Contains(s.Tags, "python"), "%s must be tagged python", name)
		}
	}

	// The demo web endpoint is not python-gated.
	demo, ok := subs["Demo"]
	if tt.True(ok, "subscription Demo not found") {
		tt.False(demo.Manual, "Demo must not be a manual subscription")
	}
}

func TestEmbedder_Embed(t *testing.T) { // MARKER: Embed
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := embedderapi.NewClient(tester)
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

			vector, err := client.Embed(ctx, text)
			assert.Expect(
				vector, expectedVector,
				err, nil,
			)
		})
	*/
}

func TestEmbedder_Similarity(t *testing.T) { // MARKER: Similarity
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := embedderapi.NewClient(tester)
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

			score, err := client.Similarity(ctx, a, b)
			assert.Expect(
				score, expectedScore,
				err, nil,
			)
		})
	*/
}

func TestEmbedder_Demo(t *testing.T) { // MARKER: Demo
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := embedderapi.NewClient(tester)
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

			res, err := client.Demo(ctx, "GET", "", nil)
			if assert.NoError(err) {
				assert.Expect(res.StatusCode, http.StatusOK)
			}
		})
	*/
}

func TestEmbedder_DemoInit(t *testing.T) { // MARKER: DemoInit
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := embedderapi.NewClient(tester)
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

			res, err := client.DemoInit(ctx, "", nil)
			if assert.NoError(err) {
				assert.Expect(res.StatusCode, http.StatusOK)
			}
		})
	*/
}

func TestEmbedder_DemoStatus(t *testing.T) { // MARKER: DemoStatus
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := embedderapi.NewClient(tester)
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

			res, err := client.DemoStatus(ctx, "")
			if assert.NoError(err) {
				assert.Expect(res.StatusCode, http.StatusOK)
			}
		})
	*/
}
