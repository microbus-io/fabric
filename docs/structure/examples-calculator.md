# Package `examples/calculator`

The `calculator.example` microservice implement two endpoints, `/arithmetic` and `/square` in order to demonstrate functional handlers. These types of handlers automatically parse incoming requests (typically JSON over HTTP) and make it appear like a function is called. Functional endpoints are best called using the client that's defined in the `calculatorapi` package. The `hello.example` discussed earlier is making use of this client.

The `/arithmetic` endpoint takes query arguments `x` and `y` of type integer, and one of four operators in the `op` argument: `+`, `-`, `/` and `*`. The response is a an echo of the input arguments and the result of the calculation. It is returned as JSON.

http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=-8 produces:

```json
{"xEcho":5,"opEcho":"*","yEcho":-8,"result":-40}
```

The `/square` endpoint takes a single integer `x` and prints its square.

http://localhost:8080/calculator.example/square?x=55 produces:

```json
{"xEcho":55,"result":3025}
```

If the arguments cannot be parsed, an error is returned.

http://localhost:8080/calculator.example/square?x=not-valid results in:

```
json: cannot unmarshal string into Go struct field .x of type int
```

The `/distance` endpoint demonstrates the use of a complex type `Point`. When an unknown type is used in any of the signatures in `service.yaml`, the code generator automatically defines an empty struct in a file of the same name (in this case `point.go`) in the API package:

```go
// Point
type Point struct {
}
```

It can then be filled appropriately for its use case:

```go
// Point is a 2D coordinate (X, Y)
type Point struct {
	X float64 `json:"x,omitempty"`
	Y float64 `json:"y,omitempty"`
}
```

Dot notation can be used in URL query arguments to denote values of nested fields.
http://localhost:8080/calculator.example/distance?p1.x=0&p1.y=0&p2.x=3&p2.y=4 produces the result:

```json
{"d":5}
```