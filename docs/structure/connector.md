# Package `connector`

The `Connector` provides key capabilities (or _building blocks_) to microservices deployed on the `Microbus` and is the most fundamental construct of the framework. In this release, the `Connector` includes the following building blocks:

* Startup and shutdown with corresponding callbacks
* Service host name and a random instance ID, both used to address the microservice
* Connectivity to NATS
* HTTP request/response model over NATS, both incoming (server) and outgoing (client)
* Rudimentary logger
* Configuration
* Deployment environment indicator (PROD, LAB, LOCAL, UNITTEST)
* Plane of communications

The `connector` package has multiple files for each functional area of the microservice but they all implement the same `Connector` class.

* `config.go` is responsible for fetching config values from environment variables or `env.yaml` file
* `connector.go` defines the `Connector` struct and provides a few getters and setters
* `interfaces.go` defines various interfaces of the microservice
* `lifecycle.go` implements the `Startup` and `Shutdown` logic
* `logger.go` implements a very basic logger
* `messaging.go` is perhaps the most interesting area of the connector. It implements an HTTP request/response model over NATS
* `subjects.go` crafts the NATS subjects (topics) that a microservice subscribes to or publishes to
