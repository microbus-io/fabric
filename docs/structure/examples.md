# Package `examples`

The `examples` package holds several examples that demonstrate how the `Connector` can be used to create microservices.

## Echo

The `echo.example` microservice implements two endpoints, `/echo` and `/who`.

`/echo` echos the incoming HTTP request in text format. This is a useful tool for looking at the HTTP control headers `Microbus-` added by the framework.

https://localhost:8080/echo.example/echo produces:
```http
GET /echo HTTP/1.1
Host: echo.example
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.126 Safari/537.36
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9
Accept-Encoding: gzip, deflate, br
Accept-Language: en-US,en;q=0.9
Connection: keep-alive
Dnt: 1
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

The `/who` endpoint prints the ID of the particular instance of `echo.example` that processed the request, out of the two instances that are created in `examples/main/main.go`.

https://localhost:8080/echo.example/who results in:

```
Request from instance tg190vjj3j of host http.ingress.sys
Handled by instance qi3v3tjpbn of host echo.example

Refresh the page to try again
```

## Calculator

The `calculator.example` microservice implement two endpoints, `/arithmetic` and `/square` in order to demonstrate parsing of query arguments and error handling.

The `/arithmetic` endpoint takes query arguments `x` and `y` of type integer, and one of four operators in the `op` argument: `+`, `-`, `/` and `*`.

https://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=-8 produces:

```
5 * -8 = -40
```

The `/square` endpoint takes a single integer `x` and prints its square.

https://localhost:8080/calculator.example/quare?x=55 produces:

```
55 ^ 2 = 3025
```

If the arguments cannot be parsed, an error is returned.

https://localhost:8080/calculator.example/quare?x=not-valid results in:

```
strconv.ParseInt: parsing "not-valid": invalid syntax

calculator.(*Service).Square
	/Users/brianwillis/Dev/Microbus/src/github.com/microbus-io/fabric/examples/calculator/service.go:75
connector.(*Connector).onRequest
	/Users/brianwillis/Dev/Microbus/src/github.com/microbus-io/fabric/connector/messaging.go:225
	calculator.example:443/square
connector.(*Connector).Publish
	/Users/brianwillis/Dev/Microbus/src/github.com/microbus-io/fabric/connector/messaging.go:94
	http.ingress.sys -> calculator.example
httpingress.(*Service).ServeHTTP
	/Users/brianwillis/Dev/Microbus/src/github.com/microbus-io/fabric/services/httpingress/service.go:124
```

## Hello World

The `helloworld.example` microservice prints a simple greeting. It demonstrates the use of configs as well as taking in arguments from the URL. The single endpoint `/hello` takes in a query argument `name` and prints the greeting specified in the `Greeting` config, repeated as many times as indicate by the `Repeat` config. The values of these configs are set in `examples/main/env.yaml`.

https://localhost:8080/helloworld.example/hello?name=John prints:

```
Ciao, John!
Ciao, John!
Ciao, John!
```
