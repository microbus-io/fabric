# Package `examples/hello`

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
Microbus-From-Host: http.ingress.core
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
0iij3m5fhf.http.ingress.core
7k9f82n45f.configurator.core
n89hmtb9iq.calculator.example
```

The `/hello` endpoint renders a simple greeting. It demonstrates the use of configs as well as taking in arguments from the URL. The single endpoint `/hello` takes in a query argument `name` and prints the greeting specified in the `Greeting` config, repeated as many times as indicate by the `Repeat` config. The values of these configs are set in `main/config.yaml`.

http://localhost:8080/hello.example/hello?name=Bella prints:

```
Ciao, Bella!
Ciao, Bella!
Ciao, Bella!
```

The `/calculator` endpoint renders a rudimentary UI of a calculator. Behind the scenes, this endpoint calls the `calculator.example/arithmetic` endpoint to perform the calculation itself, demonstrating service-to-service calls. The `calculator.example` microservice is discussed next.

<img src="examples-hello-1.png" width="315">

The `/bus.png` endpoint serves an image from the embedded resources directory.

The `/localized` endpoint demonstrates loading a localized string from the `strings.yaml` resource based on the request's `Accept-Language` header.

http://localhost:8080/hello.example/localized prints `Hello` in one of several European languages:

```
en: Hello
fr: Bonjour
es: Hola
it: Salve
de: Guten Tag
pt: Olá
da: Goddag
nl: Goedendag
pl: Dzień dobry
no: God dag
tr: Merhaba
sv: God dag
```

The `//root` endpoint demonstrates how the magic hostname `root` is used to create a subscription to the topmost root path of the web server.

http://localhost:8080/
