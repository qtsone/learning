# Design Case Studies

> `shared.systems.case-studies` · ~3-4h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Design a URL shortener end-to-end — key generation, storage, redirect
  latency, analytics — and defend every component choice.
- Design a chat system covering connection handling, message fan-out,
  ordering, and offline delivery.
- Apply the standard design structure (requirements, estimation, high-level
  design, deep dive, bottlenecks) without being prompted for the next phase.
- Name the scaling bottleneck in a design and propose the next evolution step
  when load grows 10×.

## Two designs, one method

Every lesson in this stage handed you a mechanism in isolation: a cache, a
queue, a replica set, a hash ring, an SLO, a hexagon. Real design is the
opposite exercise — a vague prompt arrives and *you* decide which mechanisms
the requirements pay for. This lesson walks two complete designs, then hands
you three prompts of your own.

Both walkthroughs follow the design-intro structure: requirements,
estimation, high-level design, deep dives, bottlenecks. Watch the structure do
work — in the shortener the estimate rules storage *out* as interesting, in
the chat system it rules connections *in*. From here on nobody prompts you
with "now do estimation"; running the phases yourself is part of the grade.

One warning, because worked examples are a dangerous way to learn: these are
*defensible* designs, not canonical ones, and the value is in the derivations,
not the diagrams. A memorized shortener is worth nothing — the moment a prompt
says "and links are editable", the memorized answer is actively wrong. Your
exercise briefs are built to punish transcription.

## Case study: a URL shortener

The prompt, in full: *"Build a link shortener — people paste a long URL and
get a short one that redirects."* That is the whole spec you get, and it is
typical.

### Requirements and scope

Functional, must-have: create a short link from a long URL and get it back;
follow a short link and land on the original; per-link click counts visible
to the creator. Nice-to-have, explicitly version two: custom aliases, expiry,
user accounts, QR codes, per-country breakdowns.

Non-functional, where the design actually comes from:

- **Read-heavy.** Assume ~100 redirects per created link.
- **Redirect latency** is the product: p99 ≤ 50 ms server-side. A shortener
  that adds 300 ms to every link in an email is a broken product.
- **Availability is asymmetric.** A failed redirect breaks links already
  printed on posters; a failed create is a retry. Four nines on the redirect
  path, three on create — different targets for different paths is legitimate
  and often skipped.
- **Durability.** A mapping is never lost and never silently changes.
- **Mappings are immutable and permanent** once created. Write that down: it
  is the assumption that makes caching trivial, and the exercise revokes it.

### Estimation

Assume 10M new links/day (stated assumption: a mid-size public service) and
the 100:1 read ratio above.

```text
writes:  10M/day / 10⁵ s        ≈ 100/s avg,   ×3 peak ≈ 300/s
reads:   1B/day  / 10⁵ s        ≈ 10,000/s avg, ×3 peak ≈ 30,000/s
record:  key 7 B + URL ~200 B + metadata ≈ 500 B
storage: 10M × 500 B = 5 GB/day ≈ 1.8 TB/year, ~10 TB at five years
egress:  30,000/s × ~500 B      ≈ 15 MB/s peak
```

Read that off the page: storage is unremarkable, bandwidth is a rounding
error, write QPS would fit on a laptop. **The dominant axis is read QPS at
very high availability** — 30k lookups/s that must not miss and must not be
slow — so every deep dive below is on the read path. A deep dive on the
storage engine would contradict your own arithmetic.

### Key generation

The one genuinely interesting write-path decision. Three real options:

**Random key + conditional insert.** Generate 7 random base62 characters,
insert only if absent, retry on conflict. The key space is 62⁷ ≈ 3.5 × 10¹²,
so at 3.65 × 10⁹ links/year it lasts centuries. Do the collision arithmetic
instead of hand-waving it: after one year, 3.65 × 10⁹ / 3.5 × 10¹² ≈ 1 insert
in 1,000 collides — at 100 writes/s, one every ten seconds — and after five
years it is 1 in 200. "Collisions never happen" is false, and it does not
matter: a compare-and-set insert plus a retry costs one extra round trip on
0.1-0.5% of writes, and every writer mints keys alone with no coordination.

