# Microbus.io fabric : Milestone 4

<img src="docs\gopher-on-bus.png" width=256>

## Introduction

`Microbus` is a framework for the development, deployment and operation of microservices in Go. Its most notable characteristic is that it leverages NATS, a messaging bus, for communications among microservices.

`fabric` is the main project that provides the core capabilities that all `Microbus` microservices require. The [milestones](docs/milestones.md) of this project will be maintained in separate branches in order to demonstrate the thinking process behind the building of this framework.

## Quick Start

[Get started quickly](docs/quick-start.md) by setting up your system and running the examples.

## Code Structure

Review each of the major project packages to get oriented in the code structure:

* [application](docs/structure/application.md) - A collector of microservices that run in a single process and share the same lifecycle
* [connector](docs/structure/connector.md) - The primary construct of the framework and the basis for all microservices
* [errors](docs/structure/errors.md) - An enhancement of the standard Go's `errors` package 
* [examples](docs/structure/examples.md) - Demo microservices 
* [frame](docs/structure/frame.md) - A utility for type-safe manipulation of the HTTP control headers used by the framework
* [pub](docs/structure/pub.md) - Options for publishing requests
* [rand](docs/structure/rand.md) - A utility for generating random numbers
* [services/httpingress](docs/structure/services-httpingress.md) - The HTTP ingress proxy service
* [sub](docs/structure/sub.md) - Options for subscribing to handle requests

## Technical Deep Dive

Go into the details with these technical guides:

* [Messaging](docs/tech/messaging.md) - How HTTP-like request/response pattern is achieved over the NATS messaging bus
* [HTTP ingress](docs/tech/httpingress.md) - The reason for and role of the HTTP ingress proxy service
* [Encapsulation pattern](docs/tech/encapsulation.md) - The reasons for encapsulating third-party technologies
* [Error capture](docs/tech/errorcapture.md) - How and why errors are captured and propagated across microservices boundaries
* [Time budget](docs/tech/timebudget.md) - The proper way to handle request timeouts
* [Configuration](docs/tech/configuration.md) - How to configure microservices via environment variables or an `env.yaml` file
* [NATS connection settings](docs/tech/natsconnection.md) - How to configure microservices to connect to NATS

Note the [shortcuts](docs/shortcuts.md) taken by this milestone. These will be addressed in future releases.

Get your hands dirty and [explore more on your own](docs/self-explore.md).
