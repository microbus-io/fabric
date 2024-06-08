# Package `errors`

The `errors` package is an enhancement of Go's standard `errors` package. It augments the standard `error` to capture and print stack traces. For this purpose, it overrides the standard `errors.New` method and adds a new `errors.Newf` (in lieu of `fmt.Errorf`).

```go
import "github.com/microbus-io/errors"

err := errors.New("my error") // err is augmented with the stack trace of this line
err = errors.Newf("error in process '%s'", processName) // err is augmented with the stack trace of this line
```

Note how it seamlessly replaces the standard `import "errors"` with `import "github.com/microbus-io/errors"`. That is made possible because `github.com/microbus-io/errors` redefines all the constructs in `errors`.

If a standard `error` was created by an unaware function, `errors.Trace` is used to augment it with the stack trace

```go
import "github.com/microbus-io/errors"

body, err := io.ReadAll("non/existent.file") // err is a standard Go error
err = errors.Trace(err) // err is now augmented with the stack trace of this line
```

HTTP status codes can be attached to errors by using `errors.Newc`, or by converting the error to the underlying `*TracedError` manually, or with `errors.Convert`. The status code is returned to upstream clients.

```go
notFound := errors.Newc(http.StatusNotFound, "nothing to see here")

body, err := io.ReadAll("non/existent.file") // err is a standard Go error
err = errors.Trace(err) // err is now augmented with the stack trace of this line
errors.Convert(err).StatusCode = http.StatusNotFound
// or
err.(*errors.TracedError).StatusCode = http.StatusNotFound
```

The `fmt` verb `%v` is equivalent to `err.Error()` and prints the error message.
The extended verb `%+v` is equivalent to `errors.Convert(err).String()` and also print the stack trace.

```
strconv.ParseInt: parsing "nan": invalid syntax
[400]

- calculator.(*Service).Square
  /src/github.com/microbus-io/fabric/examples/calculator/service.go:75
- connector.(*Connector).onRequest
  /src/github.com/microbus-io/fabric/connector/messaging.go:225
- connector.(*Connector).Publish
  /src/github.com/microbus-io/fabric/connector/messaging.go:94
- httpingress.(*Service).ServeHTTP
  /src/github.com/microbus-io/fabric/coreservices/httpingress/service.go:124
```
