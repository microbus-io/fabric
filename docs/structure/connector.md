# Package `connector`

The `Connector` is the most fundamental construct of the framework, providing key capabilities to `Microbus` microservices:

* Startup and shutdown with corresponding callbacks
* Service host name and a random instance ID, both used to address the microservice
* [Connectivity to NATS](../tech/natsconnection.md)
* HTTP-like communication over NATS, unicast (request/response) and multicast (pub/sub)
* JSON logger
* [Configuration](../tech/configuration.md)
* Mockable clock
* Tickers to execute jobs on a fixed schedule
* Distributed cache

The `connector` package includes a separate source file for each functional area of the microservice. All these source files implement the same `Connector` class.

* `config.go` is responsible for fetching config values from the configurator system microservice
* `connector.go` defines the `Connector` struct and provides a few getters and setters
* `control.go` deals with subscribing and handling the control messages on the reserved port `:888`
* `fragment.go` orchestrates the fragmentation and defragmentation of large requests and responses
* `lifecycle.go` implements the `Startup` and `Shutdown` logic
* `logger.go` provides a JSON logger for the microservice
* `metrics.go` deals with prometheus metrics
* `publish.go` deals with outbound messaging
* `subjects.go` crafts the NATS subjects (topics) that a microservice subscribes to or publishes to
* `service.go` defines the public interface of the `Connector`
* `subscribe.go` deals with inbound message handling
* `time.go` introduces tickers and a mockable clock to the microservice
