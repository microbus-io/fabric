# Load Balancing

Microservices in `Microbus` are lightweight goroutines that pull (consume) messages off a messaging bus, in which a queue is maintained for each endpoint of each microservice. Replicas of the same microservice subscribe to the same queues and consume messages as they are published. Load balancing is achieved by virtue of the bus dispatching messages from the queue to only one random consumer at a time. A separate load balancer is therefore not required.

<img src="lb-1.drawio.svg">
<p></p>

`Microbus` also allows for [multicast](../blocks/multicast.md) subscriptions in which all replicas receive all messages.

<img src="lb-2.drawio.svg">
<p></p>

When using the `Connector`'s `Subscribe` method directly, multicasting can be enabled via the `sub.NoQueue()` option.

When using [`service.yaml`](../tech/service-yaml.md) and the [code generator](../blocks/codegen.md), the `queue: none` property is used to indicate a multicast subscription.

[Events](../blocks/events.md) are by definition multicast.
