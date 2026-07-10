## Design Rationale

The `lru` package is a single-node, weight-bounded, TTL-aware LRU cache: a `map` for lookup plus an intrusive
doubly-linked list for recency, guarded by one mutex. It is the local tier under [dlru](../dlru); the notes here
cover the decisions that are not obvious from the code.

### `Load` takes the exclusive lock because a read is a write

An LRU read is not read-only. The default `Load` bumps: it unlinks the entry and relinks it at the front of the
recency list, renews its `inserted` timestamp, and updates the hit/miss counters. All of that mutates shared state,
so `Load` takes the plain `sync.Mutex` exclusively, exactly like `Store` and `Delete`. There is deliberately **no**
`RWMutex`: an `RLock` on the read path would be wrong, since the read writes. Only a `NoBump()` load (used by
`Exists`) is genuinely read-only, and it is the minority. Consequently the way to add read concurrency is not to
swap in an `RWMutex` but to first stop mutating on most reads (e.g. probabilistic/lazy bump, or a CLOCK-style
reference bit), which is a larger change and unnecessary at the volumes a roundtrip-saving cache actually sees - the
single mutex sustains millions of loads/sec (see `BenchmarkLRU_LoadBumpParallel`), orders of magnitude above real
request rates, so it is not sharded and not an `RWMutex` on purpose.

### TTL expiry is reaped proactively from the tail, exploiting an insertion-order invariant

Every `store` and every bump stamps the current time on the node **and** moves it to the front of the list. Nodes
only drift toward the tail as newer ones are inserted or bumped ahead of them. Therefore the list is ordered by
`inserted` descending from front to tail, and `c.oldest` always holds the **minimum** `inserted` - it is always the
entry closest to expiry. That makes a single O(1) tail check a complete expiry oracle: **if the tail is not expired,
nothing is.** `maybeTrimOldest` uses this to reap expired entries from the tail without ever scanning.

This matters because of the *under-the-weight-limit* case. `diet` (weight eviction) already removes from the tail,
so under memory pressure the most-expired entries go first and dead data never outranks live data. The gap was a
cache that is **not** full: `diet` never fires, so an expired entry that is never read again would linger forever,
pinning memory and inflating the reported weight. `maybeTrimOldest` closes exactly that gap - it runs on ordinary
activity regardless of weight pressure.

It is bounded (`maybeTrimOldest(k)`, currently `k == 2` at each call site) so a single operation never stalls
reaping a large backlog; the remainder is caught by later operations. It is called from the internal `load`,
`store`, **and** `delete`, so any activity - including read-heavy workloads with few stores - keeps the tail clean.

### `maybeTrimOldest` must not call `delete`

`delete` calls `maybeTrimOldest`, so `maybeTrimOldest` must not call `delete`, or the two would recurse. It instead
inlines the tail unlink (the same removal `diet` performs), operating only on `c.oldest`. Keep this constraint in
mind before "simplifying" the reaper to reuse `delete`.

### Expiry is measured against the cache-wide max age, not the per-call max age

The cache enforces a **maximum** TTL (`c.maxAge`); a caller may pass a *shorter* per-call `MaxAge` to a `Load`, but
not a longer one. Two consequences follow. A `Load` with a shorter `MaxAge` that finds the entry past that shorter
age treats it as a miss **and deletes it then and there** (the load path's own expiry check). But `maybeTrimOldest`,
which runs on unrelated activity with no per-call age, measures against `c.maxAge` alone - so an entry within the max
age is left in place even if some shorter-age read would have missed it. "Reclaimed only once it exceeds the max
age" is the contract the reaper relies on: an entry past `c.maxAge` is dead for every possible reader, so trimming
it is always safe.

### A single `now` is threaded through one operation

Each public operation reads the clock exactly once via `c.now()` and threads that `now` snapshot through the
internal `load`/`store`/`delete`/`maybeTrimOldest`. This is both a performance and a correctness choice. Performance:
the bump path would otherwise call `time.Now` up to three times (reaper check, expiry check, bump renewal); a single
read makes the reaper's cost effectively nil and leaves the bump path faster than before the reaper existed.
Correctness: every sub-step of one operation observes the same instant, rather than a clock that can advance
mid-operation. `now()` also folds in `timeOffset`, the test-only clock control that lets tests advance time
deterministically; because `now` is captured once per op and `timeOffset` changes only between ops, the snapshot is
equivalent to reading the clock live.

### Weight, not count, is the budget

Capacity is a total **weight**, not an entry count; each entry carries a caller-supplied weight (default 1), and
`SetMaxWeight`/`SetMaxMemory` (in dlru) bound the sum. `diet` evicts from the tail until the total is back under the
limit. Billing by weight is what lets dlru account real memory honestly (it weighs each value by its byte length),
so the eviction budget reflects bytes held rather than a proxy count.

A value whose own weight exceeds `maxWeight` can never be kept, and `store` rejects it up front rather than inserting
it and letting `diet` reclaim it. Inserting first is not equivalent: the oversized node pushes the total over the
limit, so `diet` would shed *other, smaller, live* entries from the tail before finally evicting the oversized one -
wiping good data to make room for a value that cannot stay. Two consequences at the seam:

- `Store` still deletes any prior entry under the key **before** the size check, so overwriting with an oversized
  value removes the old value rather than leaving a stale one the caller believes it replaced. (The distributed
  `dlru` layer above returns success regardless of whether the local keep happened, so this delete is what prevents
  a stale local read after an oversized overwrite.)
- `LoadOrStore` of an oversized value for an absent key is a clean no-op - it neither keeps the value nor disturbs
  the rest of the cache.
