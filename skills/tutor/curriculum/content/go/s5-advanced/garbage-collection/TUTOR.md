# Tutor notes — Garbage Collection

## Where the learner is

Sixth lesson of S5, straight after the scheduler. They have the GMP model
fresh (P, sysmon, work stealing), know benchmarks and pprof from S3, and have
lived under `-race` since S3's concurrency arc. This is their first look at
the allocator and collector, and their first `sync.Pool`.

The stage's shape matters for what you withhold: **escape analysis and
stack-vs-heap are the next lesson**, and pprof heap profiles get their proper
treatment in Profiling after that. When they ask "but why does that allocate
at all?", confirm the question is excellent and park it — the answer is
literally the next lesson. Keep this lesson on *the heap already exists; what
does the collector do with it, and how do I put less in it*.

Expect the theory to land faster than the exercise. The three fixes are
mechanically easy; the value is in them predicting the numbers first and
explaining the ones they get.

## Common misconceptions

- **"GC pauses scale with heap size."** Marking is concurrent; the STW phases
  are root-scanning and mark termination, sub-millisecond on healthy programs.
  A big heap costs CPU and cycle frequency, not pause length. If they say
  "Go pauses for a second on a 10 GB heap", that's Java-1990s folklore.
- **"Garbage that dies immediately is free."** Allocation rate drives pacing.
  A program with a 10 MB live set that allocates 1 GB/s runs cycles
  constantly. Point at `gcdemo`: 64 MiB live, 1 GiB churned, ~19 cycles.
- **"GOGC=off means no GC."** Only with no `GOMEMLIMIT`. The Part 1 fourth run
  collects *more* than the default; make them explain why before moving on.
- **"GOMEMLIMIT is a hard cap / a leak fix."** It's soft, it covers all
  runtime-managed memory (not just heap), and under a genuine overshoot it
  converts an OOM kill into a CPU-burning slowdown (GC capped at 50% CPU).
  Better failure mode, still a failure.
- **`sync.Pool` treated as a cache.** It is cleared at GC, `Get` may return
  anything or nothing, and pooled objects must be `Reset`. If they say "I'll
  cache parsed configs in a pool", stop and re-read the contract together.
- **Forgetting `Reset`.** The classic pooling bug and a real security bug —
  request A's bytes appearing in request B's response. Ask what `Get` returns
  and whether it can have leftover content.
- **"Fewer bytes = faster GC."** It's pointer slots, not bytes. `[]*Item` and
  `[]Item` can hold identical data at wildly different mark costs.
- **Optimizing without measuring.** They have `-benchmem` and
  `AllocsPerRun`; a claim with no number attached does not count in this
  lesson.

## Grilling points

- "Walk me through the black-to-white problem: draw three objects, do the
  write, tell me what gets freed and why that's a disaster. Now where does the
  write barrier intervene?"
- "Your `gcdemo` numbers: live set is ~68 MB in every run, but the goal moves
  from 105 to 333 MB. Compute the goal from the formula and check it against
  your paste. Where does the extra memory go, and what did you buy with it?"
- "Which line item does each of your three fixes attack — mark cost, or
  pacing frequency? Justify one of them either way." (`EventIDs` and
  `FormatEvents` mostly cut pacing pressure; a pool cuts both, and also cuts
  the allocator's work.)
- "`WriteEvents` reaches ~0 allocs/op and `FormatEvents` bottoms out at 1.
  Why can't `FormatEvents` reach 0?" (It must return a fresh string the caller
  keeps — the allocation is the API, not the implementation. A good learner
  gets to "I'd have to change the signature to a writer or a caller-supplied
  buffer", which is exactly the design lesson.)
- "Under `-race` your pooled path sometimes shows a few allocs/op instead of
  zero. Why is the gate 64 and not 0?" (Detector overhead, pools dropped at
  GC — and the honest answer: the gate must separate ~2000 from ~0, not
  measure precisely.)
- "You have a service being OOM-killed in a 512 MiB container. What do you set
  first, and what do you look at before touching any knob?"
- "When would you *not* use `sync.Pool` here?" (Cheap objects, wildly varying
  sizes, anything whose lifetime outlives the call, anything the profile
  doesn't blame.)

## Grading rubric

- **A** — All tests pass under `-race`; `FormatEvents` sums lengths and calls
  `Grow` (not just `strings.Builder` unsized); `EventIDs` preallocates
  capacity; `WriteEvents` uses a package-level `sync.Pool` with `Reset` and a
  deferred `Put`, writes pieces directly (no per-event temporary), and keeps
  the single `Write` + error propagation. `NOTES.md` is complete, including
  the pacer formula checked against their own numbers and a correct
  `GOMEMLIMIT` account. They can explain the write barrier and the
  pointer-slot argument without prompting.
- **B** — Tests pass and the benchmark deltas are real, but one part is
  shallow: `Grow` omitted (still 1-2 allocs, passes the gate), `NOTES.md`
  numbers recorded without the explanation, or the GOGC trade-off stated only
  in one direction. Explanation of concurrency/write barrier is hand-wavy but
  not wrong.
- **C** — Tests pass only after heavy hinting, or `sync.Pool` is used as
  cargo cult ("the TODO said so") without being able to state its contract, or
  Part 1 was skipped and back-filled from the lesson text rather than from
  their own runs. Pass only if remediation lands within the session.
- **Fail** — Tests failing; or a shared global buffer "fixed" by making the
  concurrency test smaller; or they cannot distinguish mark cost from pacing
  frequency; or `NOTES.md` is empty. Remediate, don't advance.

## Remediation ladder

1. "Run the failing gate and read the number aloud: 2048 allocations for 2048
   events. Which of the three functions does the gate name? Work that one
   only — and answer the question the number is asking: what in *that* loop
   body happens once per event?"
2. Now name where the garbage comes from, in their function.
   `EventIDs`: "What does `append` do when the slice is full, and how many
   times does that happen for 2048 elements? What do you already know before
   the loop starts?" `FormatEvents`: "`s += x` produces a *new* string every
   time, and the old one is garbage the instant the next `+=` runs. How many
   strings does 2048 events cost you?" `WriteEvents`: "Two separate problems —
   name both before fixing either. (a) A throwaway `line` string per event.
   (b) A buffer born and dying each *call*. Which is per-event, which is
   per-call?"
3. Now the tool. `EventIDs`: "One `make` with the length you already know, and
   `append` never re-grows." `FormatEvents`: "Which standard-library type is
   built for accumulating text without recopying, and what does its `Grow`
   method save you? You need one pass to sum the lengths first."
   `WriteEvents`: "(a) Can you write `Level`, `": "`, `Msg`, `'\n'` straight
   into the buffer, with no intermediate string? (b) Re-read the `sync.Pool`
   section: `Get`, `Reset`, defer `Put`. Why must it be a pool and not a
   package-level buffer? Run the concurrency test with `-race` and let it
   answer."
4. Only if still stuck: walk the pool shape verbally (`var bufPool =
   sync.Pool{New: func() any { return new(bytes.Buffer) }}`) and have them
   type the body themselves, then re-run the benchmarks and explain the drop.

## After passing

Preview: "You now know what the collector charges you for heap objects. Next
lesson asks the question underneath: which values end up on the heap at all —
escape analysis — plus the memory model that says when one goroutine's write
becomes visible to another."
