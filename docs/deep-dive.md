# Deep Dive

## Messaging

One of the challenges with messaging buses is that they have an unfamiliar pattern that doesn't map well to modern web standards. When it comes to microservices, most developers are accustomed to thinking in terms of HTTP.
`Microbus` overcomes this gap by implementing the familiar request/response pattern of HTTP over the asynchronous messaging pattern of NATS.

For starters, while NATS supports a purely arbitrary binary message format, `Microbus`'s messages adhere to the HTTP/1.1 request and response message formats. This is done for several reasons:
* The HTTP format includes a meta-data section in the form of headers in addition to an unrestricted binary body. The headers are ideal for sending the control information necessary to make the `Microbus` magic happen
* The HTTP format is familiar to developers
* There are plenty of tools and libraries to work with the HTTP format
* Conversion to and from "real" HTTP is trivial. The section about the ingress proxy will touch on that

Request/response is achieved by utilizing carefully crafted subjects (topics) as means of delivering messages to their destination. Each endpoing of a microservice is assigned a dedicated subject based on the path it wants to handle. For example, the URL `https://www.example.com:443/path/func` is mapped to the NATS subject `443.com.example.www.|.path.func`. With that, when a microservice wants to handle calls to any given endpoint (identified by a URL path) it will subscribe to the appropriate NATS subject and when a microservice wants to make a call to another microservice's endpoint (path), all it has to do is publish a message to the same subject. The order of the segments in the subject is by intent. NATS provides means of controlling access to subjects using an ACL and this format will enable setting permissions such as `ALLOW *.com.example.>`.

In addition to any subscriptions that a microservice makes to handle incoming calls, it also creates a subscription at which it expects to receive replies. The format of this subject is `r.com.example.zzz.up7cjo7pok`. The `r` prefix can be thought of as designating the reply port. The `up7cjo7pok` is the unique instance ID of the microservice. So now, when a serving micoservice wants to respond to a call from a client microservice, all it has to do is publish the response to the reply subject of the client.

It is necessary that the client provide a return address in order for the server to know where to respond to. To facilitate that, each request made over the bus must include two special HTTP headers, `Microbus-From-Host` and `Microbus-From-Id`, that enable the server to construct the reply subject.

Another important header included by the client in each request is `Microbus-Msg-Id`. The server is required to echo back this unique ID in the response so that the client can map it to the corresponding request. Remember that the client can be making thousands of requests in parallel whose responses can return in no particular order. In the code, a `chan *http.Response` is created for each outgoing request and indexed by the message ID in a `map[string]chan *http.Response`. Requests awaits on the channel until a response comes back or a timeout occurs.

To look at an example that puts this all together, start NATS in debug mode in another window using `./nats-server -D -V` and then run the `TestEcho` unit test located in `connector/messaging.go`. Redacted for brevity, the output looks like this:

NATS server starting

```
[INF] Starting nats-server version 1.4.1
[DBG] Go build version go1.17.3
[INF] Git commit [not set]
[INF] Listening for client connections on 0.0.0.0:4222
[DBG] Server id is 5eBkFRs9kHYJlrAib3Nrlp
[INF] Server is ready
```

The microservices `alpha.echo.connector` starts up and subscribes to the reply subject `r.connector.echo.alpha.s6gerdf3o5`.

```
[DBG] cid:1 - Client connection created
[TRC] cid:1 - ->> [CONNECT {"verbose":false,"pedantic":false,"tls_required":false,"name":"","lang":"go","version":"1.16.0","protocol":1,"echo":true,"headers":false,"no_responders":false}]
[TRC] cid:1 - ->> [PING]
[TRC] cid:1 - <<- [PONG]
[TRC] cid:1 - ->> [SUB r.connector.echo.alpha.s6gerdf3o5 s6gerdf3o5 1]
```

The microservices `beta.echo.connector` starts up and subscribes to the reply subject `r.connector.echo.beta.o6cfdjocuq` and to the endpoint subject `443.connector.echo.beta.|.echo`. If you look closely at `[SUB 443.connector.echo.beta.|.echo beta.echo.connector 2]` you'll note that the host name `beta.echo.connector` is set as the queue name of the subscription. In NATS, messages delivered on a queue are delivered to a random consumer rather than to all consumers. Queues allows us to achieve load-balancing between multiple instances of the same microservice.

```
[DBG] cid:2 - Client connection created
[TRC] cid:2 - ->> [CONNECT {"verbose":false,"pedantic":false,"tls_required":false,"name":"","lang":"go","version":"1.16.0","protocol":1,"echo":true,"headers":false,"no_responders":false}]
[TRC] cid:2 - ->> [PING]
[TRC] cid:2 - <<- [PONG]
[TRC] cid:2 - ->> [SUB r.connector.echo.beta.o6cfdjocuq o6cfdjocuq 1]
[TRC] cid:2 - ->> [SUB 443.connector.echo.beta.|.echo beta.echo.connector 2]
```

The microservices `alpha.echo.connector` makes a request to `https://beta.echo.connector/echo`, including a unique message ID in `Microbus-Msg-Id` and its identity in `Microbus-From-Host` and `Microbus-From-Id`. The binary format of the message is that of the standard HTTP/1.1 request.

