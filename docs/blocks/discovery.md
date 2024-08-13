# Dynamic Service Discovery

In traditional microservice architectures, microservices are implemented as web servers. Upstream clients who wish to make a request to a downstream microservice must know its location (IP address and port). This is further complicated when multiple replicas of the downstream microservice exist and load balancing is added to the mix. A typical solution is to have the upstream client contact a service discovery or DNS for the location (IP and port) of the downstream microservice, before making (pushing) its request. In this setup, the service discovery is a separate system from the transport layer (HTTP) and inconsistencies may occur if it is not in sync with the current state of any of the microservices.

In `Microbus`, microservices are not web servers. Instead, they are lightweight goroutines that pull (consume) messages off [carefully-crafted subscriptions](../blocks/unicast.md#notes-on-subscription-subjects) from a messaging bus. A microservice's subscriptions on the messaging bus are how it is discovered by clients (publishers). When a subscription is established, such as when the microservice starts up, the microservice becomes immediately discoverable. When the subscription is closed, such as when the microservice shuts down, the microservice is not longer discoverable. The messaging bus is in effect the service discovery. An external service discovery system is not required.

<img src="discovery-1.drawio.svg">
<p></p>
