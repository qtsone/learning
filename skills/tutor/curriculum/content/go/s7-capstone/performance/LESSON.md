# Performance Engineering

> `go.capstone.performance` · ~8-12h · Stage: Expert Capstone (Go)

## Objectives

By the end of this lesson you can:

- Define a measurable performance goal for a real bottleneck in your capstone —
  latency, throughput or memory — and pin it with a baseline benchmark.
- Diagnose that bottleneck end to end with pprof profiles, execution traces and
  benchmarks, and name the dominant cost rather than guess at it.
- Implement one optimization and prove the improvement with before/after
  comparisons over repeated runs, not with intuition.
- Explain the trade-offs the optimization introduced — complexity, memory,
  readability — and why they are acceptable at the sizes your PRD promised.
- Pass the graduation review: present the whole loop and defend the methodology
  under grilling.

## The bottleneck nobody labelled

S5's profiling lesson handed you a slow program with two planted specimens: a
hidden per-iteration cost and an allocation storm. Finding them taught you the
tools. This is the same loop against a harder target — your own code, where
nothing is labelled, most of it is fast enough, and you already have opinions
about which part is slow. Those opinions are the problem: you remember which
function was hard to write, and difficulty is not cost. The discipline is
refusing to act on that memory until a profile agrees. Expect to be wrong at
least once; being wrong cheaply, in writing, is the skill.

One constraint makes this lesson expert-level: **you only get to ship one
optimization.** One question, one measurement, one change, one honest write-up.
A single change you can defend to the last decimal is worth more than five that
made the code worse in ways nobody measured.

## Start from a question, not from code

"Make it faster" is not a goal; it has no failure condition. A performance goal
names four things:

1. **The operation.** Something a caller waits for, named the way they would
   name it: "listing notes by tag", "the /search request", "ingesting one day of
   logs". Not "the `parse` function" — that is an implementation detail you have
   not yet earned the right to care about.
2. **The input.** A size and a shape, from the non-functional requirements you
   wrote in the planning lesson: 10,000 notes, 5 MB files, 200 concurrent
   readers, tags that match one row in ten.
3. **The number today and the number you want**, both with units. The target
   comes from somewhere real — a perception threshold, a timeout you keep
   hitting, a container memory limit, an SLO from S6.
4. **One axis.** Latency, throughput or memory. They trade against each other,
   so a goal with three axes has none; track the others only to notice if you
   made them worse.

Write that down before you measure anything — it is the first section of
`PERF.md`, and it is what prevents the afternoon spent optimizing whatever
happened to look interesting in the profile.

**Pick the operation that matters, not the one that is easy to benchmark.** A
pure function with no dependencies is a lovely benchmark target and usually not
where your program spends its life. If what users wait for is an HTTP handler or
a whole CLI invocation, benchmark that, fixtures and all. Measuring the wrong
operation precisely is the most common way to waste a week.

## The baseline is an artifact, not a memory

Before you change anything, commit a benchmark that reproduces the cost. The
mechanics are S5's: setup outside the loop, `b.ResetTimer()`,
`b.ReportAllocs()`, a package-level sink so the compiler cannot delete the work,
sub-benchmarks to sweep input sizes. Two things are new because the code is
yours.

**Choosing representative inputs is now your job, and it is the hard part.**
Real data has shape: cardinality, distribution, hit rate, size skew. A store
where every note carries every tag and one where tags are rare are different
programs to the profiler, and an optimization tuned to the wrong shape can be a
pessimization in production. Take the shape from your PRD, write the assumption
down in `PERF.md`, and generate input deterministically so the benchmark means
the same thing next month.

**Sweep at least two sizes.** One number tells you where you are; two tell you
where you are going. When the input grows 100x, does the cost grow 100x or
10,000x? That is the difference between a constant you can shave and an
algorithm you have to replace, measured rather than argued.

Then take the baseline properly, and keep it:

```sh
go test -bench='BenchmarkListByTag' -run='^$' -count=10 ./internal/store \
    | tee docs/perf/bench-before.txt
```

`-count=10` because a single run is a coin flip; `-run='^$'` so ordinary tests
stay out of the timing; no `-race`, whose 2-20x instrumentation distorts the
very ratios you are about to compare. Quiet machine, laptop plugged in. That
file is evidence — commit it. "It used to be about 300 microseconds, I think"
is not a baseline.

