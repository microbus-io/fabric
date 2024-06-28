# Package `frame`

The `frame` package is responsible for the manipulation of the HTTP control headers that the `Microbus` framework is adding to the messages that travel over NATS. It defines constants for the various `Microbus-` header names and type-safe getters and setters.

A `frame.Frame` is first created `Of` an `*http.Request`, `*http.Response`, `http.ResponseWriter`, `http.Header` or `context.Context`.

For example:

```go
func Foo(w http.ResponseWriter, r *http.Request) error {
	callerHost := frame.Of(r).FromHost() // equivalent to r.Header.Get(frame.HeaderFromHost)
	callerID := frame.Of(r).FromID()     // equivalent to r.Header.Get(frame.HeaderFromId)
}
```
