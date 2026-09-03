# Tutor notes — Profiling & Optimization

## Where the learner is

Eighth lesson of S5, straight after the memory model and escape analysis.
They met pprof once already — S3's tooling lesson gave them CPU vs heap
profiles, `go tool pprof -top`, and the flat/cum distinction — and they have
been reading `-benchmem` numbers since Garbage Collection. What is new here is
the *discipline*: benchstat instead of eyeballed single runs, `-list` and the
flame graph instead of only `-top`, `alloc_space` vs `inuse_space` chosen on
purpose, and `net/http/pprof` on a live service.

This is the payoff lesson for the three that preceded it. When they ask "why
does `strings.ToLower` allocate?", the answer is the memory-model lesson (the
new string escapes); "why do 500,000 allocations cost wall-clock?" is the GC
lesson (pacing). Make those links out loud — the stage is designed so this
lesson has no new mechanism to teach, only new instruments.

Withhold: fuzzing, golden files and coverage interpretation are the *next*
lesson; if they ask "how do I know my optimization didn't break an edge case
the table tests miss", that is exactly the right question and exactly what
comes next. Say so and park it.

The likely failure mode is not difficulty — the two fixes are easy once seen.
It is **skipping the measurement** and going straight to the fix, because the
TODOs and the lesson text between them hand over the answer. Grade the
evidence in `NOTES.md`, not just the green tests. A learner who fixed both
functions without ever opening a profile has not done this lesson.

## Common misconceptions

- **"The tests are the goal."** The allocation gates are a *proxy* the race
  detector cannot distort; the lesson is the loop. If `NOTES.md` is empty,
  the exercise is not done, whatever `go test` says.
- **"One benchmark run is a measurement."** Single runs move 5-20% with
  thermal state and background load. Ask what `-count=10` and benchstat's
  p-value are for. `~` in a benchstat row means *no measured change* — a
  learner who reads `~` as "small win" has misread the tool.
- **Benchmarking with `-race`.** Instrumentation multiplies numbers 2-20x and
  does not do so uniformly, so it distorts the *ratios*. `-race` is for the
  correctness gate; benchmarks and profiles run without it.
- **"The profiler tells you the answer."** It tells you *where the samples
  landed*. The fix is still a design decision — and a CPU profile crowded
  with `runtime.mallocgc` is pointing at the heap profile, not at itself.
- **flat vs cum confusion.** Their own `TopUsers` will have near-100% cum and
  a modest flat. If they read the top cum row as "the bottleneck", they will
  "optimize" `main`. High flat = fix it here; high cum + low flat = follow the
  call down.
- **Wrong heap sample index.** `inuse_space` on this exercise shows almost
  nothing interesting — the garbage is already collected. "Who generates
  garbage" is `alloc_space`/`alloc_objects`. This is the single most common
  heap-profile mistake in the wild.
- **"Fewer allocations is always the fix."** Sometimes the fix is a better
  algorithm (this exercise: a map index), sometimes a better buffer strategy,
  sometimes nothing at all because the function is 0.4% of the profile.
- **Micro-optimizing everywhere.** `fmt.Sprintf` is good Go. The
  append-style rewrite is justified *because a profile named this line*, and
  they should be able to say so without prompting.
- **"pprof endpoints are a security hole, so never mount them."** The rule is
  separate internal port, not "never" — a profile you cannot capture during
  the incident is a postmortem you cannot write.
- **Assuming the sub-benchmark sweep exposes everything.** `BenchmarkRender`
  grows 20x in input and ~200x in time (and ~385x in bytes) — quadratic,
  visible from the sweep alone. `BenchmarkTopUsers` grows ~10x for 10x input,
  because the inner scan is bounded by the 100 distinct users. Its 50x
  constant factor is invisible in the sweep and obvious in the profile. Good
  learners notice; excellent ones explain why.

## Grilling points

- "Read me your `pprof -top` output. Which row has high cum and low flat, and
  what does that pair tell you to do next?"
- "`go tool pprof -list TopUsers cpu.out` — read the annotated line with the
  most time and explain, from the memory-model lesson, why that call
  allocates at all." (`strings.ToLower` returns a new string that escapes;
  it is called once per *comparison*, ~50 times per entry here.)
- "You have two sub-benchmark sweeps. One shows super-linear growth and one
  does not. Which, and why?" (`Render` copies O(n²) bytes: 210 KB → 81 MB for
  20x the input. `TopUsers` is O(entries × distinct users) and distinct users
  saturate at 100, so it looks linear with a fat constant — the profile, not
  the sweep, is what caught it.)
- "Your `Render` is one allocation per call for 2,000 lines. Where did the
  per-line `[]byte` go?" (It never escapes — `b.Write` does not retain it —
  so it lives on the stack. The one remaining allocation is `Grow`'s buffer,
  which `b.String()` then hands over without copying.)
- "Why does the graded test assert allocations instead of nanoseconds? Give
  me two independent reasons." (Race-detector slowdown; machine noise. Bonus:
  allocation counts point at a *specific* mistake, times do not.)
