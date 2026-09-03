# Caching

> `shared.systems.caching` · ~3h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Place caches across the layers — browser, CDN, gateway, application,
  database — and explain what each layer can and cannot absorb.
- Choose between cache-aside, read-through, write-through, and write-behind
  for a given workload and justify the choice.
- Explain why invalidation is hard — stale reads, thundering herds, cache
  stampedes — and implement mitigations: TTL, jitter, singleflight.
- Implement a TTL-based in-memory cache with an eviction policy and
  demonstrate correct behavior under concurrent access.
- Explain when a CDN or Redis is the right tool versus in-process caching.

## The cheapest read is the one you don't do

In the data-storage lesson you scaled reads with indexes and replicas. Both
still pay for a network hop and a database's machinery on every request. A
cache skips the work entirely by keeping a copy of a recent answer closer to
the asker. The latency ladder explains the payoff — rough, order-of-magnitude
numbers worth memorizing:

- read from local memory: ~100 nanoseconds
- read from SSD: ~100 microseconds
- round trip within a datacenter: ~0.5 milliseconds
- database query (parse, plan, buffer pool, maybe disk): ~1–10 milliseconds
- round trip across an ocean: ~100 milliseconds

A cache turns the bottom of the ladder into the top. And because almost every
consumer-facing workload is read-heavy (design-intro: one write, many reads),
a cache in front of a database can absorb the overwhelming majority of
traffic.

One arithmetic habit before anything else: **think in miss rate, not hit
rate**. A cache going from 99% to 98% hits sounds like a 1% change; it is a
*doubling* of the load that reaches your database (1% → 2% of all traffic).
Capacity planning behind a cache is planning for misses — including the worst
case where the cache is empty (a restart, a flush) and 100% of traffic lands
on the origin. If the origin cannot survive that even briefly, the cache is
load-bearing infrastructure, not an optimization, and must be treated with
matching care.

## One request, five caches

A single page load can be served — or partially served — from five different
layers. Each absorbs something the layers below would otherwise pay for, and
each has a hard limit on what it may hold.

1. **Browser (client) cache.** The user's own machine: zero network cost, the
   fastest possible hit. Controlled by the HTTP `Cache-Control` headers your
   API design already decides (api-design lesson: headers are contract). It
   can absorb repeat views by *one* user; it cannot help any other user, and
   once shipped, a too-long lifetime is unfixable — you cannot reach into
   browsers and invalidate.
2. **CDN (edge) cache.** Hundreds of locations near users, shared across all
   of them. Absorbs bandwidth and geography: static assets, images, video,
   and any response cacheable *by URL*. It cannot absorb personalized
   responses ("your feed") unless you carefully split personal from shared
   content, and it is operated per HTTP semantics — if your headers say
   private or uncacheable, the CDN obeys.
3. **Gateway / reverse-proxy cache.** Same idea as a CDN but in your own
   infrastructure, in front of your services. Absorbs whole-response
   recomputation for hot, shared endpoints. Same personalization limit.
4. **Application cache.** Inside or beside your service: an in-process map,
   or a shared cache server (Redis, Memcached). This is the layer *you*
   program directly, and the only one that can cache things that are not
   HTTP responses: a session, a computed aggregate, a permission check, a
   database row. It is where this lesson's exercise lives.
5. **Database caches.** The buffer pool keeping hot pages in memory, and
   materialized views precomputing expensive queries. They make queries
   cheaper but cannot make them free: the network hop and query machinery
   remain. When the database is the bottleneck, its own cache is the last
   layer to help — the win is preventing the request from arriving at all.

Design habit: when someone says "add caching", ask *which layer*. "Cache the
avatar images" is a CDN decision; "cache the permission check" is an
application decision; they share nothing but the word.

## Where writes meet the cache: four patterns

A cache holds a *copy*, so every pattern is an answer to one question: how
does the copy relate to the truth when data changes?

**Cache-aside (lazy loading).** The application talks to both sides itself:

