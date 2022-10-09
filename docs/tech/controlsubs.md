# Control Subscriptions

All microservices on the `Microbus` subscribe to handle control messages on the reserved port `:888` in addition to any subscriptions they create for their own use case.

Currently, only the `:888/ping` endpoint is implemented but more will be added in the future. The ping handler is subscribed to both the original host `www.example.com` and to the special host `all`:

To discover all instances of `www.example.com`:

```go
ch := s.Publish(r.Context(), pub.GET("https://www.example.com:888/ping"))
for r := range ch {
    res, err := r.Get()
    if err != nil {
        return errors.Trace(err)
    }
    fromID := frame.Of(res).FromID()
}
```

To discover all instances of all microservices:

```go
ch := s.Publish(r.Context(), pub.GET("https://all:888/ping"))
for r := range ch {
    res, err := r.Get()
    if err != nil {
        return errors.Trace(err)
    }
    fromHost := frame.Of(res).FromHost()
    fromID := frame.Of(res).FromID()
}
```

Note how `frame.Of(res)` is used to extract the control headers from the response. The body of the response is ignored.
