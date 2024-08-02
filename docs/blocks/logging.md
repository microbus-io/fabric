# Structured Logging

Logging is one of the pillars of observability alongside [distributed tracing](../blocks/distrib-tracing.md) and [metrics](../blocks/metrics.md). `Microbus` uses Go's standard `log/slog` for logging with output directed to `stderr`.

Logs are printed in human-friendly format in the `LOCAL` and `TESTING` [deployment environments](../tech/deployments.md) and in JSON in `PROD` and `LAB`.

Logs are automatically enriched with the microservice's hostname, version number and ID, as well by the distributed trace ID, if applicable.

In addition, logs are metered on a per-message basis to make them visible in Grafana. For this reason, the message part should be a fixed string, and all variable parts added as arguments.

The `Connector` supports 4 methods for logging at different severity levels: `LogDebug`, `LogInfo`, `LogWarn` and `LogError`. Debug logs are ignored unless the `MICROBUS_LOG_DEBUG` [environment variable](../tech/envars.md) is set.

Example:

```go
c.LogInfo(ctx, "Fixed message",
    "key1", "value",
    "key2", 1234,
)
```
