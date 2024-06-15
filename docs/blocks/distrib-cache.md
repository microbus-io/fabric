# Distributed Cache

Each microservice implements an in-memory [LRU cache](../structure/lru.md) that is distributed among all peer replicas. The cache uses pub/sub over NATS to communicate and synchronize with peers.

<img src="./distrib-cache-1.drawio.svg">
<p>

The capacity of the cache scales horizontally with the number of replicas of the microservice.

<img src="./distrib-cache-2.drawio.svg">
<p>

The cache is scoped to a single microservice, therefore isolating it from side-effects that can be caused by "noisy neighbor" microservices. Isolation also makes it possible to independently scale to the individual needs of each microservice.

<img src="./distrib-cache-3.drawio.svg">
<p>

Operations are synchronized over the network and the cache is not immune to race conditions. To help improve consistency, the `Load` operations checks with peers to ensure there are no multiple versions of the same element. This is still not a 100% guarantee of consistency.

<img src="./distrib-cache-4.drawio.svg">
<p>

Data can survive a clean shutdown of a microservice if there is at least one other replica running at that time that has enough capacity to hold its data.

<img src="./distrib-cache-5.drawio.svg">
<p>

Cached elements can get evicted for various reason and without warning. Cache only that which you can afford to lose and reconstruct from the original data source. A distributed cache is not shared memory. Do not use a distributed cache to share state among peers.
