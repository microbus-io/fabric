# Unicast Messaging

## Overview

One of the challenges with messaging buses is that they have an unfamiliar pattern that doesn't map well to modern web standards. When it comes to microservices, most developers are accustomed to thinking in terms of synchronous HTTP, not asynchronous messaging over a bus.
`Microbus` overcomes this gap by emulating the familiar synchronous request/response pattern of HTTP over the asynchronous messaging pattern of NATS.

For starters, while NATS supports a purely arbitrary binary message format, `Microbus`'s messages adhere to the HTTP/1.1 request and response message formats. This is done for several reasons:

* The HTTP format includes a meta-data section in the form of headers in addition to an unrestricted binary body. The headers are ideal for sending the control information necessary to make the `Microbus` magic happen
* The HTTP format is familiar to developers
* There are plenty of tools and libraries to work with the HTTP format
* Conversion to and from "real" HTTP by the [ingress proxy service](httpingress.md) is trivial

## Emulating Request/Response

<img src="unicast-1.drawio.svg" width="561">

Request/response is achieved by utilizing carefully crafted subjects (topics) as means of delivering messages to their destination. Each endpoint of a microservice is assigned a dedicated subject based on the method and path it handles. For example, handling any method at `https://server.host:443/path/func` is mapped to the NATS subject `microbus.443.host.server.|.*.path.func`. With that, when a microservice wants to handle calls to any given endpoint (identified by a URL and optionally a method) it will subscribe to the appropriate NATS subject. And when a microservice wants to make a call to another microservice's endpoint (method and URL), all it has to do is publish a message to the appropriate subject.

In addition to any subscriptions that a microservice makes to handle incoming calls, it also creates a subscription at which it expects to receive responses. The format of this subject is `microbus.r.host.client.up7cjo7pok`. The `r` prefix can be thought of as designating the `r`esponse port. The `up7cjo7pok` is the unique instance ID of the microservice. When a serving microservice wants to respond to a call from a client microservice, it publishes the response to the response subject of the client.

It is necessary for the client to provide its return address in order for the server to know who to respond to. Each request made over the bus therefore must include two special HTTP headers, `Microbus-From-Host` and `Microbus-From-Id`, that together enable the server to construct the response subject.

Another important header included by the client in each request is `Microbus-Msg-Id` which the server is required to echo back in the response. The client can be making thousands of requests in parallel whose responses can return in no particular order and the message ID in needed to map each response to the corresponding request. In the code, a `chan *http.Response` is created for each outgoing request and indexed by the message ID in a `map[string]chan *http.Response`. Requests await on the channel until a response comes back or a timeout occurs.

## Example Walk-Through

To look at an example that puts this all together, start NATS in debug mode in another window using `./nats-server -D -V` (debug and verbose) and then run the `TestConnector_Echo` unit test located in `connector/publish_test.go`. The output below was edited for brevity.

The `-D -V` flags of NATS slow it down considerably and are therefore disabled in `microbus.yaml`. To run with these flags enabled, it's recommended to start NATS in a separate terminal window instead of inside Docker, and only for the duration that these flags are needed.

The microservices `alpha.echo.connector` starts up and subscribes to the response subject `microbus.r.connector.echo.alpha.dvm0oofeb5`.

```
[DBG] cid:1 - Client connection created
[TRC] cid:1 - ->> [CONNECT {"verbose":false,"pedantic":false,"tls_required":false,"name":"dvm0oofeb5.alpha.echo.connector","lang":"go","version":"1.16.0","protocol":1,"echo":true,"headers":false,"no_responders":false}]
[TRC] cid:1 - ->> [PING]
[TRC] cid:1 - <<- [PONG]
[TRC] cid:1 - ->> [SUB microbus.r.connector.echo.alpha.dvm0oofeb5 dvm0oofeb5 1]
```

The microservices `beta.echo.connector` starts up and subscribes to the response subject `microbus.r.connector.echo.beta.rouq0u0mf4` and to the endpoint subject `microbus.443.connector.echo.beta.|.*.echo`. If you look closely at `[SUB microbus.443.connector.echo.beta.|.*.echo beta.echo.connector 6]` you'll note that the hostname `beta.echo.connector` is set as the queue name of the subscription (the second argument). In NATS, messages delivered on a queue are delivered to a random consumer rather than to all consumers. Queues allows us to achieve load-balancing between multiple instances of the same microservice.

