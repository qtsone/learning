# Tutor notes — API Design

## Where the learner is

Third lesson of S6, second discussion-verify in the stage (design-intro
set the requirements/estimation/trade-off habits — hold them to those
here). They have *built* both API styles: S3/S5 gave them production
HTTP services, gRPC, databases, and operations. So mechanics are cheap
for them; what is new is contract thinking — designing for consumers
they don't control and for change over time. Verify is discussion: grade
from the filled `api-sketch.md` and the review conversation. Grade the
**habits** — numbers with stated assumptions, named trade-offs, choices
defended under attack — never trivia recall.

## Review protocol

Run it as a design review, not a quiz. Read the whole worksheet silently
first and note the weakest section — start the conversation there.

1. **Estimates audit (section 0).** Recompute one of their numbers live
   with them. Reference arithmetic: 5M charges/day ≈ 58 QPS average;
   ×4 peak ≈ 230 QPS. Reads at 10× ≈ 580 average / ~2,300 peak.
   Storage: 5M/day × 1 KB ≈ 5 GB/day ≈ ~2 TB/year. Idempotency keys at
   24 h retention: ~5M keys; with a stored response of ~1 KB, single-digit
   GB — small, and *saying* it's small is the point. Order-of-magnitude
   agreement is fine; a missing assumption is not.
2. **Protocol defense (section 1).** Expected shape: REST for merchants
   (2,000 polyglot integrators + a browser dashboard = audience and
   tooling dominate), gRPC internally (Go-to-Go, fraud sits on the hot
   path of every charge with a 150 ms budget — cheap encoding and HTTP/2
   multiplexing are actual drivers; settlement's bulk reads fit
   streaming). Accept a contrary choice **if argued from the drivers**;
   then counterfactual-test it: "the merchant surface is now consumed
   only by two partner companies you co-develop with — does your answer
   flip?" and "fraud scoring moves to a third-party vendor — now what?"
3. **Versioning stress-test (section 3).** Walk the change drill against
   the reference below. Wherever they wrote NB, ask "which client does
   this break anyway?"; wherever B, demand the escape route.
4. **Pagination probe (section 4).** Make them narrate the mid-walk
   insert scenario without notes. Then: "your sort key is `created_at`
   alone — two charges in the same millisecond; what does the walk do?"
   (skip or repeat at the boundary; hence the id tie-break). Then the
   product-pressure test: "the dashboard PM wants 'jump to page 500'."
   (Expected: push back or hybrid — offset for shallow dashboard pages,
   cursor for reconciliation walks; naming the split is an A-signal.)
5. **Idempotency attack (section 5).** Run the crash story if their
   worksheet didn't: execute → crash before response → retry. Their
   scheme must recover the *stored or reconstructible* outcome, which
   forces key-recording and charge-execution into one atomic commit —
   if key storage is a separate cache (e.g. "Redis then the DB"), ask
   what happens when the crash lands between the two writes. Follow
   with: retry arrives 25 h later, after key expiry (duplicate charge —
   what do they do about it: longer retention, client contract, or
   accept-and-document); and two requests, same key, in flight at once.
