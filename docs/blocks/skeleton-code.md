# Skeleton Code

The [code generator](../blocks/codegen.md) is `Microbus`'s most powerful productivity tool. It takes as input definitions from [`service.yaml`](../tech/service-yaml.md) to produce boilerplate and skeleton code that saves valuable engineering time to do more meaningful work. Boilerplate code is generally tucked away in `*-gen.go` files and used to add type safety to the individual microservice or to automate repetitive code. Skeleton code, on the other hand, is a placeholder marked with `TODO`s that prompts the engineer to fill in meaningful code.

### Lifecycle

Skeletons of the `OnStartup` and `OnShutDown` lifecycle callbacks are created for all microservice.

```go
// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	// TO DO: Implement OnStartup
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	// TO DO: Implement OnShutdown
	return nil
}
```

These callbacks are registered with the `Connector` in the internal `NewService` method in `intermediate-gen.go`.

### Web Handlers

When a web endpoint is defined in `service.yaml`, the code generator creates the following skeleton in `service.go`:

```go
/*
WebHandler is an example of a web handler.
*/
func (svc *Service) WebHandler(w http.ResponseWriter, r *http.Request) (err error) {
	// TO DO: Implement WebHandler
	// ctx := r.Context()
    return nil
}
```

This web handler is also registered with the `Connector` in `intermediate-gen.go`.

### Functions and Sinks

For functional endpoints and event sinks, the code generator creates a type-safe skeleton that matches the definition in `service.yaml`, always adding a `ctx context.Context` input argument and an `err error` return value.

```go
/*
FuncHandler is an example of a functional handler.
*/
func (svc *Service) FuncHandler(ctx context.Context, id string) (ok bool, err error) {
	// TO DO: Implement FuncHandler
    return ok, nil
}
```

Behind the scenes, a web handler is created that performs unmarshaling of the request and marshaling of the response to match the arguments of the function. It is this web handler that is registered with the `Connector`.

### Tickers

Skeletons of tickers have a simple signature. They too are registered with the `Connector` in `intermediate-gen.go`.

```go
/*
RecurringJob is an example of a ticker handler.
*/
func (svc *Service) RecurringJob(ctx context.Context) (err error) {
	// TO DO: Implement RecurringJob
    return nil
}
```

### Config Change Callbacks

If a config designates a callback, it will be created with the following signature:

```go
// OnChangedMyConfig is triggered when the value of the MyConfig config property changes.
func (svc *Service) OnChangedMyConfig(ctx context.Context) (err error) {
	// TO DO: Implement OnChangedMyConfig
    // val := svc.MyConfig()
    return nil
}
```

### Integration Tests

A skeleton test is created in `integration_test.go` for each testable web handler, functional endpoint, event, event sink and config callback. For example:

```go
func TestMyService_FuncHandler(t *testing.T) {
	t.Parallel()
	/*
		FuncHandler(t, ctx, id).
			Expect(ok).
			NoError()
	*/
	ctx := Context()

	// TO DO: Test FuncHandler
}
```
