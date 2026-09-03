# Distributed Systems

> `shared.systems.distributed-systems` · ~3-4h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- State the CAP theorem precisely and explain why the real choice during a
  partition is between consistency and availability.
- Rank consistency models (linearizable, sequential, causal, eventual) and
  pick the weakest model that still satisfies a given feature.
- Explain at a high level how consensus (Raft-style leader election and log
  replication) achieves agreement despite node failures.
- Enumerate distributed failure modes — partial failure, network partitions,
  clock skew, split brain — and describe a defense for each.
- Explain why "the network is reliable" and the other distributed-computing
  fallacies lead to real production bugs.

## The defining property: partial failure

On one machine, failure is honest. The process crashes, everything stops,
and when it restarts it reads a consistent disk. Painful, but simple.

Distribute the system over several machines and failure changes character:
*part* of the system fails while the rest keeps running — and, worse, the
survivors cannot tell what happened. You send a request and no reply comes
back. Four different worlds produce that identical observation:

1. The request was lost on the way there.
2. The remote node crashed before processing it.
3. The remote node processed it, slowly, and is still working.
4. It processed it fine and the *reply* was lost.

Nothing you can observe locally distinguishes them. A timeout is therefore
not failure *detection*; it is a *decision to stop waiting*, made in
ignorance. Every defense in this lesson grows from that one epistemic hole.

You have already met it. In the message-queues lesson, the broker redelivers
a message when the ack doesn't arrive in time — even if the consumer
actually processed it and only the ack was lost. That is world 4, and it is
exactly why the honest promise was *at-least-once*, never exactly-once, and
why your consumer had to deduplicate. Distributed systems theory is largely
that observation, taken seriously everywhere.

## The eight fallacies

In the 1990s, engineers at Sun wrote down the assumptions that newcomers to
distributed systems make and later pay for. All eight still bill hourly:

1. The network is reliable.
2. Latency is zero.
3. Bandwidth is infinite.
4. The network is secure.
5. Topology doesn't change.
6. There is one administrator.
7. Transport cost is zero.
8. The network is homogeneous.

The first two cause the most outages, so trace them to real bugs:

**"The network is reliable"** → your code treats a remote call like a
function call. A call times out during checkout; the client retries — the
obvious fix — and the customer is charged twice, because the first attempt
actually succeeded (world 4 again). The retry was correct; sending the same
*effect* twice was the bug. Defense: retries must be paired with
idempotency, which is why the api-design lesson made you design idempotency
keys before letting you retry anything.

**"Latency is zero"** → a service is built and tested in one datacenter,
where a round trip costs ~0.5 ms (your design-intro anchor table). It makes
30 sequential internal calls per request — 15 ms, fine. Then a second
region is added an ocean away at ~100 ms per round trip, and the same
request takes 3 seconds. Nothing "broke"; an assumption silently priced in
everywhere became false. Defense: know your round-trip budget, batch and
parallelize calls, and treat every network hop in a design as a cost you
can point at.

The remaining six follow the same pattern: an assumption that holds on one
machine (or in a demo) and fails at scale. When a design review asks "what
are you assuming about the network here?", this list is the checklist.

## Replication and the disagreement problem

The data-storage lesson gave you replication's *why*: survive machine loss,
serve reads near users, scale read traffic. Here is its *cost*: the moment
two copies exist, they can disagree. A write lands on the leader; for some
window the follower hasn't heard yet. Anyone reading the follower sees the
past. Cross an ocean and that window is at least one round trip — often
seconds under load.

So every replicated system must answer: **what are readers allowed to
observe while replicas disagree?** That answer is called a consistency
model. It is a contract with the application programmer, exactly like your
caching lesson's TTL was a contract about how stale a cache entry may be —
a consistency model is the same idea, made precise and system-wide.

## The consistency ladder

Four models cover most real conversations, ordered strongest to weakest.
Stronger models are easier to program against and more expensive to
provide — the strongest ones need coordination on every operation, the
weakest need almost none.

**Linearizable.** The system behaves as if there is a single copy of the
data, and every operation takes effect atomically at some instant between
its start and its finish. Once *any* client's write completes, *every*
later read — by anyone, on any node, "later" by real time — sees it. This
is what people naively expect of any storage system, and it is the most
expensive promise on the menu.

**Sequential.** Everyone observes the *same* order of operations, and each
client's own operations appear in the order that client issued them — but
that agreed order is no longer tied to real time. Node A's completed write
may be invisible to node B for a while, as long as nobody ever observes two
different orderings.

