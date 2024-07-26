# Distributed Cache

Caching is a powerful and common technique that reduces load on downstream databases as well as latency. In a microservices environment, where there are many replicas of the same microservices, it is often needed to share the cache among replicas. One replica might store an item in the cache, while another replica might load it. An invalidation of a cached element by one replica needs to be visible to all others.

## Localized Cache

In `Microbus`, each microservice holds in-memory an [LRU cache](../structure/lru.md) that is shared with all peer replicas of the microservice, but not with other microservices. Each replica's local LRU cache is a segment of the entire cache. The cache uses pub/sub to communicate and synchronize with peers.

<img src="./distrib-cache-1.drawio.svg">
<p></p>

The capacity of the cache scales horizontally with the number of replicas of the microservice.

<img src="./distrib-cache-2.drawio.svg">
<p></p>

The cache is scoped to a single microservice, therefore isolating it from side-effects that can be caused by "noisy neighbor" microservices. Isolation also makes it possible to independently scale to the individual needs of each microservice.

<img src="./distrib-cache-3.drawio.svg">
<p></p>

Operations are synchronized over the network and the cache is not immune to race conditions. To help improve consistency, the `Load` operations checks with peers to ensure there are no multiple versions of the same element. This is still not a 100% guarantee of consistency (e.g. during a network partition) but rather a mechanism to recover from inconsistent state.

<img src="./distrib-cache-4.drawio.svg">
<p></p>

Data can survive a clean shutdown of a microservice if there is at least one other replica running at that time that has enough capacity to hold its data.

<img src="./distrib-cache-5.drawio.svg">
<p></p>

Cached elements can get evicted for various reason and without warning. Cache only that which you can afford to lose and reconstruct from the original data source. A distributed cache is not shared memory. Do not use a distributed cache to share state among peers.

## The Trouble With a Centralized Cache

Using a centralized cache is a common anti-pattern that may result in system instability or even an outage.

A centralized cache shared by multiple microservices creates a dependency among those seemingly unrelated microservices. For example, a misbehaving microservice can overwhelm the cache, resulting in evictions of elements cached by other microservices. Those in turn will experience excessive cache misses and will have to hit their data stores again and again. This can easily bring down the system to its knees or worse.

Similarly, a centralized cache is a bottleneck and a single point of failure (SPOF). If it is overwhelmed, fails or restarted, all microservices using that cache will be affected at the same time. This too will result in a high number of cache misses and consequently a high load on the data stores.

It is also a matter of security when multiple microservices can read and write to the same cache. For example, a compromised microservice may be able to access user access tokens stored in a centralized cache by the authentication microservice.

A centralized cache often does not allow for setting a different TTL or memory limits on a per-microservice basis. The "SLA" is the same for all clients.

A cache that is localized to a single microservice is isolated from other microservices. The blast radius of a failure is limited to that microservice alone.
