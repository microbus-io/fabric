# Package `examples`

The `examples` package holds several examples that demonstrate how the framework can be used to create microservices. When studying an example, start by looking at the `service.yaml` to get a quick overview of the functionality of the microservice. Then go deep into the code in `service.go`. All files with `-gen` in their name are code generated and can be ignored unless studying the internals of the generated code.

## Hello

The `hello.example` microservice demonstrates some of the key capabilities of the framework using various endpoints.

`/echo` echos the incoming HTTP request in text format. This is a useful tool for looking at the HTTP control headers `Microbus-` added by the framework.

http://localhost:8080/hello.example/echo produces:

```http
GET /echo HTTP/1.1
Host: hello.example
User-Agent: Mozilla/5.0
Accept-Encoding: gzip, deflate, br
Accept-Language: en-US,en;q=0.9
Connection: keep-alive
Microbus-Call-Depth: 1
Microbus-From-Host: http.ingress.sys
Microbus-From-Id: tg190vjj3j
Microbus-Msg-Id: UQnfaJf4
Microbus-Time-Budget: 19749
Sec-Ch-Ua: "Chromium";v="104", " Not A;Brand";v="99"
Sec-Ch-Ua-Mobile: ?0
Sec-Ch-Ua-Platform: "macOS"
Sec-Fetch-Dest: document
Sec-Fetch-Mode: navigate
Sec-Fetch-Site: none
Sec-Fetch-User: ?1
Upgrade-Insecure-Requests: 1
```

The `/ping` endpoint broadcasts a ping to discover the identity of all microservices running on the cluster.

```
bvtgii68r8.messaging.example
mv2pcoockl.messaging.example
r52l78kha4.messaging.example
pa6r5ohm5h.hello.example
0iij3m5fhf.http.ingress.sys
7k9f82n45f.configurator.sys
n89hmtb9iq.calculator.example
```

The `/hello` endpoint renders a simple greeting. It demonstrates the use of configs as well as taking in arguments from the URL. The single endpoint `/hello` takes in a query argument `name` and prints the greeting specified in the `Greeting` config, repeated as many times as indicate by the `Repeat` config. The values of these configs are set in `examples/main/env.yaml`.

http://localhost:8080/hello.example/hello?name=Bella prints:

```
Ciao, Bella!
Ciao, Bella!
Ciao, Bella!
```

The `/calculator` endpoint renders a rudimentary UI of a calculator. Behind the scenes, this endpoint calls the `calculator.example/arithmetic` endpoint to perform the calculation itself, demonstrating service-to-service calls. The `calculator.example` microservice is discussed next.

<img src="examples-1.png" width="315">

The `/bus.jpeg` endpoint serves an image from the embedded resources directory.

## Calculator

The `calculator.example` microservice implement two endpoints, `/arithmetic` and `/square` in order to demonstrate functional handlers. These types of handlers automatically parse incoming requests (typically JSON over HTTP) and make it appear like a function is called. Functional endpoints are best called using the client that's defined in the `calculatorapi` package. The `hello.example` discussed earlier is making use of this client.

The `/arithmetic` endpoint takes query arguments `x` and `y` of type integer, and one of four operators in the `op` argument: `+`, `-`, `/` and `*`. The response is a an echo of the input arguments and the result of the calculation. It is returned as JSON.

http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=-8 produces:

```
{"xEcho":5,"opEcho":"*","yEcho":-8,"result":-40}
```

The `/square` endpoint takes a single integer `x` and prints its square.

http://localhost:8080/calculator.example/square?x=55 produces:

```
{"xEcho":55,"result":3025}
```

If the arguments cannot be parsed, an error is returned.

http://localhost:8080/calculator.example/square?x=not-valid results in:

```
json: cannot unmarshal string into Go struct field .x of type int
```

The `/distance` endpoint demonstrates the use of a complex type `Point`. The type is defined in `service.yaml`:

```yaml
types:
  - name: Point
    description: Point is a 2D (X,Y) coordinate.
    define:
      x: float64
      y: float64
```

and code generated in the API package:

```go
/*
Point is a 2D coordinate (X, Y)
*/
type Point struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
}
```

Dot notation can be used in URL query arguments to denote values of nested fields.
http://localhost:8080/calculator.example/distance?p1.x=0&p1.y=0&p2.x=3&p2.y=4 produces the result:

```
{"d":5}
```

## Messaging

The `/home` endpoint of the `messaging.example` microservice demonstrates three messaging patterns: load-balanced unicast, multicast and direct addressing.

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

The first paragraph indicates the current instance ID of the microservice that is processing the `/home` request. Because there are 3 instances added to the app that are load-balanced, this ID is likely to change on each request.

```go
func main() {
	app := application.New(
		httpingress.NewService(),
		hello.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		calculator.NewService(),
	)
	app.Run()
}
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

The `messaging.example` microservice also demonstrates how multiple replicas of the same service can share a single cache by communicating over NATS.

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
