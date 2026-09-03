# Garbage Collection

> `go.advanced.garbage-collection` · ~2-4h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Explain Go's concurrent tri-color mark-and-sweep design and why it trades
  throughput for low pause times.
- Explain what `GOGC` controls, predict how changing it shifts the
  CPU-vs-memory trade-off, and describe when to set `GOMEMLIMIT`.
- Measure allocation behavior with `go test -benchmem` and identify which code
  changes reduce `allocs/op`.
- Reduce GC pressure in a hot path by reusing buffers (`sync.Pool`) and
  preallocating slices, and verify the improvement with benchmarks.
- Explain why fewer pointer-heavy data structures reduce mark-phase cost.

## The other half of the runtime

Last lesson you opened the scheduler — the part of the runtime that decides
*when your code runs*. This one opens the part that decides *when your memory
goes away*. You have never called `free`, yet the service you optimize below
recycles gigabytes an hour. That work happens on your CPU budget.

The bill has two separate line items, and keeping them apart is most of the
skill here. **Mark cost** is proportional to the *live* heap: how much
reachable memory must be traversed each cycle. **Pacing frequency** is driven
by your *allocation rate*: how fast you produce garbage decides how often
cycles happen at all. Garbage is not free just because it dies young.

## Tri-color mark and sweep

The collector's job is to find every object still reachable from your program
and reclaim everything else. It does it by coloring objects:

- **White** — not yet proven reachable. At the end of marking, white means
  garbage.
- **Grey** — proven reachable, but its own pointers have not been followed
  yet. The work queue.
- **Black** — reachable, and all its outgoing pointers already followed. Done.

The algorithm is three sentences: start with everything white; color the
**roots** grey (goroutine stacks, globals, registers); then repeatedly take a
grey object, blacken it, and grey every white object it points at. When no
grey objects remain, everything still white is unreachable and its memory is
**swept** — returned to the allocator's free lists, lazily, as spans are
needed again.

Note what "reclaim" does *not* mean here: Go's collector is **non-moving**. It
never relocates a live object to compact the heap, so a pointer's value is
stable for the object's whole life. That costs some fragmentation and rules
out a bump-pointer allocator; it buys pointer stability, which is why Go
interoperates with C and `unsafe` without a pinning ritual around every call.

## Concurrent means your code keeps running

Marking a multi-gigabyte heap takes far longer than any latency budget would
accept as a stop-the-world pause. So Go marks **concurrently**: the collector
runs on some of your CPUs while your goroutines — the *mutator*, in GC
vocabulary — keep rewriting the very pointer graph the marker is walking.

That is a genuine race with one dangerous shape. Suppose the marker has
already blackened object `A`, and your code then writes a pointer to a
still-white `W` into `A` while deleting the last other reference to `W`. `A`
is black, so the marker never revisits it; `W` stays white and is swept while
live. That is a use-after-free with a garbage collector attached.

Go prevents it with a **write barrier**: while a cycle is marking, the
compiler routes every pointer write in your program through a short runtime
hook that greys the objects involved, so no black-to-white edge is created
unnoticed. The barrier is the price of concurrency, and *your* code pays it —
pointer stores get slower for the duration of a cycle.

Two more pieces keep the pauses small and the pacing honest. There are **two
brief stop-the-world phases** per cycle rather than one long one: a short
setup pause that turns the write barrier on, and another at mark termination,
both sub-millisecond on healthy programs. Note what is *not* in them: goroutine
stacks — the bulk of the root set — are scanned during the concurrent mark
phase, one goroutine paused at a time, not during the global stop. That is why
a program with a hundred thousand goroutines still sees sub-millisecond STW
times, and why the pause you measure in `gctrace` barely moves when goroutine
count grows. And there are **mark assists**: a
goroutine that allocates during a cycle must do a proportional slice of
marking work before it gets its memory. Without assists a fast allocator could
outrun the marker forever and the heap would grow without bound; with them,
allocation pressure applies its own brake — which is why "the GC is slow"
often shows up in a profile as *your* code doing GC work.

Alongside assists, the runtime dedicates roughly **25% of `GOMAXPROCS`** to
background marking. Add it up and the design goal is explicit: Go spends CPU —
barriers, assists, a quarter of your cores during a cycle — to keep pauses
under a millisecond. A throughput-first collector (big STW pauses, generational
copying) would finish the same work using less total CPU. Go took the trade the
other way, because it was built for servers where a 250 ms pause is an incident
and a 5% CPU tax is a line item.

(Go 1.25 introduced a redesigned marker — "Green Tea" — that scans memory span
by span for better locality, and Go 1.26 made it the default. The tri-color
contract above is unchanged; what changed is how the marker walks memory.)

## The pacer: GOGC

