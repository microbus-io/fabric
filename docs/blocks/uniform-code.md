# Uniform Code Structure

The code generator creates numerous files and sub-directories in the directory of the microservice.

```
{name}
├── app
│   └── {name}
│       └── main-gen.go
├── {name}api
│   ├── clients-gen.go
│   ├── imports-gen.go
│   └── {type}.go
├── intermediate
│   ├── intermediate-gen.go
│   └── mock-gen.go
├── resources
│   └── embed-gen.go
├── integration_test.go
├── integration-gen_test.go
├── service-gen.go
├── service.go
├── service.yaml
├── version-gen_test.go
└── version-gen.go
```

Files that include `-gen` in their name are fully code generated and should not be edited. The are also marked with a `DO NOT EDIT` comment.

The `app` directory hosts `package main` of an `Application` that runs the microservice, and only the microservice. The executable will be named like the package name of the microservice. If you choose to deploy each microservice as a separate executable, this will be it.

The `{name}api` directory (and package) defines the `Client` and `MulticastClient` of the microservice and the complex types (structs) that they use. `MulticastTrigger` and `Hook` are defined if the microservice is a source of events. Together these represent the public-facing API of the microservice to upstream microservices. The name of the directory `{name}api` is derived from that of the microservice in order to make it easily distinguishable in code completion tools.

The `intermediate` directory (and package) defines the `Intermediate` and the `Mock`. The `Intermediate` serves as the base of the microservice via anonymous inclusion and in turn extends the [`Connector`](../structure/connector.md). The `Mock` is a mockable stub of the microservices that can be used in [integration testing](../blocks/integration-testing.md) when a live version of the microservice cannot.

`integration-gen_test.go` is a testing harness that facilitates the implementation of integration tests, which are expected to be implemented in `integration_test.go`

The `resources` directory is a place to put static files to be embedded (linked) into the executable of the microservice. Templates, images, scripts, etc. are some examples of what can potentially be embedded.

`service-gen.go` primarily includes the function to create a `NewService`.

`service.go` is where solution developers are expected to introduce the business logic of the microservice. `service.go` implements `Service`, which extends `Intermediate` as mentioned earlier. Most of the tools that a microservice needs are available through the receiver `(svc *Service)` which points to the `Intermediate` and by extension the `Connector`. It include the methods of the `Connector` as well as type-specific methods defined in the `Intermediate`.

```go
type Intermediate struct {
    *connector.Connector
}

type Service struct {
    *intermediate.Intermediate
}

func (svc *Service) DoSomething(ctx context.Context) (err error) {
    // svc points to the Intermediate and by extension the Connector
}
```

In addition to the standard `OnStartup` and `OnShutdown` callbacks, the code generator creates an empty function in `service.go` for each and every web handler, functional handler, event sink, ticker or config change callback defined in `service.yaml` as described earlier.

```go
// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
    return
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
    return
}
```

`service.yaml` is the input to the code generator.

`version-gen.go` holds the SHA256 of the source code and the auto-incremented version number. `version-gen_test.go` makes sure it is up to date. If the test fails, running `go generate` brings the version up to date.
