# Tutor notes — Data Storage at Scale

## Where the learner is

Fourth lesson of S6, third discussion-verify (design-intro set the
habits, api-design set the Ledgerly promises — hold them to both).
They have *operated* Postgres: S5 gave them pgx, migrations, pools,
transactions, and they profiled real queries. So single-node mechanics
are cheap; what is new is storage as a *design space* — store
selection, index economics, replication semantics, sharding. Verify is
discussion: grade from the filled `storage-design.md` and the review
conversation. Grade the **habits** — numbers with stated assumptions,
named trade-offs, promises traced from API to storage — never trivia
recall.

## Review protocol

Run it as a design review, not a quiz. Read the whole worksheet
silently first and note the weakest section — start there.

1. **Estimates audit (section 0).** Recompute one number live.
   Reference arithmetic: 20M charges/day ≈ 230 writes/s avg; ×4 peak
   ≈ ~930/s. Reads ×10 ≈ 2,300/s avg, ~9,300/s peak. Fraud: 3 lookups
   per charge ≈ 700/s avg, ~2,800/s peak. Storage: 20 GB/day ≈
   7 TB/year table, ~11 TB/year with indexes. The kicker: current
   ~9 TB + ~4.5 TB indexes ≈ 13.5 TB — the node is **already far
   outside** the 2-4 TB comfort zone; "months until" is ~zero. Dominant axis is
   **data size**, not QPS (930 writes/s and 9.3k reads/s both fit the
   anchors). Idempotency keys: 20M × 1 KB ≈ 20 GB live — trivial, and
   *saying* it is trivial is the point. A learner who computes their
   way to "we are already over; that is why this project exists" has
   the estimation habit; one who reports a comfortable runway did not
   use the brief's current-size number.
2. **Store-choice defense (section 1).** Expected: ledger relational
   (multi-row refund/charge invariant + support's ad-hoc queries;
   money); fraud features key-value (exact-key reads, 10 ms p99,
   5-min staleness tolerance — nothing needs joins or transactions);
   idempotency keys **in the same relational store as charges**
   despite the TTL-KV-shaped workload — the api-design crash story
   requires key-and-charge in one atomic commit. If they put keys in
   a separate KV, replay their own crash story: charge committed in
   the DB, crash before the KV write (or vice versa) — walk both
   orderings, break both. TTL cleanup in Postgres: background delete
   or dropping daily partitions. Counterfactual tests: "fraud now
   needs zero staleness and joins across merchant history — does the
   KV survive?" (no — it becomes a relational read replica problem);
   "support never queries the ledger ad hoc and refunds move to a
   separate service — what loosens?".
3. **Index audit + drills (section 2).** Expected index set:
   PK on id (Q1), `(merchant_id, created_at, id)` (Q2/Q3, and it is
   the cursor's spine — requirement 2), `(external_ref)` (Q4), and a
   Q5 answer from Drill B. Write bill: ~3-4 index writes per insert
   at ~930/s peak — acceptable, but they must *say* it. Probe the
   leftmost-prefix rule: "platform-wide charges in the last hour —
   which of your indexes serves it?" (none; merchant-first ordering
   scatters time).
4. **Replication probe (section 3).** The refund story must be
   narrated fluently: write → leader, dashboard read → replica 2 s
   behind, refund invisible, support clicks again — and *their own*
   api-design idempotency key is what prevents the double refund.
   Fixes accepted: pin the acting session's reads to the leader for a
   short window, or return a log position with the write and wait for
   the replica to reach it. "Refresh the page harder" is not a
   mechanism. Then the failover attack (requirement 1): async leader
   dies with unshipped commits → acknowledged charges *and their
   idempotency keys* vanish → merchant retry re-executes → promise
   broken by a durability setting. Expected repair: semi-sync on the
   commit path (ack after ≥1 follower has it; cost ~0.5 ms in-DC per
   commit, and ≥2 followers so one can be down). "Accept and
   reconcile against processor logs" passes only if argued with the
   brief's word "non-negotiable" acknowledged and rejected reasons
   given.
