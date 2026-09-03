# Tutor notes — Design Case Studies

## Where the learner is

Eleventh lesson of S6, after architecture. Every mechanism in these designs
is already theirs — caching and invalidation, queues and at-least-once
delivery, partitioning and replication, consistency models, statelessness,
backpressure, SLOs and graceful degradation, ports and adapters. What is new
is **composition under a prompt**: choosing which mechanisms the requirements
pay for, in order, without being asked for the next phase.

Verify is discussion. Grade from the three filled worksheets plus the review
conversation, and grade the *habits*: stated assumptions, visible arithmetic,
named trade-offs with flip conditions, and a 10× answer with a first symptom
and a cost. Never grade agreement with the reference numbers below, and never
grade recall of the lesson's walkthroughs — the briefs were built so that
reciting the walkthrough produces a wrong design.

This lesson plus architecture are the dress rehearsal; design-capstone next
is the performance.

## Review protocol

Run it as a design interview: you are a reviewer who is interested, informed,
and unimpressed by fluency. Timebox 45-60 minutes for all three cases. Read
all three worksheets silently first and pick your agenda from what is thin.

If a worksheet is materially empty (a missing phase in Case A or B), stop and
send them back with the remediation ladder — do not review a stub. An
unfinished **speed round** is different: that one is timeboxed on purpose,
so review what exists and probe the pacing.

1. **Open with the design statement, not the worksheet.** "Give me Trackly in
   90 seconds." A design statement that cannot be delivered aloud usually
   means the worksheet was filled section by section without ever being
   thought of as one system.
2. **Audit two calculations live, one per case.** Pick one they will have to
   re-derive rather than read. Grade the method; probe any input more than
   ~10× off the envelopes below.
3. **Attack one trade-off per case.** Argue the rejected side sincerely. Pass
   signal: they defend with costs and requirement ids, concede where the
   alternative genuinely wins, and their flip condition survives.
4. **Replay a failure from the brief.** Trackly: the analytics pipeline is
   down 20 minutes mid-campaign. Huddle: a gateway crashes holding 80k
   sockets while 400 conversations are waiting for assignment.
5. **Inject a new constraint** (list below) and make them redesign in the
   room. Watch for whether they *revisit their numbers* rather than bolting a
   box onto the diagram.
6. **Run the 10× drill** on whichever design they seem proudest of. Demand
   all three parts: what breaks, first symptom, next step and its cost.
7. Close with the quiz (quiz.json), skipping anything already demonstrated.

### Reference envelope — Trackly (Case A)

Theirs may differ; the arithmetic must be visible and the conclusions must
follow from *their* numbers.

- creates 2M/day ≈ 20/s average — deliberately trivial; noticing that the
  write path is boring is a correct finding.
- redirects 400M/day ≈ 4,000/s average, campaign spike 40,000/s ≈ **10×
  average**. Sizing to the average is the classic failure here.
- storage ≈ 2M × ~500 B ≈ 1 GB/day ≈ 0.4 TB/year — also trivial. Anyone who
  deep-dives the database has misread their own estimate.
- raw click events ≈ 400M × ~200 B ≈ 60-100 GB/day ≈ 5-9 TB over the 90-day
  window — ordinary, but it is the one growing number.
- live-link working set ≈ tens of millions × 500 B ≈ 10-30 GB — fits a cache
  tier, so a high hit rate is available.
- **Intended dominant axis:** the read path at 40k/s spikes *under a hard
  60-second global mutation-propagation rule*. Accept "read QPS + edge
  invalidation"; challenge anything that names storage.

Expected content and the probes that expose its absence:

- **Alias namespace.** Strong answer: one namespace with a uniqueness
  constraint deciding every conflict, the 2,000 reserved paths pre-inserted
  as taken rows at deploy time, and generated keys drawn from the same space
  with conditional insert and retry. Race between two customers: the
  constraint decides, the loser gets a 409. Probe: "who wins, and what
  mechanism made that decision?" — "we check first, then insert" is a
  check-then-act race; make them see it.
- **301 is now illegal.** A permanent redirect is cached in browsers and
  proxies you do not control, so the 60-second takedown cannot be honored.
  Expect 302 or 307 plus `Cache-Control: no-store` (or a max-age far below
  60 s). If they kept 301 from the lesson, that is the transcription tell —
  ask them to walk a takedown through a browser cache.
- **Propagation arithmetic.** Push purge to edges *plus* a TTL ceiling as the
  backstop: a lost purge with no TTL is unbounded staleness, and an audited
  requirement cannot rest on a best-effort message. With TTL T the worst-case
  staleness is T, so T ≤ 60 s (many will pick 20-30 s for margin) — and the
  cost is origin load ≈ distinct hot keys per PoP ÷ T, which they should at
  least gesture at. Takedown may reasonably ride a separate small denylist
  replicated faster than the main mapping.
