# Package `examples/eventsource` and Package `examples/eventsink`

The `eventsource.example` and `eventsink.example` demonstrate how [events reverse the dependency between two microservices](../blocks/events.md). The event source microservice is unaware and independent of the event sink microservice, event though technically it is the initiator of a request to the event sink. Rather, it is the event sink that is aware of and dependent on the event source.

In this example, the `eventsource.example` mocks a simple user registration microservice that fires events to see if any filtering microservices wish to block the registration. The `eventsink.example` microservice acts as a filter provider for the `eventsource.example` microservice and disallows certain registrations. Other such filter providers may be added in the future without requiring changes to the `eventsource.example`.

Try the following URLs in order:

* http://localhost:8080/eventsource.example/register?email=peter@example.com : example.com domain is allowed.
* http://localhost:8080/eventsource.example/register?email=mary@example.com : example.com domain is allowed.
* http://localhost:8080/eventsource.example/register?email=paul@gmail.com : gmail.com domain is disallowed.
* http://localhost:8080/eventsource.example/registered
