# Tutor notes — Architecture Patterns

## Where the learner is

Tenth lesson of S6, after reliability. Every mechanism this lesson
arranges is already theirs: queues with at-least-once delivery and the
outbox (message-queues), failure modes and consistency (distributed-
systems), statelessness and backpressure (scalability), SLOs and
graceful degradation (reliability), contract discipline (api-design).
What is new is the zoom level: organizational forces as design inputs,
and pattern choice as a priced trade-off. Verify is discussion: grade
from the filled `architecture-review.md` and the review conversation.
Grade the **habits** — numbers with stated assumptions, evidence-based
diagnosis, named trade-offs, choices defended under attack — never
pattern-name recall. This and case-studies are the direct rehearsal for
the design-capstone.

## Review protocol

Run it as an architecture review board, not a quiz. Read the whole
worksheet silently first; note the weakest section and start there.

1. **Estimates audit (section 0).** Recompute one number live.
   Reference arithmetic: 500k orders/day ≈ 500k/10⁵ ≈ 5 QPS average,
   ~15 peak — *deliberately unimpressive*; a learner who says "order
   write load is trivial, capacity is not why we change anything" has
   read the brief correctly. Search: 60M/day ≈ 600-700 QPS average,
   ~2,000 peak — ~120× order traffic, the only genuine capacity axis.
   Events: 4 consumers × 500k ≈ 2M deliveries/day ≈ 20-25/s average,
   ~70/s peak — any broker yawns, and saying it's small is the point.
   Post-seam checkout: inventory+order txn (250) + charge (350) +
   outbox insert (~tens of ms, same txn) ≈ 650 ms against the 800 ms
   budget. Bonus if they note p99s don't strictly add — the arithmetic
   is decision-grade anyway because it's an upper-bound sketch.
   Rollback blast: ~35 changes/release, 1-in-12 rolled back, 3/week ≈
   one mass revert of ~34 unrelated changes every month.
2. **Diagnosis probe (section 1).** Expected: big ball of mud in
   monolith clothing — folder-level modules, no enforced boundaries.
   Evidence they should cite: checkout reading catalog tables
   (dependency), a third of transactions writing multiple modules'
   tables (transaction), growth's analytics SQL on production
   (ownership). Push: "the folders say modular — why isn't it?" The
   answer must be mechanical (imports, transactions, table access),
   not aesthetic.
3. **Deployment-shape defense (section 2).** Expected shape: modular
   monolith as the spine — enforce boundaries, keep transactions —
   with at most surgical extractions (search is the defensible one:
   ~120× read traffic, wants a different datastore, read-only,
   60 s staleness tolerance). Full microservices for 26 engineers is
   8-10 services across 4 teams — make them price the on-call and
   contract overhead. Accept a contrary choice **if argued from the
   drivers**; then counterfactual-test whatever they chose: "headcount
   doubles to 60 in four teams-of-teams — flip?", "orders go 100× via
   a B2B API deal — which driver changes?", "a headcount freeze at 26
   forever — does your hybrid survive?". An answer that never flips is
   dogma — probe until something flips or the position collapses.
4. **Event-seam attack (section 3).** Sync/async split expected:
   inventory+order and charge stay inline (tolerance 2 plus the user
   must know the charge outcome); email, seller notification, search
   index, analytics become consumers (tolerances 3-5 license each).
   Then attack: replay the 90-minute email outage — checkout unaffected,
   backlog ≈ 90 min × ~20/s ≈ 100-150k events, drained on recovery;
   redelivery means duplicate emails unless the consumer dedupes on
   (order id, consumer) — if their idempotency column is empty or
   vague, this is where it shows. Dual-write: the outbox insert must
   commit **in the same transaction** as the order row; if they wrote
   "publish after commit" or "write to broker then DB", crash them
   between the two writes and watch the repair. Outbox-relay-down
   story: users notice nothing at checkout, emails and index updates
   lag, tolerance 3 (2-minute email) is the first SLO breached —
   bonus if they alert on outbox lag (reliability lesson tie-in).
5. **Hexagonal check (section 4).** Ports owned by the domain, named in
   business language (Store, Payments, Events — not PostgresClient).
   Probe ownership: "who owns the Payments interface — the domain or
   the payment adapter? Why does the direction matter?" (Domain owns
   it; otherwise vendor types leak inward and the vendor swap or the
   fake both become surgery.) Demand the mechanically checkable rule
   ("the orders module never imports the HTTP layer or the database
   driver") and the concrete test (place-order business rules against
   in-memory store + fake payment gateway). Their "where NOT to apply
   it" answer matters: growth's CRUD-ish notification preferences or
   similar — hexagon everywhere is a miss, not a flex.
6. **Migration plan walk (section 5).** Every step must be shippable
   and reversible — if step 1 is "rewrite checkout", stop them. A
   strong plan starts with boundary enforcement inside the monolith
   (branch by abstraction: introduce ports, break direct table access)
   before any process split, extracts search first (driver × safety:
   real scaling driver, read-only, staleness tolerance, dual-read
   verification is cheap), and moves data last. Probe: "how do you
   *prove* extracted search is correct before cutover?" (shadow
   traffic / dual-read diff). "When do catalog's tables stop being
   readable by checkout?" (after access is routed through the module
   interface; physical move later). "What protects never-charged-twice
   mid-migration?" (charge stays inside the one transaction-owning
   module; idempotency keys on the payment call — api-design callback).
