# Worksheet 3 — Tuesday's incident, done properly

Source: the raw log in [`brief.md`](brief.md). You are writing the
postmortem that should have been written. Blameless throughout: if a
sentence needs a person's name to work, it is describing the wrong layer.

## Timeline, classified

Rebuild the timeline and tag each entry: `impact-start`, `detection`,
`mitigation-attempt`, `mitigation`, `recovery`, or `other`. Add the two
derived numbers below with visible arithmetic.

| Time | Event | Classification |
|------|-------|----------------|
| 14:02 | … | … |
| … | … | … |

```text
time-to-detect   = … − …  = … min   (impact start → first human awareness)
time-to-mitigate = … − …  = … min   (detection → users healthy)
total user impact ≈ … min at ~…% error rate
```

## The budget bill

Against your S1 from worksheet 1, what did Tuesday cost? Show the chain.

```text
burn rate during impact = …% errors / …% budget ≈ …×
budget consumed = …× × (… min / 43,200 min) ≈ … % of the 30-day budget
```

One sentence: at this rate, how many Tuesdays does one 30-day window
tolerate before the budget policy freezes releases?

> …

## Impact summary

Two or three sentences a stakeholder could read: who was affected, how
badly, for how long, and what it cost in budget. No jargon, no blame.

> …

## What went well / what hurt

At least two of each — something *did* go well (find it in the log).

- Went well: …
- Went well: …
- Hurt: …
- Hurt: …

## Contributing factors

At least **3**, none of which is a person. Each one names a property of the
*system* — tooling, alerts, process, defaults — and says how it contributed.
"Root cause: bad deploy" is one factor at most; Tuesday has several.

1. …
2. …
3. …

## Action items

At least **4**. Each has an owner **role** (on-call, feed team, platform
team, support lead — not a name), a deadline, and the gap it closes
(`detection`, `mitigation`, `prevention`, or `communication`). At least one
item must close the communication gap.

| # | Action | Owner (role) | Deadline | Gap closed |
|---|--------|--------------|----------|------------|
| 1 | … | … | … | … |
| 2 | … | … | … | … |
| 3 | … | … | … | … |
| 4 | … | … | … | … |

## The rewrite

The manager wrote: *"Who approved the v417 deploy? This can't happen
again."* Rewrite it as the message a blameless culture sends — same
urgency, different target.

> …