## Finding the bottleneck when nothing is labelled

Work top-down. Time the whole operation first, then the layers inside it, and
only profile the layer that owns the time — profiling the whole program on the
first move gives you a picture of everything, which is a picture of nothing.
Then choose the profile by symptom:

| Symptom | Look at | Because |
|---|---|---|
| CPU busy, one path hot | CPU profile | it samples stacks; the wide frames are the work |
| GC frames near the top, memory climbing | heap profile, `alloc_space` | you are generating garbage, not computing |
| Slow while CPU is idle | execution trace, block/mutex profiles | you are waiting, not working |
| Fast alone, slow under load | trace plus a load-shaped benchmark | contention only exists when there is contention |

That third row is the one people miss. Half the latency problems in a real
service are a lock held too long, a channel nobody drains, or a database round
trip inside a loop — and a CPU profile of a waiting program is flat and
blameless. `runtime/trace` from S5's scheduler lesson is the ground truth for
"why did this goroutine wait": reach for it the moment wall-clock and CPU time
disagree. Three habits then keep the reading honest:

- **Flat versus cum.** High flat means fix it here; high cum with low flat means
  the cost is downstream — follow the call. Backwards, and you optimize a
  function that does nothing but delegate.
- **`-list` before you believe anything.** The line-level view turns "the
  aggregation is slow" into "that map lookup, 480 ms; that comparison, 380 ms".
  A hypothesis you cannot state at line granularity is not ready.
- **Amdahl's arithmetic.** A change to something that is 8% of the time buys at
  most 8%, even made infinitely fast. Read the share before the function, and be
  willing to walk away.

Write down what you expect *before* you open the profile. When it turns out you
were wrong, that sentence is the most valuable line in your write-up.

## One change, then measure again

The loop is `measure → hypothesize → change → re-measure → keep or revert`, and
each arrow is load-bearing. A usable hypothesis has a mechanism and a
prediction:

> `HasTag` is 380 ms of the 940 ms inside `List` because it runs once per note
> in the store and lower-cases the query each time. If the tag lookup is
> indexed, both the scan and the oversized result slice go away, so
> `ListByTag/n=10000` should drop roughly 10x at 10% selectivity, and
> `ListAll` should not move at all.

That prediction is falsifiable, which is the point: when the numbers come back
at 1.2x you have learned something instead of shrugging.

**Change one thing.** Two changes measured together are one unexplained result;
when you later revert the risky half, you no longer know what the safe half
bought. Three things at once is three runs of this loop, not one.

**Re-measure the way you measured before** — same benchmark, inputs, `-count`
and machine, ideally the same afternoon — into `bench-after.txt`, then
`benchstat docs/perf/bench-before.txt docs/perf/bench-after.txt`.

benchstat turns two files into a comparison with a p-value. `~` means the two
are statistically indistinguishable and your change did nothing, whatever the
means look like. If benchstat is unavailable, say so in the write-up and quote
medians and ranges from ten runs instead — an improvement smaller than the
spread between your own runs is not an improvement.

**Reverting is a normal outcome.** A change that wins 3% and costs a page of
cleverness goes back in the bin, and the fact that you tried it belongs in
`PERF.md` — it is the cheapest gift you can leave the next person, who would
otherwise try it too.

## Algorithmic wins and constant-factor wins

Both are real; they answer different questions and they age differently.

A **constant-factor** win keeps the shape of the curve and moves it down: fewer
allocations per iteration, a reused buffer, one `strings.Builder` instead of
`+=`. Doubling the input still doubles the cost. It is bounded — you can only
remove work that exists — and it is usually the safe, local, reviewable change.

An **algorithmic** win changes what the cost tracks: a map instead of a nested
scan, an index instead of a filter, batching instead of a call per row, a cache
instead of a recomputation. The gain grows with the input, which is why S2's
complexity vocabulary is worth money here.

The sweep tells you which one you got. Take the ratio of the cost at your two
sizes, before and after: an unchanged ratio with parallel curves is a
constant-factor win — honest and useful, say so; a collapsed ratio means you
changed the growth term, and then you must say *in which variable* and confirm
at a third size.