**Counter + base62 encoding.** Encode a monotonically increasing integer: no
collisions by construction, shortest possible keys. The cost is coordination —
a global counter is a singleton on your write path. The standard fix is to
hand out *ranges*: each writer claims a block of 10⁴ ids and encodes locally,
so allocation happens once per 10,000 writes and a writer holding a spare
block survives the allocator's outage. The remaining cost is real: sequential
keys are enumerable, so anyone can walk your whole link set and read your
daily volume off it. Run the counter through a reversible scramble (multiply
by an odd constant modulo 62⁷, or a format-preserving permutation) and it
stays bijective — collision-free but no longer walkable.

**Hash of the long URL, truncated.** Attractive because identical URLs
deduplicate for free. But truncation reintroduces collisions with birthday
behavior, so you need the conditional-insert machinery anyway — and dedupe is
usually *unwanted*: two customers shortening the same campaign URL expect
separate links with separate click counts.

**Pick:** random + conditional insert, because it needs no coordination and
the measured retry cost is negligible at 100 writes/s. **What flips it:** a
write rate two orders of magnitude higher (retries and the existence check
start to bite), or a hard requirement for the shortest possible keys, both of
which favor counter ranges.

**In Go:** the store's conflict error drives the retry, and `randomKey`
draws from `crypto/rand` — guessable keys would be a security bug, not a
style choice:

```go
func (s *Shortener) Create(ctx context.Context, long string) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		key, err := randomKey(7)
		if err != nil {
			return "", err
		}
		switch err := s.store.InsertIfAbsent(ctx, key, long); {
		case err == nil:
			return key, nil
		case errors.Is(err, ErrKeyExists):
			continue
		default:
			return "", err
		}
	}
	return "", errors.New("shortener: no free key in 5 attempts")
}
```

`InsertIfAbsent` is a port (architecture lesson) — a unique-index violation
in SQL or a conditional put in a KV store is an adapter detail. The bound on
the loop matters: unbounded retry turns a store outage into a spin.

### Storage and the read path

The redirect workload is a single-key lookup with no joins, no ranges, and no
transactions — which, per the data-storage lesson, means the access pattern
does not choose your database for you. A relational table keyed by the short
key and a KV store both serve it in ~1 ms; choose on what your team already
operates well.

Partitioning is the choice that matters. Hash-partition on the short key:
keys are random, so load spreads evenly and every redirect touches exactly one
partition. Note the trap in the counter design — range-partitioning sequential
keys puts every new write on the last partition, a hot spot built on purpose.
Writes go to one primary per partition and redirects read replicas; replica
lag is harmless, because a link one second old is a link nobody has shared
yet. Analytics tables live elsewhere: nothing that scans or aggregates belongs
in the store on the redirect path.

### Making the redirect fast

Budget the path a redirect actually takes: DNS, TLS handshake (often
resumed), load balancer, application, lookup, 302 response. The only part you
control at scale is the lookup, and you control it with the cache hierarchy
from the caching lesson.

Traffic is Zipf-shaped, so a small fraction of links takes most clicks. Cache
the hot set in memory — 40M mappings × 500 B ≈ 20 GB, a small cache tier — and
expect a hit rate above 95%. Because mappings are immutable, this cache has
**no invalidation problem at all**: entries cannot go stale, so eviction is
pure capacity management. That falls straight out of a requirement, and it is
the first thing you lose when the requirement changes.

Two failure modes worth pre-empting:

- **Miss storms from crawlers.** Random 7-character probes are all misses,
  and each becomes a database read. Negative-cache the 404s with a short TTL
  and a bounded size.
- **Cross-ocean latency.** One region means ~150 ms of round trip for half the
  planet — three times the entire latency budget. Replicate the hot map to
  edge locations and answer there; the cost is that a link created and shared
  within seconds may miss at the edge, so an edge miss must fall back to
  origin rather than answer 404.

