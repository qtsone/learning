# Tutor notes — Go Tooling

## Where the learner is

Second-to-last lesson of S3, right before the concurrent capstone. They have
the full intermediate toolkit: interfaces through concurrency patterns,
including the race detector from the goroutines lesson. What they lack is the
*practitioner's rig* — they've only ever run `go build`, `go test`, `go vet`
(implicitly, via `go test`), and `gofmt`. This lesson is deliberately
discussion-verified: the win condition is judgment — knowing which tool
answers which question — not code. Grade from `NOTES.md` and the
conversation. The planted bugs in `vetlab/` and `lintlab/` reuse material
from the sync, context, json-encoding, and strings lessons; use them to check
retention, not just tool mechanics.

## Common misconceptions

- **"If it compiles, the compiler approved it."** The central fallacy this
  lesson attacks. The compiler checks what the language *forbids*; vet and
  staticcheck check what is legal but wrong. If they can't name a concrete
  example (printf verbs, copied mutex, leaked cancel, broken struct tag),
  the lesson hasn't landed.
- **"Linters are style nitpickers."** Conflating gofmt (layout) with static
  analysis (bugs). SA4006 and lostcancel are real defects, not taste.
- **"go.sum is a lockfile."** The most common modules confusion. Versions are
  pinned by `go.mod` + MVS; `go.sum` is integrity hashes — tamper evidence.
  The part 5 tampering step exists to make this visceral.
- **"Go picks the latest version of a dependency."** MVS picks the maximum of
  required *minimums* — never something newer than someone asked for. If
  they say "newest compatible", push back with the sampler example.
- **"Profile = benchmark."** A benchmark measures *how long*; a profile shows
  *where*. Learners who ran part 4 mechanically often can't separate these.
- **"CPU profile for everything."** Probe: memory keeps growing — which
  profile? If they say CPU, revisit the heap profile's purpose.
- **Suppressing findings with naked `//nolint`** — accept suppression only
  with linter name and reason; otherwise it's silent debt.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "The copied-mutex finding: walk me through what actually happens at runtime
  when `Value()` runs concurrently with `Inc()`. Would `-race` have caught
  it?" (Copy means the lock guards nothing — the race detector *could* catch
  the resulting data race, but only if a test actually races it; vet catches
  it before any test exists.)
- "Why does `ctx, _ := context.WithTimeout(...)` leak? What exactly is not
  released until the deadline?" (Timer + goroutine + child-context resources;
  cancel releases them immediately — straight from the context lesson.)
- "Your `.golangci.yml` set `default: none`. Argue for and against that
  versus enabling everything." (Chosen linters keep findings signal; enabling
  all invites noise and nolint sprawl.)
- "Team A requires `x v1.4.0`, team B's library requires `x v1.6.0`, the
  newest release is `v1.9.0`. What does your build use, and could it ever use
  v1.9.0 without someone asking?" (v1.6.0; no — MVS never exceeds a stated
  minimum.)
- "You delete a line from `go.sum`. The build now *fails* with `missing
  go.sum entry` — it never silently regenerates the line; you must run
  `go mod tidy` yourself. So what does go.sum protect against, precisely?"
  (Substitution of *different bits* for a known version — with the checksum
  DB as the outside witness; not against choosing wrong versions, and not
  against deletion, which is loud by design.)
- "In the CPU profile, was the top function `JoinNaive` itself or something
  in the runtime? What does that split (flat vs cum) tell you?" (Time is in
  `runtime` memmove/mallocgc/GC on behalf of concatenation — cum attributes
  it up to JoinNaive.)
- "When would you *not* act on a profile?" (When the hot spot isn't the
  user-visible bottleneck, or the win doesn't justify complexity — measure,
  then judge.)

## Grading rubric

- **A** — All six parts done with specifics: vet findings explained with
  *why the compiler allowed each*; staticcheck codes correctly attributed to
  families; a working v2 `.golangci.yml` they can defend line by line;
  errcheck findings fixed by handling errors (e.g. returning an error from
  `WriteReport`), not `_`-discarding them; profile reading distinguishes flat
  from cum and ties allocs/op to O(n²) copying; MVS and go.sum answers
  precise and in their own words; all three tools exit clean at the end.
- **B** — Everything attempted and tools run honestly, but one conceptual
  soft spot: fuzzy MVS wording ("picks a compatible version"), heap-vs-CPU
  hesitation, or fixes that silence rather than resolve (an unjustified
  nolint, an `_ =` discard). Solid in conversation after one nudge.
- **C** — Findings pasted into NOTES.md without explanation, config
  copy-pasted without being able to say what `default: none` does, or the
  modules field trip skipped/simulated. Pass only if live remediation lands:
  have them re-explain two vet findings and re-answer the MVS question on the
  spot.
- **Fail** — Tools not actually run (findings paraphrased from the lesson
  text), or they still equate "compiles" with "checked" when asked directly.
  Redo parts 1-3 together, then re-discuss.

## Remediation ladder

1. "Read the vet finding aloud, then read the exact line it points to. What
   does the *compiler* check on that line? What didn't it check?"
2. "For the mutex: draw the two copies of `SafeCounter` after the method
   call. Which copy's lock did `Value` take? Which one does `Inc` take?"
3. For MVS: lay out the sampler versions on a number line, mark each
   requirer's minimum, and ask "what's the smallest version that satisfies
   every arrow?" — let them find the maximum-of-minimums themselves.
4. For pprof: have them rerun with `-benchmem` and read just allocs/op for
   the two functions side by side, then ask "2000 lines — how many string
   copies did each make?" before returning to the profile.

## After passing

Preview: "Next up is the S3 capstone: a concurrent tool built end to end —
goroutines, channels, context, the whole arc — and every tool from this
lesson (vet, staticcheck, golangci-lint, `-race`) is part of its definition
of done."