- **Two analytics paths.** Approximate: event to a broker, rollups, dashboard
  reads rollups, drops events under pressure, never blocks the redirect.
  Exact/billed: durable capture (edge access logs are the natural audit
  backstop), deduplicated by event id, reconciled and retained 3 years. The
  key insight to fish for: *the redirect still must not block* — losing a
  billed click is a revenue bug, not an availability bug, and the fix is a
  durable log plus reconciliation, not a synchronous write.
- **Bot classification** belongs downstream of durable capture for the billed
  path, so invoices can be restated when the classifier improves; cheap
  filtering at the edge is fine for the dashboard path.

### Reference envelope — Huddle (Case B)

- messages 12M/day ≈ 120/s average, ~360/s peak; deliveries ×3-5 sessions
  ≈ 400-1,500/s peak. Small.
- connections ≈ 250k customers × ~1 + 25k agents × ~2.5 ≈ **300-400k
  sockets** → at 100-200k per box, a handful of boxes per region. **The
  connection tier is not the hard part here** — that is the whole point of
  the case, and the lesson's 50-box messenger is the trap.
- text 12M × 200 B ≈ 2.4 GB/day ≈ 6 TB over 7 years. Attachments
  12M × 8% × 400 KB ≈ **400 GB/day ≈ 1 PB over 7 years** — two orders of
  magnitude above text.
- search index: 90 days ≈ 1.1B messages ≈ 200-250 GB of text, index a small
  multiple — modest.
- **Intended dominant axis:** attachment storage and retention/residency
  compliance, with the 60-second routing SLA as the product-risk axis. Accept
  either if argued from their lines; challenge "connections" hard, because
  their own arithmetic refutes it.
- **Planted contradiction:** 250k open conversations versus 25k agents × 6 =
  150k slots. In character, the sponsor answers: "open" includes waiting and
  idle-not-yet-closed conversations; a conversation auto-closes 20 minutes
  after the last message; roughly 40% are actively awaiting a reply. Best
  learners find it unprompted; good ones find it when you ask "does your
  agent capacity cover your concurrency?"; if nobody notices, that is a
  B-ceiling finding.

Expected content:

- **Ordering.** Partition by conversation id, sequence assigned by that
  conversation's owner, clients sort by sequence. A transfer keeps the same
  conversation id — the transcript requirement demands one ordered stream —
  and appears as an event in the log; a supervisor join reads the log from
  the start and records an access event. Probe the timestamp trap and the gap
  repair.
- **Routing.** Atomic assignment: conditional update or a lease with TTL so a
  crashed assigner releases its claim. Probe: "two assigners, one free agent,
  same millisecond" and "the assigner crashes after picking an agent and
  before telling anyone" — the answers must be mechanisms, not intentions.
- **Reconnect.** Cursors per device, dedupe by client-generated message id,
  bounded catch-up, resumable idempotent attachment upload. Page refresh
  loses nothing only if the server acked after durable commit — check the
  ordering of those two events in their flow.
- **Residency.** Per-region cells holding content; global services limited to
  metadata (availability, skill tags, routing). The EU→US transfer question
  has no single right answer: serving the agent's session from the EU cell,
  or refusing cross-region transfer for EU companies, are both defensible —
  what is not defensible is not noticing that content would move. Erasure
  versus 7-year legal hold: expect the conflict named, the hold winning, and
  the option of destroying per-company keys so retained data is unreadable.

### Reference envelope — Threadline (Case C, speed round)

- posts 8M/day ≈ 80/s average, ~240/s peak; feed opens 40M × 6 = 240M/day
  ≈ 2,400/s average, ~7,000/s peak.
- fan-out on write ≈ 8M × ~200 followers ≈ 1.6 × 10⁹ inserts/day ≈ 16,000/s
  average — feasible. One 25M-follower post is 25M writes on its own, which
  is the number that forces a hybrid.
- **Intended answer shape:** fan-out on write into bounded per-user inboxes
  for ordinary accounts, pull-at-read for the few thousand huge accounts,
  merged when the feed is rendered. Inbox capped (~hundreds to 1,000
  entries), images on a CDN, feed page from cache.
- 30 minutes is not enough for depth everywhere — reward *sequencing*: did
  they estimate before designing, did the celebrity number appear, did they
  pick a deep dive on evidence, did they stop on time.

### Constraint injections

Use one per case, after they have defended the original design. Say it
plainly and let them work; silence is fine.

- Trackly: "Legal just moved takedown from 60 seconds to 5, worldwide." /
  "A customer imports 50 million links in one hour." / "Invoiced clicks must
  now exclude bots retroactively for the past 12 months."
- Huddle: "One company signs with 6,000 agents and becomes 40% of your
  traffic." / "The regulator says EU transcripts must never be readable
  outside the EU, including by your on-call engineers." / "Product wants live
  typing indicators for every participant."
- Threadline: "A celebrity posts once a minute for an hour." / "The feed is
  now ranked by a model costing 50 ms per candidate."

Grade the *response shape*: recompute the affected number, name which
component changes, name what the change costs, and say what you would now
cut. A learner who answers a 5-second takedown by keeping their 30-second TTL
has not understood their own mechanism.

## Common misconceptions

