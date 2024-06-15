<p align="center">
<img src="docs/logo.svg" height="100" alt="Microbus.io logo">
</p>

`Microbus` is an opinionated framework for the development, deployment and operation of microservices. Its most notable characteristic is that it leverages a messaging bus for the transport layer. Microservices over a bus = microbus.

[![Test](https://github.com/microbus-io/fabric/actions/workflows/test.yaml/badge.svg?branch=main&event=push)](https://github.com/microbus-io/fabric/actions/workflows/test.yaml)

## Introduction

The framework's philosophy can be categorized into three conceptual areas:

* Common runtime - The framework specifies rules that all microservices need to comply with during runtime. This includes things like how microservices [communicate](./docs/tech/unicast.md), how they report [metrics](./docs/structure/coreservices-metrics.md), where they [pull config values](./docs/blocks/configuration.md) from, how they output logs, how they get deployed, etc. A common set of rules is important for interoperability, maintainability and stability of the system
* RAD tools - The framework leverages [code generation](./docs/blocks/codegen.md) for the rapid development of microservices with the intent that application developers focus on the business logic rather than on boilerplate code. Developer productivity is directly correlated to customer value
* Standard building blocks - Capabilities that are deemed to be the building blocks of microservices are implemented by the framework in a standard way, serving to facilitate both of the above

`fabric` is the main project that provides the basic capabilities that all `Microbus` microservices are built upon. The [milestones](./docs/milestones.md) of this project are maintained in separate branches in order to archive and demonstrate the development process of the framework and the evolution of the codebase.

## Getting Started

👉 Follow the [quick start guide](./docs/howto/quick-start.md) to set up your system and run the example app

👉 Go through the [examples](./docs/structure/examples.md) in depth

👉 Follow the step-by-step guide and [build your first microservice](./docs/howto/first-service.md)

👉 Discover the full power of [code generation](./docs/blocks/codegen.md). It's totally RAD, dude

👉 Learn how to write thorough [integration tests](./docs/blocks/integrationtesting.md) and achieve high code coverage

👉 Venture out and [explore more on your own](./docs/self-explore.md)

## Digging Deeper

👉 The [layerd architectural diagram](./docs/blocks/layers.md) is map of the many building blocks of `Microbus` and how they relate to one another

👉 If you like reading code, the [list of packages](./docs/structure/packages.md) will help you find your way

## Technical Deep Dive

Go deeper into the philosophy and technology of `Microbus`:

* [Unicast messaging](./docs/tech/unicast.md) - Unicast enables bi-directional (request and response) HTTP-like messaging between a client and a single server over NATS
* [HTTP ingress](./docs/blocks/http-ingress.md) - The reason for and role of the HTTP ingress proxy service
* [Encapsulation pattern](./docs/tech/encapsulation.md) - The reasons for encapsulating third-party technologies
* [Error capture](./docs/blocks/error-capture.md) - How and why errors are captured and propagated across microservices boundaries
* [Time budget](./docs/blocks/time-budget.md) - The proper way to manage request timeouts
* [Configuration](./docs/blocks/configuration.md) - How to configure microservices
* [NATS connection settings](./docs/tech/natsconnection.md) - How to configure microservices to connect and authenticate to NATS
* [Multicast messaging](./docs/tech/multicast.md) - Extending on the unicast pattern, multicast enables bi-directional (request and response) HTTP-like messaging between a client and multiple servers over NATS
* [Control subscriptions](./docs/tech/controlsubs.md) - Subscriptions that all microservices implement out of the box on port `:888`
* [Environment variables](./docs/tech/envars.md) - Environment variables used to initialize microservices
* [Code generation](./docs/blocks/codegen.md) - Discover the power of `Microbus`'s most powerful RAD tool
* [Events](./docs/blocks/events.md) - Using event-driven architecture to decouple microservices
* [Integration testing](./docs/blocks/integrationtesting.md) - Testing multiple microservices together
* [Distributed tracing](./docs/blocks/distrib-tracing.md) - Distributed tracing using OpenTelemetry and Jaeger
* [OpenAPI](./docs/blocks/openapi.md) - OpenAPI document generation for microservices
* [Path arguments](./docs/tech/patharguments.md) - Defining wildcard path arguments in subscriptions
* [HTTP magic arguments](./docs/tech/httparguments.md) - How to use HTTP magic arguments in functional endpoints to gain finer control over the HTTP request and response
* [RPC over JSON vs REST](./docs/tech/rpcvsrest.md) - Implementing these common web API styles

## Legal

An explicit license from `Microbus LLC` is required to use the `Microbus` framework.
Refer to the list of [third-party open source software](./docs/third-party-oss.md) for licensing information of components used by the `Microbus` framework.
