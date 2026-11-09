package kitchen

import (
	"testing"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/cmd/gencreds/testdata/kitchen/kitchenapi"
	"github.com/microbus-io/fabric/cmd/gencreds/testdata/weird/weirdapi"
	"github.com/microbus-io/fabric/connector"
)

func TestKitchen_OnChangedThreshold(t *testing.T) { // MARKER: Threshold
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.SetThreshold(value)
			assert.NoError(err)
		})
	*/
}

func TestKitchen_OnObserveQueueDepth(t *testing.T) { // MARKER: QueueDepth
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.OnObserveQueueDepth(ctx)
			assert.NoError(err)
		})
	*/
}

func TestKitchen_OnMyEvent(t *testing.T) { // MARKER: OnMyEvent
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	trigger := kitchenapi.NewMulticastTrigger(tester)
	hook := kitchenapi.NewHook(tester)
	_ = trigger
	_ = hook

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Enter distinct alphanumeric queue names in sub.Queue when hooking multiple times to simulate multiple clients.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			unsub, err := hook.WithOptions(sub.Queue("UniqueQueueName")).OnMyEvent(
				func(ctx context.Context, note string) (ok bool, err error) {
					// Implement event sink here...
					return ok, nil
				},
			)
			if assert.NoError(err) {
				defer unsub()
			}
			for e := range trigger.OnMyEvent(ctx, note) {
				if frame.Of(e.HTTPResponse).FromHost() == tester.Hostname() {
					ok, err := e.Get()
					assert.Expect(
						ok, expectedOk,
						err, nil,
					)
				}
			}
		})
	*/
}

func TestKitchen_MyFunc(t *testing.T) { // MARKER: MyFunc
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := kitchenapi.NewClient(tester)
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

			output, err := client.MyFunc(ctx, input)
			assert.Expect(
				output, expectedOutput,
				err, nil,
			)
		})
	*/
}

func TestKitchen_SelfPing(t *testing.T) { // MARKER: SelfPing
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := kitchenapi.NewClient(tester)
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

			err := client.SelfPing(ctx)
			assert.NoError(err)
		})
	*/
}

func TestKitchen_AltHostFn(t *testing.T) { // MARKER: AltHostFn
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := kitchenapi.NewClient(tester)
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

			err := client.AltHostFn(ctx)
			assert.NoError(err)
		})
	*/
}

func TestKitchen_OnSomething(t *testing.T) { // MARKER: OnSomething
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	trigger := weirdapi.NewMulticastTrigger(tester)
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

			for e := range trigger.OnSomething(ctx, detail) {
				ok, err := e.Get()
				if frame.Of(e.HTTPResponse).FromHost() == svc.Hostname() {
					assert.Expect(
						ok, expectedOk,
						err, nil,
					)
				}
			}
		})
	*/
}