When does a cycle start? The runtime keeps a **heap goal**, and `GOGC` sets
it:

```
heap goal = live heap after last cycle × (1 + GOGC/100)
```

Default `GOGC=100` means "start the next cycle when the heap has grown to
twice the live set". `GOGC=50` halves the slack; `GOGC=400` gives you five
times the live heap before collecting. `GOGC=off` disables the pacer
altogether.

Predict the trade-off before the exercise measures it. Raising `GOGC` gives
each cycle more garbage to amortize over, so cycles get rarer: **less CPU
spent collecting, more memory held**. Lowering it collects sooner and more
often: **less memory held, more CPU spent**. The live set is unaffected either
way — `GOGC` buys and sells only the *slack* above it. On one machine,
`cmd/gcdemo` (64 MiB live, 1 GiB allocated) gives:

| setting | heap goal | GC cycles | heap at exit |
|---------|-----------|-----------|--------------|
| `GOGC=50` | ~105 MB | 37 | 94 MiB |
| `GOGC=100` (default) | ~137 MB | 19 | 119 MiB |
| `GOGC=400` | ~333 MB | 5 | 212 MiB |

Halving the cycles by paying with memory is the whole knob. In code the same
control is `debug.SetGCPercent`.

## GOMEMLIMIT: the backstop GOGC cannot be

`GOGC` is *relative*, and that is its blind spot. If your live set doubles on a
legitimate traffic spike, the heap goal doubles with it — right past the
container limit, into an OOM kill. Conversely, a service with a small live set
leaves most of a generous memory allowance unused.

`GOMEMLIMIT` (Go 1.19+) sets a **soft limit on total runtime-managed memory** —
heap, stacks, and runtime metadata, not just the heap. As the total approaches
it, the collector runs more aggressively regardless of what `GOGC` says. It is
*soft* on purpose: if live memory genuinely exceeds the limit, Go keeps the
program running and caps GC at 50% of CPU, so a doomed process degrades
instead of freezing in a death spiral of back-to-back cycles.

Two configurations you will actually use. **Keep `GOGC=100` and set
`GOMEMLIMIT` a bit below the container limit** — leave headroom for non-Go
memory (the binary, cgo, OS buffers). Normal operation is unchanged; the limit
engages only under pressure, as a safety net. Start here. Or **`GOGC=off` plus
`GOMEMLIMIT`**, which turns the allowance into a target: allocate freely,
collect only near the ceiling, minimize GC CPU. Excellent for a job with a
well-understood live set, dangerous for anything whose live set can grow —
with no relative pacing left, a leak walks straight into the death spiral.

The same knob in code is `runtime/debug.SetMemoryLimit`. And a memory limit is
*not* a leak fix: it converts an out-of-memory crash into a CPU-burning
slowdown, which is a better failure mode and still a failure.

## The lever you actually control

You will rarely tune `GOGC` in anger. What you change every week is how much
your code allocates — and that moves both line items at once: fewer bytes live
means cheaper marks, fewer allocations means rarer cycles.

Measure first. From S3 you know benchmarks; `-benchmem` is the flag that
matters here:

```sh
go test -bench=. -benchmem -run='^$'
```

```
BenchmarkFormatEvents-10   200   8391317 ns/op   108437207 B/op   2054 allocs/op
```

Read `allocs/op` before `ns/op`. It is far more stable across machines and
runs than a time measurement, it points at a specific mistake, and on a hot
path it is usually *why* the time is what it is. For an automated gate,
`testing.AllocsPerRun(runs, f)` returns the average allocation count of `f` — a
deterministic number you can assert on. Times are not assertable (a loaded CI
machine or the race detector shifts them by an order of magnitude);
allocation counts are.

Three moves cover most of the wins:

**Preallocate what you can count.** `append` to a nil slice re-grows and
re-copies roughly `log₂(n)` times, and every intermediate array is garbage.
When the final length is known — and it usually is — say so:

```go
ids := make([]int64, 0, len(events))   // one allocation, no copies
```

**Build strings once.** `s += x` in a loop allocates a whole new string per
iteration and copies everything built so far: quadratic bytes, linear
allocations. `strings.Builder` writes into one growing buffer, and `Grow`
sizes it up front so even that buffer is allocated exactly once.

**Reuse buffers with `sync.Pool`.** For scratch space that a function needs on
every call and nobody keeps afterwards, a pool recycles it across calls:

```go
var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func render(w io.Writer, events []Event) error {
	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	buf.Reset()
	// … write into buf …
	_, err := w.Write(buf.Bytes())
	return err
}
```

