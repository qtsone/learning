# Scalability Patterns

> `shared.systems.scalability` · ~4-5h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Explain why statelessness enables horizontal scaling, and refactor session
  state out of an application instance.
- Implement a rate limiter (token bucket or sliding window) and choose
  appropriate limits and responses (429, `Retry-After`).
- Explain backpressure, and demonstrate how a bounded queue protects a service
  from overload where an unbounded one fails.
- Design a work-sharding scheme that distributes load across workers without
  hot spots and survives worker failure.

## Two ways to grow, and why only one keeps going

**Scale up (vertically)**: give the machine more CPU, more RAM, faster disks.
It requires no code changes, so it is always the right first move — a single
well-tuned box handles far more than most people guess. It ends for two
reasons: the biggest instance money can buy is a hard ceiling, and price grows
faster than capacity long before you reach it. It also leaves you with one
machine, which is one failure away from zero.

**Scale out (horizontally)**: add more machines and spread the work. No
ceiling, commodity prices, and redundancy comes free with the second box. The
bill is paid in design: the work has to be *divisible*, and the machines have
to be *interchangeable*.

Neither is linear. Two effects bend the curve, and naming them is half of
sounding credible in a design review:

- **The serial fraction (Amdahl).** If 5% of a request is inherently serial —
  a lock, a single-writer database, a licence check — no number of machines
  gets you past 20× the single-machine throughput. Find the serial part before
  you buy machines.
- **Coordination cost (the "universal scalability" correction).** Nodes that
  must agree with each other pay a cost that grows with the *square* of their
  number. Past some size, adding capacity makes the system slower. Every
  chatty cross-service call and every distributed lock feeds this term.

Practical consequence: the scalable design is the one that *removes*
coordination, not the one that adds machines. The four patterns in this lesson
are four ways of removing it.

## Statelessness: the precondition for everything else

"Interchangeable" has a precise meaning: **any instance can serve any request,
and losing an instance mid-flight loses nothing but that request.** That holds
exactly when the instance stores no state that another instance would need.

State is not "variables". A request-scoped variable is fine — it dies with the
request. The problem is state that *outlives* a request and that a *later*
request depends on: a session map, an uploaded file on local disk, an
in-process counter used for business decisions, a scheduled job's cursor.

Watch a stateful service fail:

```
        ┌────────────┐
user ──►│    load    │──► instance A     login  → session stored in A's memory
        │  balancer  │──► instance B     next request routed to B → logged out
        └────────────┘──► instance C
```

The usual first patch is **sticky sessions** (session affinity): the balancer
pins each user to one instance. It works, and it quietly re-creates every
problem you were escaping:

- A deploy restarts instances, so every pinned user is logged out.
- An instance dies and takes its users' state with it.
- Load is pinned too: the instance holding your loudest users stays hot while
  others idle, and autoscaling cannot rebalance existing traffic.
- Scaling in becomes destructive — removing an instance destroys live state.

The fix is to move state to somewhere shared and let instances stay empty:

- **Shared store.** Sessions, locks, counters, and job cursors go to Redis or
  a database. This is the caching lesson's "shared cache server" earning its
  keep — one copy every replica reads.
- **Client-held state.** Give the client a signed token (a JWT-style bearer
  credential) so the server can verify a claim without storing it. Nothing to
  replicate, nothing to expire from memory — at the cost that you cannot
  revoke it before it expires without... a shared store. Most systems use a
  token for identity plus a small server-side record for revocation.
- **Object storage.** Uploads go to S3-style storage, never local disk. Local
  disk in a horizontally scaled service is a bug with a delayed fuse.

State does not disappear; it *concentrates* where you can replicate and back
it up deliberately (distributed-systems: replication and its consistency
menu). Your service layer becomes stateless and therefore disposable, and
disposable is what makes autoscaling, rolling deploys, and spot instances
safe.

## Rate limiting: protecting capacity you actually have

A stateless fleet can be scaled — but not instantly, and not for free. A rate
limiter is how you stay inside the capacity you have *right now*. It serves
four distinct purposes, and confusing them produces bad limits:

1. **Protection** — one buggy client retrying in a tight loop must not consume
   the capacity of everyone else.
2. **Fairness** — per-tenant limits stop the noisiest neighbour from starving
   the rest.