6. **Trade-off log (section 7).** Reject straw men ("chose REST over
   carrier pigeons"). Each entry must name a loss they actually accepted.

## Reference answers — change drill

1. **NB** — *if* the tolerant-reader rule is written into their
   contract (section 3 asks for it). If they marked NB but wrote no such
   rule, that's the probe: safe by whose promise?
2. **B** — rename is remove + add; removal breaks parsers. Escape:
   add `amount_minor_units` alongside, dual-publish, deprecate `amount`,
   remove only at a major bump (or never, under additive-only).
3. **B — the most dangerous row.** No parser breaks, every amount is
   silently 100× off. Meaning changes are invisible to tooling; the
   escape is a *new name* (which makes #2's pattern the fix), never an
   in-place semantic swap. A learner who ranks this less severe than #2
   has it backwards — dig in.
4. **B** — old clients' previously-valid requests now rejected. Escape:
   optional with a documented default (`USD`), telemetry on who omits
   it, tighten later if ever.
5. **NB** under every policy — new surface, nobody's parser sees it.
6. **The judgment row.** Formally additive; in practice merchants
   `switch` exhaustively on `status` and an unknown value hits their
   `default: panic` path. Either classification passes *with* the
   insight: the contract wording decides — "new statuses may appear;
   treat unknown as X" makes it NB, its absence makes it B. Bonus tie
   to protobuf enums, where unknown values arrive whether you like it
   or not.
7. **B** — tightened validation rejects previously-accepted input.
   Escape: announce + measure affected merchants + sunset window; or
   soft-fail (accept, warn) first.
8. **B, wire corruption** — old binaries decode `risk_flags` bytes as
   `risk_notes`. Field numbers are the contract; the deleted number
   should have been `reserved`. Escape: reserve 4, take a fresh number.

## Common misconceptions

- **"gRPC is faster, therefore gRPC everywhere."** Speed is real
  (binary encoding, HTTP/2) but the merchant surface is decided by the
  adoption tax on 2,000 polyglot teams. Make them price the tax.
- **"PUT = update, POST = create."** The real line is the semantics:
  PUT is idempotent full-state replacement, wherever it's used. If
  their table has a PUT that appends, the protocol's retry promise is
  broken — proxies may retry it.
- **"Idempotent = returns the same response."** It's about state:
  DELETE returning 204 then 404 is idempotent. (The idempotency-*key*
  scheme returning stored responses is a stronger, contractual add-on —
  keep the two ideas distinct.)
- **"We're on /v1, so we can change things within it."** /v1 is the
  contract; the version prefix is the escape hatch for breaking
  changes, not a license for them.
- **"Adding a field is always safe."** Only under tolerant readers —
  and enum/status growth breaks exhaustive matchers (drill #6).
- **"Offset drift is fixable with a transaction."** Each page is a
  separate request; no sane server holds a snapshot open across a
  20M-row nightly walk by an external client. The fix is naming a
  position, not freezing the world.
- **"The cursor is just the last row's id."** Works only when sorting
  by id; under `created_at` ordering it must carry the composite key.
  And if it's readable, merchants will mint their own — opacity is
  Hyrum-proofing.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go
deeper):

- "Your reconciliation client crashes halfway through the nightly walk.
  What does it need to have persisted to resume without loss?" (Just
  the last cursor — and that property falls out of position-not-count.)
- "Why does the idempotency record and the charge have to commit
  atomically? Walk the two orderings of separate writes and break each."
- "Stripe has run on /v1 for over a decade. Under which versioning
  strategy from the lesson, really? What does that tell you about URI
  numbers vs change discipline?"
- "Rank drill rows 2 and 3 by blast radius and by detectability. Which
  one pages you at 3 a.m. a month later?"
- "Your PATCH for updating a merchant webhook URL — idempotent or not?
  What would make it not?"
- "What in your error contract lets a merchant *program* against a
  declined charge vs a validation failure, without string-matching your
  messages?"

## Grading rubric

- **A** — Estimates within an order of magnitude with assumptions
  stated unprompted; both protocol choices argued from audience/tooling/
  performance with real rejected alternatives; drill ≥7/8 with the #3
  severity and #6 contract-wording insights; pagination design has
  composite key + tie-break + opaque token and the drift narrative is
  fluent; idempotency scheme survives the crash story and the atomicity
  probe; trade-off log names genuine losses; counterfactuals move their
  answers for stated reasons, not vibes.
- **B** — Sound design with a gap that discussion closes: an estimate
  missing its assumption, drill wobble on #6 or #7, tie-break missing
  until probed, or idempotency retention/scope hand-waved. Corrects
  cleanly when challenged.
- **C** — Worksheet filled but habits thin: adjectives where numbers
  belong, choices asserted without alternatives, drift narrated only
  with heavy prompting, crash story breaks their scheme and the repair
  needs to be fed to them. Pass only if live remediation lands — make
  them redo section 0 and the crash story on the spot; otherwise
  another iteration on the worksheet.
- **Fail** — Empty or copy-pasted sections; cannot explain why a retry
  without a key double-charges; classifies #3 or #8 as non-breaking and
  defends it after probing; or every "trade-off" is a straw man. Redo
  the relevant sections together before re-review.

## Remediation ladder

1. "Point at the number in the brief that your weakest section should
   have used. Now use it — out loud, rounding as hard as you like."
2. "Draw the transaction list as a column, newest on top. Mark where
   offset=100 points. Insert two rows at the top. Where does it point
   now? Which rows does the client see twice?"
3. "Your server charged the card and died before replying. The retry
   arrives. Ask your own scheme, step by step: what row exists, what
   does the unique constraint do, what goes back to the client?"
4. "For each drill row: name one concrete merchant integration and the
   exact line of *their* code that breaks. If you can't, it's NB — if
   you can, say what you'd ship instead." Then walk rows 3 and 8 with
   them — meaning-change and field-number reuse — and let them write
   the escape routes themselves.

## After passing

Preview: "You've designed the promises; next lesson is what keeps them —
where the data actually lives: SQL vs NoSQL, indexes at scale,
replication, and sharding, including the tables your cursor and
idempotency keys just assumed into existence."
