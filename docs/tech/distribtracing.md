# Distributed Tracing

Distributed tracing enables the visualization of the flow of function calls among microservices and across processes. Without distributed tracing, it's extremely difficult to understand the flow of calls across microservices and to troubleshoot a distributed system.
`Microbus` microservices use [`OpenTelemetry`](https://opentelemetry.io) to automatically create and collect tracing spans for executions of endpoints, tickers and callbacks.

`Microbus` exports tracing spans to Jaeger via the OTLP gRPC collector at the default endpoint of `127.0.0.1:4317`. The `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable can be used to configure this.

Whether or not a trace is exported to the collector depends on the deployment environment of the microservice:

- In `LOCAL` deployments, all traces are exported to the collector
- In `PROD` and `LAB` deployments, only traces that contain at least one error span, or those that are explicitly selected using the query argument `trace=force` are exported to the collector
- In `TESTINGAPP` deployments, tracing is disabled

By default, `Microbus` uses [`Jaeger`](https://www.jaegertracing.io) to collect and visualize tracing data.
Although spans are created by disparate microservices and processes, `Jaeger` is able to present tracing spans belonging to a single trace as a unified stack-trace.

<img src="distribtracing-1.png" width="1011">