```
read(key):
    value = cache.get(key)
    if missing:
        value = database.read(key)
        cache.set(key, value, ttl)
    return value

write(key, value):
    database.write(key, value)
    cache.delete(key)          # invalidate; the next read repopulates
```

The default choice. Only requested data occupies memory, a cache outage
degrades to plain database reads, and it works with any cache. Costs: every
miss pays cache-miss *plus* database read, and the application carries the
invalidation logic — which, as the next section shows, is where the bugs
live.

**Read-through.** Same laziness, but the cache itself fetches from the origin
on a miss — the application only ever asks the cache. This centralizes the
loading logic (one place to add stampede protection) at the cost of a
smarter cache component. Your exercise's `GetOrLoad` is exactly this shape.

**Write-through.** Every write goes to the cache *and* the database
synchronously before the write is acknowledged. Reads after writes are
always fresh, at the price of slower writes and of caching data that may
never be read. Fits read-heavy data where staleness right after a write is
unacceptable — often paired with cache-aside reads.

**Write-behind (write-back).** Writes go to the cache and are acknowledged
immediately; the cache flushes to the database asynchronously, often
batched. Spectacular write latency and batching wins — and the sharpest
trade-off in this lesson: until the flush lands, the *only* copy of the
write is in a cache, so a crash loses acknowledged data. Choose it only when
losing a small window of writes is acceptable (view counters, metrics), and
say that out loud in the design. The async-delivery machinery it needs is
next lesson's topic.

Justifying a choice is naming the workload: read-heavy with tolerable
staleness → cache-aside with TTL. Read-your-own-write required → write-
through (or invalidate precisely). Write-heavy and lossy-tolerant →
write-behind. Interview answers without the workload named are vibes.

## Invalidation is genuinely hard

A cache entry is **derived state** — a copy that does not know when its
source changes. You have exactly two levers, and real systems pull both:

**Bounded staleness: TTL.** Every entry dies at most *d* after it was
written. You are not eliminating staleness, you are *capping* it, and the
cap is a product decision: "prices may be up to 60 seconds old" is a
requirement you can defend; an unstated TTL is a stale-data bug waiting for
a user to find it.

**Explicit invalidation.** Delete or overwrite the entry when the source
changes. Precise — and racy. The classic cache-aside race:

```
t1  reader:  cache miss for key K
t2  reader:  reads K from the database        → sees old value
t3  writer:  writes new value for K, deletes K from cache
t4  reader:  cache.set(K, old value, ttl)     → stale value cached!
```

The reader lost a race it did not know it was in, and the cache now serves
the old value until the TTL expires — which is precisely why even "we
invalidate explicitly" systems still set a TTL as a backstop. This race is
also why the write path *deletes* rather than *sets*: two concurrent writers
setting the cache can interleave against the database order and pin the
older value; a delete is always safe to repeat.

**Thundering herd / cache stampede.** The failure mode that turns caches
into outage machinery. A hot key expires (or a cold cache starts), and every
concurrent request misses *at once* — hundreds of identical expensive
queries hit the origin simultaneously, the origin slows, more requests pile
up, and the system falls over at exactly its busiest moment. Mitigations,
usually combined:

- **TTL jitter.** If ten thousand entries were cached during the same warm-up
  minute with the same TTL, they expire in the same minute too. Adding a
  random offset (say ±10%) to each entry's TTL decorrelates the expiries and
  turns a spike into a drizzle.
- **Singleflight (request collapsing).** Per key, let exactly *one* caller
  fetch from the origin; every concurrent caller for the same key waits and
  shares that result. The origin sees one query per hot key per expiry
  instead of one per request. You implement this in the exercise.
- **Serve-stale (soft TTL).** Keep serving the expired value while one
  background fetch refreshes it. Trades a little extra staleness for zero
  user-visible miss latency on hot keys.

## Eviction: deciding what to forget

Memory is finite, so a cache must also choose what to *forget* when full.

- **LRU (least recently used)** evicts the entry unused for longest —
  betting that recent past predicts near future. Right often enough that
  it is the default everywhere, and O(1) with a map plus a doubly linked
  recency list (your S2 data structures, earning rent).