3. **Cost control** — for endpoints whose work costs real money downstream.
4. **Abuse control** — credential stuffing, scraping, spam.

Four algorithms, in increasing order of quality:

- **Fixed window.** Count requests per calendar minute; reset at the boundary.
  Trivial, and wrong at the seam: a client can send the full quota at 11:59:59
  and the full quota again at 12:00:00 — *double* the intended rate for an
  instant.
- **Sliding window log.** Keep the timestamp of every request in the last
  window and count. Exact, and memory grows with the limit.
- **Sliding window counter.** Interpolate between the previous and current
  fixed windows. Nearly exact, cheap, widely used in production gateways.
- **Token bucket.** A bucket of `capacity` tokens refilling at `rate` tokens
  per second; each request spends one; empty means denied. Two numbers encode
  two policies: `rate` is the sustained throughput you promise, `capacity` is
  the burst you tolerate. It has no seam, costs O(1) memory per client, and
  a leftover-tokens state that resumes correctly after idle time. This is what
  you implement. (Its mirror image, the **leaky bucket**, drains at a constant
  rate and *smooths* traffic rather than allowing bursts — the right shape
  when the thing you are protecting hates spikes more than it hates delay.)

The important trick in a token bucket is that nothing runs on a timer. You
refill **lazily**: on each request, credit `elapsed × rate` tokens (capped at
`capacity`), then decide. No background goroutine, no clock skew between
refill and check, and the whole thing testable with an injected clock.

**Choosing the limits.** Never invent a number. Start from measured capacity —
the load test says one instance handles 500 requests/second at acceptable
latency — then work backwards through fleet size, expected tenant count, and
headroom. State the burst separately from the sustained rate: real clients are
bursty (a page load fires ten API calls at once), and a limiter with zero
burst tolerance rejects perfectly legitimate traffic. Then *observe before
enforcing*: run the limiter in log-only mode, look at who would have been
rejected, and only then turn it on.

**The response is part of the contract.** A rejected caller gets HTTP
**429 Too Many Requests** — not 403 (that means "never allowed"), not 503
(that means "the server is broken"). Include **`Retry-After`** with a real
number, and make the number *true*: after waiting exactly that long, the next
request must succeed. A limiter that says "retry in 1 second" and then denies
again teaches clients to ignore the header and hammer you. Many APIs also send
`RateLimit-Limit` / `RateLimit-Remaining` / `RateLimit-Reset` so clients can
pace themselves *before* being rejected — the api-design habit of putting the
contract in headers.

**Where to enforce.** At the edge (CDN/gateway) you reject cheaply, before the
request costs you anything — but the edge does not know about your internal
tenants. In the service you know exactly who is calling and what it costs, but
you have already paid for the connection. Real systems do both.

**In a fleet, limits are shared state.** Ten instances each enforcing "100
requests/second" enforce 1,000. The options are a shared counter (Redis; a
network hop on every request, and the limiter's availability becomes yours),
or per-instance limits set to `total / instances` (no hop, but wrong whenever
the load balancer is uneven or the fleet resizes). Consistent hashing — later
in this lesson — gives a third option: route all of a tenant's traffic to one
instance so its local counter *is* the global one.

## Backpressure: what to do when you can't keep up

Rate limiting handles clients who ask for too much. Backpressure handles the
moment when demand exceeds capacity for any reason at all — a slow dependency,
a lost instance, a genuine traffic spike.

The instinct is to queue: accept everything, buffer it, work through it. Two
laws say that fails.

**Little's Law**: `L = λ × W`. Items in the system = arrival rate × time each
spends there. Rearranged: `W = L / λ`. In a *steady* system, where work
leaves as fast as it arrives, arrival and service rate are the same number,
and the law reads: waiting time = backlog ÷ drain rate. If a queue holds
10,000 items and the worker drains 100 per second, everything entering that
queue waits 100 seconds. The queue did not absorb the overload — it
*converted* it into latency. (When arrivals genuinely outrun service, as
below, there is no steady state at all: the backlog and the wait both grow
without bound until something sheds load.)

**Conservation of work**: if arrivals exceed service rate, the backlog grows
without bound for as long as the overload lasts. An unbounded queue never
rejects anything, so it never signals a problem; it just grows, and latency
grows with it, until every response is useless (the client timed out long ago)
and the process dies of memory exhaustion. You will measure exactly this in
the exercise: at 10 arrivals against 8 served per tick, the unbounded backlog
after 1,000 ticks is *ten times* the backlog after 100 — linear growth, with
no steady state. The bounded queue reaches its cap and stays there.

