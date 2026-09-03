# Data Storage at Scale

> `shared.systems.data-storage` · ~3h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Choose between SQL and NoSQL stores for a given access pattern and
  justify the choice by consistency, query flexibility, and scaling
  needs.
- Explain how a B-tree index accelerates reads, what it costs on writes,
  and read a query plan to spot a missing index.
- Contrast leader-follower and multi-leader replication, and explain
  replication lag and its impact on read-your-writes.
- Design a sharding scheme (key choice, resharding strategy) and
  identify hot-key and cross-shard-query hazards.

## The database is where designs go to be honest

In S5 you ran Postgres in production: migrations, pools, transactions.
One node, one truth. The api-design lesson then made promises against
it — a cursor that never skips a transaction, an idempotency key that
makes retries harmless — while treating the storage underneath as a
given. This lesson removes that assumption.

At scale the storage layer stops being one box. You will choose *which
kind* of store fits each dataset, pay for reads with writes via
indexes, copy the data (replication) and then live with the copies
disagreeing for a moment, and eventually split the data itself
(sharding) and lose things you took for granted — like a transaction
that can see all of it. Every promise your API made upstairs is either
kept or broken down here.

The design-intro discipline is unchanged and non-negotiable: numbers
before architecture, every number with its assumption, every choice
with a named loser. Storage decisions are the most expensive ones to
reverse — data has inertia; code redeploys in minutes, a 50 TB
migration takes months.

## SQL or NoSQL: decide by access pattern

Like REST-vs-gRPC, this is not a religious question. It is three
practical ones:

1. **What consistency do you need?** Relational stores give you
   multi-row ACID transactions: debit one row, credit another, both or
   neither — the S5 guarantee your ledger code leaned on. Most NoSQL
   stores confine atomicity to a single key or document. If an
   invariant spans records, that is a strong pull toward SQL — or a
   redesign so it doesn't.
2. **How flexible must queries be?** SQL answers questions you had not
   thought of at design time: ad-hoc filters, joins, aggregates —
   support and analytics live on this. NoSQL stores ask you to know
   your access paths *up front* and shape (denormalize) the data
   around them; off-path queries range from expensive to impossible.
3. **How must it scale?** NoSQL systems were mostly born distributed:
   partitioning and replication are built in, and what they took out
   (joins, cross-key transactions) is exactly what is hard to
   distribute. A relational store scales reads easily (replication,
   below) but scaling *writes or data size* past one node is manual
   surgery (sharding, below).

The main NoSQL families, by the access pattern they serve:

| Family | Access pattern | Typical use |
|---|---|---|
| Key-value | get/put by exact key | sessions, feature flags, counters |
| Document | get/put a nested record by id, some secondary queries | user profiles, catalogs |
| Wide-column | rows × dynamic columns, range scans within a partition | time series, feeds at massive write rates |
| Graph | traverse relationships | social graphs, fraud rings |

A defensible default: **relational until an access pattern breaks
it** — and then move *that dataset*, not the whole system. The
break is specific and demonstrable: a write rate no single node
sustains, a latency budget joins cannot meet, data shaped so key-value
access is all you will ever need. "NoSQL because scale" without a
number is the storage version of "high traffic" — the design-intro
lesson taught you what that is worth.

One trap deserves its own sentence: an access pattern can *look* like
one family while a consistency requirement binds it to another. Your
idempotency keys look exactly like a TTL key-value workload — but the
api-design crash story required the key record and the charge to
commit **atomically**, and atomic-with-the-charge means *same store as
the charge*. Consistency needs outrank workload shape.

**In Go:** nothing about this choice is language-specific — that is
the point — but you have lived one side of it: the `pgx` transaction
in your S5 service that wrote a charge and its idempotency key in
one commit is a multi-row invariant. Before moving any dataset to a
key-value API, find every transaction in your code that spans it and
something else: each one is a consistency requirement you would be
giving up.

## Indexes: buying reads with writes

A table without an index is a heap: finding one row means reading all
of them. An index is a second, sorted structure that trades write work
and space for read speed. The workhorse is the **B-tree**.

