# Package `trc`

The `trc` package supports distributed tracing with OpenTelemetry. It defines various `Option`s that can be applied to the tracing span using the options pattern. This pattern is used in Go for expressing optional arguments. 

For example:

```go
con.StartSpan(ctx, "Job", trc.String("name", "my job"))
```
