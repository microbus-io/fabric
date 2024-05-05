# Package `services/control`

The `control.sys` microservice provides no function and in fact will not start. It is the code generated clients in `controlapi` that are the essence of this package. These clients, and in particular the `controlapi.MulticastClient`, provide a programmatic interface to the [control subscriptions](../tech/controlsubs.md) that all microservices support.

For example, to ping and discover all microservices:

```go
ch := controlapi.NewMulticastClient(svc).ForHost("all").Ping(ctx)
for r := range ch {
    fromHost := frame.Of(r.HTTPResponse).FromHost()
    fromID := frame.Of(r.HTTPResponse).FromID()
}
```

Overriding the host of the client via `ForHost` is required because the default host `control.sys` does not exist. In the example above, the special host name `all` is used to address all microservices.