Picture the index as a tree of fixed-size pages. Each internal page
holds hundreds of sorted keys and pointers to child pages; leaf pages
hold the keys and row locations, and are chained left-to-right in
sorted order. Because each level fans out by hundreds, the tree is
*shallow*: hundreds-to-the-fourth is billions, so even a
multi-billion-row index is typically 3-4 levels deep. A lookup walks
root to leaf — three or four page reads instead of scanning
terabytes. And because leaves are sorted and chained, a **range scan**
(`created_at` in the last 7 days) is a walk along the leaf chain, and
the rows come out *already ordered* — an index can satisfy an
`ORDER BY`, deleting an entire sort step.

A **composite index** on `(merchant_id, created_at, id)` sorts by
merchant first, then time, then id — like a phone book sorted by last
name, then first. It serves any *leftmost prefix*: all rows for a
merchant; a merchant's rows in a time range, newest first. It does
**not** serve "all merchants in the last hour" — those rows are
scattered across every merchant's section. Column order is a design
decision, driven by the queries you committed to in your API.

What the read side buys, the write side pays:

- Every insert updates the table **and every index on it** — one
  logical write becomes several physical ones (write amplification).
- Pages fill and **split**, and index pages must be read to be
  updated — random I/O your append-only table never needed.
- Space: indexes on a busy table routinely add 30-50% to its size.

So the rule is not "index everything"; it is "index the access paths
you promised, and be able to say what each index costs at your write
rate".

### Reading a query plan

You do not guess whether an index is missing; you ask the database how
it intends to execute the query. In relational stores this is
`EXPLAIN` (`EXPLAIN ANALYZE` runs it and shows actuals):

```text
EXPLAIN ANALYZE
SELECT * FROM charges WHERE external_ref = 'ord-20260815-0042';

Seq Scan on charges  (cost=0.00..214327.60 rows=1 width=1024)
                     (actual time=812.44..3104.77 rows=1 loops=1)
  Filter: (external_ref = 'ord-20260815-0042'::text)
  Rows Removed by Filter: 11999999
Execution Time: 3104.812 ms
```

Read it bottom-up and look for four things:

1. **Scan type.** `Seq Scan` = read the whole table. Fine for tiny
   tables or queries that really need most rows; a red flag when the
   query wants one.
2. **`Rows Removed by Filter`.** The smoking gun: the executor read
   twelve million rows to keep one. That ratio — rows touched versus
   rows returned — is the cost of the missing index.
3. **Estimated vs actual rows.** The planner chooses strategies from
   statistics; when its estimate is off by orders of magnitude, it
   picks bad plans. Wildly wrong estimates are their own finding.
4. **Sort nodes.** An explicit `Sort` step means no index provided the
   order; on big result sets it spills to disk.

After `CREATE INDEX ON charges (external_ref)`:

```text
Index Scan using charges_external_ref_idx on charges
    (cost=0.56..8.58 rows=1 width=1024)
    (actual time=0.041..0.043 rows=1 loops=1)
  Index Cond: (external_ref = 'ord-20260815-0042'::text)
Execution Time: 0.071 ms
```

Three page reads down the tree: 3,104 ms to 0.07 ms — four orders of
magnitude, and *that* is why indexes are the first tool you reach for,
long before any architecture in the rest of this lesson. Beware the
subtler failure too: a plan that *uses* an index can still be wrong —
if `Index Cond` narrows to 200,000 rows and a `Filter` line discards
all but 50, the index answered the wrong question. The exercise makes
you diagnose exactly that.

## Replication: copies that disagree

