# Architecture Patterns

> `shared.systems.architecture` · ~2-3h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Argue for a modular monolith versus microservices for a given team size
  and product stage, citing operational and organizational costs.
- Explain event-driven architecture and identify where it buys decoupling
  and where it costs debuggability and consistency.
- Describe hexagonal (ports and adapters) architecture and show how it
  keeps domain logic independent of frameworks and databases.
- Identify which pattern an existing codebase follows and propose an
  incremental migration path when it no longer fits.

## Patterns are trade-offs with names

Everything in this stage so far handed you a mechanism: queues, caches,
replicas, hash rings, SLOs. This lesson is about the recurring
*arrangements* of those mechanisms — the shapes systems settle into and
the names practitioners use for them. Two warnings before the tour:

First, a pattern is graded by **fit**, never by modernity. "Microservices"
is not a compliment and "monolith" is not a diagnosis. Each pattern buys
something by paying something, and the right question — the design-intro
question — is whether *your* team, at *your* product stage, under *your*
load, comes out ahead on the exchange.

Second, more architecture decisions are about people than about
computers. Conway's law: organizations produce designs that copy their
communication structure. A four-team company will end up with roughly
four-part software whatever the diagrams say, so team size and team
boundaries are design inputs exactly as real as QPS — your worksheets in
this lesson use both kinds of number.

A working definition for the whole lesson: architecture is the set of
decisions that are *expensive to change*. That is why every section below
keeps asking not "is this good?" but "what does it cost to undo?"

## One deployable, or many

Start by separating two axes that get conflated constantly:

- **Logical boundaries** — modules: which code may call which, who owns
  which data.
- **Deployment boundaries** — services: what ships, scales, and fails as
  an independent unit.

A **monolith** is one deployable. That says nothing about its insides: a
monolith with no enforced logical boundaries is the famous big ball of
mud, while a **modular monolith** enforces real module boundaries —
checkout may only call catalog's published interface, never reach into
its tables — inside a single deployable. **Microservices** promote the
logical boundaries to deployment boundaries: separate processes with a
network between them.

The monolith gives you things you only notice after leaving:

- Transactions across modules — reserve inventory and record the order
  in one commit, for free.
- Refactoring across boundaries with the compiler's help — move a
  function, fix the callers, done in an afternoon.
- One debug surface — a single stack trace crosses every "service".
- One thing to deploy, monitor, and page on.

Microservices charge for each of those, and the bill is itemized. Price
it before you sign:

- Every call across a boundary becomes a **network call**, importing the
  entire distributed-systems lesson voluntarily: timeouts, partial
  failure, retries — and retries demand idempotency (api-design).
- Cross-service **transactions are gone**. What was one commit becomes a
  saga or an outbox-driven workflow (message-queues lesson).
- **Operational surface multiplies**: every service wants its own
  pipeline, dashboards, alerts, capacity plan, and a share of on-call.
  You operated *one* service in S5; multiply that honestly.
- Every boundary needs **contract discipline** — versioning, tolerant
  readers, deprecation — applied *internally* now.
- And the organizational prerequisite: a service with no clear owning
  team is an orphan that pages whoever is unlucky.

What the spending buys: independent deploys (no shared release train),
independent scaling per hot spot, fault isolation (a *bulkhead*: like a
ship's watertight compartments, a flooded service does not sink the rest —
the process-level cousin of the circuit breakers and load shedding from the
reliability lesson), and team autonomy — including
different datastores or languages where justified.

So the decision drivers, in the order that usually settles it:

1. **Team count and structure.** A handful of teams stepping on each
   other rarely justifies the bill; a heuristic with teeth: services
   should not outnumber teams while you are small. Conway's law works in
   both directions — service boundaries that cut across team boundaries
   generate permanent coordination overhead.
2. **Deploy contention** — and it is measurable: changes per release ×
   rollback frequency = unrelated work reverted per incident.
3. **Differential load** — one module at 100× everyone's traffic,
   wanting a datastore the rest don't need, is a genuine extraction
   driver. Check it against your estimates, not your feelings.
4. **Domain stability.** Before product-market fit, boundaries churn —
   and a wrong *module* boundary costs a refactoring afternoon while a
   wrong *service* boundary costs API versioning plus data migration
   plus cross-team choreography.

That asymmetry is the argument for the default: **modular monolith
first**, extract a service when a specific driver pays a specific bill.
The catch — extraction is only cheap if the module boundary was kept
clean. Modules are the rehearsal for services; a codebase that skipped
the rehearsal extracts a distributed ball of mud.

## Event-driven: decoupling you can measure

The patterns above arrange code; event-driven architecture rearranges
*communication*. The distinction that carries the section: a **command**
asks a specific party to do something and cares about the answer; an
**event** states a fact — `order.placed` — and whoever cares reacts.

