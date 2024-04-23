# Distributed Tracing

Distributed tracing enables the visualization of the flow of function calls among microservices and across processes. Without distributed tracing, it's extremely difficult to understand the flow of calls across microservices and to troubleshoot a distributed system.
`Microbus` microservices use [OpenTelemetry](https://opentelemetry.io) to automatically create and collect tracing spans for executions of endpoints, tickers and callbacks.

`Microbus` exports tracing spans via the OTLP HTTP collector. The `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` or the `OTEL_EXPORTER_OTLP_ENDPOINT` environment variables may be used to configure the collector's endpoint.
By default, the endpoint is set to `http://127.0.0.1:4318` on a `LOCAL` deployment.
On all other deployments (`TESTINGAPP`, `LAB` and `PROD`), the endpoint must be explicitly specified by one of the aforementioned environment variables.

Whether or not a trace is exported to the collector depends on the deployment environment of the microservice:

- In `LOCAL` and `TESTINGAPP` deployments, all traces are exported to the collector
- In `PROD` and `LAB` deployments, only traces that contain at least one error span, or those that are explicitly selected using the query argument `trace=force` are exported to the collector

By default, `Microbus` uses [Jaeger](https://www.jaegertracing.io) to collect and visualize tracing data.
Although spans are created by disparate microservices and processes, Jaeger is able to present tracing spans belonging to a single trace as a unified stack-trace.

<img src="distribtracing-1.png" width="1011">
