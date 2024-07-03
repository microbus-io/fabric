# Package `dlru`

The `dlru` package implements a [distributed LRU cache](../blocks/distrib-cache.md) that is shared among all peer replicas of a microservice. The cache uses pub/sub over NATS to communicate and synchronize.

By default, a DLRU is created and assigned for each microservice and made available using `svc.DistribCache()`.

```go
var obj MyObject
ok, err := svc.DistribCache().LoadJSON(ctx, cacheKey, &obj)
if err != nil {
    return errors.Trace(err)
}
if !ok {
    obj, err = svc.loadObjectFromDatabase(ctx, objKey)
    if err != nil {
        return errors.Trace(err)
    }
    err = svc.DistribCache().StoreJSON(ctx, cacheKey, obj)
    if err != nil {
        return errors.Trace(err)
    }
}
```

Additional DLRU caches can be created manually.
The constructor requires a microservice in order to be able to communicate with NATS, and the path of the NATS subscription to use for synchronization among peers. The cache's capacity and TTL can be configured as well.

```go
myCache, err := dlru.NewCache(ctx, svc, ":444/my-cache")
if err != nil {
    return errors.Trace(err)
}
myCache.SetMaxMemoryMB(5)
myCache.SetMaxAge(time.Hour)
```
