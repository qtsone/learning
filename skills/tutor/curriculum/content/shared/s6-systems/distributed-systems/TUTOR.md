# Tutor notes — Distributed Systems

## Where the learner is

Seventh lesson of S6. They have the design-intro habits (stated
assumptions, visible arithmetic, flip conditions — keep grading those),
and they have *built* the concrete precursors: a framed protocol over an
unreliable-feeling transport (networking), idempotency keys (api-design),
replication/sharding vocabulary (data-storage), TTL staleness (caching),
and an at-least-once broker with acks and redelivery (message-queues).
This lesson is where those experiences get their theory. Expect the
theory to feel slippery even to someone who coded the practice — the
CAP/consistency vocabulary is new and widely misused on the internet
they learned from. No consensus implementation exists anywhere in this
stage; grade Raft *intuition* (quorum arithmetic, majority intersection,
terms), never implementation detail.

## Running the design review

Verification is a conversation against the four worksheets. Protocol:

1. **Read all four first**, silently listing: any "linearizable to be
   safe", any anomaly column that says "inconsistent data" instead of a
   user-visible sentence, any A-row without reconciliation, PACELC rows
   without arithmetic, and consensus answers with outcomes but no
   quorum reasoning. That list is your agenda.
2. **Attack the consistency menu.** Pick two rows and press "why not
   one step weaker?" — the pass signal is a named anomaly tied to a
   business ask, not re-asserted strength. Reference calls (defensible
   alternatives exist; grade the argument): F1 eventual; F2 causal or
   session-scoped (a cheap defense: "cart merge beats cart lock");
   F3 needs atomic decrement per region but tolerates region-local
   stock pools — full global linearizability is overshoot given
   business ask 4; F4 linearizable (business ask 3 is the citation);
   F5 read-your-writes (global linearizability here is classic
   overshoot — probe P1 for "scope it to the buyer's session");
   F6 eventual/monotonic reads. If they chose stronger, make them pay
   the latency/availability price aloud; if weaker on F4, quote legal.
3. **Inject partition variations live.** Their sheet answered 90 s.
   Now: "the link is down 45 minutes, and the drop started 5 minutes
   in." Watch whether F4 stays C (it must — or they must renegotiate
   stock, e.g. pre-partitioned per-region allocations, which is an A
   answer worth full credit *because it renames the invariant*). Then:
   "partition heals and two regions have divergent carts — walk me
   through the merge for a buyer who deleted an item on one side."
4. **Audit the else-case arithmetic.** Sync cross-region ≈ +100 ms
   (one RTT, more with quorum) on every write; async ≈ in-region
   latency with a lag window. Their numbers must trace to the brief's
   RTTs. Ask for the connect-the-dots defense if pairs mismatch.
5. **Probe one consensus trace.** Have them re-derive Q3-Q5 aloud with
   the node names. Reference answers: Q1 — not committed (no
   majority), may be lost, client learns nothing (timeout: the retry+
   idempotency loop from api-design is the right recovery, worth
   surfacing); Q2 — both can *run*, both cannot *win*: each node votes
   once per term and majorities intersect; Q3 — A appends and
   replicates to B only: 2/5 is no majority, so no commit — A is a
   zombie leader, safely impotent; Q4 — yes, {C,D,E} is 3/5, term 4;
   any elected leader's majority shares a node with every commit
   majority, so nothing committed is lost; Q5 — A sees term 4 > 3,
   steps down, its unreplicated partition-era entries are discarded
   (they were never committed, so no client was promised anything);
   Q6 — 5 tolerates 2; 6 needs majority 4, still tolerates 2; Q7 — C;
   both sides accepting writes = split brain, divergent "single"
   facts. Grade the *because*; a right outcome with no intersection
   argument gets one nudge, then remediation.
6. **Fallacy stories reality-check.** For their best Part-B story ask:
   "which log line or dashboard panel from your S5 observability
   lesson catches this before the customer does?" Concrete answers
   (duplicate idempotency-key hits, retry-rate spike, cross-region
   call latency histogram) show it connected to practice.
7. Close with the quiz (quiz.json), skipping anything already
   demonstrably covered.

Timebox: 30-45 minutes. Materially incomplete worksheets: stop and
remediate rather than reviewing a stub.

## Common misconceptions

- **"CAP: pick any two of three"** — the classic. P is not optional;
  the choice is C-vs-A, only during a partition, per operation. Anyone
  reciting "CA system" gets sent back to the lesson's CAP section.
- **CAP-consistency = ACID-consistency** — the C in CAP is
  linearizability; ACID's C is application invariants. Same word, two
  meanings; S6 uses the CAP one.
- **"Eventual consistency means it sorts itself out"** — it is a
  convergence promise with *no* interim guarantees; the anomaly column
  exists to make that concrete. Also the reverse error: treating
  eventual as always unacceptable.
