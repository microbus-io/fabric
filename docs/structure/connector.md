# Package `connector`

The `Connector` provides key capabilities (or _building blocks_) to microservices deployed on the `Microbus` and is the most fundamental construct of the framework. It includes the following building blocks:

* Startup and shutdown with corresponding callbacks
* Service host name and a random instance ID, both used to address the microservice
* [Connectivity to NATS](../tech/natsconnection.md)
* HTTP-like communication over NATS, unicast (request/response) and multicast (pub/sub)
* JSON logger
* [Configuration](../tech/configuration.md)

The `connector` package includes a separate source file for each functional area of the microservice. All these source files implement the same `Connector` class.

* `config.go` is responsible for fetching config values from environment variables or an `env.yaml` file
* `connector.go` defines the `Connector` struct and provides a few getters and setters
* `control.go` deals with subscribing and handling the control messages on the reserved port `:888`
* `frag.go` orchestrates the fragmentation and defragmentation of large requests and responses
* `interfaces.go` defines various interfaces of the microservice
* `lifecycle.go` implements the `Startup` and `Shutdown` logic
* `logger.go` provides a JSON logger for the microservice
* `publish.go` deals with outbound messaging
* `subjects.go` crafts the NATS subjects (topics) that a microservice subscribes to or publishes to
* `subscribe.go` deals with inbound message handling
