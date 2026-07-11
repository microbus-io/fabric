## Design Rationale

The `dlru` cache is a distributed LRU cache segmented across the replicas of a microservice. Each key is owned by
exactly one replica, chosen by rendezvous (HRW) hashing over the current membership set, and every operation is routed
to that single owner rather than multicast to all peers. It sits on top of the single-node [lru](../lru) cache (one
per replica) and is tied to the connector lifecycle: the connector constructs it during `Startup`, closes it during
`Shutdown`, and exposes it as `DistribCache()`.

These notes cover the *why*. The load-bearing idea throughout: **only the peer set must be agreed on (which is O(N)
and cheap); the key-to-owner map is never agreed on** - each peer computes it locally from the set.

### Owner-routed, not multicast

`owner(key)` is `argmax` over the members of `hash(key, memberID)` (rendezvous / highest-random-weight hashing). It is
a deterministic function of the membership set, so any peer that holds the same set computes the same owner with no
coordination. A `Store` or `Load` is therefore a single unicast to `id-<owner>.hostname/one`, not a broadcast to every
peer followed by an ack window. Ownership is inherently O(N) in the replica count - a map cannot make an `argmax`
O(1) - but N is small, so the per-operation cost is negligible next to a network hop.

Capacity still scales linearly with replica count, since each key lives on one replica - that was the original cache's
rationale too. The difference is that this design does not pay a cluster-wide fan-out on every operation to achieve it.

### The generation is a hash of the membership set, so there is no coordinator

The generation is `sha256` of the sorted member IDs. Any two peers that agree on the set derive the same generation
with zero coordination: no leader, no election, no version service. A membership change is observable as a generation
change. This is the entire consensus surface of the cache - peers agree on *who is present*, and ownership and routing
are pure functions of that agreement.

### The generation gate tolerates disagreement instead of preventing it

Owner routing is correct only when caller and owner share the same view, and during a membership change they briefly
will not. Rather than block or coordinate, each request is stamped with the caller's generation and the owner runs a
gate: accept if the stamp equals the owner's current generation, or, within an overlap window, the owner's
immediately-previous generation; otherwise reject with `417 Expectation Failed`. A rejected `Store` is dropped and
tolerated as a future miss (the value recomputes); a genuine `Load` miss is `404`. The gate is enforced at request
time off the cache-level generation. Values are stored as bare `[]byte` with no per-element version stamp; an earlier
per-element generation/monotonic pair was write-only dead code and was removed.

### Previous-generation retry covers the ownership handoff

When a key's owner changes, the key is a miss on the new owner until it is shed there. Within the overlap window, a
`Load` miss triggers one retry against the *previous* generation's owner, stamped with the previous generation, which
that owner still accepts because its own gate honors its recent-previous generation. First hit wins. It is a single
sequential retry (depth 1), not a parallel multi-generation gather. This is the read-side safety net for the handoff
window; the write side is handled by offload plus soft store, below.

### Two subscriptions: control broadcasts vs. addressed data

`/all` is a no-queue multicast (`sub.NoQueue`) reached by every peer; it carries the control plane - `ping`
(discovery), `join`, `leave`, `clear`, `deletePredicate` - plus the cluster-wide aggregates `weight` and `len`.
`/one` is addressed directly to one replica through the `id-<owner>` hostname prefix; it carries the data plane -
`store`, `load`, `delete` - to exactly the owning replica. Splitting them keeps the hot data path a unicast to one
peer, while the rare control operations that genuinely need every peer stay a broadcast. Both dispatch on a `do` query
argument, and both reject any message whose `FromHost` is not this microservice's hostname, so only same-microservice
peers participate.

Unlike the original cache, there is no `FromID` self-filter: discovery, join, and leave broadcasts deliberately reach
the sender too. A replica counts itself among the responders to its own `ping`, and its own `leave` on `Close` removes
itself from the set through the same handler that removes any other peer.

### Membership is learned by broadcast, and startup convergence is synchronous

A replica discovers peers by broadcasting `ping` on `/all` and collecting responder IDs. It re-discovers periodically
(default one minute) and immediately when a request to an owner times out (see below). On startup it announces itself
with `join`. The subtle part is that a freshly-started cluster must converge *without* waiting for the periodic ping,
or replicas would disagree on ownership for up to a minute. Two invariants make convergence complete by the time every
`NewCache` returns:

1. **The join exchange is two-way.** `broadcastJoin` both announces this replica (peers add it in their join handler)
   and absorbs each responder's ID into its own set. For any pair of replicas, at least one broadcasts after the other
   has activated, and that single broadcast teaches both directions, so a join that races ahead of a peer's activation
   can no longer leave the two permanently unaware.
