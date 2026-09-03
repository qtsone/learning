# Failure modes, degradation, and 10×

Build this by walking **your own diagram** from worksheet 1 — every box
and every arrow, including the boring ones (DNS, object store, config,
identity provider). A dependency missing from this table is the one that
will be missing from your design.

## 1 — Dependency and component inventory

| Component / dependency | "Down" looks like | "Slow" looks like | Blast radius | Detection signal | Degradation behavior | Non-negotiable threatened |
|---|---|---|---|---|---|---|
|  |  |  |  |  |  |  |
|  |  |  |  |  |  |  |
|  |  |  |  |  |  |  |
|  |  |  |  |  |  |  |
|  |  |  |  |  |  |  |
|  |  |  |  |  |  |  |

- Which single component, if unavailable, hurts the most — and what your
  design does about that concentration:
- Per dependency, fail **open** or fail **closed**, and why criticality
  decides it:
- Where you deliberately accept an outage rather than build a fallback
  (a fallback path you never exercise is a liability):

## 2 — Failure story: a partial outage

One region, one shard, one replica set, or one zone — pick the partial
failure that is most awkward for your design and write it as a narrative.
Include: what users see, what the system does automatically, what an
operator does, when the first alert fires, and what is *lost or delayed*
versus merely slow.

> …

## 3 — Failure story: correlated failure

Find the synchronizing event in your own design — a deploy that drops
every long-lived connection, a cache flush, an expiring token everyone
refreshes at once, a retry storm — and walk it through.

- The synchronizing event:
- What amplifies it (how many clients act at once; the arithmetic):
- What breaks first, and how the failure feeds back on itself:
- Your defenses, and the evidence that they are sized correctly:

## 4 — The poison input

The tenant, key, document, or device that is 1,000× the median.

- What it is, and roughly how big it can get:
- Which component notices first, and how:
- What you do about it — and whether that is a design change or an
  operational one:

## 5 — The 10× drill

- **Axis you are multiplying** (users? activity per user? concurrency on
  one key? something else?) and why that axis is the realistic one:

- **Re-run the arithmetic** with the new inputs:

```text
…
```

| Order | What breaks | Which limit and which resource | Fix | config change / design change / rewrite |
|---|---|---|---|---|
| 1st |  |  |  |  |
| 2nd |  |  |  |  |
| 3rd |  |  |  |  |

- The breakage that **cannot** be fixed by adding nodes, and why (hot
  spots and coordination do not average):
- What in your design would need a **rewrite** rather than a change, and
  how far away that day is at current growth:
- What you would measure *today* to know how close you are to the first
  breakage:

## 6 — Rollout and operations

- How this ships the first time (order of deployment, what runs in
  parallel with what, how you verify before cutting traffic over):
- The three signals you would put on a dashboard, and which of them
  pages:
- Your top SLO, its target, and the arithmetic for its error budget:
- What you would run a game day on before launch:
- Capacity headroom you keep, and the arithmetic behind the number:
