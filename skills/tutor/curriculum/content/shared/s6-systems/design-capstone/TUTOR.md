# Tutor notes — System Design Capstone

## Where the learner is

Final lesson of S6, straight after the guided case studies. Every
mechanism in this stage is theirs, and the last lesson showed the moves
performed on somebody else's briefs. What is new here is that nobody
picks the brief, nobody sequences the phases, and the artifact must stand
without them in the room. Verify is discussion: grade from the four
worksheets plus a 60-90 minute review that you run as an interviewer, not
a lecturer.

Grade **process**, never agreement with your reference design: stated
assumptions, visible arithmetic, named alternatives with honest costs,
failure stories that follow from their own diagram, and — the signature
of this lesson — coherent revision under pressure. A learner whose design
differs completely from the reference and holds up under attack scores
higher than one who reproduces it and cannot say what would flip it.

This is the last rehearsal before S7, where they design *and then build*.
A design that survives this review is the kind of document `go.capstone.
planning` expects them to write for real.

## Review protocol

**Pre-read (before the session).** Read all four worksheets silently.
Check each acceptance criterion is materially filled — if a worksheet is
a stub, or the failure table lists three of their nine boxes, stop and
send them back with the remediation ladder. Reviewing a stub teaches
nothing. Otherwise pick the three weakest points; they are your agenda.

Then run the session in this order. If you fall behind, protect steps 2,
4, 5 and 6 — those are where the grade actually comes from.

1. **The tour (5 min).** They present worksheet 4's two-minute tour. Do
   not interrupt, even for something wrong; note it. What you are
   listening for: do they lead with the dominant axis, or with a
   technology?
2. **Estimates audit (15 min).** Re-derive **two** numbers live: one
   aggregate (total write rate, storage) and one **per-key** (load on the
   busiest single document/partition/tenant). The per-key one is where
   most designs turn out to be disconnected from their numbers. Probe any
   input more than ~10× off the reference envelope below, and ask of each
   assumption: "where did that come from, and what if it's 3× higher?"
3. **Architecture walk (15 min).** Have them narrate both flows end to
   end. Then two moves: point at a box and ask "which requirement pays
   for this?"; and pick a box and **delete it** — "we can't have this;
   what breaks and what would you do instead?" Finish on state: "this
   process holding live sessions dies right now — what do users see?"
4. **Decision attack (15 min).** Take the decision they flagged as least
   certain in worksheet 4 and argue the rejected alternative as strongly
   as you honestly can. Then take a decision they are *confident* about
   and counterfactual it until something flips or the position collapses
   — a design where nothing ever flips is dogma. Reject straw-man
   options and consequences lines that are really benefits.
5. **Failure injection (15 min).** Break two things: one hard down
   (kill the component with the widest blast radius), one **gray** (a
   dependency answering in 5-10 s instead of failing). Gray failure is
   the one they will have skipped. For each: what users see, what fires,
   what an operator does, what is lost versus delayed, and which
   non-negotiable breaks first if it lasts an hour.
6. **Constraint injection (10-15 min).** One or two from the menu below.
   Rules: never before step 3 (they must own the design before you move
   it), one at a time, and always close with *"what else moves?"* The
   graded motion is requirement → estimate → decision → cost. "That
   doesn't affect my design" is acceptable only with the chain shown.
7. **10× drill (10 min).** Make them name the axis, order at least two
   breakages, and classify the fixes config change / design change /
   rewrite. Push on the one that adding nodes cannot fix.
8. **Close.** Fill quiz.json gaps the conversation did not already cover,
   then give the grade with the three specific things to fix.

### Reference envelope — Codraft (default brief)

Their numbers will differ; grade the method and probe anything off by
more than an order of magnitude. Assume 400k DAU (20% of 2M), 8 opens
and 3 editing sessions per active user per day, peak factor 3.

- **Opens:** 400k × 8 = 3.2M/day ≈ 32/s average, ~100/s peak —
  *unimpressive, and saying so is a finding.*
- **Concurrent connections:** editors 1.2M sessions × 11 min ÷ 1,440 min
  ≈ 9k average, ~28k peak; viewers 3.2M × 4 min ÷ 1,440 ≈ 9k average,
  ~27k peak. Total ~18k average, ~55k peak. Small fleet — but stateful
  and routed by document, which is the actual difficulty.
- **Operation ingest:** ~9k editors × 1 op/s ≈ 9k ops/s average, ~28k/s
  peak. **Per document: 1-5 ops/s, worst observed 60.** The intended
  insight: aggregate is large, per-key is tiny → partition by document
  id and every document's write path is trivial.
