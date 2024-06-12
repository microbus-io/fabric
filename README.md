# Microbus.io fabric : Milestone 24

[![Test](https://github.com/microbus-io/fabric/actions/workflows/test.yaml/badge.svg?branch=main&event=push)](https://github.com/microbus-io/fabric/actions/workflows/test.yaml)

<img src="docs/gopher-on-bus.png" width="256">

## 🚌 Introduction

`Microbus` is an opinionated framework for the development, deployment and operation of microservices. Its most notable characteristic is that it leverages NATS, a messaging bus, for communications among microservices. Microservices over a bus = microbus.

The framework's philosophy can be categorized into three conceptual areas:

* Common runtime - The framework specifies rules that all microservices need to comply with during runtime. This includes things like how microservices [communicate](./docs/tech/unicast.md), how they report [metrics](./docs/structure/coreservices-metrics.md), where they [pull config values](./docs/tech/configuration.md) from, how they output logs, how they get deployed, etc. A common set of rules is important for interoperability, maintainability and stability of the system
* RAD tools - The framework leverages [code generation](./docs/tech/codegen.md) for the rapid development of microservices with the intent that application developers focus on the business logic rather than on boilerplate code. Developer productivity is directly correlated to customer value
* Standard building blocks - Capabilities that are deemed to be the building blocks of microservices are implemented by the framework in a standard way, serving to facilitate both of the above

`fabric` is the main project that provides the basic capabilities that all `Microbus` microservices are built upon. The [milestones](./docs/milestones.md) of this project are maintained in separate branches in order to archive and demonstrate the development process of the framework and the evolution of the codebase.

## 🚦 Getting Started

👉 Follow the [quick start guide](./docs/quick-start.md) to set up your system and run the example app.

👉 Go through the [examples](./docs/structure/examples.md) in depth.

👉 Follow the step-by-step guide and [build your first microservice](./docs/first-service.md)!

👉 Discover the full power of [code generation](./docs/tech/codegen.md). It's totally RAD, dude!

👉 Learn how to write thorough [integration tests](./docs/tech/integrationtesting.md) and achieve high code coverage.

👉 Venture out and [explore more on your own](./docs/self-explore.md).

## 🗺 Code Structure

Review each of the major project packages to get oriented in the code structure:

* [application](./docs/structure/application.md) - A collector of microservices that run in a single process and share the same lifecycle
* [cfg](./docs/structure/cfg.md) - Options for defining config properties
* [codegen](./docs/structure/codegen.md) - The code generator
* [connector](./docs/structure/connector.md) - The primary construct of the framework and the basis for all microservices
* [coreservices/configurator](./docs/structure/coreservices-configurator.md) - The configurator core microservice
* [coreservices/control](./docs/structure/coreservices-control.md) - Client API for the `:888` control subscriptions
* [coreservices/httpegress](./docs/structure/coreservices-httpegress.md) - The HTTP egress proxy core microservice
* [coreservices/httpingress](./docs/structure/coreservices-httpingress.md) - The HTTP ingress proxy core microservice
* [coreservices/inbox](./docs/structure/coreservices-inbox.md) - The inbox microservice listens for incoming emails and fires appropriate events
* [coreservices/metrics](./docs/structure/coreservices-metrics.md) - The metrics microservice collects metrics from microservices and delivers them to Prometheus and Grafana
* [coreservices/openapiportal](./docs/structure/coreservices-openapiportal.md) - The OpenAPI portal microservice produces a portal page that lists all microservices with open endpoints
* [dlru](./docs/structure/dlru.md) - An LRU cache that is distributed among all peers of a microservice
* [env](./docs/structure/env.md) - Manages the loading of environment variables
* [errors](./docs/structure/errors.md) - An enhancement of the standard `errors` package
* [examples](./docs/structure/examples.md) - Demo microservices 
* [frame](./docs/structure/frame.md) - A utility for type-safe manipulation of the HTTP control headers used by the framework
* [httpx](./docs/structure/httpx.md) - Various HTTP utilities
* [log](./docs/structure/log.md) - Fields for attaching data to log messages
* [lru](./docs/structure/lru.md) - An LRU with with limits on age and weight
* [mtr](./docs/structure/mtr.md) - Metrics collectors
* [openapi](./docs/structure/openapi.md) - Supports the generation of OpenAPI documents
* [pub](./docs/structure/pub.md) - Options for publishing requests
* [rand](./docs/structure/rand.md) - A utility for generating random numbers
* [service](./docs/structure/service.md) - Interface definitions of microservices
* [sub](./docs/structure/sub.md) - Options for subscribing to handle requests
* [trc](./docs/structure/trc.md) - Options for creating tracing spans
* [timex](./docs/structure/timex.md) - Enhancement of the standard `time.Time`
* [utils](./docs/structure/utils.md) - Various independent utility classes and functions

## 👩‍💻 Technical Deep Dive

Go deep into the philosophy and implementation of `Microbus`:

* [Unicast messaging](./docs/tech/unicast.md) - Unicast enables bi-directional (request and response) HTTP-like messaging between a client and a single server over NATS
* [HTTP ingress](./docs/tech/httpingress.md) - The reason for and role of the HTTP ingress proxy service
* [Encapsulation pattern](./docs/tech/encapsulation.md) - The reasons for encapsulating third-party technologies
* [Error capture](./docs/tech/errorcapture.md) - How and why errors are captured and propagated across microservices boundaries
* [Time budget](./docs/tech/timebudget.md) - The proper way to manage request timeouts
* [Configuration](./docs/tech/configuration.md) - How to configure microservices
* [NATS connection settings](./docs/tech/natsconnection.md) - How to configure microservices to connect and authenticate to NATS
* [Multicast messaging](./docs/tech/multicast.md) - Extending on the unicast pattern, multicast enables bi-directional (request and response) HTTP-like messaging between a client and multiple servers over NATS
* [Control subscriptions](./docs/tech/controlsubs.md) - Subscriptions that all microservices implement out of the box on port `:888`
* [Environment variables](./docs/tech/envars.md) - Environment variables used to initialize microservices
* [Code generation](./docs/tech/codegen.md) - Discover the power of `Microbus`'s most powerful RAD tool
* [Events](./docs/tech/events.md) - Using event-driven architecture to decouple microservices
* [Integration testing](./docs/tech/integrationtesting.md) - Testing multiple microservices together
* [Distributed tracing](./docs/tech/distribtracing.md) - Distributed tracing using OpenTelemetry and Jaeger
* [OpenAPI](./docs/tech/openapi.md) - OpenAPI document generation for microservices
* [Path arguments](./docs/tech/patharguments.md) - Defining wildcard path arguments in subscriptions
* [HTTP magic arguments](./docs/tech/httparguments.md) - How to use HTTP magic arguments in functional endpoints to gain finer control over the HTTP request and response
* [RPC over JSON vs REST](./docs/tech/rpcvsrest.md) - Implementing these common web API styles

## 👩‍⚖️ Legal

An explicit license from `Microbus LLC` is required to use the `Microbus` framework.
Refer to the list of [third-party open source software](./docs/third-party-oss.md) for licensing information of components used by the `Microbus` framework.
