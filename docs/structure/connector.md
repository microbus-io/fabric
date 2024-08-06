# Package `connector`

The `Connector` is the most fundamental construct of the framework, providing key capabilities to `Microbus` microservices:

* Startup and shutdown with corresponding callbacks
* Service hostname and a random instance ID, both used to address the microservice
* [Connectivity to NATS](../tech/nats-connection.md)
* HTTP-like communication over NATS, [unicast](../blocks/unicast.md) (request/response) and [multicast](../blocks/multicast.md) (pub/sub)
* JSON logger
* [Configuration](../blocks/configuration.md)
* [Tickers](../blocks/tickers.md) to execute jobs on a recurring basis
* [Distributed cache](../blocks/distrib-cache.md)
* Bundled resource files and strings
* [Distributed tracing](../blocks/distrib-tracing.md)

The `connector` package includes a separate source file for each functional area of the microservice. All these source files implement the same `Connector` class.

* `config.go` is responsible for fetching config values from the configurator core microservice
* `connector.go` defines the `Connector` struct and provides a few getters and setters
* `control.go` deals with subscribing and handling the control messages on the reserved port `:888`
* `fragment.go` orchestrates the fragmentation and defragmentation of large requests and responses
* `lifecycle.go` implements the `Startup` and `Shutdown` logic, as well as `Go` and `Parallel` for running code in goroutines
* `logger.go` provides a JSON logger for the microservice
* `metrics.go` collects metrics using Prometheus
* `muffler.go` is an OpenTelemetry span sampler that excludes noisy spans
* `publish.go` deals with outbound messaging
* `res.go` manages the loading of files and localized strings from a resource `FS`
* `selectiveprocessor.go` is an OpenTelemetry processor of tracing spans that exports only spans that are explicitly selected
* `subjects.go` crafts the NATS subjects (topics) that a microservice subscribes to or publishes to
* `subscribe.go` deals with inbound message handling
* `telemetry.go` supports distributed tracing with OpenTelemetry
* `time.go` introduces tickers to the microservice
