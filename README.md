# Microbus.io fabric : Milestone 3

<img src="docs\gopher-on-bus.png" width=256>

## Introduction

`Microbus` is a framework for the development, deployment and operation of microservices in Go. Its most notable characteristic is that it leverages NATS, a messaging bus, for communications among microservices.

`fabric` is the main project that provides the core capabilities that all `Microbus` microservices require. Every milestone of this project will be released separately in order to demonstrate the thinking process behind the building of this framework. This is the third milestone.

[Milestone 1](https://github.com/microbus-io/fabric/tree/milestone/1):

* `Connector` construct, including startup and shutdown logic
* Config from environment or file
* Basic logger
* HTTP request/response messaging over NATS
* HTTP ingress proxy

[Milestone 2](https://github.com/microbus-io/fabric/tree/milestone/2):

* Error capture and propagation
* Time budget for requests

[Milestone 3](https://github.com/microbus-io/fabric/tree/milestone/2) <sup style="color:orange">new</sup>:

* `Application` construct to group microservices
* NATS connection settings
* Deployment environment indicator (`PROD`, `LAB`, `LOCAL`)
* Plane of communication

## Documentation

[Get started quickly](docs/quick-start.md) by setting up your system and running the examples.

Review each of the major project packages to get oriented in the code structure:

* [application](docs/structure/application.md) <sup style="color:orange">new</sup> - A collector of microservices that run in a single process and share the same lifecycle
* [connector](docs/structure/connector.md) <sup style="color:orange">updated</sup> - The primary construct of the framework and the basis for all microservices
* [errors](docs/structure/errors.md) - An enhancement of the standard Go's `errors` package 
* [examples](docs/structure/examples.md) - Demo microservices 
* [frame](docs/structure/frame.md) - A utility for type-safe manipulation of the HTTP control headers used by the framework
* [pub](docs/structure/pub.md) - Options for publishing requests
* [rand](docs/structure/rand.md) - A utility for generating random numbers
* [services/httpingress](docs/structure/services-httpingress.md) - The HTTP ingress proxy service
* [sub](docs/structure/sub.md) - Options for subscribing to handle requests

Go into the details with these technical deep dives:

* [Messaging](docs/tech/messaging.md) <sup style="color:orange">updated</sup> - How HTTP-like request/response pattern is achieved over the NATS messaging bus
* [HTTP ingress](docs/tech/httpingress.md) - The reason for and role of the HTTP ingress proxy service
* [Encapsulation pattern](docs/tech/encapsulation.md) - The reasons for encapsulating third-party technologies
* [Error capture](docs/tech/errorcapture.md) - How and why errors are captured and propagated across microservices boundaries
* [Time budget](docs/tech/timebudget.md) - The proper way to handle request timeouts
* [Environment variables](docs/tech/envars.md) <sup style="color:orange">new</sup> - A catalog of the environment variables used by the framework
* [NATS connection settings](docs/tech/natsconnection.md) <sup style="color:orange">new</sup> - How to configure microservices to connect to NATS

## Cutting Some Corners

This milestone is taking several shortcuts that will be addressed in future releases:

* The timeouts for the `OnStartup` and `OnShutdown` callbacks are hard-coded to `time.Minute`
* The network hop duration is hard-coded to `250 * time.Millisecond`
* The logger is rudimentary

## More to Explore

A few suggestions for self-guided exploration:

* Start NATS in debug mode `./nats-server -D -V`, run unit tests individually and look at the messages going over the bus
* Modify `examples/main/env.yaml` and witness the impact on the `helloworld.example` microservice
* Add an endpoint `/calculate` to the `calculator.example` microservice that operates on decimal numbers, not just integers
* Create your own microservice from scratch and add it to `examples/main/main.go`
* Put a breakpoint in any of the microservices of the example application and try debugging
