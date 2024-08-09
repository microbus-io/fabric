# Mission Statement

[`Microbus`](#microbus) is a [holistic](#holistic) [open source framework](#open-source-framework) for the [development, testing, deployment and operation](#sdlc) of [microservices](#why-microservices) [at scale](#at-scale).

`Microbus` combines [best-in-class OSS](#curated-oss), [tooling](#tooling) and [best practices](#best-practices) into an [elevated engineering experience](#elevated-engineering-experience) that eliminates much of the complexity of the [conventional practice](#conventional-practice).

`Microbus`’s [runtime substrate](#runtime-substrate) is highly [performant](#performance), strongly [reliable](#reliability) and [horizontally scalable](#horizontal-scalability).

`Microbus` conforms to [industry standards](#industry-standards) and [interoperates](#interoperability) smoothly with existing systems.

### Microbus

The name `Microbus` stems from the fact that `micro`services communicate with each other over a messaging `bus`.
The bus enables both the [request/response](../blocks/unicast.md) and [publish/subscribe](../blocks/multicast.md) patterns of communications
and is also responsible for load balancing and service discovery.

### Holistic

Holistic _adj_ /hō-ˈli-stik/ : characterized by the belief that the parts of something are interconnected and can be explained only by reference to the whole.

### Open Source Framework

`Microbus` is essentially a software library you use as a foundation for your microservice solution.
And because it is open sourced, you can feel confident betting your business on it.

### SDLC

`Microbus` delivers a dynamic range of tools designed to optimize the full SDLC of microservice solutions.

#### Develop

Run and debug an entire solution comprising a multitude of microservices on your [local development](../tech/local-dev.md) machine, just as easily as if it were a monolith.

Speed up development with [code generation](../blocks/codegen.md).

#### Test
Spin up the actual downstream microservices along with the microservice being tested into a single process and execute full live [integration tests](../blocks/integration-testing.md)

#### Deploy
From a [local development](../tech/local-dev.md) machine to a multi-region cloud deployment, `Microbus`'s [adaptable topology](../blocks/topology.md) grows with your needs. No code change required.

#### Operate
Observe system internal with the help of [distributed tracing](../blocks/distrib-tracing.md), [metrics](./docs/blocks/metrics.md) dashboards, [structured logging](../blocks/logging.md) and [error capture](../blocks/error-capture.md).

### Why Microservices?

Microservices are the architecture best suited to deal with the technical and organizational challenges of a growing business.
* Scale the engineering organization
* Elevate engineering experience
* Contain the complexity of the codebase
* Scale horizontally to handle more load
* Stay agile and adaptable to change
* Harden your solution to better deal with failures

### At Scale

`Microbus` helps you build and operate large solutions comprising dozens or even hundreds of microservices by addressing both the engineering and the operational challenges inherent in such complex systems. Unlike many other frameworks, it is not merely a helper library for coding of single microservices.

### Curated OSS

`Microbus` is powered by a small curated set of best-in-class [OSS](../blocks/layers.md#oss) technologies integrated to work together in unison.
The small number of moving parts keeps the learning curve short and operational cost and complexity low.

### Tooling
 
A powerful [code generator](../blocks/codegen.md) takes care of most of the repetitive mundane work, freeing engineers to do meaningful work and deliver business value faster.

### Best Practices

`Microbus` implements best practices that pave the road and steer engineers away from common pitfalls.

* Client stubs for calling remote microservices
* Full live Integration testing
* Time budget instead of point-to-point timeouts
* Error capture and surfacing
* Centralized configuration
* Distributed cache siloed to each microservice, not globally centralized

### Elevated Engineering Experience

In `Microbus`, microservices are not large memory-gobbling processes but rather compact worker goroutines that ultimately consume messages from a queue.
This quality allows running, testing and debugging an entire solution comprising a multitude of microservices on a [local development](../tech/local-dev.md) machine.
In fact, it takes only a few seconds to build and restart an entire solution, so code iterations can be made quickly.

A powerful [code generator](../blocks/codegen.md) takes care of most of the repetitive mundane work, freeing engineers to do meaningful work and deliver business value faster.

Observability tools such as [distributed tracing](../blocks/distrib-tracing.md), [metrics](./docs/blocks/metrics.md) dashboards, [structured logging](../blocks/logging.md) and [error capture](../blocks/error-capture.md) provide visibility into the internals of the system, allowing precision identification of bugs and performance issues.

`Microbus`'s functionality is exposed through a [simple API](../tech/encapsulation.md) that is easy to learn. Engineers are able get up to speed quickly and become productive without having to learn the internals of the system. This principle of simplicity is also carried over to `Microbus`'s runtime where the small number of moving parts dramatically reduce the operational complexity.

### Conventional Practice

The conventional practice of developing microservices is a jumble of sophisticated systems that no one truly fully understands. The high level of complexity introduces friction to the software development lifecycle, significant operational costs, and failure points that cause unexplained outages. It takes a small army of engineers to keep the system afloat.

Here's a list of some of the technologies commonly used today:

* Web server for each microservice
* DNS for discovery
* Load balancer
* Port mapping
* Health and liveness checks
* Client-side load balancing
* gRPC
* Kubernetes
* Service mesh (e.g. Istio, Envoy, Consul)
* eBPF networking (e.g. Cilium, Calico)
* Separate system for pub/sub (e.g. Redis, RabbitMQ)
* Separate system for caching (e.g. Redis, memcached)
* K3s for local development

### Runtime Substrate

All microservices running on `Microbus` comply with the same set of rules for [unicast](../blocks/unicast.md) or [multicast](../blocks/multicast.md) communications, [configuration](../blocks/configuration.md), observability, and more. This consistent behavior makes it easier to reason about the accuracy of the system, guaranteeing smooth interoperability, straightforward maintainability and verifiable stability. 

### Performance

Benchmarks indicate `Microbus` is capable of processing upward of 94,500 req/sec on a 10-core MacBook Pro M1 CPU, connected to a messaging bus on localhost.

### Reliability

Reliable communication is an imperative quality of any distributed system. In `Microbus`, microservices communicate with each other over a messaging bus. Each microservice connects to the bus over a [persistent multiplexed connection](../blocks/multiplexed.md) that is monitored constantly and kept alive with automatic reconnects if required. Locality-aware routing, [ack of fail fast](../blocks/ack-or-fail.md) and [graceful shutdowns](../blocks/graceful-shutdown.md) further enhance the reliability of communications.

It is also imperative that a distributed system remains online at all times. `Microbus` achieves that by capturing all errors and panics so that malfunctioning microservices do not crash.

As a framework, `Microbus` is expected to run business-critical solutions. It is thoroughly-tested by hundreds of unit tests.

### Horizontal scalability

Components at all [layers](../blocks/layers.md) of `Microbus` are horizontally scalable. There is no single points of failure or bottlenecks. At the transport layer, the messaging bus forms a full mesh so that any message traverses no more than two nodes regardless of the size of the cluster. At the application layer, dynamic discovery makes it trivial to add replicas of microservices and scale the solution.

### Industry Standards

`Microbus` conforms to industry standards.

* Communication over the bus conforms to the HTTP protocol
* Observability data is pushed to OpenTelemetry collectors
* [OpenAPI](../blocks/openapi.md) documents are automatically created for each microservice

### Interoperability

Because `Microbus` conforms to the familiar HTTP protocol for service-to-service communications, it is a snap to process incoming HTTP requests from non-`Microbus` microservices or from JavaScript clients, or conversely make an outbound call to non-`Microbus` microservices or third-party web services.

`Microbus` is easily deployable as containers running on a Kubernetes cluster.
