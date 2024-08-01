# Packages

Learn about each of the project's packages to get familiar with the `Microbus` codebase.

* [application](../structure/application.md) - A collector of microservices that run in a single process and share the same lifecycle
* [cfg](../structure/cfg.md) - Options for defining config properties
* [codegen](../structure/codegen.md) - The code generator
* [connector](../structure/connector.md) - The primary construct of the framework and the basis for all microservices
* [coreservices](../structure/coreservices.md) - Microservices that are required for most if not all apps
    * [configurator](../structure/coreservices-configurator.md) - The configurator core microservice
    * [control](../structure/coreservices-control.md) - Client API for the `:888` control subscriptions
    * [httpegress](../structure/coreservices-httpegress.md) - The HTTP egress proxy core microservice
    * [httpingress](../structure/coreservices-httpingress.md) - The HTTP ingress proxy core microservice
    * [metrics](../structure/coreservices-metrics.md) - The metrics microservice collects metrics from microservices and delivers them to Prometheus and Grafana
    * [openapiportal](../structure/coreservices-openapiportal.md) - The OpenAPI portal microservice produces a portal page that lists all microservices with open endpoints
    * [smtpingress](../structure/coreservices-smtpingress.md) - The SMTP ingress microservice listens for incoming emails and fires appropriate events
* [dlru](../structure/dlru.md) - A distributed LRU cache that is shared among all peers of a microservice
* [env](../structure/env.md) - Manages the loading of environment variables, with the option of overriding values for testing
* [errors](../structure/errors.md) - An enhancement of Go's standard `errors` package that adds stack tracing and status codes
* [examples](../structure/examples.md) - Demo microservices
    * [browser](../structure/examples-browser.md) is an example of a microservice that uses the [HTTP egress core microservice](../structure/coreservices-httpegress.md)
    * [calculator](../structure/examples-calculator.md) demonstrates functional handlers
    * [directory](../structure/examples-directory.md) is an example of a microservice that provides a RESTful CRUD API backed by a database
    * [eventsink and eventsource](../structure/examples-events.md) shows how events can be used to reverse the dependency between two microservices
    * [hello](../structure/examples-hello.md) demonstrates the key capabilities of the framework
    * [helloworld](../structure/examples-helloworld.md) demonstrates the minimalist classic example
    * [messaging](../structure/examples-messaging.md) demonstrates load-balanced unicast, multicast and direct addressing messaging
* [frame](../structure/frame.md) - A utility for type-safe manipulation of the HTTP control headers used by the framework
* [httpx](../structure/httpx.md) - Various HTTP utilities
* [lru](../structure/lru.md) - An LRU cache with limits on age and weight
* [openapi](../structure/openapi.md) - OpenAPI document generator
* [pub](../structure/pub.md) - Options for publishing requests over the bus
* [rand](../structure/rand.md) - A utility for generating random numbers and identifiers
* [service](../structure/service.md) - Interface definitions of microservices
* [sub](../structure/sub.md) - Options for subscribing to handle requests over the bus
* [trc](../structure/trc.md) - Options for creating tracing spans
* [timex](../structure/timex.md) - Enhancement of Go's standard `time.Time`
* [utils](../structure/utils.md) - Miscellaneous utility classes and functions

