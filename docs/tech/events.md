# Events

Events are a powerful pattern that is often neglected in microservice systems because it is not trivial to implement using HTTP and because the workaround of making direct calls is considered acceptable. Without events however, microservices end up cyclicly depending on one other, resulting in what soon becomes a spaghetti topology. An event-driven architecture done right uses events to communicate and serves to keep microservices decoupled.

To demonstrate, let's look at a common example: a user management microservices. When a user is deleted, many other resources that are tied to the user also need to be deleted. Without the benefit of events, the `DeleteUser` handler needs to make direct requests to all related service and might look similar to this (error checks omitted for brevity):

```go
func (svc *Service) DeleteUser(userID string) (err error) {
    svc.db.Execute("DELETE FROM USERS WHERE ID=?", userID)
    filestoreapi.NewClient(svc).DeleteForUser(userID)
    creditcardapi.NewClient(svc).DeleteForUser(userID)
    groupmanagerapi.NewClient(svc).DeleteForUser(userID)
    // etc.
}
```

What's more, this list may keep growing when new microservices are added in the future. Releasing a new microservice that keeps resources tied to a user now also requires releasing a new version of the user management microservice. In very large systems, with multiple teams, this may result in code conflicts, increased release complexity, or implementation delays.

In addition, the user management microservice has become dependent on a large number of microservices which are almost certainly depending back on it. The microservices dependency graph is no longer a DAG making it is challenging to reason about and test the system.

<img src="events-1.svg" width="300">

Alternatively, events take advantage of the pub/sub pattern and allow the user management microservice to publish an event without knowing who will be there to respond. The code will look similar to the following:

```go
func (svc *Service) DeleteUser(userID string) (err error) {
    svc.db.Execute("DELETE FROM USERS WHERE ID=?", userID)
    for range usermanagerapi.NewMulticastTrigger(svc).OnUserDeleted(userID) {
    }
}
```

Other microservices are able to dynamically subscribe to handle the `OnUserDeleted` event, which means that as new microservices are deployed, no change is required of the user management microservice. With this approach, the consumers (aka event sinks) depend on the producer (aka event source) and no cycles are introduced to the microservices dependency graph.

<img src="events-2.svg" width="300">

In `Microbus`, events are implemented as carefully crafted requests and subscriptions. Event sources publish a multicast request to a URL on their own host name. Event sinks subscribe to handle requests on the host name of the source rather than their own. Since they are fundamentally not any different than regular requests, events can also return values back to the source. The [events example](../structure/examples.md) uses this technique to ask for permission to perform an action. 

The [code generator](./codegen.md) makes it simple to produce and consume events using the `events` and `sinks` sections, respectively.

By default, events use port `:417` ("force eventing") to differentiate them from standard requests which default to port `:443`. This allows setting up port-based [NATS ACLs](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization) in low-trust environments where authorization of microservices is important. This way the event source can be made the only one allowed to publish to `eventsource.example:417`.

```
EVENTSOURCE_EXAMPLE = {
    publish = ["*.417.example.eventsource.>", "*.443.>"]
    subscribe = ["*.*.example.eventsource.>", "*.417.>"]
}

EVENTSINK_EXAMPLE = {
    publish = ["*.417.example.eventsink.>", "*.443.>"]
    subscribe = ["*.*.example.eventsink.>", "*.417.>"]
}
```