`sync.Pool` is per-P (that's the P from last lesson), so `Get`/`Put` are
mostly uncontended and safe for concurrent use. But read its contract
carefully, because it is not a cache:

- **The pool may be emptied at any GC.** Anything you keep there can vanish;
  `New` must be able to rebuild it. Never store something that must survive.
- **`Get` returns arbitrary state.** Always `Reset` before use — the number
  one pooling bug, and it leaks one request's data into another's.
- **Never `Put` something still referenced elsewhere**, and never substitute a
  single shared global buffer: two callers writing one buffer is a data race,
  and the exercise's race detector will say so.
- **Don't pool objects of wildly varying size.** One 100 MB buffer that lands
  in the pool keeps 100 MB alive for every small request that borrows it next.

Pool because a profile told you to. Pooling a cheap object adds
synchronization and a footgun for nothing.

## Pointers cost more than bytes

The mark phase does not care how many *bytes* you have live. It cares how many
**pointer slots** it must follow. The allocator records, per size class,
whether objects contain pointers at all; a pointer-free object lives in a
"noscan" span and the marker skips its contents entirely. A 1 MB `[]byte` is
one pointer to scan. A million-node linked list of tiny structs is a million
objects to visit, in cache-hostile order.

So `[]Item` (values, contiguous, scanned as a block) beats `[]*Item` (n
pointers, n objects, n cache misses) for anything long-lived and large. Watch
for hidden pointers too: every `string`, slice, map, interface, and function
value contains one, and so do types like `time.Time` (it carries a
`*Location`). A struct of `int64`s and fixed-size arrays is noscan; add one
`string` field and every instance becomes scannable. Big caches of
pointer-rich objects are the classic "the GC is slow" complaint, and the fix
is usually a representation change — indices into a value slice, or one
backing array instead of a million small nodes — not a GC knob.

Where those objects live in the first place, stack or heap, is the *next*
lesson's subject. This one is about what the collector does with the ones that
reach the heap.

## Exercise

Open [`exercise/`](exercise/) — a Go module with two parts.

**Part 1 — watch the pacer.** `cmd/gcdemo` keeps a fixed live set while
churning about a gigabyte of short-lived garbage. Run it under `gctrace` at
several `GOGC` settings and record what you see in `NOTES.md`:

```sh
GODEBUG=gctrace=1 go run ./cmd/gcdemo
GODEBUG=gctrace=1 GOGC=50 go run ./cmd/gcdemo
GODEBUG=gctrace=1 GOGC=400 go run ./cmd/gcdemo
GODEBUG=gctrace=1 GOGC=off GOMEMLIMIT=100MiB go run ./cmd/gcdemo
```

Each `gc` line reads: cycle number, time since start, cumulative % of CPU
spent on GC, the three clock phases (`sweep termination + concurrent mark +
mark termination`, in ms), CPU times, then `heap-at-start -> heap-at-mark-end
-> live-after-sweep MB`, the heap goal, and the P count.

**Part 2 — fix the hot path.** `hotpath.go` is the rendering layer of an
ingest service, written the obvious way. Three `TODO`s mark the work:
`FormatEvents` (string concatenation in a loop), `EventIDs` (unsized
`append`), `WriteEvents` (a per-event temporary plus a fresh scratch buffer
per call, called concurrently). `hotpath_test.go` holds correctness tests plus
allocation gates with deliberately generous bounds; `hotpath_bench_test.go`
holds the benchmarks you run by hand for evidence.

Acceptance criteria:

1. `go test -race ./...` passes, including the three `…Allocs` gates. Behavior
   is unchanged: renderings must match byte for byte, `WriteEvents` still
   makes exactly one `w.Write` call and still propagates its error.
2. `FormatEvents` builds its result in a single sized `strings.Builder`, and
   `EventIDs` preallocates the capacity it can already compute.
3. `WriteEvents` reuses scratch buffers across calls via a package-level
   `sync.Pool` and is safe under `TestWriteEventsConcurrent` with `-race`. A
   single shared global buffer is not an acceptable answer.
4. `NOTES.md` records, for each `GOGC` setting above, the cycle count, the
   heap goal, and the live heap after sweep — plus your written explanation of
   the CPU-vs-memory trade-off and of what `GOMEMLIMIT` changed.
5. `NOTES.md` records `allocs/op` and `B/op` for all three benchmarks before
   and after your changes, from `go test -bench=. -benchmem -run='^$'` (run
   without `-race`; the detector distorts everything it touches).

Run the tests from inside `exercise/`. They fail before you start — that's the
point.

## Further reading

- [A Guide to the Go Garbage Collector](https://go.dev/doc/gc-guide)
- [runtime package docs — GOGC, GOMEMLIMIT and other environment variables](https://pkg.go.dev/runtime#hdr-Environment_Variables)
- [sync.Pool documentation](https://pkg.go.dev/sync#Pool)
- [Go blog — Getting to Go: the journey of Go's garbage collector](https://go.dev/blog/ismmkeynote)
