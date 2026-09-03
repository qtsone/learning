# Memory Model & Escape Analysis

> `go.advanced.memory-model` · ~3-4h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Explain the happens-before relation and identify which Go operations
  (channel ops, mutex lock/unlock, sync/atomic) establish it.
- Identify a data race in code that "looks fine", explain why the memory model
  makes its behavior undefined, and fix it with proper synchronization.
- Explain the difference between stack and heap allocation in Go and why the
  compiler, not the programmer, decides placement.
- Use `go build -gcflags='-m'` to read escape analysis output and explain why
  specific values escape to the heap.
- Refactor a function so a value no longer escapes, and confirm the change
  with escape analysis output and benchmark allocations.

## Two contracts you've been trusting

Since S3 you've written concurrent code that passes the race detector, and in
the last lesson you watched the garbage collector pay for every heap
allocation. Both times you were leaning on machinery this lesson finally opens
up: the **memory model** (when is one goroutine's write visible to another?)
and **escape analysis** (which values land on the heap in the first place?).
They meet in the same place — what the compiler and CPU actually do with your
memory, as opposed to what the source code appears to say.

## The memory model is a visibility contract

Inside a single goroutine, Go behaves exactly as written: statements take
effect in program order, reads see the most recent write. Across goroutines,
that guarantee **disappears**. The compiler may reorder writes, the CPU may
buffer them, and a plain read in another goroutine may see an old value, a new
value — or, for multi-word values like strings, interfaces, and slices, a
torn half-and-half of both.

The Go memory model restores order through one relation: **happens-before**.
A read is *guaranteed* to observe a write only if the write happens-before the
read. Program order gives you happens-before within a goroutine; between
goroutines, only **synchronization operations** create edges:

- `go f()` happens-before the first statement of `f`.
- A channel send happens-before the corresponding receive completes. Closing
  a channel happens-before a receive that returns the zero value because of
  the close. (This is why "signal by close" from S3 is correct.)
- `mu.Unlock()` happens-before any later `mu.Lock()` — everything you did
  inside the critical section is visible to the next holder.
- `once.Do(f)`: `f` returning happens-before *any* `Do` call returns.
- `wg.Done()` happens-before `wg.Wait()` returning — after `Wait`, you may
  read what the workers wrote.
- `sync/atomic` operations behave like sequentially consistent operations: an
  atomic `Load` that observes an atomic `Store` gives you an edge from the
  storing goroutine. The typed wrappers are the ones to reach for — an
  `atomic.Pointer[T]` publishes a whole value behind one word:

  ```go
  var cfg atomic.Pointer[Config]     // zero value is ready to use
  cfg.Store(&Config{Timeout: time.Second})
  c := cfg.Load()                    // never nil once Store has happened
  ```

  Whatever the publishing goroutine wrote into that `Config` *before* the
  `Store` is visible to every goroutine whose `Load` observes it. That is the
  whole trick: one atomic word carries an arbitrarily large payload safely,
  because the pointer's publication drags the happens-before edge with it.
  Note what it does *not* give you — read-modify-write of the pointed-to
  value. Publish immutable snapshots and replace them wholesale; mutating a
  `Config` that other goroutines already hold is a race again.

Just as important is what is *not* on the list. A goroutine merely *exiting*
creates no edge. `time.Sleep` creates no edge — sleeping "long enough" is a
bet, not a guarantee. And an ordinary variable, read on one side and written
on the other with no edge between them, is a **data race**.

## "Looks fine" is not fine

Here is code you will meet in the wild, written by someone who reasoned
carefully and is still wrong:

```go
var (
	ready bool
	conn  *Conn
)

func Get() *Conn {
	if !ready {                // unsynchronized read
		conn = dial()      // unsynchronized writes
		ready = true
	}
	return conn
}
```

The author's logic: "I check the flag before touching `conn`, and setting a
bool is a single instruction — what could go wrong?" Everything, in three
ways:

1. **Reordering.** Nothing orders the write to `conn` before the write to
   `ready`. Another goroutine may observe `ready == true` while `conn` is
   still nil or points at a half-built value.
2. **Visibility.** With no happens-before edge, a goroutine may *never* see
   `ready` become true, or see it arbitrarily late. The compiler is allowed
   to hoist the read out of a loop entirely.
3. **Undefinedness.** The Go memory model says a program with a data race on
   non-trivial data has no defined behavior at all. Not "sees a stale value"
   — *undefined*. Racy reads of a string or interface can tear and crash the
   process. This is why there is no such thing as a "benign" race: you are
   not sampling possible outcomes, you are outside the language's guarantees.

The fix is always the same move: name the invariant, then pick the
synchronization that carries it. Here the invariant is "initialize once,
publish the result" — exactly `sync.Once` (or `sync.OnceValue`), which you
met in S3. Now you know *why* it works: `Do` builds the happens-before edge
that this code was missing.

## The race detector checks happens-before, not luck

You might expect races to be caught only when two goroutines collide at just
the wrong nanosecond. The race detector is better than that: it tracks the
happens-before graph of your execution (using vector clocks) and reports
whenever two accesses to the same address — at least one a write — have **no
edge** between them, *regardless of how they interleaved in time*. Because it
reads that graph rather than timing coincidences, it does not need the two
accesses to land at the same nanosecond, and in practice races on executed
paths are found reliably — which is why the exercise's race tests fail on the
starter code from your very first run, and why `go test -race` in CI is a gate
worth having rather than a dice roll.

It is still not a proof of absence. The detector only sees accesses that
actually execute, and the history it keeps per address is bounded — enough
other traffic to the same memory can evict the earlier access before its
partner arrives — so a race can slip past. What it never does is report a race
that is not there. A report is evidence; a clean run is only the absence of
evidence.

The detector's cost (2-20x slowdown, more memory) is also why this
curriculum's tests never assert wall-clock timing: under `-race` your program
is a different beast temporally, but its happens-before structure — the thing
that matters — is unchanged.

## Stack or heap? The compiler decides

Now the second contract. Every Go value lives in one of two places:

- **The stack** of a goroutine: allocation is bumping a pointer, deallocation
  is returning from the function. Near-free, and invisible to the GC.
- **The heap**: shared, outlives any frame, and — as the previous lesson made
  concrete — every heap allocation is future work for the garbage collector.

In C you choose with `malloc`; in Go you *can't* choose. There is no
"allocate on the heap" keyword — `new`, `&T{}`, and `var` are all just ways
to create values. The compiler runs **escape analysis** over your code and
proves, for each value, whether it can outlive its stack frame. If a value
provably dies with the frame, it goes on the stack, *even if you took its
address*. If it might outlive the frame — it "escapes" — it must go on the
heap.

The common ways a value escapes:

- **Returned by pointer**: `return &r` — the frame dies, the value must not.
- **Stored somewhere heap-reachable**: appended to a slice, put in a map,
  assigned to a struct field or global, sent on a channel.
- **Captured by a closure** that outlives the function.
- **Converted to an interface** that the compiler can't see through — the
  classic being every argument you pass to `fmt` functions (`...any`).
- **Too big or dynamically sized** for the stack.

None of this changes correctness. It changes *cost* — and in a hot path,
"one small allocation per call" multiplied by a million calls is the GC
pressure you learned to fear last lesson.

## Reading the compiler's mind: -gcflags='-m'

You don't have to guess: the compiler will tell you its escape decisions.

```sh
go build -a -gcflags='-m' . 2>&1 | grep -v 'can inline\|inlining call'
```

(`-a` forces a full recompile. Modern Go caches compiler diagnostics and
replays them, so you usually get the output without it — `-a` just
guarantees it. The grep drops inlining chatter; add `-m=2` when you want the
compiler's full reasoning chain.)

The lines that matter, from this exercise's starter:

```
./report.go:14:7: &Report{} escapes to heap        ← heap-allocated: returned by pointer
./format.go:15:50: value escapes to heap           ← boxed into fmt's ...any
./report.go:33:16: xs does not escape              ← param stays stack-only
./race.go:46:15: leaking param: c                  ← callers beware: c is retained
```

"escapes to heap" / "moved to heap" mean a real allocation. "does not escape"
is the compiler telling you a pointer parameter is safe. "leaking param"
means the function stores or returns the pointer you passed in — the *caller's*
value may escape because of it.

## Prove it with numbers, never with a stopwatch

Escape analysis output says *what* the compiler decided; two tools confirm
*what it costs*:

- `testing.AllocsPerRun(n, f)` runs `f` many times and returns the average
  number of heap allocations per run — a **deterministic** count, identical
  under `-race`, on a laptop or in CI. The exercise tests use it to gate
  "zero allocations" behavior.
- `go test -bench=. -benchmem` reports `allocs/op` and `B/op` next to the
  timing. The allocation columns are stable; the ns/op column is not — which
  is why benchmarks here are something you *read*, and tests never assert
  timing.

The refactoring pattern you'll practice is the one the standard library uses
everywhere (`strconv.AppendInt`, `time.Time.AppendFormat`): instead of
building and returning a new string or object, **append into a caller-owned
`[]byte`**. The caller reuses one buffer; steady-state allocations drop to
zero.

## Exercise

Open [`exercise/`](exercise/) — module `tutor.local/memory-model` with two
themes:

- `race.go` — a `Meter` (event counter + last event) and a config `Store`,
  both written in the "looks fine" style. `race_test.go` hammers them from
  several goroutines.
- `format.go` / `report.go` — a metrics-line formatter and a summary
  function that allocate more than they should. `format_test.go` and
  `report_test.go` gate correctness *and* allocation counts;
  `bench_test.go` you run by hand.
- `NOTES.md` — where you record escape-analysis output and benchmark
  numbers. **Fill in section 1 before you change any code** — once you've
  refactored, the "before" output is gone.

Acceptance criteria:

1. `go test -race ./...` passes with no data-race reports.
2. `Meter` is safe for concurrent use and its two fields stay consistent:
   after N concurrent `Record` calls, `Snapshot` returns exactly N hits, and
   never a positive count with an empty last event. The two fields form one
   invariant — protect them as one unit.
3. `Store` publishes configs safely: concurrent `Update`/`Current` are
   race-free and the last update wins. Use `atomic.Pointer[Config]` — a
   mutex also works, but single-word publication is exactly what atomics are
   for, and you should leave this lesson having used one.
4. `AppendSample(dst, name, v)` appends exactly `<name> <value>\n` and makes
   **zero** heap allocations when `dst` has capacity (the test measures with
   `testing.AllocsPerRun`). Build the line with `append` and `strconv` —
   look at `strconv.AppendInt`.
5. `Summarize` returns a correct `Report` **by value** with zero allocations.
   `SummarizeHeap` is the "before" specimen: benchmark it, read its escape
   line, but don't call it from `Summarize`.
6. `NOTES.md` is filled in: the `-m` lines before and after, one-sentence
   explanations of *why* each value escaped, and the `-benchmem` numbers for
   both `Summarize` benchmarks.

Run everything from inside `exercise/`:

```sh
go test -race ./...                                       # the gate
go build -a -gcflags='-m' . 2>&1 | grep -v 'can inline\|inlining call'
go test -bench=Summarize -benchmem                        # read, don't gate
```

The tests fail on the starter — the race tests under `-race`, the allocation
and correctness tests everywhere. Make them green, then bring your NOTES.md
to the review.

## Further reading

- [The Go Memory Model](https://go.dev/ref/mem) — the contract itself; short
  and worth reading end to end now that you have the vocabulary.
- [Data Race Detector](https://go.dev/doc/articles/race_detector) — options,
  typical races, and runtime cost.
- [A Guide to the Go Garbage Collector](https://go.dev/doc/gc-guide) — the
  "Where Go Values Live" discussion connects escape analysis to GC cost.
- [pkg.go.dev — sync/atomic](https://pkg.go.dev/sync/atomic) — the typed API
  (`atomic.Pointer[T]` and friends) you use in this exercise.