- **Fan-out:** ops × concurrent collaborators (1-2) ≈ 10-50k messages/s,
  × 100 B ≈ 1-6 MB/s. Bandwidth is a non-problem — except the
  company-wide document: one editor at 1 op/s × 3,000 readers = 3,000
  messages/s from a single key.
- **Storage:** op log 9k/s × 10⁵ × 100 B ≈ 90 GB/day → ~8 TB at 90-day
  retention before replication. Documents themselves: 12M × 20 KB ≈
  240 GB. History costs ~30× the documents — so compaction/snapshotting
  and the retention policy are load-bearing, not housekeeping.
- **Revocation (60 s):** ~55k connections revalidating each minute ≈
  ~900 checks/s, or a 60 s permission cache. Cheap — they should be able
  to show it.
- **Open latency (500 ms p95):** snapshot + replay of the tail; snapshot
  cadence is the knob, and the arithmetic (ops since last snapshot ×
  cost per op) should appear.

Reference design shape, for your own orientation only: long-lived client
connections routed by document id to a single owner process per document
(single writer ⇒ ordering for free), ops appended durably *before* the
acknowledgement (non-negotiable 1), broadcast to collaborators, periodic
snapshots plus a compacted op log for the 90-day history, async
consumers (search, notifications, analytics) fed from the log,
permissions revalidated on a 60 s cycle, each document homed in one of
the two regions. Convergence via server-ordered transforms or CRDTs are
both defensible; what is graded is the trade-off, not the pick. A design
with no single writer per document must explain how it converges and how
ordering is established; a design that stores documents as a row updated
per edit must survive the 60-editor and 1 s-latency probes.

### If they brought their own brief

Approve it before they write: hard axis, estimable, two paragraphs plus
five non-negotiables, reviewable without domain expertise. Without a
reference envelope, audit differently:

- **Derive a number a second way** ("you got storage from writes ×
  bytes; now get it from documents × size — do they agree?").
- **Check internal consistency**: does the concurrency figure match the
  session length and the traffic figure?
- **Anchor externally**: does one estimate compare sanely to something
  known (a phone photo, a datacenter round trip, one node's throughput
  from their S5 benchmarks)?
- **Check the hard axis is actually engaged**: if their design would be
  identical without the hard axis, the brief was decoration.

### Constraint injection menu

Pick by what their design is most confident about — the point is to move
a load-bearing assumption, not to trivia them.

1. **Data residency.** "EU customers' documents may never leave the EU."
   Expect: documents homed by region, routing on that key, cross-region
   collaboration paying a round trip, and an honest statement of what
   becomes impossible. Watch for "we'll replicate everywhere", which
   violates the constraint they just accepted.
2. **Offline editing becomes must-have.** Directly attacks the
   convergence decision. Server-ordered designs must either concede a
   significant redesign or bound it (offline as a branch merged on
   reconnect, with a named conflict story). A learner who says "no
   change needed" has not understood their own model.
3. **Cost cut of 40%.** They must name the biggest line item from their
   own estimates (for Codraft: history retention, then the connection
   fleet) and cut something with a stated sacrifice — shorter retention,
   coarser history, cold tiering, fewer replicas.
4. **A single document with 5,000 simultaneous editors.** The hot-key
   probe: nothing they can add fixes one key. Acceptable answers batch
   broadcasts, degrade most participants to read-only or delayed view,
   or fan out through a tree — all with the cost named.
5. **Four engineers, eight weeks.** Scope negotiation: what ships, what
   is cut, which cut is a one-way door. Cross-check against worksheet 4
   section 5 and push where the two disagree.
6. **Compliance deletion.** "A deleted document must be unrecoverable
   within 24 hours." Collides with 90-day history, snapshots, and
   backups. Strong answers reach for per-document encryption keys
   destroyed on delete, or a stated, costed backup-expiry story.
7. **A customer demands four nines with penalties.** Forces the
   reliability lesson's price-tag conversation: what the extra nine
   costs, which single points of failure block it, and whether the
   honest answer is "we sell them three nines".

## Common misconceptions

- **The doc as a slide deck.** Confident boxes, no arithmetic, no
  consequences lines. Ask for the derivation of any number; the answer
  reveals it immediately.
- **Aggregate load mistaken for per-key load.** "28k ops/s, so we need a
  big cluster" — while each document takes 1 op/s. Partitioning makes
  aggregate load a fleet-sizing question and per-key load the real
  design question.
- **Averages hiding hot keys.** Median 3 collaborators, so fan-out is
  "small" — with a 3,000-reader document in the same brief. Every design
  must handle its own tail.
- **Durability asserted, not designed.** "We never lose an edit" while
  the acknowledgement is sent before the write is durable. Ask exactly
  where the ack happens relative to the durable write, then crash the
  process between them.
- **Slow ignored in favor of down.** Failure tables with a "down" column
  and no "slow" column. Gray failure drains capacity into waiting and
  trips nothing.
- **Retention as an afterthought.** History that grows forever, with
  storage estimated for one day. Retention and compaction are design
  decisions with arithmetic attached.
- **Consistency chosen once, globally.** One model for the whole system
  instead of the weakest sufficient model per feature — overshooting is
  spent latency, not safety.
- **Rigidity mistaken for conviction.** Absorbing a new constraint by
  insisting nothing changes. Equally bad: abandoning the design entirely
  at the first push. Both are failures of the same skill.
- **Novelty as justification.** A component chosen because it is
  interesting, whose consequences line is empty. Ask who operates it at
  3 a.m.

## Grilling points

- "Delete your busiest box from the diagram. Build me the design that
  does not have it — and tell me what that version costs."
- "Your acknowledgement to the client: draw the exact line where it is
  sent. Now the process dies one instruction earlier. What does the user
  believe, and what is true?"
- "Which of your non-negotiables would you break first if you had to
  break one, and what would you tell the sponsor the morning after?"
- "You have 55,000 live connections and you deploy. Walk me through the
  next 90 seconds." (Reconnect stampede — is there jitter, draining, a
  staged rollout, and does the arithmetic support their answer?)
