# System Design Capstone

> `shared.systems.design-capstone` · ~4-6h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Produce a complete written design for a system you chose: requirements,
  estimates, architecture diagram, data model, and API sketch.
- Justify every major component choice against at least one rejected
  alternative.
- Analyze your design's failure modes and describe how it degrades under
  partial outages and 10× load.
- Defend the design in a live review, revising it coherently when your
  assumptions are challenged.

## Eleven lessons, one document

This stage handed you mechanisms one at a time: estimates and framed
trade-offs, protocols, contracts, storage and replication, caches,
queues, consistency models, partitioning, SLOs and degradation, patterns,
and — last lesson — worked case studies where somebody else picked the
brief and walked you through the moves.

Nobody walks you through this one. You choose a system, you write the
design, and you defend it against a reviewer whose job is to find the
seam. The mechanisms are no longer the difficulty; *selection* is. A
design is graded on the mechanisms you left out as much as the ones you
used, because every mechanism you add is an operational bill somebody
pays forever.

The medium shifts too: until now your design work fed a conversation,
and here the deliverable is a **document that must survive without you
in the room**.

## What a design doc is for

A design doc exists to let people disagree with you *cheaply* — before
the code, the migration, and the on-call rotation exist. It succeeds when
a reader who was not in the room can say "your fan-out numbers assume X,
and X is wrong" and be right. That is what turns intent into checkable
claims, and it makes the qualities of a good doc unglamorous: specific,
falsifiable, explicit about what it is *not* doing.

Its sections are the phases of a design discussion, frozen in writing:

| Section | Must contain | Comes from |
|---|---|---|
| Context & goals | Why now, who it is for, non-goals | design-intro |
| Requirements | Functional must/nice; non-functional with numeric targets | design-intro |
| Estimates | Traffic, storage, bandwidth, concurrency — arithmetic visible, assumptions named | design-intro |
| Architecture | Boxes, arrows, and the main flows walked end to end | architecture |
| Data model | What is stored, keyed how, partitioned how, kept how long | data-storage, scalability |
| API sketch | The handful of calls that carry the main flows, with their contracts | api-design |
| Decisions | Each major choice with alternatives, consequences, and a flip condition | all of them |
| Failure modes | Per-dependency behavior, blast radius, degradation | reliability, distributed-systems |
| Scale-out | What breaks at 10×, in what order, and the fix for each | scalability, data-storage |
| Rollout & operations | How it ships, what you watch, what pages | reliability |
| Open questions | What you don't know, and what would settle it | honesty |

Two rules keep the document honest. **Every box earns its place from a
requirement** — if you cannot name the requirement a component serves,
delete the component and see who complains. And **every number carries
its assumption**; a doc full of bare figures is a doc nobody can check,
which defeats the point of writing it.

Length is not the goal. A tight eight pages that a reviewer can attack
beats thirty pages of hedging. If a section has nothing decision-grade in
it, cut it to one line.

## Choosing your system

You pick the brief. The exercise ships a default — a collaborative
document service — plus alternates, and you may bring your own if it
clears the bar:

- **It has a hard axis.** At least one of: fan-out, contention on shared
  state, ordering or consistency requirements, statefulness (long-lived
  connections, sessions), or volume that a single node cannot hold. A
  system whose only difficulty is CRUD over a database is not a design
  problem; it is an afternoon.
- **You can put numbers on it.** If nobody can estimate its traffic —
  yours or your reviewer's — nobody can check the design.
- **It fits in two paragraphs** of description, and you can state its
  non-negotiables in five bullets.
- **Judging it needs no domain expertise.** A design for a system only
  you understand cannot be reviewed; it can only be admired.

Ambition is not the same as size. "Design a bank" is not a stronger
capstone than "design the ledger that a bank's transfers post to" — it
is a vaguer one. Pick something small enough that you can be *specific*,
and hard enough that specificity costs you something.

## Decisions, written down

You framed decisions out loud in design-intro: name the decision, name
alternatives with costs, pick, tie the pick to a requirement, say what
would flip it. A written decision record adds one section that speech
usually skips:

> **Consequences** — what the team now lives with because of this
> choice. Not the benefit; the bill. "Documents are partitioned by
> document id, so any feature that queries across documents (search,
> admin reports) needs a separate index we now have to keep in sync."