One node is a single point of failure and a read-throughput ceiling.
Replication — keeping the same data on several nodes — buys you
availability (a replica can take over), read scaling (spread queries
across copies), and latency (a copy near the user; recall the
networking lesson's ~100 ms ocean crossing). What it costs you is the
subject of this section: the copies are not always equal.

**Leader-follower** is the default topology. All writes go to one
leader; it appends them to a replication log; followers apply the log
in order and serve reads. One decision defines the design: does the
leader wait?

- **Synchronous**: the leader confirms the write only after a follower
  has it. An acknowledged write survives the leader's death — but
  every write pays a network round trip, and a stalled follower stalls
  writes. Fully-sync-to-everyone is rare; **semi-sync** (wait for at
  least one follower) is the common durability compromise.
- **Asynchronous**: the leader confirms immediately and ships the log
  in the background. Fast, and a dead follower bothers nobody — but
  now followers *lag*, and an acknowledged write can still be lost.

**Replication lag** is the followers' distance behind the leader —
milliseconds normally, seconds-to-minutes under load. Lag turns into
user-visible anomalies:

- **Read-your-writes.** Support issues a refund (write → leader), the
  dashboard refreshes the list (read → follower, 2 s behind), the
  refund is not there. To the human, their own action vanished — so
  they click refund again. (Your api-design idempotency key is what
  stands between that second click and a double refund. Layers of the
  design protecting each other — or failing together.) Fixes: route a
  user's reads to the leader for a short window after their own
  write; or have the write return a log position and make follower
  reads wait until the replica has caught up to it.
- **Monotonic reads.** Two refreshes hit two differently-lagged
  followers and the second shows *older* data — time runs backwards.
  Fix: pin a session to one replica.

**Failover** is where async replication presents its bill. The leader
dies with 500 ms of writes not yet shipped; a follower is promoted;
those acknowledged writes are gone. For Ledgerly that is not an
abstract durability footnote: the vanished rows include charges *and
their idempotency keys*, so a merchant retry re-executes — the
"never charged twice" promise from api-design is now broken by a
storage setting. Durability configuration is API contract in disguise.
(Split-brain — the old leader comes back and thinks it still leads —
is the other failover monster; the distributed-systems lesson later
this stage takes consistency and consensus properly.)

**Multi-leader** lets several nodes accept writes — usually one
leader per region, replicating asynchronously to the others. You buy
local write latency in every region and survival of a whole region's
outage. You sign up for **write conflicts**: the same record written
in two regions concurrently, both accepted, discovered only when the
logs cross. Someone must resolve it: last-write-wins silently discards
one write (data loss as a policy); application-level merge rules are
real design work. The honest default: avoid multi-leader unless a
requirement — multi-region *write* latency, regional independence —
forces it, and then partition ownership so each record has one home
region and conflicts cannot happen in the common case.

The sentence that reframes the next section: **replication scales
reads, never writes** — every replica still applies every write. When
writes or sheer data size outgrow one node, copying is not enough.
You have to split.

**In Go:** replication makes read routing an *application* concern.
Your S5 service held one `pgxpool` for one DSN; with replicas there
are several pools, and "may this query read stale data?" becomes a
parameter of your storage layer's API — the refund status read after
a support action goes to the leader; the nightly reconciliation walk
is happy on a lagging replica. Design the seam now or grep for every
query later.

## Sharding: splitting the data itself

Sharding (partitioning) splits the dataset across nodes, each owning a
subset. It is the answer when a single node cannot hold the data or
sustain the writes — and it is deliberately *last* in this lesson,
because it is the step where familiar guarantees quietly stop:
transactions and indexes now live per-shard, and any query that spans
shards is your problem.

**The key decides everything.** The sharding key maps each row to its
shard, and the goal is two-sided: spread *load* evenly, while keeping
your *common queries* on a single shard.

- **Range sharding** — each shard owns a contiguous key range. Range
  scans stay local and efficient. Hazard: a monotonic key (time,
  sequential ids) sends every new write to the last shard — perfect
  distribution of data, total concentration of load.
- **Hash sharding** — shard by hash of the key. Even spread by
  construction; cross-key range locality gone.
- **Tenant keys** — for a multi-tenant system like Ledgerly,
  `merchant_id` is the natural key: every merchant-scoped query
  (their list, their reconciliation walk, their cursor) stays on one
  shard. The hazard has a name: the **whale**. One merchant doing 30%
  of platform traffic makes one shard hot no matter how fair the
  hash. Mitigations are all trade-offs: isolate whales on dedicated
  shards; split a whale's data by a compound key
  `(merchant_id, hash(charge_id))` and accept that *their* queries now
  fan out.

**Hot keys** generalize the whale: any key the workload fixates on — a
flash sale, a viral post — concentrates load the sharding scheme
cannot spread, because the scheme sees keys, not popularity. Detection
(per-key metrics) and mitigation are part of the design, not an
incident-day improvisation.

**Cross-shard queries.** Anything not aligned with the key must ask
many shards and merge — *scatter-gather*: latency is the slowest
shard's, and cost is paid on every shard. Worse, **cross-shard
transactions**: two rows on two shards can no longer change
atomically with one commit. The realistic menu: co-locate rows that
must change together (choose the key so invariants are single-shard);
or split the operation into steps that each commit locally and
reconcile the whole asynchronously — machinery a later lesson in this
stage builds properly. What you must not do is assume the S5
transaction still covers you; at the shard boundary it ends.

**Resharding.** Load grows; shards must move. The naive scheme —
`shard = hash(key) mod N` — is a trap: change N and nearly *every* key
changes shards; adding one node reshuffles the world. The standard
escape: fix a large number of **logical partitions** up front (say
4,096), assign each row to a partition by hash, and map partitions to
physical nodes. Growth then moves *whole partitions* — a bounded,
resumable copy — and the partition count is a decision you make on
day one, because changing *it* is the reshuffle you built all this to
avoid. Range-sharded systems reshard by splitting hot ranges instead.
(The scalability lesson later this stage adds a hashing scheme that
minimizes movement even further.)

**Escalate in order.** Sharding is powerful and expensive — pay for
it last:

1. One node, good indexes, measured queries (this lesson, top half).
2. Replicas for read scale and availability.
3. Partition *by time* and archive cold data — a 7-year ledger where
   99% of reads touch the last 90 days does not need 7 years on the
   hot path.
4. Shard, when writes or working-set size leave no choice — with the
   key chosen for your queries and the resharding story written down
   *before* shard one.

(Between 2 and 3 lives caching — the next lesson. Much of the read
traffic you are provisioning replicas for never needs a database at
all.)

## Exercise

Open [`exercise/`](exercise/). Ledgerly again, two years on — traffic
is 4× and the storage layer you assumed in api-design must now be
designed for real. `brief.md` has the numbers; `storage-design.md` is
the worksheet, including two query plans to diagnose.

Acceptance criteria:

1. Estimates first: write QPS, read QPS, storage growth, and time
   until a single node's comfort limits are exceeded — arithmetic
   visible, every number with its assumption, dominant axis named.
2. A store choice for each of the three datasets (ledger, fraud
   features, idempotency keys), argued by consistency, query
   flexibility, and scaling needs — each with the rejected alternative
   and its cost.
3. An index plan for the five committed queries, with the write-side
   cost stated, plus a correct diagnosis and fix for both query-plan
   drills.
4. A replication design: topology, sync/async with a stated lag
   expectation, the refund read-your-writes story resolved with a
   named mechanism, and the failover walk-through connected to the
   "never charged twice" requirement.
5. A sharding plan: the trigger computed from your own estimates, key
   choice with the whale analysis, two cross-shard queries named with
   mitigations, and a concrete resharding strategy.
6. A trade-off log with at least three entries of the form "chose X
   over Y; it costs us Z" — real losses, not straw men.

Nothing compiles. When the worksheet is full, bring it to your tutor —
the verification is a design review, and the failover story will be
attacked first.

## Further reading

- [Designing Data-Intensive Applications](https://dataintensive.net/)
  — chapters 3 (storage & indexes), 5 (replication), and 6
  (partitioning) are this lesson in book form.
- [Use The Index, Luke](https://use-the-index-luke.com/) — the
  canonical free guide to B-tree indexes and reading execution plans.
- [PostgreSQL — Using EXPLAIN](https://www.postgresql.org/docs/current/using-explain.html)
  — the official guide to query plans.
- [Herding elephants: sharding Postgres at Notion](https://www.notion.so/blog/sharding-postgres-at-notion)
  — a real resharding story: key choice, logical partitions, and the
  migration.