A synchronous call chain couples the caller to each callee three ways:
temporally (all must be up *now*), by failure (their outage is your
outage), and by knowledge (the caller lists every party to notify).
Publishing an event through a broker cuts all three: the producer
records the fact and moves on.

What that buys, concretely:

- **Failure isolation.** If confirmation email is a consumer of
  `order.placed` rather than a step inside checkout, the email
  provider's bad day stops reaching your checkout SLO. The queue
  absorbs the outage; redelivery drains it afterward — mechanics you
  built in the message-queues lesson.
- **Latency.** Side effects leave the request path; the user waits only
  for what must be synchronous.
- **Additive extension.** A new consumer — fraud analytics, a data
  warehouse feed — subscribes without touching the producer.
- **Spike absorption.** The queue is a buffer; consumers drain at their
  own rate with backpressure (scalability lesson) instead of falling
  over together.

And the costs, which land on debuggability and consistency:

- **No call stack.** "What happened to order 123?" is now a correlation
  exercise across producer and consumers — tracing and correlation IDs
  (your S5 observability) stop being optional. Without them you get
  event soup: behavior nobody can trace end to end.
- **Eventual consistency downstream.** The search index and the
  dashboard now *lag* the order table. That is often fine — but "how
  stale may it be?" becomes a requirement you must state per consumer,
  exactly like the staleness questions in the data-storage lesson.
- **At-least-once delivery** means every consumer must be idempotent —
  dedupe on an event ID — or your retries send three confirmation
  emails (message-queues lesson, now applied at architecture scale).
- **The dual-write problem returns.** Committing the order to the
  database and publishing the event are writes to two systems; crash
  between them and they disagree. You know the fix: the outbox pattern
  — event written in the same transaction as the order, relayed to the
  broker afterward.
- **Event schemas are contracts.** Consumers you don't know parse them,
  so the api-design discipline — additive changes, tolerant readers —
  applies to every event you publish.

One vocabulary pair for multi-step flows: **choreography** lets a
workflow emerge from chained events (each consumer reacts, no owner —
flexible, but "where is order 123 stuck?" has no single answer) while
**orchestration** gives the workflow an owner that issues commands and
tracks state (a box you can point at — and a box that can fail). Rule of
thumb: independent fan-out side effects choreograph well; multi-step
workflows with compensation — refund, cancel, retry — deserve an
orchestrator.

## Hexagonal: the pattern inside one box

The first two sections argued about boxes and arrows *between*
deployables. Hexagonal architecture — also called ports and adapters —
is about the inside of one box, and it is orthogonal to the
monolith/microservices question: it applies to a module in a monolith
and to a single microservice identically.

Three roles:

- The **domain core**: your business rules — pricing, ordering,
  eligibility — expressed in plain code that imports nothing about
  HTTP, SQL, brokers, or vendor SDKs.
- **Ports**: interfaces *owned by the domain*, stating what it needs
  from the world (an order store, a payment gateway, an event
  publisher) and what it offers to the world (place order, get order).
- **Adapters**: the outer ring that connects ports to real technology —
  an HTTP handler adapting requests onto the inbound port, a database
  repository implementing the outbound store port in SQL, a broker
  publisher implementing the events port.

The load-bearing rule is the **dependency rule**: source dependencies
point inward. The core knows its ports; it never names an adapter. The
day the framework or the database appears in a domain type, the rule is
broken and every cost below comes back.

What the rule buys:

- **Testability without infrastructure.** The domain runs against
  in-memory fakes — no database container, no HTTP server, no broker.
  You have practiced exactly this since S3: injected clocks, fake
  stores, `httptest` at the edges only. Hexagonal is that habit
  promoted to a named structure with an enforceable rule.
- **Swappable edges.** Postgres to another store, or adding a gRPC
  adapter next to the HTTP one, touches adapters only.
- **Deferral.** A module with clean ports already has its seam. If it
  ever becomes a service, the port becomes the network boundary — which
  is why hexagonal-inside-a-modular-monolith is the combination that
  keeps future extraction cheap.

The costs are honest but real: indirection (every capability passes
through an interface), and mapping — domain object, database row, and
API shape are separate representations you convert between. For a
forms-over-data CRUD module that ceremony repays nothing. Spend the
hexagon where the rules are real — checkout, pricing, settlement — and
skip it where they aren't. That, too, is a framed trade-off.

**In Go:** consumer-defined interfaces make the dependency rule nearly
free — the domain package declares the small interfaces it needs, and
adapters implement them without being imported:

```text
orders/           domain: Order, PlaceOrder; ports: Store, Payments, Events
orders/postgres/  adapter: implements orders.Store using pgx
orders/httpapi/   adapter: handlers mapping JSON onto orders calls
main.go           wires concrete adapters into the domain at startup
```