**301 or 302?** A 301 (permanent) lets browsers and proxies cache the
redirect and stop asking you — cheaper and faster for the user. But you never
see the click again, so per-link counts die, and you can never repoint or
disable that link: the redirect now lives in machines you do not own. With
click analytics as a must-have, the pick is 302 (or 307), paying for every
click deliberately. **What flips it:** permanent links and no analytics
product — then 301 is free scaling.

### Analytics without taxing the redirect

The naive design writes a click row synchronously, putting your least
important requirement on your most important path: a slow analytics database
becomes a slow redirect.

Instead the redirect emits a click event to a queue and returns
(message-queues lesson). A consumer aggregates into per-link, per-hour
rollups; the dashboard reads rollups, never raw events. State the
consequences rather than discovering them:

- Delivery is at-least-once, so counts are approximate unless you deduplicate
  on an event id. Approximate is usually fine — agree the requirement as
  "within ~1%, up to 5 minutes stale" instead of leaving it implied.
- If the pipeline is down or saturated, the redirect **drops the event and
  succeeds anyway** — graceful degradation, in the reliability lesson's
  terms. The analytics SLO is lower, so analytics never fails a redirect.
- Unique-visitor counts do not need exactness either; probabilistic counters
  answer them in kilobytes instead of storing every visitor id.

### What breaks at 10×

At 300k redirects/s peak the application tier is stateless, so it scales by
adding boxes. The cache tier is next: shard it by key, and add a small
per-process cache in front for the *hot key* case, where one viral link would
otherwise aim all traffic at a single cache shard. Origins stay quiet behind
a 95%+ hit rate, and the write path at 3k/s is untouched. The remaining single
points are the edge fallback path and — in the counter design — the id
allocator, which is why local block buffers belonged in that design rather
than in a follow-up.

## Case study: a chat system

The prompt: *"Build a chat app — one-to-one and small group messaging, with
history."* The mechanisms are the same ones you know. The shape of the
problem is not, and one estimate is enough to show why.

### The estimate that reframes the problem

Assume 50M daily users, 20 messages sent each, 20% connected at any moment,
average 2.5 recipients per message (one-to-one dominates), ~300 B stored per
message.

```text
sends:       1B/day / 10⁵ s      ≈ 10,000/s avg, ×3 peak ≈ 30,000/s
deliveries:  sends × 2.5         ≈ 25,000/s avg, ~75,000/s peak
connections: 50M × 20%           ≈ 10M concurrent, held open for hours
storage:     1B × 300 B = 300 GB/day ≈ 100 TB/year before replication
```

Sends and deliveries are ordinary numbers — you have built services that
handle them. The line without precedent is **10M concurrent connections**,
with storage growth a real second axis. And the alternative prices itself:
polling for 1-second delivery from 10M clients is 10M requests/s, three
orders of magnitude above the send rate. That single line settles the
push-versus-poll debate before it starts.

### Holding millions of connections

Delivery within a second means reaching the client unprompted, so connections
are long-lived: WebSocket over TLS, or HTTP/2 and gRPC streams (networking
lesson). That breaks the statelessness rule from the scalability lesson — a
connection *is* state, pinned to one machine — so confine the damage to one
thin tier:

- **Gateway tier**: terminates TLS, owns sockets, runs no business logic.
  Sizing: 100k-250k idle connections per tuned box, so 10M connections is
  ~50 boxes plus headroom, and memory per connection is the binding limit.
- **Chat service tier**: stateless, holds the logic, scales independently.
- **Session registry**: user → gateway, so a message can find its recipient.
  Written on connect, deleted on disconnect, expiring on a heartbeat TTL so a
  crashed gateway's entries clean themselves up. The alternative — a broker
  topic per user that gateways subscribe to — removes the lookup and pays in
  subscription churn.

