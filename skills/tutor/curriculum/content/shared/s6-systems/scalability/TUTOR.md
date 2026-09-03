# Tutor notes — Scalability Patterns

## Where the learner is

Late S6, after distributed-systems. They already have the vocabulary of
replication, partitioning, CAP/PACELC, and partial failure — this lesson makes
four of those abstractions concrete enough to compile. From the Go path they
bring mutexes, generics, `-race`, and injected clocks (caching,
message-queues), and from S5 the habit of measuring instead of guessing.

This is the stage's biggest exercise: four independent files, four objectives.
Expect the ring to eat most of the time and `app.go` to take three minutes.
That asymmetry is worth naming out loud — the smallest diff in the lesson
(deleting a map) is the one that unlocks horizontal scaling at all.

In `guided` mode, work file by file in the LESSON.md order and keep the theory
attached: after each green test, ask what production failure it just prevented.

## Common misconceptions

- **"Stateless means the service stores nothing."** It means no *instance*
  holds state a later request depends on. State moves to a shared store; it
  does not evaporate. Learners who cannot say where the state went have
  memorised a slogan.
- **"Sticky sessions solve it."** They solve routing and re-introduce every
  problem: deploys log users out, an instance death takes state with it, load
  cannot be rebalanced, scale-in is destructive. Sticky sessions are a
  *performance* optimisation (cache locality), never a correctness mechanism.
- **"Rate limiting = capacity planning."** A limiter enforces a policy you
  derived from measured capacity. Ask where their number came from; "1000
  seems reasonable" is the wrong answer.
- **"429 and 503 are interchangeable."** 429 = you asked too often, the
  system is fine. 503 = the system cannot serve right now. 403 = never
  allowed. Clients (and dashboards) behave differently on each.
- **"A bigger queue absorbs the spike."** A queue converts overload into
  latency (Little's Law) and, if unbounded, into an OOM. Bigger buffers make
  the failure later and worse. The exercise's 100-vs-1000-tick comparison is
  the demonstration — make them read those two numbers aloud.
- **"Shedding load is losing requests."** It is *choosing* which requests to
  lose, quickly, instead of failing all of them slowly. Push on which work
  they would shed first and what must never be shed (health checks).
- **"Consistent hashing balances load."** It balances *keys*, assuming keys
  are roughly equal in traffic. A hot key defeats it entirely — that needs key
  splitting or replication, not a better ring.
- **"More virtual nodes is strictly better."** Better balance, but bigger ring
  state, slower lookups, and slower membership changes. 100-200 is the usual
  compromise; the knob has a cost.
- **"The ring makes the shard durable."** It reassigns *ownership* in
  milliseconds; anything that lived only in the dead worker's memory is gone.
  Ownership ≠ replication.
- **"Refill needs a background timer."** Lazy refill on access is simpler,
  race-free, and testable. If they wrote a goroutine with a ticker, ask how
  they would test it deterministically.

## Grilling points

- "Point at the exact line in the starter `app.go` that prevents horizontal
  scaling, and explain what a user sees when it's there."
- "Your fleet is behind a round-robin balancer with sticky sessions on. I
  deploy. Walk me through what your users experience, minute by minute."
- "You store sessions in Redis now. What did you just make load-bearing, and
  what happens to logins when Redis is unreachable?"
- "Your bucket is capacity 20, rate 5/s. Client sends 20 instantly, then 5/s
  forever — allowed? Then it idles 10 minutes and sends 100 at once — how many
  get through, and why not 3,000?"
- "Why must `RetryAfter` round *up*? What does a client learn from a header
  that lies by one millisecond?"
- "Ten instances, each enforcing 100 req/s per tenant. What limit is the
  tenant actually getting, and what are your three options for fixing it?"
  (Shared counter; divide by fleet size; route by consistent hash.)
- "Arrivals 10/tick, service 8/tick, unbounded queue. Give me the backlog and
  the wait time at tick 100 and tick 1000 — then tell me which SLO broke
  first." (Little's Law arithmetic out loud.)
- "You bounded the queue to 50. Where did that 50 come from?" (Target latency
  ÷ service rate — a queue bound is a latency budget in disguise.)
- "Show me your `Take`. If it ends in `q.items = q.items[1:]`, what does
  `Bounded[*Job]` still hold onto after a job is served?" (S2's trap: the
  backing array keeps taken elements reachable, so the one type that promises
  never to grow quietly does. A ring buffer, or a head index that zeroes the
  slot it leaves, fixes it.)