**Causal.** Only operations that are *related* stay ordered: if one write
could have influenced another (you read a question, then posted the
answer), everyone sees them in that order. Genuinely concurrent writes may
be seen in different orders by different readers. This is the strongest
model that stays available during partitions — which makes it a sweet spot.

**Eventual.** If writes stop, replicas converge to the same value,
eventually. Until then: no ordering promises at all. A reader may see new
data, then old data, then new again. "Eventual" is a convergence promise,
not a freshness promise — treat it as the *absence* of a consistency
guarantee, tolerable only where anomalies are cheap.

In practice you will also meet **session guarantees** — read-your-writes
("after I post, *I* see my post"), monotonic reads ("I never see time go
backwards") — which scope promises to one user's session. They are cheap,
causal-family guarantees, and often exactly what the feature needed.

The design skill this lesson drills: **pick the weakest model that still
satisfies the feature**, one feature at a time — never one model for the
whole system. Anomalies are feature-specific:

| Feature | Weakest sufficient model | One step weaker looks like… |
|---|---|---|
| Claiming a unique username | Linearizable | Two users both "win" the name |
| Reply threading in comments | Causal | An answer appears before its question |
| "My orders" after checkout | Read-your-writes | You buy, and your order list says empty |
| A like counter | Eventual | Count jitters ±3 for a minute — who cares |

Overshooting is not safety; it is spent latency and availability you could
have kept. Every strengthening must name the anomaly it prevents.

## CAP, precisely

CAP is the most misquoted theorem in the field, so get the words right.
Its three properties, as the proof defines them:

- **C — consistency**: linearizability, the top of the ladder above.
- **A — availability**: every request that reaches a *non-failed* node gets
  a non-error response — without waiting out the problem.
- **P — partition tolerance**: the system keeps operating even when the
  network splits nodes into groups that cannot talk to each other.

The theorem: no system can guarantee all three. The popular reading —
"pick any two" — is wrong, because **P is not a menu item**. Partitions are
an empirical fact of networks; a "CA system" is a system that has decided
partitions won't happen to it, which is a hope, not a property. The honest
statement is:

> **When** a partition happens, each operation must choose: refuse to
> answer (keep C, sacrifice A) or answer from what this side knows (keep
> A, sacrifice C). **When there is no partition, CAP constrains nothing.**

Two consequences worth internalizing. First, the choice is **per
operation, not per system**: your checkout can refuse writes during a
partition while your catalog keeps serving stale reads — same system, both
letters. Labeling whole databases "CP" or "AP" is shorthand at best.
Second, because CAP is silent in normal operation, it is a poor tool for
everyday design. The extension **PACELC** fills the gap: if **P**artition,
trade **A** vs **C** — **E**lse, trade **L**atency vs **C**onsistency.
The else-case is the one you live with daily: replicate synchronously
across regions (every write pays the ocean round trip, replicas stay
current) or asynchronously (fast writes, lag when you least want it). That
is the same trade-off as the partition one, paid in milliseconds instead
of outages — which is why systems that choose latency in the else-case
usually choose availability under partition too.

## Consensus: agreeing when nodes fail

Some facts must have exactly one value even while machines crash: who is
the leader replica? did order #4711 commit or not? which node owns this
shard? Getting a cluster to agree on such facts — despite crashes, lost
messages, and restarts — is the **consensus problem**. Raft is the
algorithm you will meet most often (etcd, and in spirit most modern
replicated stores). You need its intuition, not its implementation:

**A replicated log.** Nodes agree not on one value but on a *log* of
commands, applied in order by every node — agree on the log and you agree
on everything derived from it.

**Leader election with terms.** Time divides into numbered **terms**; each
term has at most one leader. Followers expect heartbeats from the leader;
a follower that hears silence too long assumes the leader is gone (a
guess! — world 3 vs world 2 again), increments the term, and asks the
others for votes. A node grants one vote per term, and only to a candidate
whose log is at least as up-to-date as its own. A **majority** of votes
makes a leader.

**Log replication.** Clients send commands to the leader; it appends them
to its log and replicates to followers. Once a **majority** hold an entry,
it is *committed* — applied, and guaranteed to survive.

Why majorities? Because **any two majorities of the same cluster share at
least one node**. Two leaders can't be elected in the same term (their
majorities would share a voter, and each node votes once per term). A new
leader's majority overlaps every commit majority, so — with the
up-to-date-log voting rule — committed entries are never lost. That
intersection argument is the entire trick; if you retain one sentence from
this section, keep that one.

The arithmetic: a cluster of **2f + 1** nodes tolerates **f** failures —
5 nodes ride out 2, and 4 nodes tolerate exactly as many as 3 do (1),
which is why cluster sizes are odd. And note what happens under partition:
the side holding a majority elects and commits; the minority side can do
*neither* — it stalls rather than diverge. Consensus systems answer CAP by
choosing C. That stall is also the price tag: every committed write costs
a quorum round trip, so consensus is reserved for low-volume,
high-stakes *coordination* facts — leadership, locks, configuration —
not for every user write in a busy system.

## Failure modes and defenses

The catalogue you must be able to recite, with defenses — most of which
you have already built in miniature earlier in this stage:

| Failure mode | What it is | Defense |
|---|---|---|
| Partial failure | A dependency is down/slow while you are fine; slow and dead are indistinguishable | Timeouts on every remote call; retries **with idempotency**; fail fast and degrade rather than hang |
| Network partition | Node groups can't reach each other, each side healthy internally | Decide C-vs-A per operation *in advance*; majority quorum for anything that must stay single-valued |
| Clock skew | Wall clocks on different machines disagree (NTP helps, milliseconds-to-seconds of skew remain) | Never order cross-node events by wall-clock timestamps; use a single leader's sequence, term/epoch numbers, or logical clocks; wall time is for humans |
| Split brain | Two nodes each believe they are the leader (e.g. the old leader was slow, not dead) | Majority election makes the second leader impossible *per term*; an epoch/term number accompanies every action so stale leaders' writes are rejected; leases must expire before takeover |

Clock skew deserves one extra beat, because "last write wins by
timestamp" looks so reasonable: with skewed clocks, a *later* write can
carry an *earlier* timestamp and be silently discarded — data loss with no
error anywhere. Ordering across machines must come from the system's own
logic (one leader assigning sequence numbers, or version vectors), never
from comparing wall clocks.

In Go: the standard library has this scar tissue built in. `time.Now()`
returns a value carrying both a wall clock and a monotonic clock reading,
and `time.Since`/`Sub` use the monotonic part — so *durations measured on
one machine* are immune to NTP stepping the clock. That protection ends at
the network: serialize a timestamp, send it to another machine, and the
monotonic part is gone — comparing it there is exactly the cross-node
wall-clock comparison the table forbids. Likewise `context.WithTimeout`,
which you have used since S3, is the "timeouts on every remote call"
defense: the deadline propagates down the call chain, turning an
indistinguishable hang into a bounded, handleable error.

## Exercise

Open [`exercise/`](exercise/). Cartwheel, an e-commerce platform you can
picture as the grown-up sibling of everything you built through S5, is
expanding from one region to two. You will work four worksheets against
its brief: a consistency menu, a partition drill, a failure-mode audit,
and a consensus walkthrough. Start with `README.md`; the scenario is in
`brief.md`.

Acceptance criteria:

1. `01-consistency-menu.md`: every feature has a chosen model that is
   defensibly the *weakest* sufficient one, a justification tied to the
   brief, and a concrete named anomaly for one-step-weaker. No feature
   answered "linearizable, to be safe" without a costed reason.
2. `02-partition-drill.md`: every feature has an explicit C-or-A call for
   the partition scenario, a concrete user-visible behavior (not
   "degrades gracefully"), a reconciliation note for the AP choices, and
   the PACELC else-case table is filled with latency numbers derived from
   the brief's RTTs.
3. `03-failure-modes.md`: all four failure modes mapped to concrete
   Cartwheel scenarios with a defense *and* its cost; three fallacies
   turned into plausible production bug stories, at least one involving
   retry-induced duplication.
4. `04-consensus-walkthrough.md`: election and partition traces answered
   with correct quorum arithmetic and *because*-reasoning (majority
   intersection, one vote per term), plus a defensible judgment on where
   Cartwheel genuinely needs consensus and where it is overkill.

There is no automated check. When the worksheets are done, tell your
tutor — verification is a design review: expect partition scenarios
injected live and "why not one model weaker?" pressed on every row.

## Further reading

- [Designing Data-Intensive Applications](https://dataintensive.net/) —
  chapters 8 ("The Trouble with Distributed Systems") and 9 ("Consistency
  and Consensus") are this lesson in long form; the best follow-up you
  can read.
- [The Secret Lives of Data — Raft](https://thesecretlivesofdata.com/raft/)
  — a visual, step-by-step walkthrough of leader election and log
  replication; run it before the consensus worksheet.
- [The Raft paper](https://raft.github.io/raft.pdf) — "In Search of an
  Understandable Consensus Algorithm"; sections 1-5 are readable with
  exactly what you know now.
- [Jepsen: Consistency Models](https://jepsen.io/consistency) — the full
  map this lesson's four-rung ladder is drawn from, with formal
  definitions one click away.