So: **bound every queue**, and decide what happens when it is full.

- **Shed load.** Reject immediately with 429/503 + `Retry-After`. Fast failure
  is a *feature*: the client can retry elsewhere, or degrade, while you stay
  healthy for everyone else. Shed the cheapest, least valuable work first —
  and never shed health checks, or the load balancer will finish you off.
- **Block the producer.** When the producer is your own code (a channel with a
  full buffer, a connection pool with no free connection), blocking propagates
  the pressure back up the chain to whoever can actually slow down. This is
  backpressure in the strict sense.
- **Bound the concurrency, not just the queue.** A limit on in-flight requests
  per instance (a semaphore) is the most effective single knob: it keeps you at
  the throughput peak instead of past it, where context switching and memory
  pressure make the machine *slower* under more load.

Three habits complete the picture. **Timeouts and deadlines everywhere**: a
queued item whose caller has already given up is pure waste — check the
deadline before starting work, not after. **Retry with exponential backoff and
jitter**, and add a circuit breaker: naive retries multiply load exactly when
the system is weakest, which is how a blip becomes an outage. And **measure
queue depth**, because it is the earliest honest signal of trouble — CPU and
error rate move *after* users are already waiting.

## Sharding work: dividing without hot spots

The last pattern splits work that cannot be split arbitrarily: a cache with
per-key state, workers each owning a set of accounts, a partitioned stream.
Some deterministic rule must map key → owner, so every node agrees.

The obvious rule is `owner = hash(key) mod N`. It is uniform, O(1), and
catastrophic to change: go from 5 nodes to 6 and *almost every key* changes
owner, because the modulus changed for all of them. Every cache is cold, every
in-flight ownership assumption is wrong, and the stampede takes you down at
the exact moment you were trying to add capacity.

**Consistent hashing** fixes the resize. Picture the hash space as a ring
(0 … 2⁶⁴−1 wrapping around). Hash each *node* onto the ring; hash each *key*
onto the ring; a key belongs to the first node clockwise from it. Now adding a
node captures only the arc between it and its counter-clockwise neighbour:
roughly `1/N` of the keys move, and they move *onto the new node only* —
existing nodes never trade keys with each other. Removing a node hands exactly
its own arc to the next node clockwise; nothing else moves.

```
        n2
     ·   ▲   ·
   ·     │     ·        key k hashes here ─┐
  ·      │      ·                          ▼
 ·       │       ·   ─────────────────────► n3 owns k
  ·      │      ·      (first node clockwise)
   ·     │     ·
     ·   ▼   ·
        n1
```

With one point per node the arcs are wildly uneven — random points on a circle
clump, and the node after a big gap gets a disproportionate share of the keys
*and* absorbs a dead neighbour's entire load. The fix is **virtual nodes**:
place each physical node at many positions (100–200 is typical) by hashing
labels like `"n3#0"`, `"n3#1"`, …. Many small arcs average out, so each node's
share converges on `1/N`, and when a node dies its many arcs are inherited by
*many* different survivors instead of one. Virtual nodes also let you weight
capacity: give a machine twice the hardware twice the positions.

The hash function matters. It must be deterministic across processes and
restarts (every node must compute the same ring), and it must avalanche —
similar inputs landing far apart. A weak hash puts `key-101` and `key-102`
next to each other, and sequential keys pile onto one node no matter how many
virtual nodes you configure.

Two failure modes survive all of this. **Hot keys**: if one key is 30% of
traffic, no hashing scheme helps, because the unit of division is the key —
you split the key itself (`user42#0` … `user42#9`) or cache it everywhere.
**Correlated failure**: with a shard-per-tenant scheme, one poisonous request
takes down one shard and all the tenants on it; *shuffle sharding* — giving
each tenant a random small subset of nodes — makes the odds of two tenants
sharing their whole subset vanishingly small.

And ownership is not durability. When a worker dies, the ring reassigns its
keys in milliseconds, but whatever lived only in that worker's memory is gone.
Shards that hold state need replication (distributed-systems), typically to
the next R distinct physical nodes clockwise.

In Go:

```go
// The three pieces you build, sketched. Note the injected clock in the
// limiter — same testing habit as the caching and message-queue lessons.
tb := NewTokenBucket(20, 5, time.Now) // burst 20, sustained 5/s
if !tb.Allow() {
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(tb.RetryAfter().Seconds()))))
	http.Error(w, "rate limited", http.StatusTooManyRequests)
	return
}

if !queue.Offer(job) { // bounded: full means shed, never grow
	http.Error(w, "overloaded", http.StatusServiceUnavailable)
	return
}

owner, _ := ring.Get(job.Key) // consistent hashing decides which worker
```

In production you would reach for `golang.org/x/time/rate` rather than writing
a limiter — but a limiter you have not built is a limiter whose burst
behaviour you cannot predict.

## Exercise

Open [`exercises/go/`](exercises/go/) — one module, four files, each mapping
to one objective. `app.go` is a service instance with a scaling bug;
`limiter.go`, `queue.go`, and `ring.go` are skeletons. The `*_test.go` files
are the specification: read them first. The limiter takes an injected
`now func() time.Time` and the tests drive a fake clock, so nothing sleeps and
nothing depends on wall-clock timing.

Acceptance criteria:

1. **Stateless instances.** `App` keeps no session state of its own: a token
   from `a1.Login` resolves through `a2.Whoami` when both share a
   `SessionStore`. Unknown tokens report `ok == false`, and concurrent logins
   across instances stay correct under `-race`.
2. **Token bucket.** A fresh bucket allows a burst of `capacity`, then denies.
   Refill is lazy and proportional to elapsed time (half a token is not a
   token), and never exceeds `capacity` however long the client idles.
3. **`Retry-After` that tells the truth.** `RetryAfter()` returns 0 when a
   token is available, and otherwise a duration rounded *up*, such that a
   caller who waits it is guaranteed to be allowed.
4. **Bounded queue.** `Offer` appends and reports true, or reports false when
   the queue already holds `capacity` items — refusing, never growing. `Take`
   is FIFO and reports `ok == false` when empty.
5. **Overload measured.** `SimulateLoad` runs the arrivals-then-service loop
   and reports `Served`, `Shed`, `MaxQueue`, and `Backlog`. The tests compare
   a bounded run against an effectively unbounded one: the bounded backlog is
   identical after 100 and 1,000 ticks, while the unbounded backlog grows
   linearly with the length of the overload.
6. **Consistent hash ring.** `AddNode` places `replicas` virtual nodes,
   `RemoveNode` takes them all off, and `Get` returns the first virtual node
   clockwise from the key's hash, wrapping past the top. Ownership is
   deterministic; an empty ring reports `ok == false`.
7. **Load spread.** On a ring built with 200 virtual nodes per node, four
   nodes each own within 30% of the mean share of 8000 keys — that even
   spread is what the virtual nodes buy.
8. **Rebalancing measured.** On a ring built with 100 virtual nodes holding
   five nodes, adding a sixth moves roughly 1/6 of keys and moves them *only
   onto the new node*; removing a node moves exactly that node's keys and
   nothing else. `MovedFraction` quantifies both, and the tests contrast it
   with hash-mod-N on the same change.

Run the tests from inside `exercises/go/`:

```sh
cd exercises/go
go test -race ./...
```

They fail on the skeleton. Start with `app.go` — it is the smallest change and
the largest idea.

## Further reading

- [Amazon Builders' Library — Using load shedding to avoid overload](https://aws.amazon.com/builders-library/using-load-shedding-to-avoid-overload/)
  — bounded queues, admission control, and why fast rejection is kindness.
- [Amazon Builders' Library — Workload isolation using shuffle-sharding](https://aws.amazon.com/builders-library/workload-isolation-using-shuffle-sharding/)
  — the blast-radius arithmetic behind the shuffle-sharding paragraph.
- [Dynamo: Amazon's Highly Available Key-value Store (SOSP '07)](https://www.allthingsdistributed.com/files/amazon-dynamo-sosp2007.pdf)
  — section 4.2 is consistent hashing with virtual nodes, in the paper that
  made the technique standard.
- [RFC 6585 §4 — 429 Too Many Requests](https://www.rfc-editor.org/rfc/rfc6585#section-4)
  and [pkg.go.dev — golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
  — the status code's definition, and the limiter you would use in production.
