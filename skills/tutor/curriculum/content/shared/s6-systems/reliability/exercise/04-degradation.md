# Worksheet 4 — Degradation plan

Decide *now*, in writing, how Framepost behaves when each dependency fails
— not at 3 a.m. Toolbox from the lesson: deadlines, bounded retries with
backoff+jitter, fallbacks (stale/default/placeholder), load shedding,
circuit breaking, fail open vs fail closed.

For every scenario: which SLO is threatened (cite S1/S2/S3), the strategy
(usually a combination), what the user sees while degraded, what you
knowingly sacrifice, and what would flip your choice. "Return an error and
wait" is an answer — but only if you argue it beats every alternative.

## Scenario 1 — resize workers down for 2 h

The queue backs up; no ~300 KB feed renditions are produced. Originals
(~3 MB) still land in object storage; metadata writes still work.

| Threatened SLO(s) | Strategy | User sees | You sacrifice |
|-------------------|----------|-----------|----------------|
| … | … | … | … |

**Why this beats the alternatives (name at least one you rejected):**

> …

**Flip condition:**

> …

## Scenario 2 — feed cache cluster lost (cold restart, ~30 min to refill)

The replicas alone sustain roughly a third of peak feed read load
(brief.md). Assume it happens at peak.

| Threatened SLO(s) | Strategy | User sees | You sacrifice |
|-------------------|----------|-----------|----------------|
| … | … | … | … |

**Why this beats the alternatives:** (Careful: "just let all reads through
to the replicas" has a failure mode you studied in the caching lesson —
name it.)

> …

**Flip condition:**

> …

## Scenario 3 — push provider outage, requests hang until the 10 s timeout

Recall from brief.md that the upload service calls it *synchronously*
after a successful upload.

| Threatened SLO(s) | Strategy | User sees | You sacrifice |
|-------------------|----------|-----------|----------------|
| … | … | … | … |

**Why this beats the alternatives:** (Address explicitly: is notification
delivery worth failing J2 for? Fail open or fail closed — and why is
retrying harder the wrong first move?)

> …

**Flip condition:**

> …

## Scenario 4 — DB primary fails over (writes down 60-90 s, reads fine)

| Threatened SLO(s) | Strategy | User sees | You sacrifice |
|-------------------|----------|-----------|----------------|
| … | … | … | … |

**Why this beats the alternatives:** (At least consider: fail fast with a
client-visible retry, vs queueing uploads for later write-back. Both are
defensible — cost them honestly, including what "posted" promises the
user under each.)

> …

**Flip condition:**

> …

## The one-liner

Finish the sentence, citing SLO IDs: "Under partial failure, Framepost
always protects … , degrades … first, and never … ."

> …
