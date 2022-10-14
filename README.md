# Microbus.io fabric : Milestone 6

<img src="docs\gopher-on-bus.png" width=256>

## Introduction

`Microbus` is a framework for the development, deployment and operation of microservices in Go. Its most notable characteristic is that it leverages NATS, a messaging bus, for communications among microservices.

`fabric` is the main project that provides the core capabilities that all `Microbus` microservices require. The [milestones](docs/milestones.md) of this project are maintained in separate branches in order to demonstrate the thinking process behind the building of this framework.

## Quick Start

[Get started quickly](docs/quick-start.md) by setting up your system and running the examples.

## Code Structure

Review each of the major project packages to get oriented in the code structure:

* [application](docs/structure/application.md) - A collector of microservices that run in a single process and share the same lifecycle
* [connector](docs/structure/connector.md) <sup color="orange">updated</sup> - The primary construct of the framework and the basis for all microservices
* [errors](docs/structure/errors.md) - An enhancement of Go's standard `errors` package 
* [examples](docs/structure/examples.md) - Demo microservices 
* [frag](docs/structure/frag.md) <sup color="orange">new</sup> - Means to break large HTTP requests and responses into fragments that can then be reassembled
* [frame](docs/structure/frame.md) - A utility for type-safe manipulation of the HTTP control headers used by the framework
* [log](docs/structure/log.md) - Fields for attaching data to log messages
* [pub](docs/structure/pub.md) - Options for publishing requests
* [rand](docs/structure/rand.md) - A utility for generating random numbers
* [services/httpingress](docs/structure/services-httpingress.md) - The HTTP ingress proxy service
* [sub](docs/structure/sub.md) - Options for subscribing to handle requests
* [utils](docs/structure/utils.md) <sup color="orange">new</sup> - Various independent utility classes and functions

## Technical Deep Dive

Go into the details with these technical guides:

* [Unicast messaging](docs/tech/unicast.md) - Unicast enables bi-directional (request and response) HTTP-like messaging between a client and a single server over NATS
* [HTTP ingress](docs/tech/httpingress.md) - The reason for and role of the HTTP ingress proxy service
* [Encapsulation pattern](docs/tech/encapsulation.md) - The reasons for encapsulating third-party technologies
* [Error capture](docs/tech/errorcapture.md) - How and why errors are captured and propagated across microservices boundaries
* [Time budget](docs/tech/timebudget.md) - The proper way to manage request timeouts
* [Configuration](docs/tech/configuration.md) - How to configure microservices via environment variables or an `env.yaml` file
* [NATS connection settings](docs/tech/natsconnection.md) - How to configure microservices to connect and authenticate to NATS
* [Multicast messaging](docs/tech/multicast.md) - Extending on the unicast pattern, multicast enables bi-directional (request and response) HTTP-like messaging between a client and multiple servers over NATS
* [Control subscriptions](docs/tech/controlsubs.md) - Subscriptions that all microservices implement out of the box on port `:888`

Note the [shortcuts](docs/shortcuts.md) <sup color="orange">updated</sup> taken by this milestone. These will be addressed in future releases.

Get your hands dirty and [explore more on your own](docs/self-explore.md).
