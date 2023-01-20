# Milestones

Note: These milestones are maintained in separate branches in order to demonstrate the thinking process behind the building of this framework. Milestone are not releases and they are not production-ready.

[Milestone 1](https://github.com/microbus-io/fabric/tree/milestone/1):

* `Connector` construct, including startup and shutdown logic
* Config from environment or file
* Rudimentary logger
* Bi-directional (request/response) HTTP-like unicast messaging over NATS
* HTTP ingress proxy

[Milestone 2](https://github.com/microbus-io/fabric/tree/milestone/2):

* Error capture and propagation
* Time budget for requests

[Milestone 3](https://github.com/microbus-io/fabric/tree/milestone/3):

* `Application` construct to group microservices
* NATS connection settings
* Deployment environment indicator (`PROD`, `LAB`, `LOCAL`)
* Plane of communication

[Milestone 4](https://github.com/microbus-io/fabric/tree/milestone/4):

* Acks and quick timeouts
* Bi-directional (request/response) HTTP-like *multicast* messaging over NATS
* Direct addressing
* Control messages on port `:888`

[Milestone 5](https://github.com/microbus-io/fabric/tree/milestone/5):

* Advanced logger with JSON output

[Milestone 6](https://github.com/microbus-io/fabric/tree/milestone/6):

* Fragmentation of large messages
* Optimized HTTP utilities (`BodyReader` and `ResponseRecorder`)

[Milestone 7](https://github.com/microbus-io/fabric/tree/milestone/7):

* Tickers
* Mockable clock
* Context for the `Lifetime` of the microservice
* Customizable time budget for callbacks
* Drain pending operations during shutdown

[Milestone 8](https://github.com/microbus-io/fabric/tree/milestone/8):

* Configurator microservice to centralize the dissemination of configuration values to other microservices
* Config property definition with value validation
* `Connector` description
* Update of NATS connection environment variable names
* Update of deployment and plane environment variable names

[Milestone 9](https://github.com/microbus-io/fabric/tree/milestone/9):

* LRU cache
* Distributed LRU cache

[Milestone 10](https://github.com/microbus-io/fabric/tree/milestone/10):

* Code generation tool
* Code generation to bootstrap new microservices
* Code generation of config definitions and accessors
* Code generation of web handlers
* Code generation of functional handlers (JSON over HTTP)
* Code generation of tickers
* Code generation of complex types
* Code change detection and automatic microservice versioning
* Embedded resources
* Clients for the port `:888` control subscriptions
* Code generator automatically adding `errors.Trace` to returned errors
* Capturing errors during initialization and failing startup

[Milestone 11](https://github.com/microbus-io/fabric/tree/milestone/11):

* Code generation of event sources
* Code generation of event sinks
* Use `InfiniteChan`s to avoid blocking messaging channels in high-load situations

[Milestone 12](https://github.com/microbus-io/fabric/tree/milestone/12):

* Improvements to how the `Application` manages the lifecycle of microservices
* Code generation of an integration test harness
* Code generation of a mockable stub for microservices
* Restarting of microservices that were previously shutdown
* `TESTINGAPP` deployment environment in which tickers and the configurator are disabled
* Integration tests for the example microservices

[Milestone 13](https://github.com/microbus-io/fabric/tree/milestone/13):

* Sharded `MySQL` database
* Sharding key allocation management
* Differential schema migration
* Code generation of `MySQL` boilerplate code
* Allow attaching multiple lifecycle and config change callbacks to the `Connector`
* `NullTime` utility to better handle serialization of the zero time value

[Milestone 14](https://github.com/microbus-io/fabric/tree/milestone/14):

* Extended `TracedError` with HTTP status code
* Handle CORS preflight and origin
* HTTP ingress protections: memory usage limit, read timeout, read header timeout, write timeout
* Compress if seeing `Accept-Encoding` header 
* Handle `X-Forwarded` headers
* TLS in HTTP ingress
* Multiple HTTP ports for HTTP ingress, with mapping for internal ports

[Milestone 15](https://github.com/microbus-io/fabric/tree/milestone/15):

* Support in `Connector` for collecting system metrics
* Metrics system microservice that collects and delivers metrics to `Prometheus` and `Grafana`
* Code generation for application custom metrics
* Quick start with Docker Compose

[Milestone 16](https://github.com/microbus-io/fabric/tree/milestone/16):

* License and copyright notices
* `MariaDB` database support
* HTTP ingress middleware