5. **Sharding stress-test (section 4).** The A-grade shape: data size
   triggers action *now*, but requirement 4 (99% of reads in 90 days,
   cold data may take 24 h) means the first move is **time
   partitioning + archive** — hot set ≈ 20 GB/day × 90 ≈ 1.8 TB
   (~2.7 TB with indexes), back inside comfort — and sharding is
   *deferred*, with the plan written now. Sharding straight away is
   defensible only if they price what archiving would have bought.
   Key: `merchant_id` keeps Q2/Q3/Q5 and the refund-charge invariant
   single-shard. Probe cross-shard: Q1/Q4 arrive with no merchant id
   — expected: embed a shard hint in the charge id (their api-design
   ids were opaque — connect it) or a lookup directory. **The whale
   probe rewards arithmetic over reflex**: 30% of 930/s ≈ 280
   writes/s on one shard — under the 5,000/s anchor, so today the
   whale is *fine*; the A-answer computes that, says "monitor, and
   here is the plan for 10× growth" (dedicated shard, or split the
   whale by compound key accepting fan-out on their own list). Panic
   without numbers is the C-signal. Resharding: mod-N rejected with
   the reshuffle argument; expected: fixed logical partitions moved
   whole (or range splits). Last probe: "you split the whale across
   two shards — what happens to their reconciliation cursor?" (the
   walk now merges two shards; the cursor must carry a position per
   sub-shard or the merge re-sorts — noticing this is an A-signal).
6. **Trade-off log (section 5).** Reject straw men ("chose a database
   over flat files"). Each entry must name a loss actually accepted —
   e.g. "semi-sync over async costs commit latency and a
   failover-availability edge", "merchant_id sharding over hash(id)
   costs whale exposure".

## Reference answers — plan drills

**Drill A.** Seq scan reads all ~9 B rows keeping 31.6 M
(`Rows Removed by Filter: 8968415898`), then a top-N sort finds the
newest 100 (`Sort Key: created_at DESC, id DESC`). Fix:
`(merchant_id, created_at, id)`. After: Index Scan (Backward) on that
index, **no Sort node** — the index yields rows already in order, so
the Limit stops after ~100 leaf entries. If they add the index but
cannot explain why the Sort disappeared, the B-tree section did not
land — the ordered-leaf-chain point is the one to re-teach.

**Drill B.** The index narrows to merchant + 7 days (~184 k rows) but
`status` is not in it: `Filter: (status = 'declined')` discards
183,917 rows to keep 50 — the index answered the wrong question. Two
fixes: (a) composite `(merchant_id, status, created_at)` — serves Q5
exactly; costs a fourth full index maintained on every insert;
(b) partial index on `(merchant_id, created_at)` `WHERE status =
'declined'` — tiny (declines are a small fraction), near-zero write
cost for non-declined rows, but serves only this shape. Either ships
with reasoning; the partial index is the sharper answer. Bonus probe:
"why is `Rows Removed by Filter` the line you scan for first in any
plan?" (it is the gap between work done and work needed.)

## Common misconceptions

- **"NoSQL scales, SQL doesn't."** The numbers say otherwise here:
  930 writes/s is a fifth of one node's anchor. The binding
  constraint is data size and ops, and NoSQL's price — no multi-row
  transactions, no ad-hoc queries — lands exactly on the ledger's
  requirements. Make them price the trade, not recite it.
- **"Indexes are free speed."** Every index taxes every insert and
  eats space (the brief's "half again"). The worksheet asks for an
  index they *refused* — a learner with no answer there is
  index-hoarding.
- **"The plan used an index, so it's optimal."** Drill B exists for
  this. `Rows Removed by Filter` after an `Index Cond` is the tell.
- **"Replication scales writes."** Every replica applies every write;
  replication buys reads and availability. Sharding is the only
  write-scaling move on the menu.
- **"Replicas are equivalent for reads."** Lag makes reads
  time-travel — read-your-writes (the refund story) and monotonic
  reads. Reads must be *routed*, not load-balanced blindly.
- **"Failover is an ops detail."** Async failover silently voids
  requirement 1 — durability settings are API contract. This is the
  lesson's deepest cross-lesson tie; do not let it stay abstract.
- **"Shard by hash(id) — even spread, done."** Even spread of *data*,
  and every merchant list query becomes scatter-gather. Key choice is
  a query-locality decision first, a balance decision second.
- **"Hot key → immediate crisis."** 280 writes/s is not a crisis; run
  the number, then decide. Reflex without arithmetic fails the
  design-intro habit.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go
deeper):

- "Your index on 9 B rows is how many levels deep? Why does a lookup
  touch 4 pages and not 30?" (Fan-out of hundreds per page —
  log base ~500.)