- **"Availability = uptime"** — CAP's A is every request to a
  non-failed node getting a non-error answer; a five-nines system that
  refuses writes during partitions chose C, not A.
- **Timeouts as failure detection** — a timeout is a decision to stop
  waiting; slow-vs-dead is undecidable. This is the root of both
  duplicate-effect bugs and split brain; learners who internalized the
  message-queues redelivery behavior have felt it.
- **Ordering by wall-clock timestamps** — "NTP makes clocks close
  enough" until last-write-wins silently drops a write. Monotonic
  clocks help one machine, never two.
- **"Consensus everywhere / Raft makes it scale"** — consensus is a
  coordination tool that *costs* a quorum round trip per write and
  chooses C under partition; it is not a horizontal-scaling device.
  Watch for Q8 answers putting every order through Raft.
- **Per-system instead of per-feature thinking** — one consistency
  model or one CAP letter for all of Cartwheel. The whole exercise is
  built to break this.

## Grilling points

- "Your F2 cart is causal. Buyer adds shoes on their phone in a Paris
  airport, lands in New York, opens the laptop. What may they see, and
  why is that acceptable — cite the brief."
- "Sell me linearizable carts. Now tell me what that costs during the
  drill's 90 seconds — which region's buyers stop shopping?"
- "The drop is 5,000 units and your F4 is C during partitions. Marketing
  asks: 'can we keep selling on both continents if we accept 1%
  oversell?' What design does that unlock, and what did marketing just
  renegotiate?" (Invariant relaxation → per-region stock allocation —
  an A answer bought by changing the requirement.)
- "Why does a 4-node cluster tolerate exactly as many failures as a
  3-node one? Show the arithmetic."
- "Two majorities of five nodes: prove to me they share a node, and
  name the two disasters that intersection prevents." (Double leader
  per term; losing committed entries.)
- "Your split-brain defense says 'leases'. The old leader's clock runs
  2% slow — does your lease still expire before the takeover? What
  extra mechanism makes stale-leader writes harmless anyway?" (Epoch/
  term/fencing numbers checked by the receiver.)
- "Which of the eight fallacies did *your own* S5 project assume? Point
  at the code area from memory."

## Grading rubric

Grade reasoning habits — cited requirements, named anomalies, quorum
arithmetic, costed choices — never recall of definitions or agreement
with the reference calls.

- **A** — Menu choices are minimal-and-sufficient with vivid anomaly
  sentences, or overshoots are consciously costed; partition drill has
  concrete behaviors and honest reconciliation for every A row; PACELC
  arithmetic traces to the brief's RTTs and pairs coherently with the
  partition calls (or the mismatch is defended); all consensus traces
  carry the majority-intersection *because*; live variations (45-min
  partition, weaker-model pressure) are absorbed by updating the
  design, not defending the sheet; fallacy stories are specific enough
  to have observability hooks.
- **B** — Sound method with lapses: one uncosted overshoot, one
  "degrades gracefully"-grade cell, a consensus answer right on
  outcome but needing a nudge to produce the quorum argument, or
  PACELC numbers present but not connected to the partition column.
  Recovers under review pressure with at most one hint per issue.
- **C** — Tables filled but reasoning thin: models chosen by strength
  ("safer"), anomalies generic, reconciliation hand-waved, quorum
  arithmetic shaky on re-derivation. Pass only if a focused redo of
  the weakest worksheet lands within this session; otherwise iterate.
- **Fail** — "Pick two of three" CAP, one model for the whole system
  defended as such, consensus traces wrong on outcomes with no
  self-correction, or worksheets materially incomplete. Remediate —
  scalability, reliability, and the capstone all build on this
  vocabulary.

## Remediation ladder

1. "Take your weakest anomaly cell and turn it into a support ticket:
   'I did X, then I saw Y.' If you can't write the ticket, the
   weakening is free — take it; if you can, you've justified the
   strength. Redo the column in that voice."
2. "Cover worksheet 4. Five nodes, leader crashes after replicating an
   entry to one follower. Walk the election with named nodes and
   counted votes, out loud, and tell me the fate of that entry —
   *because* of which rule?"
3. "For F4 during the partition: complete both sentences. 'If we stay
   available, the brief's ask #3 breaks like this: …' and 'If we stay
   consistent, buyers experience: …'. The CAP call is whichever
   sentence the business can live with — now generalize that method to
   the rows you found hard."
4. Walk the messaging-app case together — "message says *sent*, then
   the sender's other device doesn't show it" — you supply the
   question sequence (which guarantee is violated? weakest model that
   fixes it? cost during a partition?), they supply every answer. Then
   they redo worksheet 1 alone.

## After passing

Preview: "You can now say precisely what a replicated system promises.
Next: scalability patterns — consistent hashing with virtual nodes, in
code — where you'll build the machinery that decides *which* node owns
the data you just learned to keep consistent."
