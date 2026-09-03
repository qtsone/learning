# Selection worksheet

Fill this in before the PRD. Score at least one candidate; two is better,
because the comparison is what makes your choice defensible. Keep the rejected
one — "why not that" is the first thing a reviewer asks.

Answer in your own words. "Yes" on its own is not an answer; each row wants the
*specific* thing in *your* project that satisfies it.

---

## Candidate A — <name>

**One-sentence pitch (what it does, for whom):**

> …

**The one flow that makes it worth having** — the single path from input to
value; if only this worked, would you still want the tool?

> …

### Rubric

| # | Requirement | Your answer | Pass? |
|---|---|---|---|
| 1 | Needs more than one meaningful package — name at least three you expect, and what each owns | | |
| 2 | Concurrency that earns its cost — what runs concurrently, and what breaks if it is all sequential | | |
| 3 | State that survives a restart — what is stored, and what is lost if the file is deleted | | |
| 4 | At least one interface with two implementations — name the interface and both | | |
| 5 | One command to run, value visible in under a minute — write the exact command | | |
| 6 | Needs no data, credentials or hardware you lack — list what it needs and confirm you have it | | |
| 7 | You want to build it — one honest sentence on why | | |

### Size check

- **Hours you actually have** for the core build (be pessimistic; count evenings
  you will lose): …
- **Your gut estimate for the core**, in hours: …
- If the second number is more than 60% of the first, what comes out?
  …

### Is it secretly three projects?

Answer each; any "yes" needs a sentence on what you are cutting.

- Does the pitch join two nouns serving different users? …
- Do you need more than one walking skeleton? …
- Would two of your ADRs cover parts that never touch? …
- Did the one-flow answer need more than one sentence? …

### The hard part

Where is the genuine difficulty, and why is it hard? (If the answer is "there
isn't one", this is a warm-up, not a capstone.)

> …

---

## Candidate B — <name>

*(Same sections. Delete this block only if you truly have one candidate — and
then say in review why nothing else was in contention.)*

---

## Decision

**Chosen:** …

**Why this one over the other** (cite rubric rows, not enthusiasm):

> …

**What I am giving up by not choosing the other:**

> …

**The first thing I will cut if I lose a third of my hours, and what the
project still does without it:**

> …
