# Integration Testing

Thorough testing is an important cornerstone of good software. Testing a microservice is generally difficult because it almost always depends on downstream microservices which are not easy to spin up during testing. Common workarounds include mocking the downstream microservices or testing against a live test deployment but each of these comes with its own drawbacks. A mock doesn't test the actual internal business logic of the microservice, obscuring changes made to it over time. A live test deployment doesn't suffer from the drawbacks of a mock but it is a single point of failure that can block an entire development team when it's down. It also tends to become inconsistent over time and is expensive to run 24/7.

## Testing App

`Microbus` takes a different approach and spins up the actual downstream microservices along with the microservice being tested into a single process. The microservices are collected into an isolated [`Application`](../structure/application.md) that is started up for the duration of running the test suite and shutdown immediately thereafter. The microservices communicate over NATS on a random [plane of communications](../blocks/unicast.md), which keeps them isolated from other test suites that may run in parallel.

Mocks can be added to the application when it's impractical to run the actual downstream microservice, for example if that microservice is calling a third-party web service such as a payment processor. The preference however should be to include the actual microservice whenever possible and not rely on mocks. Note that in `Microbus` microservices are mocked rather than clients. The upstream microservice still sends messages over the bus, which are responded to by the mock of the downstream microservice.

<img src="./integration-testing-1.drawio.svg">

## Code Generated Test Harness

This is all rather complicated to set up which is where the [code generator](../blocks/codegen.md) comes into the picture and automatically creates a test harness (`integration-gen_test.go`) and placeholder tests (`integration_test.go`) for each of the microservice's endpoints out of the specification of the microservice (`service.yaml`). It is then left for the developer to initialize the testing app and implement the tests.

### Initializing the Testing App

The code generator prepares the testing app `App` and includes in it the microservice being tested `Svc`. All dependencies on downstream microservices must be added to the app manually, using the `NewService` constructor of that service. During testing, the [configurator](../structure/coreservices-configurator.md) core microservice is disabled and microservices must be configured directly. The `Init` method is a convenient one-statement pattern for initialization. If the microservice under test defines any configuration properties, they are pre-listed commented-out inside a call to `Svc.Init`.

```go
// Initialize starts up the testing app.
func Initialize() (err error) {
	App.Init(func(svc service.Service) {
		// Initialize all microservices
		svc.SetConfig("SQL", sqlConnectionString)
	})

	// Add microservices to the testing app
	err = App.AddAndStartup(
		downstream.NewService().Init(func(svc *downstream.Service) {
			downstream.SetTimeout(2*time.Minute)
		}),
	)
	if err != nil {
		return err
	}
	err = App.AddAndStartup(
		Svc.Init(func(svc *Service) {
			// Initialize the microservice under test
			svc.SetNumLines(10)
		}),
	)
	if err != nil {
		return err
	}
	return nil
}
```

`App.Init` is a convenient way to initialize all microservices as they are included in or joined to the `App`. At this level, only the generic `service.Service` interface of the microservices is accessible. Setting a configuration property must therefore be done using `SetConfig`.

The `Init` method at the microservice level, including `Svc.Init` for the service under test, have access to the interface of the individual microservice along with all its customizations.   

### Testing Functions and Event Sinks

For each endpoint, the testing harness `integration-gen_test.go` defines a corresponding test case which invokes the underlying endpoint and provides asserters on the result. In the following example, `Arithmetic` calls `Svc.Arithmetic` behind the scenes and returns an `ArithmeticTestCase` with asserters that are customized for its return values. It only takes a few lines of code to run various test cases against the endpoint.

```go
func TestCalculator_Arithmetic(t *testing.T) {
	t.Parallel()
	/*
		Arithmetic(t, ctx, x, op, y).
			Expect(xEcho, opEcho, yEcho, result).
			NoError()
	*/
	ctx := Context()
	Arithmetic(t, ctx, 3, "-", 8).Expect(3, "-", 8, -5)
	Arithmetic(t, ctx, -9, "+", 9).Expect(-9, "+", 9, 0)
	Arithmetic(t, ctx, -9, " ", 9).Expect(-9, "+", 9, 0)
	Arithmetic(t, ctx, 5, "*", 5).Expect(5, "*", 5, 25)
	Arithmetic(t, ctx, 5, "*", -6).Expect(5, "*", -6, -30)
	Arithmetic(t, ctx, 15, "/", 5).Expect(15, "/", 5, 3)
	Arithmetic(t, ctx, 15, "/", 0).Error("zero")
	Arithmetic(t, ctx, 15, "z", 0).Error("operator")
}
```

