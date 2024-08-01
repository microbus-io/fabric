# Package `coreservices/metrics`

The `Microbus` framework relies on Prometheus for the collection of [metrics](../blocks/metrics.md). Prometheus pulls metrics from the metrics core microservice, which in turn pulls and aggregates metrics from all microservices it can reach on the messaging bus.

<img src="coreservices-metrics-1.drawio.svg">
<p></p>

The endpoint to obtain metrics from the metrics microservice is `https://localhost:8080/metrics.core/collect`. An optional argument `service` can be used to obtain the metrics of an individual service. The `secretkey` argument is mandatory except in local development and testing. It must match the value set for the `SecretKey` configuration property or else the request will be refused.

Metrics can also be obtained from a microservice directly at `https://localhost:8080/hello.example:888/metrics`.
