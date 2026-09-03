# Architecture review — Larder

Fill every section. Short and reasoned beats long and hedged. Lines
marked *assumption*, *evidence*, or *cost* are graded content, not
padding.

## 0 — Estimates first

Back-of-the-envelope, from the brief's numbers. Show the arithmetic;
round aggressively. Part of the exercise is noticing which numbers are
*small* — say so when they are.

- Order QPS, average and peak: <!-- number + one line of arithmetic -->
- Search QPS, average and peak: <!-- number + assumption -->
- Search-to-order traffic ratio, and what it tells you:
- Event deliveries per second if checkout's side effects become events
  (state how many consumers): <!-- number + assumption + is it big? -->
- Checkout p99 after your event seam — which steps remain inline, and
  what they sum to against the 800 ms budget:
  <!-- arithmetic; note any caveat about adding p99s -->
- Rollback blast radius today: unrelated changes reverted per bad
  release, and how often: <!-- number + arithmetic -->
- One estimate the brief does NOT ask for but your design depends on:
  <!-- name it, estimate it, state the assumption -->

## 1 — Diagnosis

What pattern is Larder today? Name it, then prove it — at least
three pieces of evidence from the brief, of different kinds
(dependency evidence, transaction evidence, ownership evidence).

- Pattern:
- Evidence 1:
- Evidence 2:
- Evidence 3:
- Which boundaries exist on paper but not in practice, and how you can
  tell:
- Where do transactions currently cross module lines, and why does that
  matter for any extraction you might propose later:

## 2 — Deployment shape

Your call: modular monolith, microservices, or a hybrid. Argue it from
the four drivers (team count/structure, deploy contention, differential
load, domain stability) — using section 0's numbers, not adjectives.

- Decision:
- Driver 1 — teams (include the services-per-team arithmetic at 26 and
  at 40 engineers):
- Driver 2 — deploy contention (use the rollback blast radius):
- Driver 3 — differential load (which module, and by how much):
- Driver 4 — domain stability:
- Organizational cost of your choice (who coordinates with whom, who is
  on call for what):
- Operational cost of your choice (pipelines, dashboards, alerts,
  contracts — priced honestly):
- Rejected alternative and what rejecting it costs us:
- What would flip this decision (a concrete change in team, product, or
  load):

## 3 — The event seam

Redesign order placement against the 800 ms budget and tolerance list.

- Steps that STAY synchronous, and why each must:
- Steps that become event consumers:

| Consumer | Triggering event | Staleness tolerance (cite the brief) | Idempotency mechanism | What happens on redelivery |
|---|---|---|---|---|
|  |  |  |  |  |
|  |  |  |  |  |
|  |  |  |  |  |
|  |  |  |  |  |

- The dual-write problem in this design — where it appears, and your
  mechanism for closing it (be specific about what commits together):
- Choreography or orchestration for these consumers, and why:
- Failure story 1 — the brief's incident, replayed: the email provider
  degrades for 90 minutes under YOUR design. Walk through what checkout
  does, what the queue does, what happens on recovery, and how many
  emails the backlog holds (use section 0):
- Failure story 2 — your outbox relay (or equivalent) is down for two
  hours. What do users notice, what does the business notice, and what
  drains when it returns?
- One new consumer the Growth team adds next quarter — what has to
  change in checkout's code?

## 4 — Hexagonal: the orders module

Sketch the ports-and-adapters structure for the orders/checkout module
(inside whatever deployment shape you chose in section 2).

- Domain core responsibilities (3-5 bullet points, business language
  only):
- Ports the domain OWNS:

| Port | Direction (inbound/outbound) | Operations (2-3 each) |
|---|---|---|
|  |  |  |
|  |  |  |
|  |  |  |

- Adapters implementing them:
- Two dependency-rule statements for this module, in the form "package/
  module X must never depend on Y" — rules a reviewer could mechanically
  check:
- One test that this structure makes possible without a database,
  broker, or HTTP server — name what it verifies and what fakes it
  uses:
- One place in Larder where you would NOT apply hexagonal, and why:

## 5 — Migration plan

From today's codebase to your target, with the product shipping
throughout. 5-8 ordered steps; each row must be shippable on its own
and reversible.

| # | Step | Shippable because | Reversible because | Verification signal |
|---|---|---|---|---|
| 1 |  |  |  |  |
| 2 |  |  |  |  |
| 3 |  |  |  |  |
| 4 |  |  |  |  |
| 5 |  |  |  |  |

- First extraction (if any) and the driver × safety argument for
  choosing it over the alternatives:
- How and WHEN the shared database untangles (what moves behind an
  interface first, what moves physically, and what stays):
- Explicit not-extracting-yet list, each with the condition that would
  change your mind:
- What in this plan protects tolerance 2 (never lost, never charged
  twice) while pieces are moving?

## 6 — Trade-off log

At least three, in the form "Chose X over Y; it costs us Z." Real
losses, not straw men.

1.
2.
3.
