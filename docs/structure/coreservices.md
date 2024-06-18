# Package `coreservices`

The `coreservices` package is a grouping of microservices that are required by most if not all applications running on top of `Microbus`.

* The [configurator](../structure/coreservices-configurator.md) is responsible for delivering configuration values to microservices that define configuration properties. Such microservices will not start if they cannot reach the configurator
* [Control](../structure/coreservices-control.md) is not actually a microservice but rather a stub microservice used to generate a client for the `:888` [control subscriptions](../tech/controlsubs.md)
* The [HTTP egress proxy](../structure/coreservices-httpegress.md) relays HTTP requests to non-`Microbus` URLs
* The [HTTP ingress proxy](../structure/coreservices-httpingress.md) bridges the gap between HTTP clients and the microservices running on `Microbus`
* The [inbox](../structure/coreservices-inbox.md) microservice transforms incoming emails to actionable events
* The [metrics](../structure/coreservices-metrics.md) microservice aggregates metrics from all microservices in response to a request from Prometheus
* The [OpenAPI portal](../structure/coreservices-openapiportal.md) microservice renders a catalog of the OpenAPI endpoints of all microservices.