- "Two of your requirements conflict under load — find the pair before I
  tell you which." (For Codraft: never-lose-an-edit versus 1 s
  visibility when the store is slow.)
- "Which number in this document would you most want to measure in week
  one, and what would you do differently at 10× that value?"
- "You are the reviewer. What is the most embarrassing question you
  could be asked about this design?" (Their answer usually names the
  real weakness faster than you can find it.)

## Grading rubric

Grade estimation habits, decision quality, failure reasoning and
revision under pressure — never recall of stage vocabulary.

- **A** — Document complete against all seven acceptance criteria;
  estimates carry assumptions and visible arithmetic, including per-key
  load, and the dominant axis is named with evidence (including any axis
  found unimpressive); every box traces to a requirement; both flows
  walk end to end including the durability boundary; the data model
  states keys, partitioning and retention with arithmetic that bounds
  storage; the API sketch answers the retry question; 5-7 decisions with
  real alternatives, honest costs on the chosen side, consequences and
  concrete flip conditions; failure table covers their whole diagram
  with down *and* slow; both failure narratives are specific; the 10×
  drill orders breakages and classifies fixes, including one that adding
  nodes cannot fix; under injection they propagate the change through
  requirement, estimate, decision and cost unprompted, conceding
  cleanly where the challenge lands.
- **B** — Sound design with gaps the conversation closes: an assumption
  missing its justification, per-key arithmetic produced only when
  asked, one decision whose flip condition needs prompting, a failure
  table thin on gray failure, or a 10× ordering that arrives after a
  nudge. Constraint injection is absorbed, but the downstream chain
  takes a hint.
- **C** — The paperwork is there and the reasoning is thin: numbers
  present but unconnected to the boxes, alternatives named without
  costs, consequences lines that are benefits, failure stories only for
  the components they find interesting, hot keys unremarked. Pass only
  if a live remediation lands in this session — pick one weak section
  and rebuild it together, with them holding the pen; otherwise iterate.
- **Fail** — Worksheets incomplete or generic enough to belong to any
  system; cannot re-derive their own arithmetic; a component whose
  purpose they cannot name; durability or convergence asserted with no
  mechanism; every trade-off a straw man; or the design does not move
  under any constraint and no chain is offered. Remediate — this is the
  document S7 planning builds on, so advancing a hollow one costs them
  twice.

## Remediation ladder

1. "Pick the number in your document you are least sure of. Where did it
   come from? Now say 'I assume X because Y' and tell me what breaks if
   X is 10× off." (Rebuilds the connection between numbers and design.)
2. "Take your busiest key — one document, one tenant, one shard. Compute
   its load alone, out loud. Compare it to your aggregate figure. Which
   of those two numbers does your architecture actually answer?"
3. "Walk one write from the client's keypress to the moment you would
   swear the data is safe. Stop at every hop and tell me what a crash
   there means." (Surfaces missing durability boundaries, dual writes,
   and unacknowledged buffers faster than any question about the
   diagram.)
4. Rebuild one section together — usually the failure table. You supply
   the structure (component → down → slow → blast radius → detection →
   degradation → promise threatened), they supply every entry, walking
   their own diagram left to right. Then they redo the 10× drill alone
   and bring it back.

## After passing

Preview: "That is Systems & Design done — you can take a vague prompt to
a defensible written design and hold it under fire. S7 removes the
safety: you pick a real project, write this document for it, and then
build the thing, with the design reviewed against what production
actually does to it."
