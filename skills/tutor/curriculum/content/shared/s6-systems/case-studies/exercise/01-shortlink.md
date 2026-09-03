# Case A — Trackly

Brief A. Budget ~70 minutes. Phases in order; fill every line marked
*assumption*, *arithmetic*, *cost*, or *flips when*.

The lesson's shortener assumed links are immutable and permanent. Trackly's
are neither. Every place that assumption was load-bearing is a place your
design must differ — find them.

## 1. Requirements and scope

### Functional

| # | Capability | Must / Nice | Why |
|---|---|---|---|
| F1 | … | … | … |
| F2 | … | … | … |
| F3 | … | … | … |
| F4 | … | … | … |
| F5 | … | … | … |

### Non-functional

Concrete targets only — "fast" and "reliable" are not entries.

| # | Property | Target | Source (brief clause or your assumption) |
|---|---|---|---|
| N1 | Redirect latency | … | … |
| N2 | Redirect availability | … | … |
| N3 | Propagation of edits/takedowns | … | … |
| N4 | Analytics freshness and accuracy | … | … |
| N5 | … | … | … |

### Out of scope for v1

- …

### Questions for the sponsor

Three, each with the line of your design that the answer would move.

| # | Question | What changes if the answer is X |
|---|---|---|
| Q1 | … | … |
| Q2 | … | … |
| Q3 | … | … |

## 2. Estimation

| # | Assumption | Value | Justification |
|---|---|---|---|
| A1 | Bytes stored per link (URL + metadata + edit history) | … | … |
| A2 | Live links (created and not expired) at any time | … | … |
| A3 | Distinct links receiving traffic in a given minute | … | … |
| A4 | Bytes per click event | … | … |
| A5 | … | … | … |

```text
creates:        2M/day / 10⁵ s          ≈ …/s avg,  peak ≈ …/s
redirects:      400M/day / 10⁵ s        ≈ …/s avg
campaign spike: (brief)                 = 40,000/s — ratio to average ≈ …
storage/year:   2M × A1 × 365           ≈ …
click events:   400M × A4               ≈ …/day, ≈ … over the 90-day window
hot working set: A2 × A1                ≈ …   → fits in … of cache
```

**Dominant axis** (one sentence, naming the line above that proves it):

> …

**Second axis, if you have one**, and why it is second:

> …

## 3. High-level design

Boxes and arrows as indented text. Label every arrow with what flows over
it. Every box must trace to an F or N above.

```text
…
```

Now walk two flows, step by step, with a latency figure per step:

**Flow 1 — a customer creates a link with the custom alias `spring-sale`,
which is already taken.**

1. …

**Flow 2 — a click on an edited link arrives at an edge location 20 seconds
after the edit.**

1. …

## 4. Deep dives

### D1 — Key generation and the alias namespace

| Option | How it works | Costs | Verdict |
|---|---|---|---|
| … | … | … | … |
| … | … | … | … |
| … | … | … | … |

- Collision or exhaustion arithmetic for your pick (show it):

  ```text
  …
  ```

- How generated keys and 2,000 reserved paths and customer aliases stay in
  one namespace without ever colliding: …
- What happens when two customers request the same alias in the same second
  (name the mechanism, not the intention): …
- **Chose … over …; it costs us …**
- **Flips when**: …

### D2 — Cache, propagation, and the 60-second rules

- Where a redirect is answered from, in order, with the hit rate you expect
  at each layer: …
- Your invalidation mechanism for edits, expiry, and takedown — and the
  arithmetic showing 60 seconds is met in the worst case:

  ```text
  …
  ```

- What happens when the invalidation for one edge location is lost. (An
  answer of "it retries" must say how the retry is bounded and what the
  ceiling on staleness becomes.)
- **Redirect status code** (301 / 302 / 307) with the reason, and what the
  brief's takedown clause does to the alternative: …
- Cache headers you send to browsers and proxies, and why: …
- Negative caching for keys that do not exist: …
- **Chose … over …; it costs us …** · **Flips when**: …

### D3 — Analytics: one pipeline or two

- The approximate path (dashboard counts, ±1%, ≤ 60 s stale): where the
  event is emitted, how it travels, where it is aggregated, what the
  dashboard reads.

  ```text
  …
  ```

- The exact path (usage-billed links, invoiced, auditable for 3 years).
  State plainly what "exact" forces that "approximate" did not: …
- Your handling of duplicate events under at-least-once delivery: …
- Where bot and crawler classification runs, and which of the two paths it
  changes: …
- **The pipeline is down for 20 minutes during a campaign.** What does the
  redirect do? What do the counts show afterwards? What does the invoice
  show? …
- **Chose … over …; it costs us …** · **Flips when**: …

## 5. Bottlenecks and 10×

Traffic grows 10×: 4M links/day, 4B redirects/day, campaign spikes of
400,000/s.

Run the drill and fill each row with a *component from your own diagram*:

| Drill check | Where it lives in your design | Breaks at what point |
|---|---|---|
| The singleton | … | … |
| The hot key or partition | … | … |
| The unbounded thing | … | … |
| The synchronous fan-out | … | … |
| The coordination point | … | … |

**What breaks first**: …

**First symptom on a dashboard**: …

**Next evolution step**: …

**What that step costs**: …

**Deliberately left out, and when that decision expires**: …

## 6. Design statement

At most 200 words, written for a reviewer who has not read the worksheet:
what you are building, the dominant axis, the two decisions you would defend
hardest, and the first thing that will break.

> …