```
[TRC] cid:1 - ->> [PUB 443.connector.echo.beta.|.echo 205]
[TRC] cid:1 - ->> MSG_PAYLOAD: [POST /echo HTTP/1.1
Host: beta.echo.connector
User-Agent: Go-http-client/1.1
Content-Length: 5
Microbus-From-Host: alpha.echo.connector
Microbus-From-Id: s6gerdf3o5
Microbus-Msg-Id: iu2fogwS

Hello]
[TRC] cid:2 - <<- [MSG 443.connector.echo.beta.|.echo 2 205]
```

The microservices `beta.echo.connector` responds to the request by publishing a message to the reply channel of `alpha.echo.connector`, making sure to echo back the message ID in `Microbus-Msg-Id` and to include its identity in `Microbus-From-Host` and `Microbus-From-Id`. The binary format of the message is that of the standard HTTP/1.1 response.

```
[TRC] cid:2 - ->> [PUB r.connector.echo.alpha.s6gerdf3o5 182]
[TRC] cid:2 - ->> MSG_PAYLOAD: [HTTP/1.1 200 OK
Connection: close
Content-Type: text/plain; charset=utf-8
Microbus-From-Host: beta.echo.connector
Microbus-From-Id: o6cfdjocuq
Microbus-Msg-Id: iu2fogwS

Hello]
[TRC] cid:1 - <<- [MSG r.connector.echo.alpha.s6gerdf3o5 1 182]
```

The microservices shutdown and disconnect from NATS.

```
[TRC] cid:2 - ->> [UNSUB 2 ]
[TRC] cid:2 - <-> [DELSUB 2]
[TRC] cid:2 - ->> [UNSUB 1 ]
[TRC] cid:1 - ->> [UNSUB 1 ]
[TRC] cid:2 - <-> [DELSUB 1]
[TRC] cid:1 - <-> [DELSUB 1]
[DBG] cid:2 - Client connection closed
[DBG] cid:1 - Client connection closed
```

## HTTP Ingress Proxy

Think of NATS as a closed garden that requires a special key to access. In order to send and receive messages over NATS, it's necessary to use the NATS libraries to connect to NATS. This is basically what the `Connector` is facilitating for service-to-service calls.

Practically all systems will require interaction from a source that is outside the NATS bus. The most common scenario is perhaps a request generated from a web browser to a public API endpoint. In this case, something needs to bridge the gap between the incoming real HTTP request and the HTTP messages that travel over the bus. This is exactly the role of the HTTP ingress proxy.


```
+---------+  Real HTTP  +---------+  HTTP/NATS  +---------+
| Browser | ----------> | Ingress | ----------> | Micro-  |
|         | <---------- |  Proxy  | <---------- | service |
+---------+             +---------+             +---------+
```

On one end, the HTTP ingress proxy listens on port `:8080` for real HTTP requests; on the other end it is conected to NATS. The ingress proxy converts real requests into requests on the bus; and vice versa, it converts responses from the bus to real responses. Because the bus messages in `Microbus` are formatted themselves as HTTP messages, this conversion is trivial, with two caveats:
* The proxy filters out `Microbus-*` control headers from coming in or leaking out
* The first segment of the path of the real HTTP request is treated as the host name of the microservice on the bus. So for example, `http://localhost:8080/echo.example/echo` is translated to the bus address `https://echo.example/echo` which is then mapped to the NATS subject `443.example.echo.|.echo`.

## Encapsulaton

The `Microbus` framework aims to provide a consistent experience to developers, the users of the framework. It is opinionated about the interfaces (APIs) that are exposed to the developer and therefore opts to wrap underlying technologies behind its own interfaces.

One examples of this approach in this milestone is the config. Rather than leave things up to each individual developer how to fetch config values, the framework defines an interface that abstracts the underlying implementation. A similar approach was taken with the logger.

In addition to the consistent developer experience, there are various technical reasons behind this philosophy:
* Enforcing uniformity across all microservices brings familiarity when looking at someone else's code, lowers the learning curve, and ultimately increases velocity
* The underlying technology can be changed with little impact to the microservices. For example, the source of the configs can be extended to include a remote config service in addition to the environment or the file system
* Oftentimes the underlying technology is more extensive than the basic functionality that is needed by the framework. Wrapping the underlying API enables exposing only certain functions to the developer. For example, the logger for now is limited to only `LogInfo` and `LogError`
* The framework is in control of when and how the underlying technology is initialized. For example, future milestones will customize the logger based on the runtime environment (PROD, LAB, LOCAL, etc)
* The framework is able to seamlessly integrate building blocks together. This will take shape as more building blocks are introduced. A simple example in this milestone is how the detected config keys are logged during startup
* Bugs or CVEs in the underlying technologies are quicker to fix because there is only one source of truth. A bug such as Log4Shell (CVE-2021-44228) would require no code changes to the microservices, only to the framework

## Shortcuts

This milestone is taking several shortcuts that will be addressed in future releases:

* The timeouts for the `OnStartup` and `OnShutdown` callbacks and for outgoing requests are hard-coded to `time.Minute`
* The NATS server URL is hard-coded to localhost `nats://127.0.0.1:4222`
* The logger is quite basic
* The HTTP ingress proxy is hard-coded to port `:8080`

## More to Explore

A few suggestions for self-guided exploration:

* Start NATS in debug mode `./nats-server -D -V`, run unit tests individually and look at the messages going over the bus
* Add an endpoint `/calculate` to the `calculator.example` microservice that operates on decimal numbers, not just integers
* Create your own microservice from scratch and add it to `examples/main/main.go`
* Modify `examples/main/env.yaml` and witness the impact on the `helloworld.example` microservice
