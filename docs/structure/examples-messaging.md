# Package `examples/messaging`

## Messaging

The `/home` endpoint of the `messaging.example` microservice demonstrates three messaging patterns:
* Load-balanced unicast
* Multicast
* Direct addressing

The output of http://localhost:8080/messaging.example/home looks like this:

```
Processed by: 4l284tgjfk

Unicast
GET https://messaging.example/default-queue
> DefaultQueue 4l284tgjfk

Direct addressing unicast
GET https://4l284tgjfk.messaging.example/default-queue
> DefaultQueue 4l284tgjfk

Multicast
GET https://messaging.example/no-queue
> NoQueue 4kfei93btu
> NoQueue ba62j2gjjp
> NoQueue 4l284tgjfk

Direct addressing multicast
GET https://4l284tgjfk.messaging.example/no-queue
> NoQueue 4l284tgjfk

Refresh the page to try again
```

The first paragraph indicates the current instance ID of the microservice that is processing the `/home` request. Because `main/main.go` includes 3 instances of the `messaging.example`, this ID is likely to change on each request with load-balancing.

```go
app.Add(
	// ...
	messaging.NewService(),
	messaging.NewService(),
	messaging.NewService(),
	// ...
)
```

The second paragraph is showing the result of making a standard unicast request to the `/default-queue` endpoint. Only one of the 3 instances responds. A random instance is chosen by NATS, effectively load-balancing between the instances.

The third paragraph is showing the result of making a direct addressing unicast request. Even though there are 3 instances that may serve `/default-queue`, only the one specific instance responds.

The fourth paragraph is showing the result of making a multicast request to the `/no-queue` endpoint. All 3 of the instances respond. The order of arrival of the responses is random.

The fifth paragraph is showing the result of making a direct addressing multicast request. Even though there are 3 instances that serve `/no-queue`, only the one specific instance responds. This effectively transforms the request to a unicast.

Refresh the page to see the IDs change:

* The processor ID may change
* The responder to the unicast request may change
* The order of the responses to the multicast may change

## Distributed Cache

The `messaging.example` microservice also demonstrates how multiple replicas of the same microservice can share a single cache by communicating over NATS.

To store an element, use the http://localhost:8080/messaging.example/cache-store?key=foo&value=bar endpoint. The output shows which of the replicas handled the request.

```
key: foo
value: bar

Stored by qtshc434b7
```

To load an element, use the http://localhost:8080/messaging.example/cache-load?key=foo endpoint. Refresh the page a few times and notice how all replicas are able to locate the element.

```
key: foo
found: yes
value: bar

Loaded by ucarmsii56
```

Or if the element can't be located, e.g. http://localhost:8080/messaging.example/cache-load?key=fox :

```
key: fox
found: no

Loaded by pv7lqgeu98
```