- **Transcription.** The lesson's shortener recited over Trackly: immutable
  mappings, an invalidation-free cache, 301s. The tell is a design that would
  be identical without the brief's twist.
- **"The connection tier is always the hard part."** Huddle's own numbers say
  otherwise. Generalizing scale from a remembered example instead of from the
  estimate is the same error as transcription, one level up.
- **Sizing to the average.** Trackly's 4,000/s average is irrelevant; the
  40,000/s spike is the system. Watch for capacity plans that quietly use the
  mean.
- **Purge as a guarantee.** Treating a best-effort invalidation message as if
  it were a contract, with no TTL backstop and no answer for a lost purge.
- **Exactly-once thinking.** "The queue will deliver each click once."
  Exactness comes from durable capture plus deduplication, or it does not
  exist.
- **Confusing ordering with time.** Sorting a conversation by timestamps,
  server or client. Both are broken; sequence numbers are the mechanism.
- **Per-user queues for offline delivery.** Reinventing a broker per user
  when the durable log plus a cursor already solves it — and creates an
  unbounded thing they will not notice until the 10× drill.
- **"Add a cache" / "shard it" as answers** with no hit rate, no key, and no
  cost.
- **Compliance as a footnote.** Residency and retention treated as ops
  concerns rather than as partitioning and lifecycle decisions.

## Grilling points

- "Point at the clause in Brief A that makes 301 wrong. Now walk a takedown
  through a browser that cached one."
- "Your TTL is 30 seconds. Compute the origin request rate that implies, then
  tell me what you would change if the origin can't take it."
- "Which of your Trackly components would disappear if links were immutable?
  That set is the twist you were graded on."
- "Your Huddle estimate says a handful of gateway boxes. So why did you spend
  a deep dive there — or, if you didn't, what did the connection tier cost
  you in effort you could have spent on retention?"
- "A conversation is transferred twice and a supervisor joins. Tell me the
  sequence numbers each participant sees, and whether any two disagree."
- "Your assignment mechanism: two assigners, one free agent, same
  millisecond. What actually prevents a double assignment?"
- "Attachments are a petabyte over seven years. What is the monthly bill
  shaped like, and which tier decision moves it most?"
- "In the speed round, which line of your estimate chose your deep dive? If
  you can't point at one, you chose by taste."
- "Same functional spec, different numbers, different systems — Huddle versus
  the lesson's messenger. Name the two non-functional requirements that did
  the most damage to the resemblance."

## Grading rubric

Grade estimation habits, composition, and defense under pressure — never
recall of a reference design.

- **A** — All three worksheets complete (speed round complete or honestly
  timeboxed with the gap named). Every number traces to a stated assumption
  with visible arithmetic; both dominant axes are correct *and* used to pick
  the deep dives; the twist in each brief visibly shaped the design (mutation
  propagation in Trackly, retention/residency and routing in Huddle); trade-
  offs carry honest costs on the picked side and flip conditions that survive
  attack; the 10× answers name a component, a first symptom, a next step, and
  its cost; the injected constraint produced a recomputation, not a bolted-on
  box. Bonus, not required: spotting Huddle's capacity contradiction
  unprompted.
- **B** — Designs are sound and the method is visible, with one or two
  lapses: a bare number, a dominant axis defended weakly, one deep dive that
  the estimates did not justify, a flip condition that is vague, or an
  injected constraint absorbed with one nudge. Recovers under questioning.
- **C** — The structure is present but the reasoning is thin: arithmetic they
  cannot re-derive, one design visibly transcribed from the lesson, "add a
  cache" as a bottleneck answer, or compliance treated as a footnote. Pass
  only if a focused redo of one section lands in this session; otherwise
  iterate.
- **Fail** — A worksheet materially incomplete (outside the timed sheet);
  numbers back-filled to justify a design already drawn; the brief's twist
  absent from the design; cannot say what breaks at 10× beyond naming a
  component. Remediate, do not advance — design-capstone is this lesson
  without scaffolding.

## Remediation ladder

1. "Take Brief A and list every sentence that would be false about the
   shortener in the lesson. That list is your design's job description —
   which items does your worksheet actually address?"
2. "Cover your estimate sheet. Re-derive redirect QPS and the campaign spike
   ratio out loud, rounding as you go. Now say which one you would build
   for." (Rebuilds the arithmetic habit and the peak/average distinction in
   one move.)
3. "Pick your weakest deep dive and write only the decision table: options,
   costs per option, pick, flip condition. Nothing else." (Framing is the
   skill; prose hides missing framing.)
4. "Run the 10× drill on Huddle out loud with me: singleton, hot partition,
   unbounded thing, synchronous fan-out, coordination point. Say 'none' where
   there is none — an honest 'none' is worth more than an invented answer."
5. If two or more of the above stall, send them back to redo one full case
   from a clean sheet with a 45-minute timebox, then review only that one.

## After passing

Preview: "Last lesson of the stage — design-capstone. One brief, no
worksheet, no scaffolding: you produce the full design document and defend it
end to end, with constraints arriving while you defend it."