Beware the middle case: cost that was proportional to the size of the
collection becomes proportional to the size of the *answer*. Both are linear, so
no complexity class changed — but the constant now depends on selectivity, so
"10x faster" is only true at the selectivity you measured. State the condition
or the claim is not checkable.

One ordering rule: a constant-factor optimization applied to the wrong algorithm
is work you throw away when you fix the algorithm. Shape first, then shave.

## When not to optimize

Refusing is a senior move, and it needs to be as well argued as acting.

- **It is not hot.** The profile says 2%. Amdahl says stop.
- **The win is inside the noise.** benchstat says `~`. There is nothing there.
- **The cost is correctness risk.** A cache, a hand-rolled data structure or a
  concurrency change buys speed with new failure modes — stale reads, an
  invariant maintained in two places, a race. Pay only for a number that matters.
- **The real fix is elsewhere.** From S6: don't compute it (cache), don't do it
  now (queue), don't do it per item (batch), don't do it at all (delete the
  feature). A tuned inner loop inside an N+1 query is polish on the wrong layer.
- **It isn't finished or correct yet.** Optimizing code whose behaviour is still
  moving means optimizing something you are about to delete.
- **The bottleneck is I/O or the network.** A flat CPU profile with a long wall
  clock is not a Go problem; it is a round-trip count, a missing index, or a
  serialised fan-out.

The credible version of "I did not optimize this" always carries a number:
"the profile puts it at 3% and the p99 is 40 ms under our SLO of 200 ms."
Without the number it is an excuse.

## A performance claim a reviewer can check

`PERF.md` is this lesson's deliverable, and it exists because performance work
is the easiest engineering to lie about — mostly to yourself. Write it for
someone who was not there and does not trust you yet. The template in
`exercise/PERF-template.md` has the shape; here is what fails review:

- **Question** — no target number, or three axes at once.
- **Method** — no input sizes, no `-count`, no machine, no admission of what is
  unrepresentative. A benchmark nobody can rerun is an anecdote.
- **Evidence** — a profile that only ever existed in your terminal. Commit
  `go tool pprof -top` and `-list` output, or the profile file itself.
- **Change** — a link to a diff instead of an explanation. A reviewer wants the
  mechanism in a paragraph.
- **Result** — adjectives. Every number carries a unit *and* the input size it
  belongs to, with allocs/op and B/op beside ns/op: allocation counts survive
  being run on someone else's laptop, wall-clock numbers do not.
- **Trade-off** — "none". There is always one, if only that a reader now needs
  a paragraph of context to understand the function.
- **Correctness** — no named tests. See below.

Underneath all seven: **every claim states the conditions under which it is
true** — input size, selectivity, hardware, Go version. A claim with conditions
can be checked and can be wrong; a claim without them is marketing.

## Proving behaviour did not change

A faster wrong answer is not an optimization, and optimizations break behaviour
in the corners: the empty input, the duplicate, the value that needed
normalising, the concurrent writer. Your existing tests were written against the
code you just replaced, so passing them proves less than it feels like.

The strongest tool is a **differential test**: keep the old, obvious
implementation in the test file as an oracle — `linearScan` below is the
pre-index `List` — and assert the new one agrees with it over generated input.

```go
for _, query := range append(tags, "WORK", "  Urgent  ", "absent") {
	if got, want := ids(s.List(query)), ids(linearScan(all, query)); got != want {
		t.Errorf("List(%q) = %s, linear scan = %s", query, got, want)
	}
}
```

The oracle is allowed to be slow — that is why it is the oracle. Generate input
deterministically from a fixed seed, include the queries that need normalising
and the ones that match nothing, and this one test replaces a dozen you would
otherwise have to imagine. Where the input space is wide, fuzz old against new;
where the output is a document, golden files from S5 do the job.

Then name those tests in `PERF.md`. The harness runs exactly the ones you name —
a literal reading of "covered by a correctness test": if you cannot name it, it
does not exist.

## The graduation review

This is the last lesson of the Go track, and the review at the end is not about
this optimization. It is about whether you can talk about a system you built the
way a senior engineer talks about one — so it ranges over the whole capstone:
planning, core, hardening, operations.

What a senior engineer can do, unprompted, about their own system:

- **Say what it does and who for**, in two sentences, without describing the
  implementation.
- **Draw the boundaries** and defend each: what each package owns, which way
  dependencies point, which seam swaps a piece out.
- **Quote numbers instead of adjectives** — how big it gets, how fast the path
  people wait for is, how much memory — and say how they know.
- **Name the failure modes**: what breaks first under load, what happens when a
  dependency is down, what data loss is possible, what the operator does.
- **Name what is wrong with it.** An engineer who cannot list three known
  weaknesses has not looked.
- **Explain a decision they reversed**, with the evidence that changed their
  mind, and one they held under pressure, with the reason.
- **Say what they would cut** under a third less time, and what still works.
- **Name the next bottleneck** and how they would confirm it. There is always
  one; knowing where it is separates having measured from having guessed.

Nothing in that list is about Go. It is the same list whatever you build next,
which is rather the point of the six lessons you have just finished.

## Exercise

Optimize one real bottleneck in your capstone, end to end, and write it up. The
harness in [`exercise/`](exercise/) grades whatever it finds at your project
directory, resolved as in the previous lessons: `TUTOR_CAPSTONE_DIR`, then the
first line of `exercise/capstone.path`, then `projects/capstone` at your
workspace root, which the harness finds by walking up. If none resolves to a
directory containing a `go.mod`, every check fails with instructions. Copy
[`exercise/PERF-template.md`](exercise/PERF-template.md) into your project root
as `PERF.md` and fill it in as you go — not at the end, when you have forgotten
which run produced which number.

Acceptance criteria — the first six are exactly what the harness checks:

1. **The benchmark is committed.** At least one `Benchmark*` function taking
   `*testing.B` lives in your project, beside the code it measures, and at
   least one of them calls `b.ReportAllocs()`.
2. **The benchmarks run.** `go test -bench=. -benchtime=1x -run='^$' ./...`
   exits zero and produces at least one result line. The harness never asserts
   how fast anything is — wall-clock numbers belong to the machine, not to your
   code.
3. **`PERF.md` is complete.** At the project root, with all seven sections —
   Question, Method, Evidence, Change, Result, Trade-off, Correctness — and no
   template `TODO:` lines left. "A paragraph or more" is enforced as a
   minimum, so the numbers are on the table: at least 120 bytes of body per
   section and 800 across the document. They are floors against a heading
   with one word under it, not targets — a tight, honest section clears them
   without trying.
4. **The result is numbers with units.** The Result section states before and
   after values with units (`ns/op`, `B/op`, `allocs/op`, ms, MB, req/s, %).
5. **The evidence is committed.** A pprof artifact in the project — `cpu.out`,
   `*.pprof`, or `go tool pprof -top`/`-list` output saved as text — named by
   path in the Evidence section. Pasting the pprof output into that section
   instead also counts.
6. **The optimization is covered.** The Correctness section names the tests that
   prove behaviour is unchanged; the harness runs exactly those, and they pass.

The rest is graded in the graduation review:

7. **The goal was measurable before you started**, on an operation somebody
   waits for.
8. **The profile, not intuition, picked the target** — and you can say what you
   expected to find and whether you were right.
9. **You changed one thing at a time**, and can say what you tried that failed,
   or why nothing was reverted.
10. **You can classify the win** as algorithmic or constant-factor from the
    shape of the sweep, and state the conditions under which the claim holds.
11. **The trade-off is real and priced**, with the condition that would make you
    undo it.
12. **The graduation review**: the eight questions above, about the whole
    capstone.

```sh
cd exercise
go test ./...        # the six checks
go test -v ./...     # plus the benchmarks and tests it found
```

## Further reading

- [Go Blog — Profiling Go Programs](https://go.dev/blog/pprof) — the original
  worked example of the loop, still the clearest.
- [pkg.go.dev — testing.B](https://pkg.go.dev/testing#B) — the benchmark
  contract, `ReportAllocs`, `ResetTimer`, and what `b.N` promises.
- [golang.org/x/perf — benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
  — how to compare two runs without fooling yourself.
- [Go Blog — More powerful Go execution traces](https://go.dev/blog/execution-traces-2024)
  — reaching for the tracer when the CPU profile is flat but the clock is not.