- **LFU** counts uses instead of recency: keeps steady favorites, but slow
  to forget yesterday's hits and costlier to track.
- **FIFO / random** are nearly free and surprisingly decent; Redis offers
  approximated-LRU/LFU sampling because exact tracking has real overhead.

LRU's known weakness: a one-off scan over many keys (an export, a crawler)
touches everything once and flushes the genuinely hot set. If your hit ratio
craters at 3 a.m., look for a batch job.

## In-process, Redis, or CDN?

Three tools, one decision: *how far away is the copy, and who shares it?*

- **In-process** (a map in your service): nanosecond hits, no moving parts —
  but per-replica. Ten replicas hold ten independent copies, each with its
  own misses and its own staleness, and there is no way to invalidate them
  all at once. Right for small, hot, staleness-tolerant data: config, feature
  flags, permission lookups.
- **Redis / Memcached** (shared cache server): one copy all replicas share —
  a hit for one is a hit for all, and one delete invalidates for everyone.
  Costs a network round trip (~0.5 ms — still 10× faster than the query it
  replaces) and is a service you now operate: memory limits, eviction
  policy, and its own availability story. The default for session data and
  database-query results in any multi-replica system.
- **CDN**: for HTTP responses cacheable by URL, nothing else touches its
  combination of geography and bandwidth offload. Not a tool for application
  state.

These stack: CDN for the static 90%, Redis for shared hot data, in-process
for the tiny scorching subset — each layer shrinking what the next must
absorb.

In Go:

```go
// The exercise's cache API. Two S3/S5 habits show up here:
// inject the clock (func() time.Time) so tests control time, and
// keep loader calls outside any lock you hold.
c := cache.New[string, User](10_000, 5*time.Minute, time.Now)
u, err := c.GetOrLoad(id, func() (User, error) {
    return fetchUserFromDB(ctx, id) // runs once per key, however many callers
})
```

In production Go you would reach for `golang.org/x/sync/singleflight` for
request collapsing rather than hand-rolling it — but you are about to
hand-roll it, because you cannot reason about a stampede you have never
collapsed yourself.

## Exercise

Open [`exercises/go/`](exercises/go/) — a module with a skeleton in
`cache.go` and the specification in `cache_test.go`. Build a concurrency-safe
generic cache with TTL expiry, LRU eviction, and singleflight loading. The
constructor takes an injected `now func() time.Time`; the tests drive a fake
clock, so no test ever sleeps or depends on real time.

Acceptance criteria:

1. `Get` on an absent key reports a miss; `Set` then `Get` round-trips;
   `Set` on an existing key overwrites its value.
2. TTL: an entry written at time T is live strictly before T+ttl and gone at
   or after T+ttl. `Set` on an existing key resets its TTL as well as its
   value.
3. LRU: the cache never holds more than `capacity` entries; inserting a new
   key into a full cache evicts the least-recently-used entry, where both
   `Get` hits and `Set` count as use.
4. `Delete` removes a key; deleting an absent key is a no-op.
5. `GetOrLoad` returns a cached value without calling the loader; on a miss
   it calls the loader, caches a successful result, and returns it. A loader
   error is returned to the caller and nothing is cached.
6. Singleflight: concurrent `GetOrLoad` calls for the same missing key run
   the loader exactly once, and every caller receives its result.
7. All of the above is safe under the race detector with concurrent use.

Run the tests from inside `exercises/go/`:

```sh
cd exercises/go
go test -race ./...
```

They fail on the skeleton — that is the specification talking. Read the test
file before you write a line: it is the precise contract, including the
expiry boundary and what "counts as use" means.

## Further reading

- [Amazon Builders' Library — Caching challenges and strategies](https://aws.amazon.com/builders-library/caching-challenges-and-strategies/)
- [MDN — HTTP caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching)
- [pkg.go.dev — golang.org/x/sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [Scaling Memcache at Facebook (NSDI '13)](https://www.usenix.org/conference/nsdi13/technical-sessions/presentation/nishtala)
