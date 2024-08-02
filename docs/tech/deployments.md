# Deployments

`Microbus` recognizes four deployment environments:

* `PROD` represents a production deployment
* `LAB` represents a fully-functional non-production deployments such as dev integration, testing, staging, etc.
* `LOCAL` represents development on an engineer's local machine
* `TESTING` represents a unit test running a testing application

The deployment environment impacts certain aspects of the framework such as [structured logging](../blocks/logging.md) and [distributed tracing](../blocks/distrib-tracing.md).

| |`PROD`|`LAB`|`LOCAL`|`TESTING`|
|--------|----|---|-----|----------|
|Logging level|INFO|DEBUG|DEBUG|DEBUG|
|Logging format|JSON|JSON|Human-friendly|Human-friendly|
|Logging errors|Standard|Standard|Emphasized|Emphasized|
|Distributed tracing|Selective|Everything|Everything|Everything|
|Configurator|Enabled|Enabled|Enabled|Disabled|
|Tickers|Enabled|Enabled|Enabled|Disabled|
|Error output|Redacted|Stack trace|Stack trace|Stack trace|

The deployment environment is set according to the value of the `MICROBUS_DEPLOYMENT` [environment variable](../tech/envars.md). If not specified, `PROD` is assumed, unless connecting to NATS on `nats://localhost:4222` or `nats://127.0.0.1:4222` in which case `LOCAL` is assumed.