A decision record without consequences is a sales pitch. The consequence
line is also where your reviewer starts, because it is the only part you
had no incentive to write.

Keep records short and uniform — context, options with costs, decision
and the requirement it serves, consequences, flip condition. Five to
seven cover a real design; twenty means most are implementation details
belonging to whoever writes the code. And **write the decision you
actually agonized over**, not the one that is easy to write up: if the
storage engine took ten seconds and the concurrency model took two
hours, a doc whose longest record is the storage engine misrepresents
where its risk lives.

## Failure modes: enumerate, don't imagine

Brainstorming failures produces the failures you find interesting.
Enumeration produces the ones you have. Walk your own diagram and list
every dependency and every stateful component — no exceptions, including
the boring ones (DNS, the object store, the config service). For each,
answer five questions:

1. What does **down** look like to the user?
2. What does **slow** look like? Gray failure — a dependency answering
   in 8 s instead of failing — is usually worse than an outage, because
   nothing trips and your own capacity drains into waiting.
3. What is the **blast radius**: one user, one shard, one region,
   everyone?
4. What **detects** it, and how fast? (A failure with no signal is one
   you learn about from users.)
5. What does the system **do** about it — the degradation you chose in
   advance: fallback, stale read, shed load, open the circuit, fail
   closed — and which non-negotiable is threatened if it lasts an hour?

Then run the failures that only exist in real systems:

- **Partial outages.** One availability zone, one shard, one replica set
  behind. Partial is the normal case; total outage is the easy case.
- **Correlated failure.** Anything that makes many clients act at once:
  a deploy that drops every long-lived connection and invites fifty
  thousand simultaneous reconnects; a cache flush that sends every miss
  to the database; retries synchronizing into a storm. Your defenses —
  jitter, backoff, load shedding — are from the reliability lesson;
  what is new is *finding* the synchronizing event in your own design.
- **The poison input.** One document, key, or tenant that is 1,000×
  the size of the median. Averages hide it; your design must not.

A failure table is not paperwork. It is the fastest way to discover that
a component you drew casually — the one everything routes through — has
no story for being unavailable.

## The 10× drill

Growth does not break systems gently; it breaks one thing first, loudly.
The drill finds the order:

1. **Pick the axis that actually grows.** Ten times the users is one
   scenario; ten times the activity *per* user, or ten times the
   concurrency on a single hot key, is a different one — and usually the
   nastier. Say which axis you are multiplying.
2. **Multiply the estimates**, not the vibes. Re-run the arithmetic from
   your estimates section with the new inputs.
3. **Walk every component and name its limit and the resource behind
   it**: connections per node, ops per second per partition, bytes per
   second, IOPS, memory per active object, lock contention on a hot row.
4. **Order the breakages.** First, second, third. Only the first one is
   urgent — the rest are a roadmap, and saying so is part of the answer.
5. **Classify each fix**: a *config change* (add nodes, raise a limit),
   a *design change* (repartition, add a cache tier, split a service),
   or a *rewrite* (change the algorithm or the consistency model). This
   classification is what a reviewer is really buying: a design where
   growth costs config changes is a good design, and one where the first
   doubling forces a rewrite is not — however elegant it looks today.

Two traps the drill exists to expose. **Hot spots do not average**:
adding nodes never helps a single document, key, or partition that is
itself the bottleneck — the fix has a different shape (split the key,
batch its work, or serialize it deliberately). And **coordination does
not get cheaper with scale**: anything requiring agreement across nodes
gets *worse* as nodes multiply, so the 10× answer for a coordinated
component is usually "coordinate less", not "buy more".

## Defending a design

The review is not a quiz with an answer key. The reviewer picks at
seams to see whether your design is *connected* — whether the numbers,
the boxes, and the promises move together. There are exactly three
legitimate responses to a challenge:

- **Defend with evidence.** "It holds, because estimate A3 says the
  per-document write rate is under five per second even at peak."
- **Concede and revise.** "You're right — and here is what else moves."
  This is the response that scores highest, because it proves the design
  is a mechanism, not a memorized picture.
- **Park it.** "I don't know. It would take a measurement of X, and
  until then I assume Y — here's what changes if Y is wrong."

