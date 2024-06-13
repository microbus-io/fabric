# Layered Architecture

Onions have layers, ogres have layers, and so does any decent software architecture.

`Microbus` can be divided into 4 layers, each providing a set of building blocks (capabilities) to the microservices.

* At the bottom of the `Microbus` stack is a curated selection of OSS technologies that are utilized and abstracted away by the next layer, the connector
* The [connector](./docs/structure/connector.md) construct is the base class from which all microservices are derived. It provides a consistent API to many of the building blocks that are required for a microservice to operate and mesh with other microservices. More often than not, they rely on OSS under the hood
* A [code generator](./docs/tech/codegen.md) brings type-safe RAD that is specific to the semantics of each individual microservice
* The core microservices and the application microservices are built using the code generator

Each of the building blocks is described below the diagram.

<img src="./layers-1.drawio.svg" width="741">
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
[Error capture and propagation](../blocks/error-capture.md)
Distributed cache
Tickers
Linked static resources
Internationalization (i18n)

Logging
[Distributed tracing](../blocks/distrib-tracing.md)
Metrics

Request/response
Publish/subscribe
Eventing
[Time budget](../blocks/time-budget.md)
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