Test cases support the following asserters:

* `Expect` - asserts the return values
* `Error`, `ErrorCode`, `NoError` - assert the error returned
* `CompletedIn` - assert the execution time
* `Assert` - custom asserter

They also support a `Get` function that returns the values returned by the call to the underlying endpoint.

```go
x, op, y, sum, err := Arithmetic(t, ctx, 3, "-", 8).Get()
```

It is not required to use the provided test cases and asserters. For example, `Arithmetic(t, ctx, 3, "-", 8).Expect(3, "-", 8, -5)` can also be expressed as:

```go
x, op, y, sum, err := Svc.Arithmetic(ctx, 3, "-", 8)
if testarossa.NoError(t, err) {
	testarossa.Equal(t, 3, x)
	testarossa.Equal(t, "-", op)
	testarossa.Equal(t, 8, y)
	testarossa.Equal(t, -5, sum)
}
```

### Testing Webs

Raw web endpoints are tested in a similar fashion, except that their asserters are customized for a web request. In the following example, the `Hello` endpoint is method-anostic and can be tested with various HTTP methods. The resulting `HelloTestCase` includes asserters that are tailored to an `http.Response` return value. Note how asserters can be chained.

```go
func TestHello_Hello(t *testing.T) {
	t.Parallel()
	/*
		Hello_Get(t, ctx, "").
			BodyContains(value).
			NoError()
		Hello_Post(t, ctx, "", "", body).
			BodyContains(value).
			NoError()
		Hello(t, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	Hello_Get(t, ctx, "").
		BodyContains(Svc.Greeting()).
		BodyNotContains("Maria Chavez")
	Hello_Get(t, ctx, "?"+httpx.QArgs{"name": "Maria Chavez"}.Encode()).
		BodyContains(Svc.Greeting()).
		BodyContains("Maria Chavez")
	Hello_Post(t, ctx, "", "", httpx.QArgs{"name": "Maria Chavez"}).
		BodyContains(Svc.Greeting()).
		BodyContains("Maria Chavez")
	Hello(t, httpx.MustNewRequestWithContext(ctx, "PATCH", "application/json", `{"name":"Maria Chavez"}`)).
		BodyContains(Svc.Greeting()).
		BodyContains("Maria Chavez")
}
```

URLs are resolved relative to the URL of the endpoint. The empty URL `""` therefore resolves to the exact URL of the endpoint. A URL starting with `?` is the way to pass query arguments. The example uses `httpx.QArgs` to properly encode the query arguments in the `GET` test case, and to pass in form values in the `POST` example. `httpx.MustNewRequestWithContext` is a thin wrapper of the standard `http.NewRequestWithContext` that panics instead of returning an error.

Available asserters:

* `StatusOK`, `StatusCode` - assert the HTTP response status code
* `BodyContains`, `BodyNotContains` - assert the HTTP response body content
* `HeaderExists`, `HeaderNotExists`, `HeaderEqual`, `HeaderNotEqual`,`HeaderContains`, `HeaderNotContains` - assert the headers of the HTTP response
* `ContentType` - assert the `Content-Type` header of the HTTP response
* `TagExists`, `TagNotExists`, `TagEqual`, `TagNotEqual`, `TagContains`, `TagNotContains` - parse the HTTP response as HTML and assert HTML tags
* `Error`, `ErrorCode`, `NoError` - assert the error returned
* `CompletedIn` - assert the execution time
* `Assert` - custom asserter

The `Get` function returns the HTTP response and error returned by the call to the underlying endpoint.

```go
res, err := Hello_Get(t, ctx, "")
```

### Testing Tickers

[Tickers](../blocks/tickers.md) are disabled during testing in order to avoid the unpredictability of their running schedule. Instead, tickers can be tested manually like other endpoints. Tickers don't take arguments nor return values so the only testing possible is error validation.

