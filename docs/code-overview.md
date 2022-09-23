# Code Overview

## Connector

The `Connector` provides key capabilities (or _building blocks_) to microservices deployed on the `Microbus` and is the most fundamental construct of the framework. In this release, the `Connector` includes the following building blocks:

* Startup and shutdown with corresponding callbacks
* Service host name and a random instance ID, both used to address the microservice
* Connectivity to NATS
* HTTP request/response model over NATS, both incoming (server) and outgoing (client)
* Basic logger
* Configuration

The `connector` package has multiple files for each functional area of the microservice but they all implement the same `Connector` class.

* `config.go` is responsible for fetching config values from environment variables or `env.yaml` file
* `connector.go` defines the `Connector` struct and provides a few getters and setters
* `lifecycle.go` implements the `Startup` and `Shutdown` logic
* `logger.go` implements a very basic logger
* `messaging.go` is perhaps the most interesting area of the connector. It implements an HTTP request/response model over NATS
* `subjects.go` crafts the NATS subjects (topics) that a microservice subscribes to or publishes to

## Examples

The `examples` package holds several examples that demonstrate how the `Connector` can be used to create microservices:

* Echo microservice
* Calculator microservice
* Hello World microservice

## Rand

The `rand` package is a utility that combines `crypto.rand` and `math.rand` for more secure random number generation with reduced performance impact.

## HTTP Ingress Proxy

The `services` package is reserved for essential microservices, the first of which - the HTTP ingress proxy `services/httpingress` - is included in this release. More services will be added in the future.

An HTTP ingress proxy is needed in order to bridge the gap between the browser and NATS because NATS is a closed network that requires a special type of connection. On one end the ingress proxy is listening to HTTP requests and on the other end it is connected to the NATS network. More on that in the [Deep Dive](docs/deep-dive.md).