- "Five nodes to six with `hash mod N`: what fraction of keys move? With the
  ring? Your test measured both — quote the numbers."
- "Node n3 dies with 200 virtual nodes. Who picks up its keys, and how is that
  different from a ring with one point per node?"
- "One tenant is 30% of your traffic. Does the ring help? What actually does?"
- "Which of the four exercise files would you reach for first if a dependency
  suddenly got 5× slower, and why?" (Concurrency limit/queue — the limiter
  protects against clients, not against yourself.)

## Grading rubric

Exercise-specific; all four files must pass under `-race`.

- **A** — All tests green. `App` has *no* session field (not merely unused);
  refill is lazy, capped, and shares one helper with `RetryAfter`; the ring
  keeps positions sorted and uses binary search with an explicit wrap, and
  `RemoveNode` cleans both `hashes` and `owner`. `SimulateLoad` reproduces the
  expected tallies without special-casing the test inputs. The learner
  volunteers the theory: Little's Law on the backlog numbers, why keys move
  only onto the new node, why 429 differs from 503.
- **B** — Tests pass with a defensible-but-unexplained wart: a linear scan in
  `Get` instead of binary search, refill duplicated in `Allow` and
  `RetryAfter`, `RemoveNode` leaking `owner` entries, `Take` implemented by
  copying the whole slice each call or by reslicing the front away. Theory solid on statelessness and rate
  limiting, hand-wavy on rebalancing arithmetic.
- **C** — Green only after heavy hinting, or the ring was tuned against the
  test thresholds without understanding virtual nodes. Cannot explain why the
  unbounded backlog grows linearly. Time-box remediation on the queue
  simulation and the mod-N contrast before passing.
- **Fail** — Race-detector failures, red tests, sticky sessions defended as a
  correct design, "make the queue bigger" as the overload answer, or an
  inability to explain their own ring lookup. Do not advance: reliability,
  architecture, and the capstone all assume this reasoning.

## Remediation ladder

1. "Run one test at a time: `go test -race -run TestAnyInstance`. Read the
   failure message aloud — which instance stored what, and who was asked?"
2. Targeted questions, not answers:
   - `app.go`: "Which field in `App` does instance 2 not have a copy of?"
   - `limiter.go`: "How many tokens should exist after 500 ms at 1 token/s?
     Where is that fraction stored between calls?"
   - `queue.go`: "Write the tick loop as three sentences in English first —
     arrivals, sample, service. Which sentence increments `Shed`?"
   - `ring.go`: "You have a sorted `[]uint64` and a key hash. Which standard
     library call finds the first entry ≥ it, and what does it return when
     there is none?"
3. Structural hints:
   - "Both `Allow` and `RetryAfter` need current tokens. Extract one `refill`
     method that both call first."
   - "`MaxQueue` is sampled *after* arrivals and *before* service — check the
     order of your three steps against the doc comment."
   - "Virtual nodes come from hashing distinct labels per node. What string
     would you hash for node `n3`'s 7th position, and how does `RemoveNode`
     find them all again later?"
4. Verbal walkthrough of the shape only — for the ring: hash labels, record
   `position → owner`, keep positions sorted, binary search, wrap to index 0
   — and let them type it. Never open `solutions/go/`.

If they are stuck for time, prioritise `app.go` and `queue.go`: statelessness
and backpressure are what the remaining S6 lessons build on.

## After passing

Preview: "You can now add capacity and shed what exceeds it. Next lesson asks
the question underneath all of it — how reliable does this need to be, how do
you know, and what do you do at 3 a.m. when it isn't: SLOs, error budgets,
alerting, and graceful degradation."
