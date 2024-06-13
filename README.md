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

## 🧅 Layers

Onions have layers, ogres have layers, and so does any decent software architecture.

`Microbus` is conceptually divisible into 4 layers, each providing a set of building blocks (capabilities) to the microservices.

* At the bottom of the `Microbus` stack is a curated selection of OSS technologies that are utilized and abstracted away by the next layer, the connector
* The [connector](./docs/structure/connector.md) construct is the base class from which all microservices are derived. It provides a consistent API to many of the building blocks that are required for a microservice to operate and mesh with other microservices. More often than not, they rely on OSS under the hood
* A [code generator](./docs/tech/codegen.md) brings type-safe RAD that is specific to the semantics of each individual microservice
* The core microservices and the application microservices are built using the code generator

Each of the building blocks is described below the diagram.

<img src="./docs/readme-1.drawio.svg" width="741">
<p>

### Core Services

TODO

HTTP ingress
HTTP egress
Configurator
Metrics
SMTP ingress

### Code Generator

TODO

RAD
JSON over HTTP (marshaling and unmarshaling)
RPC client stubs
Integration test harness
OpenAPI document
Uniform code structure

### Connector Construct

TODO

Configuration
Error capture and propagation
Distributed cache
Tickers
Linked static resources
Internationalization (i18n)

Logging
Distributed tracing
Metrics

Request/response
Publish/subscribe
Eventing
Timeout budget
Ack or fail fast
Discovery
Load balancing
Geo-affinity failover routing
Connectivity health checks

### OSS

[NATS](https://www.nats.io) sits at the core of `Microbus` and makes much of its magic possible. NATS is a full-mesh, highly-available, lighting-fast, real-time, at-most-once, messaging bus. It enables request/response, publish/subscribe, load-balancing, discovery and geo affinity routing.

[OpenTelemetry](https://opentelemetry.io) is a standard for the collection of metrics, distributed tracing and logs.

[Jaeger](https://www.jaegertracing.io) is a distributed tracing observability platform that maps the flow of requests as they traverse a distributed system such as `Microbus`. It is an implementation of the OpenTelemetry standards.

[Prometheus](https://prometheus.io) is a database for storing highly-dimensional time-series data, specifically system and application-level metrics.

[Grafana](https://grafana.com) is a dashboard that provides visibility into the metrics collected by Prometheus.

[Zap](https://github.com/uber-go/zap) is a high-performance, structured (JSON), leveled logger.

[OpenAPI](https://www.openapis.org) is a widely used API description standard. The endpoints of all microservices on `Microbus` are publicly described with OpenAPI.

[Testify](https://github.com/stretchr/testify) is a Go library that helps with making assertions in unit and integration tests.

[Cascadia](https://github.com/andybalholm/cascadia) implements CSS selectors for use with parsing HTML trees produced by Go's `html` package. Used in unit and integration tests, it facilitates assertions against an HTML document. 

## 🗺 Code Orientation

Review each of the major project packages to get oriented in the code structure:

* [application](./docs/structure/application.md) - A collector of microservices that run in a single process and share the same lifecycle
* [cfg](./docs/structure/cfg.md) - Options for defining config properties
* [codegen](./docs/structure/codegen.md) - The code generator
* [connector](./docs/structure/connector.md) - The primary construct of the framework and the basis for all microservices
* coreservices - Microservices that are common for most if not all apps
    * [configurator](./docs/structure/coreservices-configurator.md) - The configurator core microservice
    * [control](./docs/structure/coreservices-control.md) - Client API for the `:888` control subscriptions
    * [httpegress](./docs/structure/coreservices-httpegress.md) - The HTTP egress proxy core microservice
    * [httpingress](./docs/structure/coreservices-httpingress.md) - The HTTP ingress proxy core microservice
    * [inbox](./docs/structure/coreservices-inbox.md) - The inbox microservice listens for incoming emails and fires appropriate events
    * [metrics](./docs/structure/coreservices-metrics.md) - The metrics microservice collects metrics from microservices and delivers them to Prometheus and Grafana
    * [openapiportal](./docs/structure/coreservices-openapiportal.md) - The OpenAPI portal microservice produces a portal page that lists all microservices with open endpoints
* [dlru](./docs/structure/dlru.md) - An LRU cache that is distributed among all peers of a microservice
* [env](./docs/structure/env.md) - Manages the loading of environment variables, with the option of overriding values for testing
* [errors](./docs/structure/errors.md) - An enhancement of Go's standard `errors` package that adds stack tracing and status codes
* [examples](./docs/structure/examples.md) - Demo microservices
    * [Hello](./docs/structure/examples-hello.md) demonstrates the key capabilities of the framework
    * [Calculator](./docs/structure/examples-calculator.md) demonstrates functional handlers
    * [Messaging](./docs/structure/examples-messaging.md) demonstrates load-balanced unicast, multicast and direct addressing messaging
    * [Event source and sink](./docs/structure/examples-events.md) shows how events can be used to reverse the dependency between two microservices
    * [Directory](./docs/structure/examples-directory.md) is an example of a microservice that provides a CRUD API backed by a database
    * [Browser](./docs/structure/examples-browser.md) is an example of a microservice that uses the [HTTP egress core microservice](./coreservices-httpegress.md)
* [frame](./docs/structure/frame.md) - A utility for type-safe manipulation of the HTTP control headers used by the framework
* [httpx](./docs/structure/httpx.md) - Various HTTP utilities
* [log](./docs/structure/log.md) - Fields for attaching data to log messages
* [lru](./docs/structure/lru.md) - An LRU cache with limits on age and weight
* [mtr](./docs/structure/mtr.md) - Metrics collectors
* [openapi](./docs/structure/openapi.md) - OpenAPI document generator
* [pub](./docs/structure/pub.md) - Options for publishing requests over the bus
* [rand](./docs/structure/rand.md) - A utility for generating random numbers and identifiers
* [service](./docs/structure/service.md) - Interface definitions of microservices
* [sub](./docs/structure/sub.md) - Options for subscribing to handle requests over the bus
* [trc](./docs/structure/trc.md) - Options for creating tracing spans
* [timex](./docs/structure/timex.md) - Enhancement of Go's standard `time.Time`
* [utils](./docs/structure/utils.md) - Miscellaneous utility classes and functions

## 👩‍💻 Technical Deep Dive

Go deeper into the philosophy and technology of `Microbus`:

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
