# Issue triage and orientation

Three issues triaged, one chosen, reproduced, and understood — all before you
edit a line. Fill sections 1 and 2 first; section 3 is written *after* you have
read the code and *before* you change it.

---

## 0. Their process, in their words

Read `CONTRIBUTING.md` (and the PR template, and the last three merged PRs).
Summarize what it actually requires. If the project has no contribution docs,
say what you inferred from recent merged PRs and where you inferred it from.

| Question | Answer | Where it says so |
|---|---|---|
| How do I run the tests exactly as CI does? | | |
| Formatting and lint requirements (config file? `gofmt` only?) | | |
| Commit message convention (format, sign-off, squash or series?) | | |
| PR conventions (template, one-issue rule, draft first?) | | |
| Legal step: DCO, CLA, neither | | |
| Do they want an issue or a claim before a PR? | | |
| Anything they explicitly say *not* to send | | |

---

## 1. Three candidates

Run each through the five triage questions. Stop at the first "no" and say so —
a rejected candidate with a one-line reason is a complete answer.

### Issue 1 — `<link>` · title

| # | Question | Answer |
|---|---|---|
| 1 | Still real? Reproduced on the current default branch? | |
| 2 | Claimed — anyone in the thread or in an open PR referencing it? | |
| 3 | Does a maintainer want it, and have they said what should happen? | |
| 4 | In scope? (README non-goals, similar PRs closed with a reason) | |
| 5 | Size: diff you imagine + test + docs + two review rounds, in hours | |

**Verdict:** …

### Issue 2 — `<link>` · title

*(Same table.)*

### Issue 3 — `<link>` · title

*(Same table.)*

---

## 2. Chosen issue

**Link:** … **Type:** bug with reproducer / docs fix / missing test /
maintainer-requested change

**Why this one** (cite the triage answers):

> …

### Reproduction

```sh
# exact commands, from a clean clone, including the commit you are on
```

**Observed:**

```
…
```

**Expected, and where that expectation comes from** (docs, the issue, a
maintainer comment, the tests):

```
…
```

### The claim comment

Posted on `<date>` · link: …

Paste the exact text you posted. It must show evidence you reproduced it, the
code path you found, a proposed approach, and one or two narrow questions.

> …

**Reply received on `<date>`:** … (or: no reply after N days, and what you did
about it)

---

## 3. Orientation — written before you edit

Five to ten sentences on how the affected subsystem works. Not what the change
is: what the *code* does today, in your own words. If you cannot write this,
you have not read enough to be trusted with the patch.

> …

**How I found the code path** (the grep, the failing test, the call chain):

> …

**Files and functions I expect to change:**

| File | Function / area | Why it changes |
|---|---|---|
| | | |

**What `git log` / `git blame` told me about why this code is the way it is:**

> …

**What could break** — callers, platforms, downstream behaviour, exported API:

> …

**What I will deliberately not touch**, even though I noticed it:

> …