What you must not do is invent a number under pressure. A fabricated
figure poisons everything downstream of it, and reviewers catch it by
asking for the derivation.

**Revising coherently** is the skill being graded. When an assumption
changes, the change propagates: the estimate moves, then the component
sized by that estimate, then possibly the decision that component
implements, then the failure story. Say the chain out loud. If nothing
downstream moves when an input changes by 10×, either the input was
inert — fine, say so — or your design was never derived from its numbers.

Expect **new constraints mid-review**: a compliance rule, a cost cut, a
launch date, a customer with a pathological workload. This is not
hazing; it is the job. Absorb one the same way every time: which
requirement does it add or change, which estimate moves, which decision
does it reopen, and what does honoring it cost? "That doesn't affect my
design" is almost always the wrong answer — and when it genuinely is the
right one, prove it with the same chain.

**In Go:** your API sketch does not need an implementation, but it does
need to be concrete enough to argue about — request and response
shapes, the idempotency key, the error cases. A handful of handler
signatures or a short `service`/`message` sketch is a fine level of
detail. Similarly, when your estimates need "what one node does",
prefer numbers you measured in S5 (your own benchmarks and profiles)
over the generic anchors from design-intro, and say which they are.

## Exercise

Open [`exercise/`](exercise/). You will produce a complete design doc and
then defend it. `brief.md` carries the default brief — **Codraft**, a
collaborative document service — plus three alternates and the bar your
own brief must clear. The four worksheets are the document:

1. `01-design-doc.md` — context and goals, requirements, estimates,
   architecture and flows, data model, API sketch.
2. `02-decisions.md` — five to seven decision records.
3. `03-failure-and-scale.md` — dependency failure table, degradation,
   the 10× drill, rollout and operations.
4. `04-review-prep.md` — your own attack on your design before someone
   else's.

Acceptance criteria:

1. `01-design-doc.md` is complete: at least 6 functional requirements
   split must/nice with an explicit non-goals list; at least 5
   non-functional requirements with numeric targets; estimates covering
   traffic, storage, and whatever concurrency or connection counts your
   system implies — every one with visible arithmetic and a named
   assumption; a dominant-axis statement backed by a specific line.
2. The architecture section has a diagram (text or Mermaid), every box
   annotated with the requirement that pays for it, and the two most
   important flows walked step by step end to end.
3. The data model names each stored entity, its key, its partitioning
   scheme, its retention, and the access patterns it serves; the API
   sketch gives the calls carrying those flows with their contracts —
   including what happens when a client retries.
4. `02-decisions.md` has 5-7 records, each with at least two real
   alternatives and their costs, the requirement or estimate ID that
   decides it, a consequences line naming what you now live with, and a
   concrete flip condition. "Nothing would flip it" is an automatic redo.
5. `03-failure-and-scale.md` covers every dependency and stateful
   component from your own diagram with down/slow behavior, blast
   radius, detection signal, degradation, and the threatened
   non-negotiable; plus one partial-outage story and one correlated-
   failure story written as narratives.
6. The 10× drill names the axis, re-runs the arithmetic, and orders at
   least three breakages, each with a fix classified config change /
   design change / rewrite.
7. `04-review-prep.md`: your weakest assumption, a pre-mortem, the
   version-one you would ship in eight weeks with four engineers
   (including which cut is a one-way door), and a two-minute tour of the
   design in your own words.

Budget 3-4 hours for the document and expect the review to take 60-90
minutes. There is no automated check: verification is a live design
review. Your tutor will audit your arithmetic, argue the alternatives you
rejected, break your dependencies one at a time, and **inject new
constraints partway through** — the grade is largely how coherently the
document moves when they do.

## Further reading

- [Design Docs at Google](https://www.industrialempathy.com/posts/design-docs-at-google/)
  — what these documents contain, and why the writing is the point.
- [Michael Nygard — Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
  — the original ADR format your decision records follow.
- [AWS — Avoiding fallback in distributed systems](https://aws.amazon.com/builders-library/avoiding-fallback-in-distributed-systems/)
  — a sharp argument that the degradation paths you never exercise are
  the ones that fail you.
- [AWS Well-Architected Framework](https://docs.aws.amazon.com/wellarchitected/latest/framework/welcome.html)
  — a review checklist you can run against your own doc before someone
  else does.