```
[DBG] cid:2 - Client connection created
[TRC] cid:2 - ->> [CONNECT {"verbose":false,"pedantic":false,"tls_required":false,"name":"rouq0u0mf4.beta.echo.connector","lang":"go","version":"1.16.0","protocol":1,"echo":true,"headers":false,"no_responders":false}]
[TRC] cid:2 - ->> [PING]
[TRC] cid:2 - <<- [PONG]
[TRC] cid:2 - ->> [SUB microbus.r.connector.echo.beta.rouq0u0mf4 rouq0u0mf4 1]
[TRC] cid:2 - ->> [SUB microbus.443.connector.echo.beta.|.*.echo beta.echo.connector 6]
[TRC] cid:2 - ->> [SUB microbus.443.connector.echo.beta.rouq0u0mf4.|.*.echo beta.echo.connector 7]
```

The microservices `alpha.echo.connector` makes a `POST` request to `https://beta.echo.connector/echo`, including a unique message ID in `Microbus-Msg-Id` and its identity in `Microbus-From-Host` and `Microbus-From-Id`. The binary format of the message is that of the standard HTTP/1.1 request.

```
[TRC] cid:1 - ->> [PUB microbus.443.connector.echo.beta.|.POST.echo 281]
[TRC] cid:1 - ->> MSG_PAYLOAD: [POST /echo HTTP/1.1
Host: beta.echo.connector
Content-Length: 5
Microbus-Call-Depth: 1
Microbus-From-Host: alpha.echo.connector
Microbus-From-Id: dvm0oofeb5
Microbus-Msg-Id: 3t0gkasY
Microbus-Op-Code: Req
Microbus-Time-Budget: 19999

Hello]
[TRC] cid:2 - <<- [MSG microbus.443.connector.echo.beta.|.POST.echo 6 281]
```

Before handling the request, microservice `beta.echo.connector` responds to it by immediately publishing an ack message to the response channel of `alpha.echo.connector`, making sure to echo back the message ID in `Microbus-Msg-Id` and to include its identity in `Microbus-From-Host` and `Microbus-From-Id`. The binary format of the message is that of the standard HTTP/1.1 response.

The ack message is used to inform the client that the request has been received and that it should expect a response. If an ack message is not received quickly, the client times out. Acks enable clients to quickly differentiate between the situations of not having a server that can respond, or having a server that responds slowly. This concept is unique to `Microbus`. In essence, the server responds with two responses to each request.

```
[TRC] cid:2 - ->> MSG_PAYLOAD: [HTTP/1.1 202 Accepted
Connection: close
Microbus-Op-Code: Ack
Microbus-From-Host: beta.echo.connector
Microbus-From-Id: rouq0u0mf4
Microbus-Msg-Id: 3t0gkasY
Microbus-Queue: beta.echo.connector

]
[TRC] cid:1 - <<- [MSG microbus.r.connector.echo.alpha.dvm0oofeb5 1 202]
```

After handling the request, microservice `beta.echo.connector` responds to it by publishing a message to the same response channel of `alpha.echo.connector`, again echoing back the message ID in `Microbus-Msg-Id` and including its own address in `Microbus-From-Host` and `Microbus-From-Id`. The binary format of the message is that of the standard HTTP/1.1 response.

```
[TRC] cid:2 - ->> [PUB microbus.r.connector.echo.alpha.dvm0oofeb5 242]
[TRC] cid:2 - ->> MSG_PAYLOAD: [HTTP/1.1 200 OK
Connection: close
Content-Type: text/plain; charset=utf-8
Microbus-From-Host: beta.echo.connector
Microbus-From-Id: rouq0u0mf4
Microbus-Msg-Id: 3t0gkasY
Microbus-Op-Code: Res
Microbus-Queue: beta.echo.connector

Hello]
[TRC] cid:1 - <<- [MSG microbus.r.connector.echo.alpha.dvm0oofeb5 1 242]
```

## Notes on Subscription Subjects

The pattern of the subscription subject of an endpoint is `<plane>.<port>.<reversed hostname>.|.<method>.<path>`.
A second subscription `<plane>.<port>.<reversed hostname>.<id>.|.<method>.<path>` is created to allow targeting the individual microservice by its ID.
The pattern of the response subscription is `<plane>.r.<reversed hostname>.<id>`

[NATS provides means of controlling access to subjects using ACLs](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization). Reversing the order of the segments of the hostname Yoda-style enables setting permissions such as `subscribe = "*.*.com.example.>"` which restricts a microservice to communicate only under the `example.com` domain. The two asterisks stand for any plane and any port.

The `microbus` prefix seen in the subscription subjects is referred to as the plane of communication. Microservices on a given plane can only talk to other services on the same plane. Planes therefore provide isolation for groups of microservices that share a single NATS cluster with other groups of unrelated microservices. For example, testing apps use a randomly generated plane to prevent unit tests from conflicting when running in parallel with other unit tests.
