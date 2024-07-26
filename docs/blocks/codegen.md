# Code Generation

Code generation is one of `Microbus`'s most powerful tools. It facilitates rapid development (RAD) of microservices and significantly increases developer productivity. Although it's possible to create a microservice by working directly with the `Connector`, the abstraction added by the code generator makes things simpler by taking care of much of the repetitive boilerplate code.

In `Microbus`, everything starts with a `service.yaml` file that semantically defines the microservice. The code generator then takes this definition and produces a code skeleton for the implementation of microservice itself, a client stub that is used by upstream microservices to call it, an integration test harness that helps to thoroughly test it along with its downstream dependencies, and an OpenAPI document that describes its API.

<img src="./codegen-1.drawio.svg">
<p></p>

Code generation in `Microbus` is additive and idempotent. When new functionality is added,
code changes are generated incrementally without impacting the existing code.

## Bootstrapping

Code generation starts by introducing the `//go:generate` directive into any source file in the directory of the microservice. The recommendation is to add it to a `doc.go` file:

```go
//go:generate go run github.com/microbus-io/fabric/codegen

package myservice
```

The next step is to create a `service.yaml` file which will be used to specify the functionality of the microservice. If the directory contains only a `doc.go` or an empty `service.yaml`, running `go generate` inside the directory will automatically populate `service.yaml`.

```cmd
go generate
```

## `service.yaml`

`service.yaml` contains sections that [define the characteristics of the microservice](../tech/service-yaml.md) in a declarative fashion. These definitions serve as the input to the code generator to produces the skeleton and boilerplate code. It is then up to the developer to fill in the gaps.

## Client Stubs

In addition to the server side of things, the code generator also creates client stubs to facilitate calling the microservice. A unicast `Client` and a `MulticastClient` are placed in a separate API package to avoid cyclical dependencies between upstream and downstream microservices.

Using clients, an upstream microservice remotely calling a downstream microservice looks very much like a standard local function call.

```go
result, err := downstreamapi.NewClient(upstreamSvc).Add(ctx, x, y)
```

For microservices that fire events (i.e. event sources), the API package implements a `MulticastTrigger` and a `Hook`. `MulticastTrigger` is used to facilitate the triggering of events and should generally be used only by the microservice itself. `Hook` is used to facilitate other microservices in subscribing to the events of this microservice.

## Integration Testing

Placeholder [integration tests](../blocks/integration-testing.md) are generated for each of the microservice's handlers to encourage developers to test each of them and achieve high code coverage.

## OpenAPI Document

For applications that have a front-end such as a single-page application, the OpenAPI document streamlines communications with the front-end engineering team. It serves as the source of truth of the backend API that is always kept up to date with the latest code.

<img src="./codegen-2.drawio.svg">
<p></p>

## Embedded Resources

A `resources` directory is automatically created with a `//go:embed` directive to allow microservices to bundle resource files along with the executable. The `embed.FS` is made available to the service via `svc.SetResFS()`.

## Versioning

The code generator tool calculates a hash of the source code of the microservice. Upon detecting a change in the code, the tool increments the version number of the microservice, storing it in `version-gen.go`. This version number is used to identify different builds of the microservice.

## Uniform Code Structure

As a byproduct of code generation, all microservices share a [uniform code structure](../blocks/uniform-code.md). A familiar code structure helps engineers to get oriented quickly in the code even if they are not its original authors. Often a quick glace at `service.yaml` is worth reading a thousand lines of code. Engineers spend a majority of their time reading code so this is of huge value. It makes engineers more portable and versatile.
