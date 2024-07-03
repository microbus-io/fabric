# Multicast Messaging

A communication pattern that is often underappreciated is publish/subscribe which enables a publisher to send a message while remaining unaware of who the subscribers are. A microservices system is ever evolving and subscribers can be added over time by distributed teams. By leveraging publish/subscribe, the client does not need to be changed when new subscribers are added. These use cases are well-suited for publish/subscribe:

* Discovery - The client is trying to detect other microservices that provide a known function. For example, an authenticator service might look for authentication providers
* On change events - The client is informing dependent microservices that an object it owns has changed so they can take the appropriate action. For example, if a user is deleted, all microservices that store information about a user need to be informed so they also delete the corresponding data they own.

`Microbus` takes publish/subscribe to the next level. Whereas traditional publish/subscribe is unidirectional, multicasting via the `Microbus` enables subscribers to also respond as if they had received a standard HTTP request. It is up to the client whether to iterate over the responses, or ignore them and just fire and forget.

Typical client code that processes multiple responses will look similar to the following:

```go
ch := svc.Publish(ctx, pub.GET("https://authprovider/discover"))
for r := range ch {
    res, err := r.Get()
    if err != nil {
        return errors.Trace(err)
    }
    // do something with res
}
```

In the case of fire and forget, it may look similar to this:

```go
svc.Go(ctx, func(ctx context.Context) (err error) {
    svc.Publish(ctx, pub.GET("https://users.storage/ondeleted?id=12345"))
    return nil
})
```

Note that a multicast may return no results at all. Unlike a unicast, a `404` error is not returned in this case.