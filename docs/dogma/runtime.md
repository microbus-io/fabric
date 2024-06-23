# Robust Runtime

It's the arrow between the boxes that's the challenge
A consistent behavior makes it easier to reason about the accuracy of the system
If some services would use thrift, some grpc, some http/1.1, some http/2, things would get dicey.
If some configs would come from X and some from Y, it will be difficult to address in time of crisis
With uniformity, it's easy to collect metrics from everything

TODO
Observe every detail of interaction between microservices
Uniformly configure your microservices
Communication protocol
Observability: logs. Metrics, tracing
configuration
Multiplexed connection



The framework employs various measures that improve reliability of running microservices:

* [Ack or fail fast](../blocks/unicast.md) detects `404` errors quickly
* [Time budget](../blocks/time-budget.md) is the correct pattern for handling client/server timeouts
* All pending operations are drained before a microservice shuts down in an attempt to avoid dropping requests
* Microservices leverage a single multiplexed TCP connection to achieve high throughput efficiently