7. **Trade-off log (section 6).** Reject straw men ("chose modular
   monolith over rewriting in a weekend"). Each entry names a genuine
   loss: e.g. keeping the monolith spine costs Catalog & Search their
   independent deploy cadence for another two quarters; the event seam
   costs read-your-writes on the seller dashboard; the outbox costs an
   extra table and a relay to operate.

## Common misconceptions

- **"Microservices are how you scale."** Here the write path is 5 QPS.
  The genuine drivers at Larder are deploy contention and failure
  isolation — organizational and reliability drivers, not capacity.
  Make them say which driver actually applies.
- **"The monolith is the problem."** The *mud* is the problem — the
  unenforced boundaries and cross-module transactions. Extracting a
  mud module yields a distributed ball of mud: same coupling, plus
  network failure modes. Boundary enforcement precedes extraction.
- **"Events make the system reliable."** At-least-once delivery makes
  duplicate side effects the *default*; without per-consumer
  idempotency the email outage becomes an email flood on recovery.
  Events move failures around and change their shape — the queue is a
  buffer, not absolution.
- **"Publish the event after the transaction commits — done."** Crash
  between commit and publish and the order exists but no email, no
  index update, ever. That is the dual-write problem; the outbox
  closes it because event and order commit atomically.
- **"Hexagonal = more layers."** It is a *direction*, not a count: one
  rule (dependencies point inward) and domain-owned interfaces. If
  their sketch is lasagna with the domain importing the ORM, the rule
  is broken regardless of layer count.
- **"Shared database between services is a pragmatic shortcut."** It
  silently reintroduces the coupling the extraction paid to remove:
  schema changes need cross-service coordination, and independent
  deployability is fiction.
- **"Choreography everywhere"** (or orchestration everywhere). Fan-out
  side effects choreograph well; a refund saga with compensation wants
  an owner. One rule for everything is the tell of pattern-matching
  over reasoning.
- **"Migration = rewrite, but carefully."** If any step is not
  independently shippable and reversible, it is a big-bang rewrite
  wearing a phased costume. The strangler works because the old path
  keeps serving until the new one is *proven*.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go
deeper):

- "Larder's CTO mandates full microservices anyway. What do you
  extract first, what last, and what do you refuse to extract until
  the org changes — and why in that order?"
- "Your event seam is live. A seller says: 'my dashboard showed the
  order 40 seconds after my buyer's confirmation email arrived.'
  Walk me through why, and whether it's a bug." (Consumer lag ordering
  is not guaranteed; it's within stated tolerances — but only because
  the tolerances were stated.)
- "Conway's law is usually quoted as fate. Use it as a *tool*: you
  know the org grows to 40 engineers — what team boundaries would you
  lobby for, given the architecture you want?" (Inverse Conway.)
- "Why does 'checkout reads catalog's tables' block extracting
  *catalog*, not just checkout? Who actually owns that data?"
- "The search extraction is done and the diff on dual-reads is clean
  for a week. What can still go wrong at cutover that shadow traffic
  never exercised?" (Writes/reindex path, cache warmth, failure
  handling under real load — shadow verifies results, not operations.)
- "Name the moment in your migration where you'd be most tempted to
  big-bang, and what you do instead."

## Grading rubric

- **A** — Estimates within an order of magnitude with assumptions
  stated unprompted, including the insight that order QPS is trivial
  and search is the only capacity axis; diagnosis cites mechanical
  evidence of all three kinds; deployment decision argued from the
  four drivers with honest pricing of its own costs and a real flip
  condition; event seam survives both failure stories, has
  per-consumer idempotency, and closes the dual-write with same-
  transaction reasoning; hexagonal ports are domain-owned with a
  mechanically checkable rule and a concrete infrastructure-free test;
  migration steps are individually shippable/reversible with named
  verification signals; counterfactuals move their answers for stated
  reasons.
- **B** — Sound design with gaps discussion closes: an estimate
  missing its assumption, diagnosis leaning on one kind of evidence,
  outbox present but atomicity fuzzy until probed, migration ordering
  right but verification signals thin, or a flip condition that needs
  prompting. Corrects cleanly when challenged.
- **C** — Worksheet filled but habits thin: pattern names without
  evidence, "microservices because scale" with 5-QPS writes
  unremarked, event seam that re-sends emails on redelivery, or a
  migration plan whose first step is a rewrite. Pass only if live
  remediation lands — make them redo the diagnosis and one failure
  story on the spot; otherwise another iteration on the worksheet.
- **Fail** — Empty or copy-pasted sections; cannot explain why
  publish-after-commit loses events or why extracting a mud module
  distributes the mud; deployment choice defended only by fashion
  after probing; or every trade-off is a straw man. Redo the relevant
  sections together before re-review.

## Remediation ladder

1. "Point at the two traffic numbers in the brief. Compute both QPS
   out loud. Which one is a problem, and what does that do to the
   'we need microservices to scale' theory?"
2. "List checkout's six steps. For each: does the buyer need its
   result before we can honestly say 'order placed'? The ones where
   the answer is no — what does the brief's tolerance list let you do
   with them?"
3. "Your server commits the order, then crashes before telling the
   broker. Walk your own design: what row exists where, who ever
   learns about this order, and what would have to be true for the
   email to still go out?" (Let them rediscover the outbox they built
   in message-queues.)
4. "Take your migration's scariest step. Split it until every piece is
   something you could ship on a Tuesday and revert on a Wednesday —
   what's the first piece?" Then walk the search extraction with them
   — seam, shadow reads, cutover, data move — and let them write the
   verification column themselves.

## After passing

Preview: "Next lesson you stop designing one system and start reading
the classics — a URL shortener and a chat system — as guided case studies,
then run a social feed against the clock, putting today's habits to work on
briefs you'll meet again in interviews and in the capstone."
