# Worksheet 1 — Requirements & scope

Source: [`brief.md`](brief.md). Turn the wish into requirements. Short
capability sentences; no architecture yet — nothing in this sheet should
mention servers, databases, or caches.

## Functional requirements

At least 5 total. Every row is marked `must` or `nice` — if you want to
argue a row could be either, the Notes column is where you argue it.

| # | Capability ("a user can …") | must / nice | Notes |
|---|------------------------------|-------------|-------|
| F1 | … | … | … |
| F2 | … | … | … |
| F3 | … | … | … |
| F4 | … | … | … |
| F5 | … | … | … |

## Non-functional requirements

At least 4, each with a **concrete target**, not an adjective. "Feels
instant" is the sponsor's phrasing — your job is to translate it into a
number you could later verify (e.g. "feed loads in < X ms at p95"). Where
the brief gives no target, propose one and mark it `assumed`.

| # | Quality | Target (measurable) | Source (brief / assumed) | Why this target |
|---|---------|---------------------|--------------------------|-----------------|
| N1 | Scale | … | … | … |
| N2 | Latency | … | … | … |
| N3 | Availability | … | … | … |
| N4 | Durability | … | … | … |

## Out of scope (version one)

List at least 3 things you are explicitly *not* building now, with one line
each on why deferring is safe. (Things the brief hints at but that would
sink a v1, or classic adjacent features the sponsor didn't ask for.)

- …
- …
- …

## Questions for the sponsor

At least 3 questions whose answers would change your design, and for each:
what you will **assume** until it is answered. Your tutor plays the sponsor
during review and will answer these — be ready to say which parts of your
design move when the answer surprises you.

| # | Question | Why it matters | Working assumption |
|---|----------|----------------|--------------------|
| Q1 | … | … | … |
| Q2 | … | … | … |
| Q3 | … | … | … |
