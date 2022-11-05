# Microbus.io fabric : Milestone 10

<img src="docs/gopher-on-bus.png" width=256>

## Introduction

`Microbus` is an opinionated framework for the development, deployment and operation of microservices. Its most notable characteristic is that it leverages NATS, a messaging bus, for communications among microservices. Microservices over a bus = microbus.

The framework involves itself in three conceptual areas:

* Common runtime - The framework specifies rules that all microservices need to comply with during runtime. This includes things like how microservices [communicate](docs/tech/unicast.md), how they report metrics, where they [pull config values](docs/tech/configuration.md) from, how they get deployed, etc. A common set of rules is important for proper interoperability and the stability of the system as a whole
* RAD tools - The framework leverages [code generation](docs/tech/codegen.md) for the rapid development of microservices with the intent that application developers focus on the business logic rather than on boilerplate code. Developer productivity is directly correlated to customer value
* Building blocks - Capabilities that are deemed to be the building blocks of microservices are implemented by the framework in a standard way, serving to facilitate both of the above

`fabric` is the main project that provides the core capabilities that all `Microbus` microservices are built on. The [milestones](docs/milestones.md) of this project are maintained in separate branches in order to demonstrate the thinking process behind the building of this framework.

## Getting Started

Follow the [quick start guide](docs/quick-start.md) <sup color="orange">updated</sup> to set up your system and run the example app.

Go through the [examples](docs/structure/examples.md) <sup color="orange">updated</sup> in depth.

Follow the step-by-step guide and [build your first microservice](docs/first-service.md) <sup color="orange">new</sup>!

Discover the full power of [code generation](docs/tech/codegen.md) <sup color="orange">new</sup> in `Microbus`. It's totally RAD, dude!

Venture out and [explore more on your own](docs/self-explore.md).

## Code Structure

Review each of the major project packages to get oriented in the code structure:

* [application](docs/structure/application.md) - A collector of microservices that run in a single process and share the same lifecycle
* [cb](docs/structure/cb.md) - Options for callbacks
* [cfg](docs/structure/cb.md) - Options for defining config properties
* [clock](docs/structure/clock.md) - An abstraction of the functions in the standard library time package to allow for mocking
* [codegen](docs/structure/codegen.md) <sup color="orange">new</sup> - The code generator
* [connector](docs/structure/connector.md) - The primary construct of the framework and the basis for all microservices
* [dlru](docs/structure/dlru.md) - An LRU cache that is distributed among all peers of a microservice
* [errors](docs/structure/errors.md) - An enhancement of Go's standard `errors` package 
* [examples](docs/structure/examples.md) <sup color="orange">updated</sup> - Demo microservices 
* [frag](docs/structure/frag.md) - Means to break large HTTP requests and responses into fragments that can then be reassembled
* [frame](docs/structure/frame.md) - A utility for type-safe manipulation of the HTTP control headers used by the framework
* [log](docs/structure/log.md) - Fields for attaching data to log messages
* [lru](docs/structure/lru.md) - An LRU with with limits on age and weight
* [pub](docs/structure/pub.md) - Options for publishing requests
* [rand](docs/structure/rand.md) - A utility for generating random numbers
* [services/configurator](docs/structure/services-configurator.md) - The configurator system microservice
* [services/control](docs/structure/services-control.md) <sup color="orange">new</sup> - Client API for the [control subscriptions](docs/tech/controlsubs.md)
* [services/httpingress](docs/structure/services-httpingress.md) - The HTTP ingress proxy system microservice
* [sub](docs/structure/sub.md) - Options for subscribing to handle requests
* [utils](docs/structure/utils.md) <sup color="orange">updated</sup> - Various independent utility classes and functions

## Technical Deep Dive

Get a deeper understanding of the principals and philosophy of `Microbus`:

* [Unicast messaging](docs/tech/unicast.md) - Unicast enables bi-directional (request and response) HTTP-like messaging between a client and a single server over NATS
* [HTTP ingress](docs/tech/httpingress.md) - The reason for and role of the HTTP ingress proxy service
* [Encapsulation pattern](docs/tech/encapsulation.md) - The reasons for encapsulating third-party technologies
* [Error capture](docs/tech/errorcapture.md) - How and why errors are captured and propagated across microservices boundaries
* [Time budget](docs/tech/timebudget.md) - The proper way to manage request timeouts
* [Configuration](docs/tech/configuration.md) - How to configure microservices
* [NATS connection settings](docs/tech/natsconnection.md) - How to configure microservices to connect and authenticate to NATS
* [Multicast messaging](docs/tech/multicast.md) - Extending on the unicast pattern, multicast enables bi-directional (request and response) HTTP-like messaging between a client and multiple servers over NATS
* [Control subscriptions](docs/tech/controlsubs.md) <sup color="orange">updated</sup> - Subscriptions that all microservices implement out of the box on port `:888`
* [Environment variables](docs/tech/envars.md) - Environment variables used to initialize microservices
* [Code generation](docs/tech/codegen.md) <sup color="orange">new</sup> - Discover the power of `Microbus`'s most powerful RAD tool
