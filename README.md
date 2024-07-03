<p align="center">
<img src="./microbus-logo.svg" height="100" alt="Microbus.io logo">
</p>

`Microbus` is a holistic open source framework for the development, testing, deployment and operation of microservices at scale. It combines tooling, guides, patterns and best practices into an elevated engineering experience.
Build entire cloud-enabled, enterprise-class and web-scalable solutions comprising a multitude of microservices, all on your local development machine.
Deploy to suit your needs, as a standalone executable or as individual pods on Kubernetes.

## 🍴 Table of Content

🚌 [Introduction](#-introduction)\
🐶 [Dogma](#-dogma)\
🚦 [Get Started](#-get-started)\
📚 [Learn More](#-learn-more)\
✋ [Get involved](#-get-involved)\
☎️ [Contact us](#-contact-us)\
📃 [Legal](#-legal)

## 🚌 Introduction

A microservice architecture is essential for addressing the technical and organizational scalability challenges of a business as it scales. Without such architecture, the complexity of the codebase will grow to a point where the engineering team can no longer innovate and collaborate efficiently. In most likelihood the entire solution will have to be rewritten at a critical point of the business - when it is growing rapidly - and at a prohibitive cost. Investing in microservices from the get-go is a wise investment that mitigates this upside risk.

Building and operating microservices at scale, however, is quite difficult and beyond the skills of most engineering teams. It's easy to spin up one web server and call it a microservice but things get exponentially more complicated the more microservices are added to the mix. Many teams at some point either call it quits and stop adding microservices, or introduce complex tooling such as service meshes to help manage the complexity. Adding complexity to solve complexity is a self-defeating strategy: the chickens eventually come home to roost.

`Microbus` thinks differently. By taking a novel approach to the development, testing, deployment and troubleshooting of microservices, it eliminates much of the complexity of the current state of the art. `Microbus` is a holistic framework that packages together various OSS technologies, tooling, best practices, patterns and guides into a cohesive engineering experience that elevates productivity up to 4x.

`Microbus` is the culmination of a decade of research and has been successfully battle-tested in production settings running SaaS solutions comprising many dozens of microservices.

## 🐶 Dogma

These core tenets were top of mind in the design of `Microbus`.

### Elevated Engineering Experience

`Microbus` includes tooling, guides, patterns and best practices that facilitate rapid application development (RAD), thorough testing and pinpoint troubleshooting. The greater engineering velocity yields more value delivered to the customer faster.

* Speed up development with [code generation](./docs/blocks/codegen.md)
* Run, test and debug an entire solution comprising a multitude of microservices in your [local development](./docs/tech/local-dev.md) machine
* Observe your system with the help of [distributed tracing](./docs/blocks/distrib-tracing.md), metrics dashboards and logging

### Adaptable Deployment Topology

From a local development machine to a multi-region cloud deployment, `Microbus`'s [adaptable topology](./docs/blocks/topology.md) grows with your needs. No code changes required.

* Deploy as a resilient modular monolith, or a multi-region cloud deployment
* Run as a standalone executable or as individual pods on Kubernetes
* Include one or many microservices in each executable or pod

### Simplified OSS Tech Stack

`Microbus` is powered by a small curated set of [OSS](./docs/blocks/layers.md#oss) technologies integrated to work together in unison and exposed through a [simplified API](./docs/tech/encapsulation.md) that keeps the learning curve short and operational complexity low.

* Get up to speed quickly by virtue of a small tech stack and reasonable defaults
* Keep the number of moving parts small and control operational complexity and cost
* Rely on a chef's choice of best in class OSS

### Scalable, Resilient and Performant

Components at all levels of `Microbus`, from individual microservices down to NATS, can be scaled horizontally. No single points of failure or bottlenecks.

* Serve 1000's of RPS (requests per second) per machine right out of the box
* Scale your system horizontally as your requirements grow
* Rest assured knowing that your code is running on top of a thoroughly-tested substrate

### Common Runtime

All microservices running on `Microbus` comply with the same set of rules for [unicast](./docs/blocks/unicast.md) or [multicast](./docs/blocks/multicast.md) communications, [configuration](./docs/blocks/configuration.md), observability, and more. This consistent behavior makes it easier to reason about the accuracy of the system, guaranteeing smooth interoperability, straightforward maintainability and verifiable stability. 

* Observe every detail of interaction between microservices
* Benefit from improved reliability with multiplexed connections, graceful shutdown, [time budgets](./docs/blocks/time-budget.md) and [acks](./docs/blocks/unicast.md)
* Centralize the configuration of all microservices

## 🚦 Get Started

👉 Follow the [quick start guide](./docs/howto/quick-start.md) to set up your system and run the example app

👉 Go through the various [examples](./docs/structure/examples.md)

👉 Follow the step-by-step guide and [build your first microservice](./docs/howto/first-service.md)

👉 Discover the power of [code generation](./docs/blocks/codegen.md). It's totally RAD, dude

👉 Learn how to write thorough [integration tests](./docs/blocks/integrationtesting.md) and achieve high code coverage

👉 Venture out and [explore more on your own](./docs/howto/self-explore.md)


## 📚 Learn More

Dig deeper into the technology of `Microbus` and its philosophy.

### Architecture

* [Layered architectural diagram](./docs/blocks/layers.md) - A map of the building blocks of `Microbus` and how they relate to one another
* [Catalog of packages](./docs/structure/packages.md) - Find your way around the codebase

### Guides

* [Code generation](./docs/blocks/codegen.md) - Discover the power of `Microbus`'s powerful RAD tool
* [Configuration](./docs/blocks/configuration.md) - How to configure microservices
* [Path arguments](./docs/tech/path-arguments.md) - Define wildcard path arguments in subscriptions
* [HTTP magic arguments](./docs/tech/http-arguments.md) - Use HTTP magic arguments in functional endpoints to gain finer control over the HTTP request and response
* [Integration testing](./docs/blocks/integrationtesting.md) - Test a multitude of microservices together
* [Environment variables](./docs/tech/envars.md) - Environment variables used to initialize microservices
* [NATS connection settings](./docs/tech/nats-connection.md) - How to configure microservices to connect and authenticate to NATS
* [RPC over JSON vs REST](./docs/tech/rpcvsrest.md) - Implement these common web API styles
* [Adaptable topology](./docs/blocks/topology.md) - Grow the topology of your system to match your requirements
* [Bootstrap a new project](./docs/howto/new-project.md) - Create a project for your solution
* [Create a new microservice](./docs/howto/create-microservice.md) - Create a new microservice and add it to your solution

### Under the Hood

* [HTTP ingress proxy](./docs/structure/coreservices-httpingress.md) - The HTTP ingress proxy bridges the gap between HTTP and `Microbus`
* [Unicast messaging](./docs/blocks/unicast.md) - Unicast enables bi-directional 1:1 request/response HTTP messaging between a client and a single server over the bus
* [Multicast messaging](./docs/tech/multicast.md) - Extending on the unicast pattern, multicast enables bi-directional 1:N publish/subscribe HTTP messaging between a client and a multitude of servers over the bus
* [Error capture](./docs/blocks/error-capture.md) - How and why errors are captured and propagated across microservices boundaries
* [Time budget](./docs/blocks/time-budget.md) - The right way to manage client-to-server request timeouts
* [Control subscriptions](./docs/tech/controlsubs.md) - Subscriptions that all microservices implement out of the box on port `:888`
* [Deployment environments](./docs/tech/deployments.md) - An application can run in one of 4 deployment environments: `PROD`, `LAB`, `LOCAL` and `TESTING`
* [Events](./docs/blocks/events.md) - How event-driven architecture can be used to decouple microservices
* [Distributed tracing](./docs/blocks/distrib-tracing.md) - Visualizing stack traces across microservices using OpenTelemetry and Jaeger
* [OpenAPI](./docs/blocks/openapi.md) - OpenAPI document generation for microservices
* [Local development](./docs/tech/local-dev.md) - Run an entire solution comprising a multitude of microservices in your local IDE

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

The `Microbus` framework is the copyrighted work of various contributors. It is licensed by `Microbus LLC` - a Delaware company formed to hold the combined intellectual property - under the [Apache License 2.0](./LICENSE).

Refer to the list of [third-party open source software](./docs/general/third-party-oss.md) for licensing information of components used by the `Microbus` framework.

[![Test](https://github.com/microbus-io/fabric/actions/workflows/test.yaml/badge.svg?branch=main&event=push)](https://github.com/microbus-io/fabric/actions/workflows/test.yaml)
