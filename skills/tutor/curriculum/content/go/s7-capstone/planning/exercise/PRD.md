# PRD — <project name>

> Copy to `projects/capstone/docs/PRD.md` and fill in. Date it; when you change
> it later, note what changed and why at the bottom. This document is the
> arbiter when you have to cut something at hour 40 — write it so it can settle
> an argument.

**Author:** … **Date:** … **Status:** draft | agreed

---

## 1. Problem

State it with **no technology nouns**. Who hurts, what it costs them today,
what they do instead, and how you would know it stopped.

> Today, … has to …, which costs them … . They tolerate it because … . We will
> know this worked when … .

**My learning goal** (separate from the problem — the technique you want to
practise, honestly stated):

> …

## 2. Users

Who this is for, and what they already know. If it is only you, say so and
describe yourself as a user: what do you have installed, what do you tolerate,
what would make you stop using it?

> …

## 3. Functional requirements

Numbered and stable. Five to nine. Each one is a thing the system *does*,
observable from outside, phrased so somebody else could tell whether it works.
`Must` / `Should` marks what survives a cut.

| ID | Requirement | Must/Should |
|---|---|---|
| F1 | | Must |
| F2 | | Must |
| F3 | | Must |
| F4 | | Should |
| F5 | | Should |

## 4. Non-functional requirements

Numbers, not adjectives. If you cannot put a number on it, either measure
something related or drop it — an unmeasurable requirement cannot be met or
missed. Say where each number came from (a measurement, a guess, an external
constraint).

| ID | Requirement | Number | Where the number comes from |
|---|---|---|---|
| N1 | | | |
| N2 | | | |
| N3 | | | |

Useful axes for a project this size: throughput on your own hardware, latency
for the interactive path, memory ceiling, startup time, data volume it must
still work at, recovery time after a crash.

## 5. Non-goals

At least five. At least two must be things a reasonable person would expect you
to build — the disappointing ones are the ones doing work. Tag each **Never**,
**Not now**, or **Someone else's**.

| # | We will not … | Kind | Why not |
|---|---|---|---|
| X1 | | Not now | |
| X2 | | Never | |
| X3 | | Someone else's | |
| X4 | | | |
| X5 | | | |

## 6. What "done" looks like

The demo you will give at the end of the stage, in five lines or fewer:
commands typed, output seen, why an observer should be impressed.

> …

## 7. Open questions

What you do not know yet, and what would settle each one. Anything here that
could invalidate the design belongs in `RISKS.md` with a spike.

- …
- …

---

## Change log

| Date | What changed | Why |
|---|---|---|
| | | |