```go
func TestHello_TickTock(t *testing.T) {
	t.Parallel()
	/*
		TickTock(t, ctx).
			NoError()
	*/
	ctx := Context()
	TickTock(t, ctx).NoError()
}
```

Tickers can be also be called inside other tests via `Svc`.

Available asserters:

* `Error`, `ErrorCode`, `NoError` - assert the error returned
* `CompletedIn` - assert the execution time
* `Assert` - custom asserter

### Testing Config Callbacks

Callbacks that handle changes to config property values are similarly tested.

```go
func TestExample_OnChangedConnectionString(t *testing.T) {
	t.Parallel()
	/*
		OnChangedConnectionString(t, ctx).
			NoError()
	*/
	ctx := Context()
	OnChangedConnectionString(t, ctx).NoError()
}
```

Available asserters:

* `Error`, `ErrorCode`, `NoError` - assert the error returned
* `CompletedIn` - assert the execution time
* `Assert` - custom asserter

### Testing Event Sources

Events are tested through a corresponding event sink. The event test case must be defined prior to the firing of the event, then `Wait`-ed on after the event is triggered. In the following example, `OnAllowRegister` defines the event test case and `Register` fires the event.

```go
func TestExample_OnAllowRegister(t *testing.T) {
	// No parallel: event sinks might clash across tests
	/*
		OnAllowRegister(t).
			Expect(email).
			Return(allow, err)
	*/
	ctx := Context()
	tc := OnAllowRegister(t).
		Expect("barb@example.com").
		Return(true, nil)
	Register(t, ctx, "barb@example.com").Expect(true)
	tc.Wait()
	tc = OnAllowRegister(t).
		Expect("josh@example.com").
		Return(false, nil)
	Register(t, ctx, "josh@example.com").Expect(false)
	tc.Wait()
}
```

Notice how the assertion of an event is reversed: input arguments of the event are `Expect`-ed whereas its output is `Return`-ed.

`Wait`-ing is necessary for events that fire asynchronously (e.g. using a goroutine) and can be be omitted for synchronous events.

Available asserters:

* `Expect` - asserts the input arguments
* `Return` - defines the return values of the event sink
* `Assert` - custom asserter
* `Wait` - awaits completion of execution

### Skipping Tests

A removed test will be regenerated on the next run of the code generator, so disabling a test is best achieved by placing a call to `t.Skip()` along with an explanation of why the test was skipped.

```go
func TestEventsink_OnRegistered(t *testing.T) {
	t.Skip() // Tested elsewhere
}
```

### Parallelism

The code generator specifies to run all tests (except for events) in parallel by default. The assumption is that tests written in a single test suite are implemented as to not interfere with one another. Commenting out `t.Parallel()` runs that test separately from other tests, however the order of execution of tests is not guaranteed and care must be taken to reset the state at the end of a test that may interfere with another.

## Mocking

Sometimes, using the actual microservice is not possible because it depends on a resource that is not available in the testing environment. For example, a microservice that makes requests to a third-party web service should be mocked in order to avoid depending on that service for development.

In order to more easily mock microservices, the code generator creates a `Mock` for every microservice. This mock includes type-safe methods for mocking all the endpoints of the microservice. If mocking is going to be the same for all tests, the mock can be permanently included in the application in the initialization phase.

```go
// Initialize starts up the testing app.
func Initialize() error {
	// Add microservices to the testing app
	err = App.AddAndStartup(
		Svc.Init(func(svc *Service) {
			// Initialize the microservice under test
			svc.SetNumLines(10),
		}),
		webpay.NewMock().
			MockCharge(func(ctx context.Context, userID string, amount int) (success bool, balance int, err error) {
				return true, 100, nil
			}),
	)
	if err != nil {
		return err
	}
	return nil
}
```

If mocking is going to be different for individual tests, a mock should be temporarily joined to the app in each relevant test instead. More likely than not, these tests should not run in parallel. In the following fictitious example, the `ChargeUser` endpoint of the `payment` microservice is calling a downstream microservice `webpay` that wraps the functionality of a third-party payment processor cloud service. `webpay` is mocked to fail payments over $200 and emulate an error if the amount is $503.