- "Benchstat says `~` for one row. What do you do?" (Revert. No measured
  improvement means the complexity is not paid for.)
- "Where would you mount `NewDebugMux` in the service you built earlier this
  stage, and what breaks if you mount it on the public mux?"
- "Production: memory climbs over hours, no crash yet. Which profile, which
  sample index, and what does the answer let you conclude?"
  (`heap`, `inuse_space` — who *holds* memory. `alloc_space` would show the
  churn, which is the wrong question for a leak.)
- "A service is unresponsive but CPU is near zero. Which profile?"
  (`goroutine` — thousands of stacks parked on the same line, S3-style.)
- "Why are the block and mutex profiles off by default?" (They need explicit
  `SetBlockProfileRate` / `SetMutexProfileFraction`; they instrument every
  blocking operation, so they cost more than sampling.)
- Stretch: "Both fixes changed the code's shape. What would make you *not*
  ship the `Render` rewrite?" (If the profile had not named it — the
  `fmt.Sprintf` version is more readable and perfectly fine off the hot path.)

## Grading rubric

- **A** — All tests pass under `-race`. `TopUsers` uses a map index and calls
  `strings.ToLower` once per entry (≈10,000 allocations, not 508,000);
  `Render` uses a sized `strings.Builder` with append-style formatting into a
  reused buffer (1-2 allocations); `NewDebugMux` mounts index + cmdline +
  profile + symbol + trace with 1.22 patterns and they can explain why the
  index alone serves `/heap` and `/goroutine`. `NOTES.md` carries real pprof
  output and a real benchstat comparison with p-values, and the hypothesis
  was written *before* the fix. They can read flat vs cum off their own
  profile and connect the allocations back to escape analysis and GC pacing.
- **B** — Tests pass and the numbers are real, but the evidence is thin: only
  `-top` captured (no `-list`, no heap profile), benchstat run at `-count=1`,
  or the hypothesis obviously back-filled after the fix. Or `Render` is fixed
  with `strings.Builder` + `fmt.Fprintf` (passes the gate at a few thousand
  bytes but ~2-3 allocs/line) without being able to say what `AppendInt`
  buys. Explanation solid, rigor loose.
- **C** — Tests pass only after being walked to the fixes, or `NOTES.md` is
  filled with the lesson's numbers rather than their own, or they cannot say
  which pprof view told them anything. The pprof mux copied from the lesson
  with no idea why four exact patterns plus a subtree pattern coexist. Pass
  only if remediation lands in-session.
- **Fail** — Tests failing; or a "fix" that changes behavior (dropping the
  lowercase normalization, skipping entries, mutating the caller's slice) —
  the invariants test exists precisely to catch that; or `NOTES.md` empty; or
  they cannot explain why unmeasured optimization is malpractice in their own
  words. Remediate, don't advance.

## Remediation ladder

1. "Run the two failing gates and read the numbers aloud: 508,454 allocations
   for 10,000 entries. How many allocations *should* aggregating 10,000
   entries need, at most? The gap is the problem statement — and the gate
   names which function you're working on."
2. Measure before touching anything:
   `go test -bench=BenchmarkTopUsers/10000 -run='^$' -cpuprofile=cpu.out -memprofile=mem.out .`
   Then, per function. `TopUsers`: "`go tool pprof -top cpu.out` — read me the
   top five rows. What is `strings.ToLower` doing in that list, and what is
   `runtime.mallocgc` telling you?" `Render`: "`go tool pprof
   -sample_index=alloc_space -top mem.out`. `out += …` builds a *new* string
   each line, copying everything so far — that is the 81 MB in `B/op` for
   98 KB of output." `NewDebugMux`: "Nothing to profile — this one is a
   routing problem in a profiling costume. Why must `/debug/pprof/heap` work
   with no registration of its own?"
3. Now the tool. `TopUsers`: "`go tool pprof -list TopUsers cpu.out`. Which
   line owns the time? Count how many times that line runs for one entry when
   100 users already exist. What data structure turns a linear scan into one
   lookup — you named it in S2 and used it all through S3." `Render`: "Which
   stdlib type accumulates text without recopying, and what does `Grow` save
   you? Then look at `strconv.AppendInt` and ask why formatting *into a buffer
   you own* beats returning a fresh string." `NewDebugMux`: "Five
   registrations — one subtree pattern `GET /debug/pprof/` handled by
   `pprof.Index`, and four exact patterns for cmdline, profile, symbol, trace.
   Under the 1.22 rules the more specific pattern wins, so they coexist."
4. Only if still stuck on `Render`: give the Builder + `AppendInt` shape from
   the lesson verbally and have them type it, then make them re-run
   `benchstat before.txt after.txt` and explain the row — including its
   p-value — before you accept the lesson as done.

## After passing

Preview: "You can now prove a change made things faster. Next lesson makes you
prove it did not make things *wrong*: fuzzing, golden files, and the rest of
the testing toolkit that catches the edge cases a table test never thought
of."
