# Client Stubs

The `GET`, `POST`, `Request` and `Publish` methods of the `Connector` allow an upstream microservice to make requests over the messaging bus. These methods provide HTTP-like semantics: the caller needs to explicitly designate the URL, headers and payload in HTTP form. This approach is broadly applicable but suffers from lack of forward compatibility and type safety. If the downstream microservice changes the signature of its endpoints, the upstream's requests may start failing. It is also somewhat inconvenience to work at the HTTP level, especially when dealing with marshaling.

To address these challenges, the [code generator](../blocks/codegen.md) creates a client stub for each of the endpoints of the downstream microservice. A stub is a type-safe function that wraps the call to the `Connector`'s low-level publishing methods. It is customized for each endpoint based on its type and signature. In the code, the stubs almost appear to be regular function calls.

Client stubs are defined in a sub-package of the microservice named after the package of the service with an `api` suffix added. For example, the clients of the `calculator` microservice are defined in `calculator/calculatorapi`. This naming convention is intended to facilitate with type-ahead code completion.

The standard `Client` is used to make [unicast](../blocks/unicast.md) requests and is the more commonly used.

```go
sum, err := calculatorapi.NewClient(svc).Add(ctx, x, y)
```

The aptly-named `MulticastClient` is used to make [multicast](../blocks/multicast.md) requests.

```go
for ri := range providerapi.NewMulticastClient(svc).Discover(ctx) {
    name, err := ri.Get()
    // ...
}
```

The `MulticastTrigger` is to be used by the microservice itself to fire its own events.

```go
for ri := range userstoreapi.NewMulticastTrigger(svc).OnCanDelete(ctx, id) {
    allowed, err := ri.Get()
    // ...
}
```

The `Hook` facilitate registration of event sinks by downstream microservices. The code generator utilizes this `Hook` automatically when an event sink is defined in `service.yaml`.

```go
userstoreapi.NewHook(svc).OnCanDelete(svc.OnCanDelete)
```

The API package also includes all objects uses by the endpoints that are owned by the downstream microservice. For example, a user store microservice will likely define a `type User struct`. When a microservice refers to an object owned by another microservice, an alias to it is defined. For example, if the registration microservice accepts a `User` object in any of its endpoints, it defines `type User = userstoreapi.User` to alias the original definition.