Operational vocabulary comes with the territory. Application heartbeats every
20-30 s detect half-open sockets TCP will not report, and keep NAT and
load-balancer idle timers from dropping them silently. Balance by *least
connections*, never round-robin: connections last hours, so what matters is
where they accumulated. And every deploy disconnects everyone a gateway held —
drain slowly and require jittered client backoff, or the deploy becomes a
self-inflicted reconnect storm (reliability lesson).

**In Go:** one goroutine per connection with a buffered outbound channel,
where the `default` case is the whole point — a slow client must never block
the fan-out serving everyone else:

```go
func (g *Gateway) deliver(userID string, msg []byte) {
	g.mu.RLock()
	sessions := g.sessions[userID] // one entry per connected device
	g.mu.RUnlock()

	for _, s := range sessions {
		select {
		case s.out <- msg:
		default:
			s.close() // the client resyncs from its cursor on reconnect
		}
	}
}
```

Dropping a laggard is safe *only* because the durable log is the source of
truth and the client can replay from its cursor — the same reasoning as
your S3 worker-pool backpressure, applied to a socket.

### Fan-out

A send does three things, and their order is the design: append to the
conversation's durable log (the commit point — after it, the message exists),
push to each recipient's live sessions, leave the rest to the offline path.
Delivery is best-effort; the log is truth. Getting that backwards — treating
the push as the commit — is how messages get lost.

Then the dichotomy: **fan-out on write** versus **fan-out on read**. Writing a
copy into each recipient's inbox makes reads trivial and costs
recipients-per-message writes; one shared log per conversation makes writes
trivial and costs a read per participant. For one-to-one and small groups
either is cheap. For a 100k-member broadcast channel, fan-out on write means
100k writes for one send — the classic celebrity problem — so large channels
read the shared log instead. A hybrid that switches on conversation size is a
legitimate design, and should be stated as a decision rather than left
implicit.

### Ordering

The requirement is narrower than it sounds: every participant must see *one
conversation* in the same order. Nobody needs a global order across all
conversations, and that difference is worth a great deal of infrastructure —
a global total order needs consensus (distributed-systems lesson), while a
per-conversation order needs one authority per conversation.

So partition by **conversation id**, not user id, and let the partition owner
assign a monotonically increasing per-conversation sequence number. Clients
sort by sequence number and nothing else. Two things fall out:

- **Client and server clocks are unusable as an ordering key.** Device clocks
  are skewed, and an offline send arrives with a timestamp minutes in the
  past. Timestamps are for display; sequence numbers are for order.
- **Gaps are detectable.** A client that holds 41 and 43 knows it missed 42
  and asks for the range — which is why messages may arrive over any path in
  any order without the design caring.

Editing and reply threading reference message *ids*, not positions, so
"reply to 42" survives whatever renumbering the UI does.

### Offline delivery and the acknowledgement ladder

Once the log is durable and ordered, offline delivery stops being a queueing
problem and becomes a **sync** problem. Each device stores a cursor — the
last sequence number it holds per conversation — and on reconnect sends its
cursors and receives everything after them. No per-user queue to grow, no
message stuck in a broker for a user who reinstalled.

The details that separate a working design from a plausible one:

- **Duplicates are guaranteed.** A lost acknowledgement causes redelivery.
  The sender attaches a client-generated message id, which makes the send
  idempotent (api-design lesson) and lets devices dedupe on render.
- **Multiple devices per user.** Delivery is per session, cursors are per
  device, read state is per user. Conflating those three is why "read on my
  phone, still bold on my laptop" bugs exist.
- **Four distinct acknowledgements** — accepted, durably stored, delivered to
  a device, read by a human — are four facts with four owners. Product ticks
  map onto them; do not collapse them into one flag.
- **Bounded catch-up.** A user returning after six months must not receive six
  months of messages in one stream: serve the last N per conversation and
  paginate history on demand.
- **Truly offline devices** get a push notification from a third-party
  service — best-effort, unordered, never on the critical path, never
  carrying content you would not want on a lock screen.

### What breaks at 10×

