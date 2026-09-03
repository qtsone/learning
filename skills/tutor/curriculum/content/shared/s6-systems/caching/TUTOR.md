# Tutor notes — Caching

## Where the learner is

Mid-S6, after data-storage: they can already reason about replicas, indexes,
and sharding, and design-intro drilled the estimation habit. From the Go path
(S1–S5) they have real concurrency experience — mutexes, channels, `-race` —
and S2's linked lists and hash maps, which this exercise finally combines
into something production-shaped. This is the stage's second implementation
lesson (after networking); expect relief at writing code again, and use the
implementation to anchor the theory: every test maps to a concept from the
lesson.

## Common misconceptions

- **"High hit ratio = done."** Hit ratio is a means; the goals are tail
  latency and origin load. Push the miss-rate framing: 99% → 98% hits
  *doubles* database traffic. Also ask about the cold-cache worst case.
- **"TTL solves invalidation."** TTL *caps* staleness; it does not eliminate
  it. The cap is a product decision that must be stated, not defaulted.
- **"Set the cache on write" as obviously correct.** The cache-aside write
  path deletes rather than sets because concurrent writers can interleave
  against database order and pin an older value; deletes are safely
  repeatable. Many learners flip this without noticing.
- **"Just add Redis."** Redis is a shared copy with a network hop and its
  own operational surface. For per-replica hot data an in-process map wins;
  for URL-addressable HTTP responses the CDN wins. "Which layer?" is the
  question.
- **"Singleflight helps across replicas."** Their `GetOrLoad` collapses
  callers *within one process*. Ten replicas still make ten origin calls per
  hot key. Distributed collapsing needs a shared cache or lock — plant the
  seed, don't chase it.
- **"Expired entries free their memory."** With lazy expiry (what the tests
  specify), a dead entry lingers until a Get touches it or LRU evicts it.
  Good learners notice this; excellent ones can say why it is acceptable.

## Grilling points

- "Walk me through the stampede: a hot key expires under 1,000 concurrent
  readers. What happens with your `GetOrLoad`, and what would happen
  without it?"
- "Why does the constructor take `now func() time.Time`? What would the
  expiry tests look like without it?" (They'd sleep — slow, flaky, and
  banned; injected clocks are the S5 testing habit generalized.)
- "Your loader runs outside the mutex. What breaks if you hold the lock
  across it?" (Every operation on every key stalls behind one slow origin
  call — a cache outage caused by the cache.)
- "Your cache is full and a new key arrives. Which entry leaves, and why is
  that bet the default? Show me the two structures that make the choice O(1)
  instead of a scan, and tell me what LFU or FIFO would buy or cost you."
  (Recency predicts reuse; map + doubly linked list from S2; Redis
  approximates by sampling because exact tracking is not free.)
- "Your hit ratio craters every night at 3 a.m. What do you go looking for?"
  (Scan pollution: a nightly export or crawler touches every key once, and
  LRU dutifully evicts the hot set for entries nobody will ask for again.)
- "A loader fails. Why must the error *not* be cached — and when would you
  actually want negative caching, briefly?"
- "Two writers update the same row and both refresh the cache. Draw the
  interleaving where the cache ends up with the older value."
- "Your service has 10 replicas, each with this in-process cache. A user
  updates their profile. What do the other 9 replicas serve, and for how
  long? What are your options?"
- "Which pattern for: a product page read 10,000× per write with 60s
  staleness budget; a view counter at 5,000 writes/s; a user's own settings
  page that must reflect their edit instantly?"

## Grading rubric

- **A** — All tests pass under `-race`. One mutex (or equivalent) guards
  map + list coherently; loader runs outside the lock; flight results are
  published via channel close (or equivalent happens-before edge); LRU is
  O(1) via map + linked list, not a scan; expiry boundary (`age >= ttl`
  misses) handled exactly. Learner explains the stampede, the delete-vs-set
  race, and layer placement unprompted.
- **B** — Tests pass, but with a design wart they cannot defend: loader
  held under the lock, O(n) recency scan on every Get, duplicated
  lock/unlock logic instead of locked helpers, or waiters polling a flag
  instead of blocking on the flight. Theory solid on patterns and layers,
  shakier on the race interleavings.
- **C** — Tests pass only after heavy hinting, or singleflight was
  reverse-engineered from the test without being able to say what problem
  it solves. Quiz shows the layers but not the invalidation reasoning.
  Time-boxed remediation on the stampede story before passing.
- **Fail** — Race detector failures, tests failing, or the learner cannot
  explain their own locking. Do not advance: S6's remaining lessons and the
  capstone lean on exactly this concurrent-correctness reasoning.

## Remediation ladder

1. "Run `go test -race -run TestExpiry`. Read the failure aloud — what did
   the test do to the clock, and what did your cache answer?"
2. "You need O(1) lookup *and* a recency order. Which S2 structure gives
   each? How do you keep a map entry and a list element pointing at the
   same thing?" (Map of key → list element; the element holds the entry.)
3. "For `GetOrLoad`: what marks 'a load for key K is in progress', where
   does it live, and how does a second caller *wait* on it without
   spinning?" (A per-key struct in a map; block on a channel the owner
   closes.)
4. Sketch the three-phase shape verbally — locked check (cache, then
   flights), unlocked loader call, locked publish (set on success, delete
   flight, close done) — and let them write it. Do not show the solution
   file.

## After passing

Preview: "Your `GetOrLoad` made concurrent readers cooperate inside one
process. Next lesson decouples work across processes — message queues, acks,
and what at-least-once delivery really commits you to."
