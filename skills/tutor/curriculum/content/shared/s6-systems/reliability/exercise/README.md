# Exercise: take over Framepost's operations

Framepost — the product you scoped in the design-intro lesson — is live, and
you are its new operations owner. [`brief.md`](brief.md) holds the production
facts and a raw log of last Tuesday's incident. Work through the worksheets
**in order** — each feeds the next:

1. [`01-slos.md`](01-slos.md) — choose SLIs, set SLOs, compute the error
   budgets everything else spends.
2. [`02-alerts.md`](02-alerts.md) — triage the existing alerts, then design
   the pages your SLOs deserve.
3. [`03-incident.md`](03-incident.md) — turn Tuesday's mess into a timeline,
   a budget bill, and a blameless postmortem.
4. [`04-degradation.md`](04-degradation.md) — decide, in advance, how
   Framepost degrades when each dependency fails.

Ground rules:

- Fill the worksheets in place — replace every `…` with your answer. Keep
  the tables; add rows freely.
- **Show your arithmetic.** `0.1% × 43,200 min ≈ 43 min` earns credit; a
  bare `43 min` does not. Round aggressively, as always.
- Targets are numbers, not adjectives. "Reliable" and "fast" are the
  sponsor's words; yours have units and windows.
- Justify by ID. Alerts cite the SLO they guard (S1, S2, …); degradation
  choices cite the SLO they protect; action items cite the gap they close.
- Write postmortem prose about *systems*, not people. If a sentence needs a
  person's name to work, rewrite it until it doesn't.

Budget roughly 90-120 minutes for all four. When you are done, tell your
tutor: verification is an **operations review conversation** against your
worksheets. Expect to re-derive any budget number aloud, to defend each
page against "would you want to be woken for this?", and to argue every
degradation pick against its alternative.
