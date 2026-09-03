# Profiling & Optimization

> `go.advanced.profiling` · ~3-5h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Write meaningful benchmarks with `testing.B` (including `b.ReportAllocs`
  and sub-benchmarks) and compare results with benchstat.
- Capture and interpret CPU and heap profiles with pprof, navigating top,
  list, and flame graph views to find the actual bottleneck.
- Explain the measure → hypothesize → change → re-measure optimization loop
  and why optimizing without a profile is malpractice.
- Diagnose a provided slow program, attribute the cost to a specific
  function via profile evidence, and fix it with a measured improvement.
- Expose `net/http/pprof` endpoints on a running service and explain what
  each profile type (cpu, heap, goroutine, block, mutex) reveals.

## The loop, and why guessing is malpractice

You have spent three lessons under the hood: the scheduler, the garbage
collector, escape analysis. You now know *why* Go programs cost what they
cost. This lesson is about finding *where* — because on that question,
intuition has a miserable track record. Decades of profiling folklore agree:
the bottleneck is almost never where the author thinks it is, and programmers
who "optimize" on instinct routinely complicate the wrong function for a 0%
win. That's the precise sense in which unmeasured optimization is malpractice:
you pay real costs — readability, risk of new bugs, your time — for benefits
you never verified.

The discipline is the same hypothesis loop you learned in the S4 debugging
lesson, pointed at performance instead of correctness:

1. **Measure** — a benchmark establishes the baseline; a profile says where
   the time and memory actually go.
2. **Hypothesize** — "the inner loop lowercases on every comparison; hoisting
   it should cut allocations by ~50x."
3. **Change** — the smallest edit that tests the hypothesis.
4. **Re-measure** — same benchmark, benchstat comparison. No improvement?
   Revert. The correctness tests must still pass — an optimization that
   changes behavior is just a bug with good PR.

One more habit from that lesson carries over: write the hypothesis down
*before* you edit. The exercise's `NOTES.md` has a slot for it.

## Benchmarks: measurement you can rerun

A benchmark is a test whose body Go runs in a calibrated loop:

```go
func BenchmarkTopUsers(b *testing.B) {
	entries := genEntries(10_000)      // setup, outside the loop
	b.ReportAllocs()
	b.ResetTimer()                     // don't bill the setup
	for range b.N {
		sinkStats = TopUsers(entries, 10)
	}
}
```

The contract of `b.N`: the framework picks it, runs your loop, and keeps
increasing it until the run lasts long enough (default 1s) to produce a
stable ns/op. Your loop body must therefore do *the same work every
iteration* — no accumulating state that makes iteration 1000 slower than
iteration 1.

Three details separate a meaningful benchmark from a random number:

- **`b.ReportAllocs()`** adds allocs/op and B/op to the output. After the
  garbage-collection lesson you know allocations are the currency of GC
  pressure — for many server workloads allocs/op predicts production behavior
  better than ns/op does.
- **The sink.** Assigning the result to a package-level variable
  (`sinkStats`) stops the compiler from noticing the result is unused and
  deleting the very work you're measuring. An inlined, side-effect-free call
  can vanish entirely; benchmarks that report 0.3 ns/op have measured an
  empty loop.
- **Sub-benchmarks** (`b.Run`, same shape as `t.Run`) let one benchmark
  sweep input sizes. Watch ns/op as n grows 10x: if time grows 100x, you're
  staring at the O(n²) you learned to name in S2 — measured, not guessed.

