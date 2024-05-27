# Control Subscriptions

All microservices on the `Microbus` subscribe to handle control messages on the reserved port `:888` in addition to any subscriptions they create for their own use case.

## Ping

The `:888/ping` endpoint enables the dynamic discovery of microservices. The ping handler is subscribed to both the hostname of the service and to the special host `all`, allowing the discovery of replicas of a specific microservice, or of all microservices.

To discover all instances of `www.example.com` using the `Publish` method of the connector:

```go
ch := con.Publish(r.Context(), pub.GET("https://www.example.com:888/ping"))
for r := range ch {
    res, err := r.Get()
    if err != nil {
        return errors.Trace(err)
    }
    fromHost := frame.Of(res).FromHost()
    fromID := frame.Of(res).FromID()
}
```

Or use the `controlapi.Client` that abstracts the internals:

```go
ch := controlapi.NewMulticastClient(svc).ForHost("www.example.com").Ping(ctx)
for r := range ch {
    fromHost := frame.Of(r.HTTPResponse).FromHost()
    fromID := frame.Of(r.HTTPResponse).FromID()
}
```

To discover instances of all microservices replace the hostname `www.example.com` with the special hostname `all`:

```go
ch := con.Publish(r.Context(), pub.GET("https://all:888/ping"))
```

Or with `controlapi`:

```go
ch := controlapi.NewMulticastClient(svc).ForHost("all").Ping(ctx)
```

## Config Refresh

The `:888/config-refresh` endpoint indicates to the microservice to contact the configurator to refetch the values of its config properties.

## Trace

The `:888/trace` endpoint indicates to the microservice to export all tracing spans belonging to the requested trace ID (as indicated by the `id` argument) to the OLTP collector.