2. **The owner sub is activated before the broadcast sub, and the join is broadcast after both.** A peer only ever
   answers a ping or join through its `/all` handler, so activating `/one` first guarantees that any replica another
   node learns about is already able to serve store and load - nothing routes to an owner whose data handler is not
   yet up. The join must be broadcast *after* `/all` is active, not between the two activations, because a node must
   be able to receive joins (not only send them) for the two-way exchange to reconcile both directions.

`TestDLRU_ConcurrentStartupConvergence` pins this at the default one-minute ping interval, so a regression cannot hide
behind periodic re-discovery.

### One offload serves both shutdown and join; soft store makes it safe

When ownership moves, the old owner must ship the displaced keys to the new owner. A single `offload` pass covers both
triggers: on `Close` this replica has already removed itself from the set, so every key moves; on a `join` only the
keys the new peer now owns move. It snapshots the membership and generation, then, for each key it no longer owns,
fires a **soft** store (`?do=store&soft=true`) to the new owner and deletes the key locally.

Soft store is the conflict-safety mechanism, and it is why the cache needs no versions. A soft store is store-if-absent
(the lru's atomic `LoadOrStore`): it fills the key on the new owner only if nothing is there. Combined with
delete-on-shed (a replica never keeps a key it no longer owns), this resolves the shed race - old owner leaves
rotation, an upstream write of V2 lands on the new owner, the old owner then offloads its stale V1 - in favor of the
live write, in either interleaving. The offloaded V1 cannot overwrite V2 (soft store yields), and a subsequent hard
write of V2 always wins over a soft-stored V1. `TestDLRU_ShedInFlight` freezes the shed mid-pass, via a checkpoint, to
pin both this no-clobber property and the read liveness the previous-generation retry provides during the window.

### Delete tombstones keep deletes deleted during a shed

Soft store resolves the write/shed race, but a *delete* leaves absence, and absence is exactly what a soft store
fills: a shed of key K fired by the old owner but not yet landed races a `Delete` of K - the delete removes the key
from both owners, then the late soft store lands on the new owner and resurrects the stale value. The owner
therefore records a tombstone when it deletes a key within the transition window (the only time sheds can be in
flight), and the soft store consults it: fill-if-absent becomes fill-if-absent-and-not-recently-deleted. Each
tombstone stores the timestamp until which it is valid (the transition window), and the soft store re-checks after
filling - a delete records its tombstone *before* removing the value, so any interleaving of a concurrent delete is
caught by one of the two checks.

`Clear`, `DeletePrefix`, and `DeleteContains` cannot tombstone the keys they must suppress by name: a key mid-shed
sits on no replica, so no walk can match it. They instead record a blanket "accept no soft stores until" timestamp
covering the window - heavier, but these are rare invalidation events, and a rejected shed is just a miss.
Tombstones live in a side map rather than as marker values in the lru, so `Len` and `Weight` stay truthful. The map
only accrues entries within the window; expired entries are swept on the next delete, and the whole map is dropped
on the first delete outside the window.

### Offload fires and drains within a bounded budget

The soft stores are fired without waiting for each response (throughput), then *all* responses are drained once at the
end. Draining is what makes it safe: it waits roughly one round trip in total rather than one per key, and it ensures
the fire-and-forget response goroutines have settled before `offload` returns, since otherwise they would outlive
`Close` and race the connector's teardown. The whole pass is bounded by `OffloadDuration` through a context timeout,
so `Close` cannot linger, and the connector clamps `OffloadDuration` to fit inside its mandatory teardown budget. That
same knob defines the overlap window used by the gate and the previous-generation retry: the window is
`2 x OffloadDuration` (8s by default), enough for an offload to drain plus margin for the change to propagate.

### A dead owner is a 404, and it coalesces into one re-discovery

If a request routes to an owner that has crashed, the owner acks with a 404 timeout. The caller treats it as a miss (a
`Store` is dropped, a `Load` misses, a `Delete` is done, since a dead owner cannot hold the key) and signals the
discovery loop to re-derive membership immediately rather than waiting for the next ping. The signal is coalesced
through a capacity-1 channel, so a burst of timeouts to a departed owner queues at most one re-derivation. A 417
gen-mismatch triggers the same signal: mid-transition it is redundant (the caller converges through the join and
leave broadcasts anyway, and re-derivation is a no-op once views agree), but a caller that missed a membership
broadcast would otherwise keep stamping a stale generation until the next periodic ping.

### The failure boundary: crash is safe, partition is lossy

A crashed replica takes its share of the cache with it; those keys become misses and are recomputed, which is safe for
a cache. A network partition is the hard case: each side sees a different membership set, derives a different
generation, and computes different owners, so the same key can be written independently on both sides (a lossy
split-brain). The cache does not attempt to prevent this - preventing it would require the coordination the design
deliberately avoids - and it is CAP-unavoidable for an available cache. Cache only what you can afford to lose and
recompute.

### Subscriptions are owned by the cache; Close unsubscribes before offloading

`start` registers `/all` and `/one` with `sub.Manual()` so the connector's automatic activation passes skip them, then
activates them itself so the cache is reachable from inside `OnStartup`. `Close` symmetrically ends the discovery loop,
announces departure, drops itself from the set, `Unsubscribe`s the subs (a full unsubscribe, not just a deactivate),
then offloads and clears. Unsubscribing before the offload guarantees the offloaded stores route only to peers, and a
full unsubscribe rather than a deactivate is what makes connector restart safe: the connector resets
`distribCache = nil` after `Close` and constructs a fresh cache on the next `Startup`, which would fail on a duplicate
subscription name if the prior subs were left registered. The restart contract covers both a clean
`Shutdown`-`Startup` cycle and the case where the first `Startup` failed inside `OnStartup` and the connector still ran
`Shutdown` to clean up.

### Options that do not fit the single-owner model are no-ops

For API compatibility with callers of the original cache, `Store` and `Load` accept the same option set, but two are
deliberate no-ops. `ConsistencyCheck` is moot: with a single owner there is no second copy to disagree, so the
generation gate replaces the original's checksum broadcast entirely. `Replicate` (write-to-all) contradicts the
single-owner model and its capacity rationale, so it is accepted and ignored rather than silently changing routing.
`Compress` (brotli behind a four-byte magic-word prefix, so compressed and uncompressed values coexist) and the load
options `Bump` / `NoBump` / `MaxAge` are honored, threaded through to the owner's local lru. The magic-word scheme
has a known collision: an uncompressed value that happens to begin with the prefix bytes is misread as compressed on
load and fails to decode. At roughly 2^-32 per arbitrary binary value this is an accepted trade-off of coexistence;
callers storing adversarial or high-volume raw binary should compress uniformly.

### Weight and Len sum disjoint shards

Because each key lives on exactly one owner, cluster-wide `Weight` and `Len` are a broadcast over `/all` that sums each
replica's disjoint local total, with no de-duplication needed (unlike a replicated cache). `Hits` / `Misses` and the
`microbus_cache_operations` counter are per-replica; the counter carries `op` (`store` / `load` / `delete`) and, for
loads, `hit` (`local` / `remote` / `miss`).

### Where the design deliberately stops

This is the minimal lossy design, and several natural extensions were considered and intentionally left out because
the simpler machinery already keeps every failure in the miss bucket (safe) rather than the stale-serve bucket
(dangerous). Recorded here so they read as chosen scope, not oversights:

- **Eager delete-on-shed, not retain-then-delete or migrate-on-read.** An old owner deletes each key the instant it
  ships it, and ships every value it no longer owns. A brief retain window plus pull-on-read would avoid shipping the
  cold tail that is never read again, at the cost of per-transition bookkeeping. It was not built because eager shed
  plus the previous-generation retry is simpler and a transient miss is cache-tolerable.
- **Depth-1 previous-generation retry, not a parallel multi-generation gather.** Two topology changes inside the
  overlap window can strand a key, which degrades to a miss, never to a stale serve. Surviving deeper churn would
  need a parallel gather across several generations' owners (reconciled by version), which in turn would need the
  per-key version stamps below.
- **No cross-owner version reconciliation.** There is no per-key version (node-ID-tiebroken counter, wall-clock LWW,
  or HLC). It would only matter if concurrent writes to the same key landed on two different owners and had to be
  ordered by value; soft store plus delete-on-shed makes that unnecessary in the single-owner steady state.
- **404 re-derive is local, not a coalesced cross-node broadcast.** A tagged, de-duplicated broadcast would shorten
  the post-crash miss window cluster-wide, but it is an availability optimization, not a correctness one (a crash only
  ever produces misses), and a naive version risks an N-squared ping burst.

These become worth revisiting only at large N with a hot cache; the correctness machinery (gate, soft store,
delete-on-shed, prev-gen retry) does not change if they are added.

### Test seams: faults and checkpoints

The cache embeds a `seamster.Seamster` (inert in production). Faults make a site misbehave: `skipLeave` (drop the
departure announcement, exercising periodic rediscovery), `staleGen` (stamp a mismatching generation, exercising the
gate), and `skipOffload` (skip shedding, parking a stale copy on the old owner for previous-generation tests).
Checkpoints make a site pausable so a test can drive a precise interleaving deterministically: `offloadBeforeStore`
(freeze a shed just before a displaced key ships) and `beforeSend` (freeze a `Store` after the owner and generation
are stamped onto the request, before it is sent). Both are reached from tests through the `Seams()` accessor plus
exported fault and checkpoint name constants. All are free in production, gated on the seams' disabled state.