(Toolchains from Go 1.24 on add `b.Loop()`: `for b.Loop() { … }` replaces the
`b.N` loop, excludes setup from the timing by itself, and keeps the call from
being optimized away — so the sink becomes unnecessary. This exercise's module
pins `go 1.22`, so keep the sink here, but recognize both shapes when you read
other people's benchmarks.)

Run them with the test binary — `-bench` selects by regex, and `-run='^$'`
keeps ordinary tests out of the way:

```sh
go test -bench=. -run='^$' -count=10 | tee before.txt
```

### benchstat: statistics or it didn't happen

One benchmark run is a coin flip — thermal throttling, background processes,
CPU frequency scaling all move single measurements by 5-20%. That's why the
command above says `-count=10`: ten samples per benchmark. benchstat
(`go install golang.org/x/perf/cmd/benchstat@latest`) turns two such files
into an honest comparison:

```
                  │  before.txt  │              after.txt              │
TopUsers/10000-8    5.037m ± 2%     0.363m ± 1%   -92.79% (p=0.000 n=10)
```

It reports the change *with a p-value*: `p=0.000` means the difference is
real; `~` means statistically indistinguishable — your "optimization" did
nothing, whatever the two means say. Never claim a win from single runs.

Two rules while benchmarking: **no `-race`** (instrumentation multiplies
every number 2-20x and distorts the ratios you care about), and a machine
that's as quiet as you can make it.

## Capturing a profile

Benchstat tells you *whether* you're fast. A profile tells you *where* you're
slow — and the benchmark you just wrote doubles as the capture harness:

```sh
go test -bench=BenchmarkTopUsers/10000 -run='^$' \
    -cpuprofile=cpu.out -memprofile=mem.out .
```

The CPU profiler is a *sampler*: ~100 times per second it interrupts the
program and records the current call stack. Costs accumulate statistically —
a function with 40% of samples costs 40% of the CPU. Two consequences: a
profile of a fast run is mostly noise (you want seconds of busy CPU, which
the benchmark loop provides), and the ~1% overhead is low enough to use in
production. The heap profile works differently — it samples allocation sites
(every ~512KB allocated on average) and knows both the bytes and the *count*
of allocations attributed to each site.

## Reading a profile

S3's tooling lesson gave you `go tool pprof -top` and the flat/cum pair for a
first look. Here is the rest of the toolkit, and what each view is for:

```sh
go tool pprof -top cpu.out          # table view
go tool pprof cpu.out               # interactive: top, list, web
go tool pprof -http=:8080 cpu.out   # browser UI: flame graph
```

`top` shows two numbers per function, and the distinction is the whole skill:

- **flat** — time spent *in this function's own code*.
- **cum** (cumulative) — time in this function *plus everything it called*.

Your own `TopUsers` will have a huge **cum** (everything happens under it)
but its **flat** may be small — the actual work sits in callees like
`strings.ToLower` and `runtime.mallocgc`. High flat = "this code is hot, fix
it here." High cum, low flat = "the problem is downstream — follow the call."
Seeing `runtime.mallocgc`, `runtime.growslice`, or GC functions
(`runtime.gcBgMarkWorker`) near the top is the profiler telling you what the
GC lesson predicted: you allocate too much; go look at the heap profile.

`list` zooms to lines — `go tool pprof -list 'TopUsers' cpu.out` prints the
function's source annotated with per-line cost. This is where "the inner
loop's `ToLower` call" stops being a theory and becomes a number.

The **flame graph** (in the `-http` UI) is the same data drawn as nested
rectangles: width = share of samples, vertical stacking = call depth. Read
widths, not colors and not height: the widest boxes on top of your own
frames are your bottleneck. It's the fastest way to spot "one callee
dominates" versus "death by a thousand cuts."

For the heap profile, choose the view deliberately:

```sh
go tool pprof -sample_index=alloc_space -top mem.out   # total bytes ever allocated
go tool pprof -sample_index=inuse_space -top mem.out   # bytes live right now
```

`alloc_space` (and `alloc_objects`) answers "who is generating garbage?" —
that's the GC-pressure question, and the one this lesson's exercise needs.
`inuse_space` answers "who is holding memory?" — the leak question. Picking
the wrong index is the classic heap-profile mistake.

## The two bottleneck archetypes

The exercise hands you a working, slow report generator with one specimen of
each of the two most common findings.

**Hidden per-iteration work.** Code that looks O(n) but re-does something
expensive inside a loop that a profile exposes instantly. The starter's
aggregation scans a slice per entry *and calls `strings.ToLower` on every
comparison* — O(entries × users) calls, each one allocating (you know why
from the memory-model lesson: the new string escapes). The fix is the
S2 reflex: index with a map, and hoist the lowering to once per entry.

**The allocation storm.** Building a string with `+=` in a loop allocates a
fresh, ever-longer string per iteration — O(n) allocations, O(n²) bytes
copied — and `fmt.Sprintf` adds interface boxing for every argument. The fix
is `strings.Builder` plus the *append-style* API family:

```go
var b strings.Builder
b.Grow(len(stats) * 48)            // one allocation, roughly right-sized
line := make([]byte, 0, 64)        // reused every iteration
for _, s := range stats {
	line = line[:0]
	line = append(line, s.User...)
	line = append(line, " requests="...)
	line = strconv.AppendInt(line, int64(s.Requests), 10)
	line = append(line, '\n')
	b.Write(line)
}
return b.String()                  // zero-copy
```

`strconv.AppendInt` formats *into a buffer you own* instead of returning a
fresh string — that's the pattern stdlib hot paths use, and it's why the
whole render costs a handful of allocations instead of thousands. To be
clear: in code that isn't hot, `fmt.Fprintf(&b, …)` is perfectly good Go —
you reach for append-style *because the profile said so*, not by default.

Because the graded tests run under `-race` (2-20x slowdown, remember), they
never assert wall-clock time. They assert **allocation counts** via
`testing.AllocsPerRun`, which are deterministic under any slowdown — the
starter's ~500,000 allocations versus your target's ~10,000 is a bigger,
more reliable signal than any stopwatch. The wall-clock win is real too;
you'll demonstrate it yourself with benchstat and record it in `NOTES.md`.

## pprof in production: net/http/pprof

Benchmarks profile code you can extract. Production problems — the 3 a.m.
memory climb, the deadlocked pod — need profiles *from the running service*.
Importing `net/http/pprof` gives you HTTP handlers for exactly that. The
import's side effect registers them on `http.DefaultServeMux`; in a real
service you mount them explicitly on a mux you control (with the 1.22
patterns you've used all stage):

```go
mux.HandleFunc("GET /debug/pprof/", pprof.Index)
mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
```

`Index` owns the subtree: `/debug/pprof/heap`, `/goroutine` and friends all
route through it by profile name. What each profile answers:

- **profile** (CPU) — "where is the CPU going?" Samples for `?seconds=30`,
  then returns the file: `go tool pprof http://host/debug/pprof/profile`.
- **heap** — "who allocates / who holds memory?" (`alloc_*` vs `inuse_*`,
  as above).
- **goroutine** — every goroutine's stack. *The* deadlock and
  goroutine-leak tool: ten thousand goroutines parked on the same channel
  receive name the guilty line, S3-style.
- **block** — where goroutines wait on channels and locks; off until
  `runtime.SetBlockProfileRate` enables it.
- **mutex** — who *holds* contended mutexes; off until
  `runtime.SetMutexProfileFraction` enables it.

Two production rules. First: these endpoints are diagnostic surface, not
API — serve them on a separate internal port (a second `http.Server` on
`localhost:6060` or a cluster-internal listener), never on the public mux;
your S4 security lesson explains what an attacker learns from a heap dump.
Second: profiles cost little, so teams leave the endpoints on — the profile
you can capture *during* the incident is worth ten postmortems.

## Exercise

Open [`exercise/`](exercise/) — a working but slow access-log report
generator (`report.go`), an unwired diagnostics mux (`debug.go`), benchmarks
(`bench_test.go`), and `NOTES.md` for your evidence. Read the tests first,
then work the loop:

```sh
cd exercise
go install golang.org/x/perf/cmd/benchstat@latest
go test -bench=. -run='^$' -count=10 | tee before.txt
go test -bench=BenchmarkTopUsers/10000 -run='^$' -cpuprofile=cpu.out -memprofile=mem.out .
go tool pprof -top cpu.out            # then: -list, -http=:8080
```

Only after the profile has named the guilty lines (paste the evidence into
`NOTES.md`), fix `TopUsers` and `Render`, re-run the benchmarks into
`after.txt`, and compare with `benchstat before.txt after.txt`. Then mount
the pprof handlers in `NewDebugMux`.

Acceptance criteria:

1. All correctness tests keep passing: case-insensitive aggregation with
   lowercase names, sort by requests descending then user ascending,
   truncation to n, and byte-identical `Render` output.
2. `TopUsers` on the 10,000-entry workload performs at most 50,000
   allocations (the starter does ~500,000).
3. `Render` on 2,000 stats performs at most 300 allocations (the starter
   does ~10,000).
4. `NewDebugMux` serves the pprof index, the named profiles (heap,
   goroutine, …), and cmdline under `/debug/pprof/` — verified via
   `httptest`, no real network.
5. `NOTES.md` contains your profile evidence and the benchstat before/after
   — the tutor will ask for both.
6. `go test -race ./...` passes and the code is `gofmt`-clean.

Graded tests run with `-race`; benchmarks are yours to run without it.

## Further reading

- [Go Blog — Profiling Go Programs](https://go.dev/blog/pprof) — the
  original tour of top/list/web on a real program; still the best worked
  example.
- [go.dev — Diagnostics](https://go.dev/doc/diagnostics) — the map of the
  whole tooling landscape: profiling vs tracing vs debugging, and when each
  applies.
- [pkg.go.dev — net/http/pprof](https://pkg.go.dev/net/http/pprof) — the
  endpoint reference, including the parameters every profile type accepts.
- [pkg.go.dev — benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
  — how to read its tables, and why it insists on multiple samples.
