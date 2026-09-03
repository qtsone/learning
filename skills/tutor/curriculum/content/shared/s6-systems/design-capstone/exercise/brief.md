# Briefs — pick one

Take the default unless you have a reason not to. It is tuned so that the
obvious answer is wrong in an instructive way, and your tutor has
reference numbers for it.

---

## Default brief — Codraft

From the product sponsor:

> **Codraft** is where teams write together. You open a document, other
> people open the same document, and everyone sees everyone's edits as
> they type. Comments in the margin, version history you can roll back,
> and sharing controls per document. We have a working prototype and
> 2 million registered users on the waitlist product; we need the design
> for the real thing.

### Facts the sponsor confirmed (use these; do not re-derive them)

- **2,000,000 registered users.** Daily active share is not measured yet
   — assume it and say so.
- **12,000,000 documents** exist today, growing ~500,000 per month.
- Average document is **20 KB** of text; the 99th percentile is 400 KB.
- Prototype telemetry:
  - Clients batch keystrokes and send about **1 operation per second**
    while a user is actively editing; an operation averages **100 bytes**
    on the wire.
  - Median **editing session: 11 minutes**. Median **viewing session:
    4 minutes**.
  - A document has a median of **3 people with access**, 12 at the 90th
    percentile, and a handful of company-wide documents have thousands
    of readers.
  - Simultaneous editors on one document are usually **1**, occasionally
    5-10; the worst observed was **60** during a workshop.
- Traffic peaks at roughly **3× the daily average**, concentrated in
  business hours per region.
- The company runs in **two regions** today and will not add more this
  year.

### Non-negotiables (from the business)

1. An edit the client was told was accepted is **never lost**.
2. Everyone editing a document **converges on the same content** — no
   permanent divergence, no "my copy differs from yours" support tickets.
3. An edit is visible to other people with the document open in
   **≤ 1 s median, ≤ 3 s p99**.
4. Opening a document takes **≤ 500 ms at p95** (from request to
   readable content).
5. Version history is restorable for **90 days**.
6. Losing a document is a company-ending event.
7. Revoking someone's access takes effect within **60 s**, including for
   sessions they already have open.

### Out of scope

The identity provider's internals, billing, rich media embeds, native
mobile apps, and the internal mathematics of whichever convergence
approach you choose — you must pick one and defend the choice, but you
are not asked to prove it correct.

---

## Alternate briefs

Same rules; you supply the confirmed facts your tutor will hold you to,
by writing a "facts" list as specific as Codraft's before you start.

- **Ticketing for high-demand events.** 50,000 seats go on sale at a
  fixed instant; 500,000 people arrive in the first minute. No seat is
  ever sold twice, nobody holds a seat forever, and the queue is fair
  enough to defend publicly. *Hard axis: contention on scarce inventory.*
- **Dispatch for a ride service.** Match riders to nearby drivers in one
  metro area: 40,000 active drivers reporting location every 4 s,
  200,000 ride requests per day, a match promised within 10 s.
  *Hard axis: geospatial fan-out plus write-heavy state that is stale in
  seconds.*
- **Telemetry ingestion.** Accept 500,000 events per second from a
  million devices, keep them queryable for 30 days, and answer dashboard
  queries over the last hour in under 2 s. *Hard axis: volume, retention
  cost, and the read/write asymmetry.*

---

## Bringing your own

Allowed, and encouraged if you have a system you actually care about.
Clear it with your tutor first; it must satisfy all four:

1. **A hard axis** — fan-out, contention on shared state, ordering or
   consistency requirements, statefulness, or volume beyond one node.
   CRUD over a database with a login screen does not qualify.
2. **Estimable** — you can state user counts and activity rates that
   someone else can sanity-check.
3. **Two paragraphs of description and five non-negotiables**, written
   before you design anything. That list is graded like the rest.
4. **Reviewable without domain expertise** — if judging your design
   requires knowing your industry, nobody can review it.

Scope, not ambition, is what makes a capstone strong. Designing the
ledger a payment system posts to beats designing "a payment system".