`orders` imports neither `net/http` nor `pgx` — check its import list
to *verify* the dependency rule. Its tests use an in-memory `Store`
and a fake clock: the same shape as your S5 service tests.

## Reading a codebase, and leaving one

You will join far more architectures than you will found. Diagnosis
first — evidence, not vibes:

- **Count deployables.** One unit or many is a fact, not an opinion.
- **Map the dependency graph.** Who imports whom, and in which
  direction? Cycles and everybody-imports-everybody are the mud
  signature regardless of folder names.
- **Find the transactions.** A commit that writes two "modules'" tables
  is real coupling, whatever the diagram claims.
- **Check table ownership.** Two modules writing the same table are one
  module in disguise; a "microservice" sharing its database with
  another has the operational costs of two services and the coupling of
  one.
- **Classify the edges.** Synchronous call or event? The synchronous
  ones are your failure-coupling map.

Then the symptoms that a pattern no longer fits — put numbers on them,
because "the monolith is slowing us down" is not evidence:

- Release-train arithmetic: merges per release × rollback rate =
  unrelated changes reverted per bad deploy.
- A hot module forcing the whole application to scale for one path's
  traffic.
- Incident blast radius: a non-critical dependency (email, analytics)
  taking down the core flow.
- Teams blocked on each other's reviews, tests, and deploy windows; a
  test suite whose duration reflects everyone's coupling.

When the evidence says move, move **incrementally**. The big-bang
rewrite fails on schedule for predictable reasons: a feature freeze
nobody honors, a moving target, and a single riskiest-possible cutover
day. The alternatives you should reach for instead:

1. **Draw the seam as a module boundary first** (branch by
   abstraction): introduce the port inside the monolith, move every
   caller onto it. No deployment change yet — pure refactoring with
   compiler support, and it already pays off in tests.
2. **Strangler fig**: stand up the new implementation beside the old,
   route a small slice of traffic to it — one endpoint, one read path —
   verify, widen the slice, let the old path atrophy, delete it. The
   routing layer from your networking lesson is the enabling mechanism;
   shadow traffic or dual-read diffing is the verification.
3. **Extract data last.** A service extracted while sharing the old
   database is still coupled where it hurts — schema changes now need
   cross-service coordination. Route access through the owning module's
   interface first; move the tables once nothing else touches them.
4. **Pick the first extraction by driver × safety**: a module with a
   genuine independent-scaling or ownership driver *and* low blast
   radius. Read-mostly paths with staleness tolerance beat the
   transactional core every time.

Every step ships to production and every step can be rolled back. A
migration you can pause indefinitely is a migration you will survive.

## Exercise

Open [`exercise/`](exercise/). You are the architect brought into
**Larder**, a seven-year-old marketplace running as a single
deployable at an inflection point. `brief.md` has the codebase
description and the numbers; `architecture-review.md` is the worksheet
you fill.

Acceptance criteria:

1. Estimates first: order QPS (average and peak), search QPS, event
   deliveries per second if checkout's side effects become events, the
   post-change checkout latency arithmetic, and the rollback blast
   radius — each with its assumption stated.
2. A diagnosis of Larder's current pattern with at least three
   pieces of evidence from the brief (dependency, transaction, or
   ownership evidence — not adjectives).
3. A deployment-shape decision (modular monolith / microservices /
   hybrid) argued from the four drivers, with organizational *and*
   operational costs priced on both sides, a rejected alternative with
   its cost, and a condition that would flip your choice.
4. An event seam for checkout: which steps stay synchronous and which
   become events, with the latency arithmetic; per-consumer delivery
   semantics, idempotency plan, and staleness tolerance; your answer to
   the dual-write problem; and the brief's email outage replayed under
   your design.
5. A hexagonal sketch of the orders module: named ports with their
   operations, adapters, two dependency-rule statements, and one test
   that becomes possible without infrastructure.
6. A migration plan of 5-8 ordered steps — each shippable, reversible,
   and with a verification signal — including a justified first
   extraction, a data-untangling step, and an explicit
   not-extracting-yet list.
7. A trade-off log with at least three entries of the form "chose X
   over Y; it costs us Z" naming real losses.

There is nothing to compile. When the worksheet is full, bring it to
your tutor — the verification is a design review, and the brief's
numbers are the ammunition you will be expected to fire.

## Further reading

- [Martin Fowler — MonolithFirst](https://martinfowler.com/bliki/MonolithFirst.html)
- [Alistair Cockburn — Hexagonal architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [Martin Fowler — Strangler Fig Application](https://martinfowler.com/bliki/StranglerFigApplication.html)
- [Shopify Engineering — Deconstructing the monolith](https://shopify.engineering/deconstructing-monolith-designing-software-maximizes-developer-productivity)
