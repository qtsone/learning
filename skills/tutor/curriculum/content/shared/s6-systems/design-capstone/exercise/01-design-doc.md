# Design doc — <your system>

Fill every section in order. Short and reasoned beats long and hedged.
Anything marked *assumption*, *arithmetic*, or *requirement* is graded
content, not decoration.

## 1 — Context and goals

- What is this system, in three sentences a reviewer outside your team
  would understand:
- Who uses it and what do they get:
- **Non-goals** — at least 3 things this design deliberately does not
  do, each with one line on why deferring is safe:

## 2 — Requirements

### Functional

At least 6, marked `must` or `nice`. Capability sentences; no
architecture in this table.

| # | Capability ("a user can …") | must / nice | Notes |
|---|---|---|---|
| F1 |  |  |  |
| F2 |  |  |  |
| F3 |  |  |  |
| F4 |  |  |  |
| F5 |  |  |  |
| F6 |  |  |  |

### Non-functional

At least 5, each with a **numeric target**. Adjectives are not targets.
Mark each source as `brief` or `assumed`.

| # | Quality | Target (measurable) | Source | Why this target |
|---|---|---|---|---|
| N1 |  |  |  |  |
| N2 |  |  |  |  |
| N3 |  |  |  |  |
| N4 |  |  |  |  |
| N5 |  |  |  |  |

### Consistency and freshness, per feature

The distributed-systems lesson's discipline: the weakest model that still
satisfies the feature. One row per feature where it matters.

| Feature | Weakest sufficient model | Anomaly one step weaker | Why acceptable / not |
|---|---|---|---|
|  |  |  |  |
|  |  |  |  |
|  |  |  |  |

## 3 — Estimates

Every estimate traces to an assumption row. Show the arithmetic; round
aggressively. A day is ~10⁵ seconds.

### Assumptions

| # | Assumption | Value | Justification |
|---|---|---|---|
| A1 |  |  |  |
| A2 |  |  |  |
| A3 |  |  |  |
| A4 |  |  |  |
| A5 |  |  |  |

### The numbers

Cover, at minimum: the main write rate, the main read rate, storage
(total and growth), and whatever *concurrency* your system implies —
simultaneous connections, in-flight sessions, or contenders per key.
Average and peak for each.

```text
write path  = … × …            ≈ … /s average, … /s peak
read path   = … × …            ≈ … /s average, … /s peak
concurrency = … × … / …        ≈ … at average, … at peak
storage/day = … × …            ≈ …
retained    = … × retention     ≈ …            (before replication)
bandwidth   = … × …            ≈ … /s
```

### Per-key arithmetic

Aggregate load and per-key load are different systems. Compute the load
on the *busiest single* document, key, partition, or shard — and note
whether it is trivial or the whole problem.

> …

### Sanity checks

Two, in prose, each comparing a result to an anchor you trust (the
design-intro table, or your own S5 measurements — say which).

1.
2.

### Dominant axis

One sentence: which quantity dominates this design, and which line above
proves it. Name any axis that turned out to be unimpressive — that is a
finding too.

> …

## 4 — Architecture

### Diagram

Text or Mermaid. Every box gets an annotation naming the requirement (by
ID) that pays for it.

```text
(sketch here)
```

| Box | What it owns | Requirement that pays for it |
|---|---|---|
|  |  |  |
|  |  |  |
|  |  |  |
|  |  |  |

### Flow 1 — <the most important write path>

Step by step, from client to durable, including what the client is told
and when.

1.
2.
3.

### Flow 2 — <the most important read path>

1.
2.
3.

### What is stateful, and where does state live

Name every component that holds state a request depends on (sessions,
in-memory documents, leases, connections). For each: what happens to
that state when the process holding it dies?

- …
- …

## 5 — Data model

| Entity | Key | Partitioned by | Retention | Access patterns it serves |
|---|---|---|---|---|
|  |  |  |  |  |
|  |  |  |  |  |
|  |  |  |  |  |
|  |  |  |  |  |

- Which store type each entity lives in, and why that type (not which
  vendor):
- The one query you expect to be expensive, and what makes it fast (index,
  denormalization, precomputation, cache):
- What is written on the hot path versus asynchronously, and how the
  asynchronous side cannot silently lose work:
- How retention is enforced (compaction, tiering, deletion), and the
  arithmetic showing it keeps storage bounded:

## 6 — API sketch

The handful of calls that carry flows 1 and 2. Contracts, not
implementations.

| Call | Request (key fields) | Response | Errors that matter |
|---|---|---|---|
|  |  |  |  |
|  |  |  |  |
|  |  |  |  |

- Protocol choice per call (request/response, streaming, long-lived
  connection) and what each costs:
- **Retry semantics**: a client sends a request, times out, and retries.
  For each call above, what prevents a duplicate effect:
- Pagination or streaming for anything unbounded:
- How the contract evolves without breaking existing clients:
