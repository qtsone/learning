# Risk register — <project name>

> Copy to `projects/capstone/docs/RISKS.md`. Revisit it at every milestone
> review: close what did not happen, add what you learned, and note what fired.
>
> Delete this quoted block in your copy.

A risk is an **uncertainty with a cost**. "There might be bugs" is neither — it
is certain and unpriced. Each entry needs a cost in hours if it happens, a
likelihood you are willing to defend, an **early warning trigger** you would
actually notice, and a mitigation you can act on *now*.

At least five risks. At least one non-technical. At least one mitigation is a
timeboxed **spike**, scheduled before the milestone that depends on the answer.

| ID | Risk — what could happen | Cost (h) | Likelihood | Trigger — how I find out early | Mitigation — what I do now | Owner milestone |
|---|---|---|---|---|---|---|
| R1 | | | High/Med/Low | | | M1 |
| R2 | | | | | | |
| R3 | | | | | | |
| R4 | | | | | | |
| R5 | | | | | | |

## Spikes

A spike is a timeboxed throwaway experiment that answers **one** question. Give
each a box and a hard stop, and schedule it before the milestone that assumes
the answer. If the timebox expires without an answer, that *is* the answer:
the assumption is unsafe, and the plan changes.

| Spike | Question it answers | Timebox | Before | What "no" means for the plan |
|---|---|---|---|---|
| S1 | | 2h | M1 | |
| S2 | | | | |

The code from a spike is thrown away. Keeping it means you skipped the
experiment and started the milestone.

## Prompts, if the register looks thin

Not a checklist to copy — questions to ask yourself. Keep the ones that are
real for your project, priced in your hours.

- The library or API I am counting on cannot actually do the thing I assumed.
- The concurrency design does not survive contact with the race detector.
- The data format I picked turns out to be painful to query, migrate or grow.
- The interesting part is 80% of the work and I planned it as one milestone.
- I lose evenings to work, travel or illness and get half the hours I planned.
- I get bored at hour 30 because the remaining work is all plumbing.
- The scope grows back: something I wrote as a non-goal returns as "quick".

## Accepted risks

Risks you looked at and decided to live with, with the reason. This section is
not a failure — deciding *not* to mitigate is a decision, and writing it down
is what separates it from an oversight.

- …
