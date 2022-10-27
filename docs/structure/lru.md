# Package `lru`

The `lru` package implements a thread-safe in-memory LRU cache that automatically evicts elements to control overflow. Elements are evicted if either they have reached a certain age or when the cache as a whole has reached its weight capacity.

The cache capacity and age limit can be changed using `SetMaxWeight` and `SetMaxAge` anytime during the lifespan of the cache, although it is typical to do so right after creation.

`Store(key K, value V, options ...StoreOption)` stores an element in the cache. The weight of the element can be indicated using the `Weight` option. If a weight is not specified it defaults to 1, which effectively limits the cache by the count of elements.

`Load(key K, options ...LoadOption) (value V, ok bool)` loads an element from the cache. If the element is found, it is moved to the head of the cache unless the `NoBump` option is specified. Bumping elements extends the life of frequently-used elements and can increase the cache's hit ratio. Not bumping elements forces them to be forgotten and refreshed periodically. This pattern is useful for automatically healing inconsistencies between the cache and the original source of the data.

`LoadOrStore(key K, newValue V, options ...LoadOrStoreOption) (value V, found bool)` combines both of the above atomically.

Additional operations such as `Delete`, `Clear`, `Exists`, `Weight` and `Len` are also available.

Typical usage:

```go
cache := lru.NewCache[int, string]()
cache.SetMaxAge(5*time.Minute)
cache.SetMaxWeight(1000)
cache.Store(1, "one", lru.Weight(len("one")))
cache.Store(2, "two", lru.Weight(len("two")))
cache.Store(3, "three", lru.Weight(len("three")))
v, ok := cache.Load(1)
```

It's important to note that in a microservice architecture each replica of a microservice runs in a separate process. If a microservice creates an LRU cache, each of its peers will have a separate copy that is not in sync with its own. Usage of this type of cache is therefore beneficial for immutable data or when inconsistencies can be tolerated for a certain time.
