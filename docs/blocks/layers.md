# Layered Architecture

Onions have layers, ogres have layers, and so does any decent software architecture.

`Microbus` applications are constructed in 4 layers:

* At the bottom of the stack is a curated selection of OSS technologies that are utilized and abstracted away by the next layer, the connector
* The [connector](./docs/structure/connector.md) construct is the base class from which all microservices are derived. It provides a consistent API to most of the building blocks that are required for a microservice to operate and mesh with other microservices. Quite often they rely on OSS under the hood
* A [code generator](./docs/blocks/codegen.md) brings type-safe RAD that is specific to the semantics of each individual microservice
* The core microservices and the application microservices are built using the code generator

Each of the building blocks is described below the diagram.

<img src="./layers-1.drawio.svg">
<p>

## Application Microservices

A `Microbus` application is a collection of microservices that implement the business logic of the solution.

## Core Microservices

`Microbus` comes bundled with a few core microservices that implement common functionality required by most if not all `Microbus` applications.

The [HTTP ingress proxy](../structure/coreservices-httpingress.md) bridges the gap between HTTP-based clients and microservices running on `Microbus`.

The [HTTP egress proxy](../structure/coreservices-httpegress.md) relays HTTP requests to non-`Microbus` URLs.

The [SMTP ingress](../structure/coreservices-inbox.md) microservice captures incoming emails and transforms them to actionable events.

The [configurator](../structure/coreservices-configurator.md) is responsible for delivering configuration values to microservices that define configuration properties. It is a must-have in practically all applications.

The [metrics](../structure/coreservices-metrics.md) microservice aggregates metrics from all microservices in response to a request from Prometheus.

The [OpenAPI portal](../structure/coreservices-openapiportal.md) microservice renders a catalog of the OpenAPI documents of each and every microservices.

## Code Generator

TODO: break codegen document?

RAD
TODO ^

RPC client stubs
TODO ^

RPC server stubs
TODO ^

[Events](../blocks/events.md) are a type-safe abstraction of publish/subscribe.

JSON marshaling and unmarshaling
TODO ^

The [integration test harness](../blocks/integration-testing.md) spins up the microservice under test along with the actual downstream microservices it depends on into a single testable application, allowing full-blown integration tests to run inside `go test`.

An [OpenAPI document](../blocks/openapi.md) is automatically created for each of the microservice's endpoints.

A [uniform code structure](../blocks/uniform-code.md) is a byproduct of using code generation. A familiar code structure helps engineers get oriented quickly also when they are not the original authors of the code.

## Connector Construct

TODO: a lot of the desc is in the structure doc. move?

### General

[Configuration](../blocks/configuration.md) properties are common means to initialize and customize microservices without the need for code changes. In `Microbus`, microservices define their configuration properties but obtain the runtime values of those properties from the [configurator](../structure/coreservices-configurator.md).

[Errors](../blocks/error-capture.md) are unavoidable. When they occur, they are captured, augmented with a full stack trace, logged, metered (Grafana), [traced](../blocks/distrib-tracing.md) (Jaeger) and propagated up the stack to the upstream microservice.

The [distributed cache](../blocks/distrib-cache.md) is an in-memory cache that is shared among the replica peers of microservice.

Similar to cron jobs, __tickers__ run jobs on a scheduled interval. 

Images, scripts, templates and any other __static resources__ are made available to each microservice by association of a file system (FS).

A specially named resource `strings.yaml` enables __internationalization (i18n)__ of user-facing display strings.

### Observability

Structured (JSON), leveled __logs__ are sent to `stderr`.

[Distributed tracing](../blocks/distrib-tracing.md) enables the visualization of the flow of function calls across microservices and processes. Tracing spans are automatically captured for each endpoint call.

Metrics such as latency, duration, byte size and count are collected automatically for all endpoint calls. Application-specific metrics may be defined by the app. Metrics are shipped to Prometheus via the [metrics core microservice](../structure/coreservices-metrics.md) and visualized in Grafana.

### Transport

[Unicast request/response](../blocks/unicast.md) is an emulation of the familiar synchronous 1:1 request/response pattern of HTTP over the asynchronous messaging pattern of the underlying transport bus.

[Multicast publish/subscribe](../blocks/multicast.md) enhances the publish/subscribe pattern of the bus by introducing a familiar HTTP interface and a bi-directional 1:N request/response pattern.

[Time budget](../blocks/time-budget.md) is a depleting timeout that is passed downstream along the call stack. It is the proper way to handle client-to-server timeouts.

__Ack or fail fast__ is a pattern by which the server responds with an ack to the client before processing the request. This way, the client knows to wait for the response only if an ack is received, and fail if it's not.

A microservices transparently makes itself __discoverable__ by subscribing to the messaging bus. Once subscribed to a subject it immediately starts receiving message from the corresponding queue. An external discovery service is not required.

__Load balancing__ is handled transparently by the messaging bus. Multiple microservices that subscribe to the same queue are delivered messages randomly. An external load balancer is not required.

__Geo-aware failover__ can be setup using [NATS super-clusters](https://docs.nats.io/running-a-nats-service/configuration/gateways).

A microservice is __alive__ when it is connected to the messaging bus and can send and receive messages. The bus validates the connection using regular pings. Explicit liveness checks are unnecessary. 

## OSS

[NATS](https://www.nats.io) sits at the core of `Microbus` and makes much of its magic possible. NATS is a full-mesh, highly-available, lighting-fast, real-time, at-most-once, messaging bus that supports dynamic subscriptions. It enables request/response, publish/subscribe, load-balancing, discovery and geo-aware failover routing.

[OpenTelemetry](https://opentelemetry.io) is a standard for the collection of metrics, distributed tracing and logs.

[Jaeger](https://www.jaegertracing.io) is a distributed tracing observability platform that maps the flow of requests as they traverse a distributed system such as `Microbus`. It is an implementation of the OpenTelemetry standards.

[Prometheus](https://prometheus.io) is a database for storing highly-dimensional time-series data, specifically system and application-level metrics.

[Grafana](https://grafana.com) is a dashboard that provides visibility into the metrics collected by Prometheus.

[Zap](https://github.com/uber-go/zap) is a high-performance, structured (JSON), leveled logger.

[OpenAPI](https://www.openapis.org) is a widely used API description standard. The endpoints of all microservices on `Microbus` are publicly described with OpenAPI.

[Testify](https://github.com/stretchr/testify) is a Go library that simplifies making assertions in unit and integration tests.

[Cascadia](https://github.com/andybalholm/cascadia) implements CSS selectors for use with parsed HTML trees produced by Go's `html` package. Used in unit and integration tests, it facilitates assertions against an HTML document. 