- "The composite index is `(merchant_id, created_at, id)`. Which of
  these does it serve: a merchant's last week? all merchants' last
  hour? Why?" (Leftmost prefix.)
- "Walk both orderings of 'write idempotency key to a separate KV'
  and 'commit the charge'. Break each with one crash." (From
  api-design, now in storage vocabulary — they should recognize it.)
- "Replica lag hits 30 s during a flash sale. What does the nightly
  reconciliation walk see — is requirement 2 violated?" (No: the
  cursor names a position; the walk just does not see the newest
  rows yet. Bounded staleness is fine for a nightly job — but they
  must reason it, not assert it.)
- "A merchant disputes a 4-year-old charge. Access path and latency
  promise under your archive design?" (Requirement 4's 24 h window is
  the escape — did they use it?)
- "When would multi-leader enter this design, and what new problem
  would you be signing up for?" (Multi-region writes; conflicts —
  out of scope by the brief, and *saying* it is out of scope is the
  right answer.)

## Grading rubric

- **A** — Estimates land within an order of magnitude with
  assumptions unprompted, including the "already over comfort"
  realization; all three store choices argued by the drivers, with
  the idempotency-keys atomicity trap caught unaided; both drills
  diagnosed with the correct line quoted and the Sort-node
  disappearance explained; refund story fixed with a named mechanism
  and the failover walk connects async loss to requirement 1 with a
  priced repair; sharding plan uses the 90-day archive insight, runs
  the whale arithmetic, and has a real resharding story; trade-offs
  name genuine losses.
- **B** — Sound design with gaps discussion closes: an estimate
  missing its assumption, keys placed in a KV but repaired quickly
  when the crash story is replayed, Drill B fixed with only one
  option considered, whale answered by reflex but corrected when
  asked to compute. Corrects cleanly under challenge.
- **C** — Worksheet filled but habits thin: adjectives where numbers
  belong, store choices asserted without losers, drills answered
  "add an index" without reading the plan, failover story breaks the
  design and the repair must be fed to them. Pass only if live
  remediation lands — make them redo section 0 and the failover walk
  on the spot; otherwise another iteration on the worksheet.
- **Fail** — Empty or copy-pasted sections; cannot explain why the
  Seq Scan in Drill A is slow; defends async replication for the
  charge path after the failover story is walked; or every
  "trade-off" is a straw man. Redo the relevant sections together
  before re-review.

## Remediation ladder

1. "Point at the number in the brief your weakest section should have
   used. Now use it — out loud, rounding as hard as you like."
2. "Draw a 3-level tree: root page, internal pages, leaf pages, 100
   keys per page. How many rows can it hold? Now find one key —
   count the page reads. Now read 50 keys in order — where are they?"
3. "Timeline on paper: leader and one replica as two lines. Mark the
   refund write on the leader, the replication delay, and the
   dashboard read. Where must the read land to see the refund? Now
   mark the leader's crash before the ship — what exists on the
   surviving line?"
4. "Ten keys, `hash mod 3`, three nodes — write down which node owns
   each key. Add a fourth node, recompute. How many keys moved? Now
   put the ten keys into ten fixed partitions and move one partition
   instead. That difference is the whole resharding story."

## After passing

Preview: "Most of the read traffic you just provisioned replicas for
never needs to touch the database at all — next lesson is caching:
the layer that intercepts it, and the invalidation bill that comes
with it."