```go
func TestPayment_ChargeUser(t *testing.T) {
	// No parallel: side effects of mocking
	/*
		ChargeUser(ctx, userID, amount).
			Expect(t, success)
	*/

	mockWebPaySvc := webpay.NewMock().
		MockCharge(func(ctx context.Context, userID string, amount int) (success bool, balance int, err error) {
			if amount >= 200 {
				return false, 100, nil
			}
			if amount == 503 {
				return false, 0, errors.New("service unavailable")
			}
			return true, 100, nil
		})

	// Join the mock to the app
	App.AddAndStartup(mockWebPaySvc)
	defer mockWebPaySvc.Shutdown()

	ctx := Context()
	ChargeUser(ctx, "123", 500).Expect(t, false)
	ChargeUser(ctx, "123", 100).Expect(t, true)
	ChargeUser(ctx, "123", 503).Error(t, "service unavailable")
}
```

## Shifting the Clock

At times it is desirable to test aspects of the application that have a temporal dimension. For example, an algorithm may place more weight on newer rather than older content, or perhaps generate a daily histogram of time-series data over a period of a year. In such cases, one would want to perform operations as if they occurred at different times, not only now.

`Microbus` enables this scenario by attaching a clock shift (offset) to the context using the `SetClockShift` method of the [frame](../structure/frame.md). The connector's `Now(ctx)` method then takes the clock shift into account before returning the "current" time.

To shift the clock in the test:

```go
func TestFoo_DoSomething(t *testing.T) {
	ctx := Context()

	frame.Of(ctx).SetClockShift(-time.Hour * 24) // Yesterday
	Svc.DoSomething(ctx, 2).NoError()

	tm, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z07:00")
	frame.Of(ctx).SetClockShift(time.Until(tm))
	Svc.DoSomething(ctx, 3).NoError()
}
```

To obtain the "current" time in the microservice:

```go
func (svc *Service) DoSomething(ctx context.Context, n int) (err error) {
	now := svc.Now(ctx) // Now is offset by the clock shift
	// ...
	barapi.NewClient(svc).DoSomethingElse(ctx, n) // Clock shift is propagated downstream
	// ...
}
```

The clock shift is propagated down the call chain to downstream microservices. The success of this pattern depends on each of the microservices involved in the transaction using `connector.Now(ctx)` instead of the standard `time.Now()` to obtain the current time.

Shifting the clock outside the `TESTING` deployment should be done with extreme caution. Unlike the `TESTING` deployment, tickers are enabled in the `LOCAL`, `LAB` and `PROD` [deployments](../tech/deployments.md) and always executed at the real time.

Note that shifting the clock will not cause any timeouts or deadlines to be triggered. It is simply a mechanism of transferring an offset down the call chain.

## Manipulating the Context

`Microbus` uses the `ctx` or `r.Context()` to pass-in adjunct data to that does not affect the business logic of the endpoint. The context is extended with a [frame](../structure/frame.md) which internally holds an `http.Header` that includes various `Microbus` key-value pairs. Shifting the clock is one common example, another is the language.

Use the `frame.Frame` to access and manipulate this header:

```go
frm := frame.Of(ctx) // or frame.Of(r)
frm.SetClockShift(-time.Hour)
frm.SetLanguages("it", "fr")
```

## Maximizing Results

Some tips for maximizing the effectiveness of your testing:

### Code Coverage

The testing harness automatically creates a test for each one of the microservice's endpoints. Use them to define numerous test cases and cover all aspects of the endpoint, including its edge cases. This is a quick way to achieve high code coverage. 

### Downstream Microservices

Take advantage of `Microbus`'s unique ability to run integration tests inside Go's unit test framework.
Include in the testing app all the downstream microservices that the microservice under test is dependent upon. Create tests for any of the assumptions that the microservice under test is making about the behavior of the downstream microservices.

### Scenarios

Don't be satisfied with the automatically created tests. High code coverage is not enough. Write tests that perform complex scenarios based on the business logic of the solution. For example, if the microservice under test is a CRUD microservice, perform a test that goes through a sequence of steps such as `Create`, `Load`, `List`, `Update`, `Load`, `List`, `Delete`, `Load`, `List` and check for integrity after each step. Involve as many of the downstream microservices as possible, if applicable.
