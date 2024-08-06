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
* Mockable clock (deprecated in milestone 19)
* Context for the `Lifetime` of the microservice
* Customizable time budget for callbacks (deprecated in milestone 19)
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
* Use `InfiniteChan`s to avoid blocking messaging channels in high-load situations (deprecated in milestone 23)

[Milestone 12](https://github.com/microbus-io/fabric/tree/milestone/12):

* Improvements to how the `Application` manages the lifecycle of microservices
* Code generation of an integration test harness
* Code generation of a mockable stub for microservices
* Restarting of microservices that were previously shutdown
* `TESTING` deployment environment in which tickers and the configurator are disabled
* Integration tests for the example microservices

[Milestone 13](https://github.com/microbus-io/fabric/tree/milestone/13):

* Sharded MySQL database (deprecated in milestone 21)
* Sharding key allocation management
* Differential schema migration
* Code generation of MySQL boilerplate code
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
* Metrics core microservice that collects and delivers metrics to Prometheus and Grafana
* Code generation for solution-specific metrics
* Quick start with Docker Compose

[Milestone 16](https://github.com/microbus-io/fabric/tree/milestone/16):

* License and copyright notices
* MariaDB database support (deprecated in milestone 21)
* HTTP ingress middleware

[Milestone 17](https://github.com/microbus-io/fabric/tree/milestone/17):

* Code generation of tests for event sources
* `any` error annotation types instead of `string`
* `rand.Read`
* Do not return an error if a multicast returns zero responses
* HTTP ingress proxy to respect `Request-Timeout` header
* Capture full stack trace during panics
* Reset ack timeout during debugging
* Fixed bug in conveying the time budget downstream
* Fixed bug in conveying baggage and `X-Forwarded` headers downstream
* Brotli compression by HTTP ingress proxy
* HTTP ingress proxy to bypass the middleware when contacting the metrics microservice
* HTTP ingress proxy to transform `/` requests to `/root`
* Deprecated the need to define types in `service.yaml`
* Simplified distributed stack trace
* Ignore debug level log message unless `MICROBUS_LOG_DEBUG` is set
* Blocking paths in the ingress proxy
* Inbox microservice (renamed in milestone 26)
* Introduced `Go` in `Connector` to run goroutines safely in the context of the microservice
* HTTP ingress proxy adds `X-Forwarded-Path` header

[Milestone 18](https://github.com/microbus-io/fabric/tree/milestone/18):

* Handle service resources natively in the `Connector` rather than in the code generation layer
* Enable initializing the `Connector` with a custom `fs.FS`
* `ServerLanguages` configuration in the HTTP Ingress microservice determines the best language to display to the user based on the `Accept-Language` request header
* `LoadResString` in `Connector` loads a string from the `strings.yaml` resource file localized to the language of the request
* Introduced `Parallel` in `Connector` to run multiple jobs in parallel
* Startup `Group`s instead of `Connector` startup sequence

[Milestone 19](https://github.com/microbus-io/fabric/tree/milestone/19):

* Distributed tracing with OpenTelemetry and Jaeger
* Deprecated the concept of the time budget for callbacks and removed the `cb` package
* Deprecated the concept of the mockable clock and removed the `clock` package
* Introduced clock shifting via the `ctx` and adjusted `connector.Now` to accept a `ctx`

[Milestone 20](https://github.com/microbus-io/fabric/tree/milestone/20):

* Code generation of OpenAPI endpoints for microservices at `openapi.json`
* OpenAPI portal core microservice to aggregate and display links to all microservices with an OpenAPI
* Subscriptions to specific HTTP methods
* Subscriptions to any port using an `:*` in the path

[Milestone 21](https://github.com/microbus-io/fabric/tree/milestone/21):

* HTTP egress core microservice
* Added browser example to demonstrate use of HTTP egress
* Deprecate `BatchLookup` in the `shardedsql` package
* Block all requests to internal port `:888` in ingress proxy
* Improve performance in `rand` package
* Renamed the `services` package to `coreservices`
* Replaced the `Service` interface in the `connector` package with various interfaces in a new `service` package
* Adjusted the pattern of including microservices in an `Application`
* Deprecated the SQL database dependency and its corresponding code generation
* Handling of `errors.Join`
* Deprecated error annotations
* Auto-detect `TESTING` deployment based on call stack
* Distributed tracing improvements

[Milestone 22](https://github.com/microbus-io/fabric/tree/milestone/22):

* Create type-safe setter functions for each config property
* Refactor the interface exposed by the client for web endpoints
* New test case asserters, mostly for web endpoints
* Simulate a database in memory for the directory example if MariaDB is not available
* Support multi-value headers in requests and responses

[Milestone 23](https://github.com/microbus-io/fabric/tree/milestone/23):

* Bug fixes and resiliency improvements
* Hello example microservice to handle the topmost root path
* Deprecated `InfiniteChan` and replaced with an internal `transferChan`
* Middleware chain inside HTTP ingress proxy instead of delegating to a remote microservice
* Enhancements to distributed tracing and metric collection

[Milestone 24](https://github.com/microbus-io/fabric/tree/milestone/24):

* Use port `:0` to subscribe to any port instead of `:*`
* Use method `ANY` to subscribe to any method instead of `*`
* Path arguments `{arg}` and `{greedy+}`
* `httpRequestBody` and `httpResponseBody` magic arguments in functions
* Change the directory example to RESTful API style
* Introduced a microservice for testing of the code generator
* Deprecated variadic options in code-generated clients

[Milestone 25](https://github.com/microbus-io/fabric/tree/milestone/25):

* Documentation

[Milestone 26](https://github.com/microbus-io/fabric/tree/milestone/26):

* Locality-aware routing given a `MICROBUS_LOCALITY` environment variable
* Determine locality automatically from availability zone name in AWS or GCP
* Renamed the inbox core microservice to SMTP ingress
* Change hostname suffix of core microservices to `.core`
* New pattern for adding and removing microservices in an `Application`

[Milestone 27](https://github.com/microbus-io/fabric/tree/milestone/27):

* Remove dependency on Testify

[Milestone 28](https://github.com/microbus-io/fabric/tree/milestone/28):

* Remove dependency on Zap logger and replaced with standard slog

[Milestone 29](https://github.com/microbus-io/fabric/tree/milestone/29):

* Documentation
* Fixed data race in metrics collection
* Added `connector.StopTicker`
* Fixed deadlock when running tickers
