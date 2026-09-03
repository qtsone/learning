# Worksheet 4 — Design discussion plan

Assemble worksheets 1-3 into the plan you would follow in a one-hour design
discussion about Framepost. This is the sheet your tutor will drive the
review from.

## Agenda

Five phases, with your time budget (should total ~60 min) and, per phase,
the one thing you must have agreed/produced before moving on.

| Phase | Minutes | Exit condition ("we don't move on until …") |
|-------|---------|---------------------------------------------|
| 1. Requirements & scope | … | … |
| 2. Estimation | … | … |
| 3. High-level architecture | … | … |
| 4. Deep dives | … | … |
| 5. Bottlenecks & evolution | … | … |

## High-level architecture sketch

Text is fine (`client -> load balancer -> api -> …`), a Mermaid diagram is
fine too. Show the boxes and, separately, walk the two main flows step by
step. Every box must be justifiable by a requirement ID — expect to be
asked "which requirement pays for this box?".

```text
(sketch here)
```

**Flow: a user uploads a photo**

1. …
2. …

**Flow: a follower views their feed**

1. …
2. …

## Deep dives you would choose

Two, each justified by a line from worksheet 2 ("reads dominate 50:1, so
…"). You are *choosing and justifying* the dives here, not performing them —
later lessons in this stage give you the tools to go deep.

| Deep dive | Which estimate sends you there |
|-----------|--------------------------------|
| … | … |
| … | … |

## Bottlenecks & open questions

At least two: what breaks first at 10× your estimates, and any single point
of failure in your sketch. One line each on the earliest symptom you'd see.

- …
- …
