<p align="center">
<a href="https://www.microbus.io"><img src="./microbus-logo.svg" height="100" alt="Microbus.io logo"></a>
</p>

### Build, Test, Deploy & Operate Microservice Architectures Smarter at Scale

`Microbus` is a holistic open source framework for the development, testing, deployment and operation of microservices at scale. It combines best-in-class OSS, tooling and best practices into a dramatically-simplified engineering experience.

Build entire cloud-enabled, enterprise-class and web-scalable solutions comprising a multitude of microservices, all on your local development machine. Deploy to suit your needs, as a standalone executable or individual containers.

[![License Apache 2](https://img.shields.io/badge/License-Apache2-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Reference](https://pkg.go.dev/badge/github.com/minio/simdjson-go)](https://pkg.go.dev/github.com/microbus-io/fabric)
[![Test](https://github.com/microbus-io/fabric/actions/workflows/test.yaml/badge.svg?branch=main&event=push)](https://github.com/microbus-io/fabric/actions/workflows/test.yaml)
[![Reference](https://goreportcard.com/badge/github.com/microbus-io/fabric)](https://goreportcard.com/report/github.com/microbus-io/fabric)

## 🍴 Table of Content

🚌 [Introduction](#-introduction)\
⚙️ [How It Works](#️-how-it-works)\
🎯 [Mission Statement](#-mission-statement)\
🚦 [Get Started](#-get-started)\
📚 [Learn More](#-learn-more)\
✋ [Get involved](#-get-involved)\
☎️ [Contact us](#-contact-us)\
📃 [Legal](#-legal)

## 🚌 Introduction

A microservice architecture is best suited for addressing the technical and organizational scalability challenges of a business as it scales. Without microservices, the complexity of a monolithic codebase often grows to a point where the engineering team can no longer innovate and collaborate efficiently. In most likelihood the entire solution will have to be rewritten at a critical point of the business - when it is growing rapidly - and at prohibitive cost. Investing in microservices from the get-go is a wise investment that mitigates this upside risk.

Building and operating microservices at scale, however, is quite difficult and beyond the skills of most engineering teams. It's easy to spin up one web server and call it a microservice but things get exponentially more complicated the more microservices are added to the mix. Many teams at some point either call it quits and stop adding microservices, or introduce complex tooling such as service meshes to help manage the complexity. Adding complexity to solve complexity is a self-defeating strategy: the chickens eventually come home to roost.

`Microbus` takes a novel approach to the development, testing, deployment and troubleshooting of microservices, and eliminates much of the complexity of the conventional practice. `Microbus` is a holistic open source framework that combines best-in-class OSS, tooling and best practices into a dramatically-simplified engineering experience that boosts productivity 4x.

`Microbus` is the culmination of a decade of research and has been successfully battle-tested in production settings running SaaS solutions comprising many dozens of microservices.

## ⚙️ How It Works

<img src="./docs/tech/nutshell-1.drawio.svg">
<p></p>

Build your microservices on top of a `Connector` construct and use its simple API to communicate with other microservices using familiar HTTP semantics. Under the hood, communication happens over a real-time messaging bus.

`Microbus` brings together the patterns and best practices that get it right from the get-go, all in a developer-friendly holistic framework that throws complexity under the bus:

#### Reliable Transport
* [Unicast](./docs/blocks/unicast.md) 1:1 request/response
* [Multicast](./docs/blocks/multicast.md) 1:N publish/subscribe
* [Persistent multiplexed connections](./docs/blocks/multiplexed.md)
* Dynamic service discovery
* Load balancing
* [Time budget](./docs/blocks/time-budget.md)
* [Ack or fail fast](./docs/blocks/ack-or-fail.md)
* Locality-aware routing
* Connectivity liveness check

#### Precision Observability
* [Structured logging](./docs/blocks/logging.md)
* [Distributed tracing](./docs/blocks/distrib-tracing.md)
* [Metrics](./docs/blocks/metrics.md)
* [Error capture](./docs/blocks/error-capture.md) and propagation

#### And more
* [Configuration](./docs/blocks/configuration.md)
* Client stubs
* [Live integration tests](./docs/blocks/integration-testing.md)
* [OpenAPI](./docs/blocks/openapi.md)
* [Graceful shutdown](./docs/blocks/graceful-shutdown.md)
* [Distributed caching](./docs/blocks/distrib-cache.md)
* Linked static resources
* [Recurring jobs](./docs/blocks/tickers.md)

## 🎯 Mission Statement

[`Microbus`](./docs/general/mission-statement.md#microbus) is a [holistic](./docs/general/mission-statement.md#holistic) [open source framework](./docs/general/mission-statement.md#open-source-framework) for the [development, testing, deployment and operation](./docs/general/mission-statement.md#sdlc) of [microservices](./docs/general/mission-statement.md#why-microservices) [at scale](./docs/general/mission-statement.md#at-scale).

`Microbus` combines [best-in-class OSS](./docs/general/mission-statement.md#curated-oss), [tooling](./docs/general/mission-statement.md#tooling) and [best practices](./docs/general/mission-statement.md#best-practices) into an [elevated engineering experience](./docs/general/mission-statement.md#elevated-engineering-experience) that eliminates much of the complexity of the [conventional practice](./docs/general/mission-statement.md#conventional-practice).

`Microbus`’s [runtime substrate](./docs/general/mission-statement.md#runtime-substrate) is highly [performant](./docs/general/mission-statement.md#performance), strongly [reliable](./docs/general/mission-statement.md#reliability) and [horizontally scalable](./docs/general/mission-statement.md#horizontal-scalability).

`Microbus` conforms to [industry standards](./docs/general/mission-statement.md#industry-standards) and [interoperates](./docs/general/mission-statement.md#interoperability) smoothly with existing systems.

## 🚦 Get Started

👉 Follow the [quick start guide](./docs/howto/quick-start.md) to set up your system and run the example app

👉 Go through the various [examples](./docs/structure/examples.md)

👉 Follow the step-by-step guide and [build your first microservice](./docs/howto/first-service.md)

👉 Discover the power of [code generation](./docs/blocks/codegen.md). It's totally RAD, dude

👉 Learn how to write thorough [integration tests](./docs/blocks/integration-testing.md) and achieve high code coverage

👉 Venture out and [explore more on your own](./docs/howto/self-explore.md)

👉 Ready? [Build your own solution](./docs/howto/new-project.md) from scratch

## 📚 Learn More

Dig deeper into the technology of `Microbus` and its philosophy.

### Architecture

* [Architectural diagram](./docs/blocks/layers.md) - A map of the building blocks of `Microbus` and how they stack up
* [Catalog of packages](./docs/structure/packages.md) - Find your way around the codebase

### Guides

* [Code generation](./docs/blocks/codegen.md) - Discover the power of `Microbus`'s powerful RAD tool
* [Configuration](./docs/blocks/configuration.md) - How to configure microservices
* [Path arguments](./docs/tech/path-arguments.md) - Define wildcard path arguments in subscriptions
* [HTTP magic arguments](./docs/tech/http-arguments.md) - Use HTTP magic arguments in functional endpoints to gain finer control over the HTTP request and response
* [Integration testing](./docs/blocks/integration-testing.md) - Test a multitude of microservices together
* [Environment variables](./docs/tech/envars.md) - Environment variables used to initialize microservices
* [NATS connection settings](./docs/tech/nats-connection.md) - How to configure microservices to connect and authenticate to NATS
* [RPC over JSON vs REST](./docs/tech/rpc-vs-rest.md) - Implement these common web API styles
* [Adaptable topology](./docs/blocks/topology.md) - Grow the topology of your system to match your requirements
* [Bootstrap a new project](./docs/howto/new-project.md) - Create a project for your solution
* [Create a new microservice](./docs/howto/create-microservice.md) - Create a new microservice and add it to your solution

### Under the Hood

* [HTTP ingress proxy](./docs/structure/coreservices-httpingress.md) - The HTTP ingress proxy bridges the gap between HTTP and `Microbus`
* [Unicast messaging](./docs/blocks/unicast.md) - Unicast enables bi-directional 1:1 request/response HTTP messaging between a client and a single server over the bus
* [Multicast messaging](./docs/tech/multicast.md) - Extending on the unicast pattern, multicast enables bi-directional 1:N publish/subscribe HTTP messaging between a client and a multitude of servers over the bus
* [Error capture](./docs/blocks/error-capture.md) - How and why errors are captured and propagated across microservices boundaries
* [Time budget](./docs/blocks/time-budget.md) - The right way to manage client-to-server request timeouts
* [Control subscriptions](./docs/tech/control-subs.md) - Subscriptions that all microservices implement out of the box on port `:888`
* [Deployment environments](./docs/tech/deployments.md) - An application can run in one of 4 deployment environments: `PROD`, `LAB`, `LOCAL` and `TESTING`
* [Events](./docs/blocks/events.md) - How event-driven architecture can be used to decouple microservices
* [Distributed tracing](./docs/blocks/distrib-tracing.md) - Visualizing stack traces across microservices using OpenTelemetry and Jaeger
* [OpenAPI](./docs/blocks/openapi.md) - OpenAPI document generation for microservices
* [Local development](./docs/tech/local-dev.md) - Run an entire solution comprising a multitude of microservices in your local IDE
* [Structured logging](./docs/blocks/logging.md) - JSON logging to `stderr`
* [Ack or fail fast](./docs/blocks/ack-or-fail.md) - Acks signal the sender if its request was received
* [Graceful shutdown](./docs/blocks/graceful-shutdown.md) - Graceful shutdown drains pending operations before termination
* [Tickers](./docs/blocks/tickers.md) - Tickers are jobs that run on a schedule
* [Multiplexed connections](./docs/blocks/multiplexed.md) - Multiplexed connections are more efficient than HTTP/1.1

### Design Choices

* [Encapsulation pattern](./docs/tech/encapsulation.md) - The reasons for encapsulating third-party technologies
* [JSON vs Protobuf](./docs/tech/jsonvsprotobuf.md) - Why JSON over HTTP was chosen as the protocol
* [Out of scope](./docs/tech/out-of-scope.md) - Areas that `Microbus` stays out of

### Miscellaneous

* [Milestones](./docs/general/milestones.md) - Each milestone of `Microbus` is maintained in a separate branch for archival purposes and to demonstrate the development process and evolution of the code.

## ✋ Get Involved

We want your feedback. Clone the repo, try things out and let us know what worked for you, what didn't and what you'd like to see improved.

Help us spread the word. Let your peers and the Go community know about `Microbus`.

Give us a Github ⭐. And ask all your friends to give us one too!

Reach out if you'd like to contribute code.

Corporation? Contact us for sponsorship opportunities. 

## ☎️ Contact Us

Find us at any of the following channels. We're looking forward to hearing from you so don't hesitate to drop us a line.

| Find us at... | |
|------------|-----|
| Website    | [www.microbus.io](https://www.microbus.io) |
| Email      | in<span>fo</span>@microbus<span>.io</span> |
| Github     | [github.com/microbus-io](https://www.github.com/microbus-io) |
| LinkedIn   | [linkedin.com/company/microbus-io](https://www.linkedin.com/company/microbus-io) |
| Slack      | [microbus-io.slack.com](https://microbus-io.slack.com) |
| Discord    | [discord.gg/FAJHnGkNqJ](https://discord.gg/FAJHnGkNqJ) |
| Reddit     | [r/microbus](https://reddit.com/r/microbus) |
| YouTube    | [@microbus-io](https://www.youtube.com/@microbus-io) |

## 📃 Legal

The `Microbus` framework is the copyrighted work of various contributors. It is licensed by `Microbus LLC` - a Delaware limited liability company formed to hold rights to the combined intellectual property of all contributors - under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).

Refer to the list of [third-party open source software](./docs/general/third-party-oss.md) for licensing information of components used by the `Microbus` framework.
