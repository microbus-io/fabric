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

package helloworld

import (
	"io"
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/exampleservices/helloworld/helloworldapi"
)

// MARKER: HelloWorld

func TestHelloworld_HelloWorld(t *testing.T) { // MARKER: HelloWorld
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	client := helloworldapi.NewClient(tester)
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
		HINT: Use the following pattern for each test case

		res, err := client.HelloWorld(ctx, "")
		if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
			body, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				// For HTML responses
				assert.HTMLMatch(body, `DIV.class > DIV#id`, "")
				// For text responses
				assert.Contains(body, "")
			}
		}
	*/

	res, err := client.HelloWorld(ctx, "")
	if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(err) {
			assert.Contains(body, "Hello, World!")
		}
	}
}

func TestHelloWorld_HelloWorld(t *testing.T) { // MARKER: HelloWorld
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := helloworldapi.NewClient(tester)
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

			res, err := client.HelloWorld(ctx, "")
			if assert.NoError(err) {
				assert.Expect(res.StatusCode, http.StatusOK)
			}
		})
	*/
}