100M connections is 500+ gateway boxes, which makes the session registry the
busiest component in the system: shard it by user id and keep in it nothing
that reconnects cannot rebuild. Storage at 1 PB/year needs a tiering plan —
recent messages hot, older ones in cheap archival storage with slower reads —
decided before the disks decide for you. And the hot partition is now the huge
group chat, one conversation whose sequence authority is a single owner. That
limit is built into the ordering choice; naming it is part of defending it.

## The 10× drill

"What breaks first at 10×?" has a repeatable procedure. Walk your own diagram
looking for five things:

1. **The singleton** — one allocator, one leader, one primary, one cron: a
   bottleneck and an availability risk in the same box.
2. **The hot key or partition** — the viral link, the celebrity account, the
   huge group. Uniform-load assumptions die here first.
3. **The unbounded thing** — a backlog, a missing retention policy, per-user
   state that only grows.
4. **The synchronous fan-out** — request-path work proportional to
   recipients, followers, or shards.
5. **The coordination point** — a lock, a consensus round, a cross-shard
   transaction; coordination costs grow superlinearly with participants.

Then finish the answer: the **first symptom** you would see (which dashboard
line moves — your S5 observability), the **next evolution step**, and what
that step **costs**. "Add a cache" is not an answer; "p99 on the lookup path
rises while origin CPU saturates, so we shard the cache by key and accept a
colder start after deploys" is.

## How designs fail in review

- **Transcription** — a memorized design recited over a prompt it does not
  fit. The tell is a component nobody can tie to a requirement.
- **Ignoring the twist** — every real prompt has a clause that invalidates the
  textbook answer (editable links, a residency rule, a retention period), and
  reviewers put it there on purpose.
- **Numbers after the fact** — estimation performed to justify a design
  already drawn. It shows, because the estimate never changes anything.
- **Reflex components** — "add a cache", "put it on a queue", "shard it", with
  no hit rate, no depth, no key.
- **Silence about what was cut** — no out-of-scope list and no known limits
  reads as an author who has not looked for them.

## Exercise

Open [`exercise/`](exercise/). Three prompts, each ending in a written design
statement your tutor will grill. Read [`brief.md`](exercise/brief.md) first —
it holds Briefs A and B and their numbers, and each one twists this lesson's
walkthrough on purpose. The speed-round brief stays sealed in
[`brief-c.md`](exercise/brief-c.md) until you start the timer.

Acceptance criteria:

1. `01-shortlink.md` (Trackly, a campaign-link service): all five phases with
   visible arithmetic and stated assumptions; a key-generation decision that
   also handles reserved and custom aliases; a cache and edge plan that
   survives editable links and a 60-second global takedown rule; a
   redirect-status decision; a near-real-time analytics path.
2. `02-chat.md` (Huddle, a support-desk chat): connection-tier sizing from
   your own numbers; agent routing; the ordering scheme across transfers;
   multi-device and supervisor fan-out; reconnect behavior on flaky mobile
   networks; the seven-year retention and residency plan.
3. `03-speed-round.md` (a social feed): the full structure in 30 timed
   minutes with no scaffolding beyond phase headings.
4. Each sheet ends in a design statement (≤ 200 words; ≤ 150 for the speed
   round), names its dominant axis, carries at least two framed trade-offs
   with flip conditions, and answers 10× with the drill above — first
   symptom, next step, cost.

Nothing compiles. When the worksheets are complete, tell your tutor: the
review is a design interview, and your constraints will change during it.

## Further reading

- [Cloudflare — How we built Cloudflare Workers KV](https://blog.cloudflare.com/building-with-workers-kv/)
  — edge key-value reads, the mechanism behind the shortener's edge tier.
- [Slack Engineering — Real-time messaging](https://slack.engineering/real-time-messaging/)
  — connection tier, presence, and reconnect storms at production scale.
- [Discord — How Discord stores trillions of messages](https://discord.com/blog/how-discord-stores-trillions-of-messages)
  — partitioning and retention for exactly the chat storage axis above.
- [Designing Data-Intensive Applications](https://dataintensive.net/) —
  chapter 12 ("The Future of Data Systems") pairs with these end-to-end
  compositions.
