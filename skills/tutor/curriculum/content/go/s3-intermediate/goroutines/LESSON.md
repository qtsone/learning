# Goroutines

> `go.intermediate.goroutines` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement concurrent work with the `go` keyword and coordinate completion
  with `sync.WaitGroup`.
- Explain why goroutines are cheaper than OS threads and what happens when
  `main` returns while goroutines still run.
- Detect a data race with `go test -race`, explain the report, and fix the
  race.
- Explain what a goroutine leak is and identify one in given code.
- Explain why `WaitGroup.Add` must happen before starting the goroutine, not
  inside it.

## One keyword

Everything you have written so far does one thing at a time. Go's entire
concurrency story starts with a single keyword in front of a function call:

```go
go download(url)
```

This starts `download` in a new **goroutine** — an independently scheduled
thread of execution — and moves on *immediately*. The statement returns
nothing: no result, no handle you can join on, no way to ask "are you done?"
Two details are worth pinning down now:

- The *arguments* are evaluated right away, in the current goroutine. Only
  the call itself runs concurrently.
- Any function works: named, method value, or (most commonly) a closure —
  the closures lesson is about to pay off, because goroutine bodies usually
  capture the variables around them.

Completion and results are entirely your problem. This lesson gives you the
first tool for it; the next lesson (Channels) gives you the second.

## Why goroutines are cheap

You may have heard "don't spawn thousands of threads." Goroutines are not
threads, and the difference is why Go programs casually run hundreds of
thousands of them:

- **Stack size.** An OS thread reserves a fixed stack up front — megabytes.
  A goroutine starts with a tiny stack (about 2KB) that grows and shrinks as
  needed. A million goroutines can fit where a few thousand threads would
  exhaust memory.
- **Scheduling.** Threads are scheduled by the kernel; switching between
  them means a context switch into kernel space, saving and restoring
  register state. Goroutines are scheduled by the Go runtime, in user space,
  multiplexed onto a small pool of OS threads (sized by `GOMAXPROCS`,
  defaulting to your CPU count). Parking a goroutine that is waiting costs
  roughly a function call.

One distinction to keep straight: **concurrency** is structure — your
program is composed of independently executing pieces. **Parallelism** is
execution — pieces literally running at the same instant on different cores.
Goroutines give you concurrency always; parallelism only when cores are free
and the work is big enough to be worth spreading.

## main does not wait

Run this and you will probably see one line, not two:

```go
func main() {
	go fmt.Println("from the goroutine")
	fmt.Println("from main")
}
```

When `main` returns, the program exits — *immediately*. Every other
goroutine is killed mid-flight: no panic, no message, no deferred functions
run. The goroutine above usually never gets scheduled before the process is
gone. The runtime never waits for goroutines you started; if the program
should outlive a piece of work, something must explicitly wait for it.

The beginner "fix" is `time.Sleep(100 * time.Millisecond)` before returning.
It makes the demo print — and it is wrong. A sleep is a guess: too short and
you are flaky, too long and you are slow, and either way it says nothing
about whether the work actually finished. Write this rule down, because the
whole concurrency arc enforces it: **a sleep is never synchronization.**
Every exercise from here to the capstone must pass with zero sleeps.

## sync.WaitGroup

The real tool is `sync.WaitGroup` — a thread-safe counter with three
methods. `Add(n)` increases the counter, `Done()` decreases it by one, and
`Wait()` blocks until it reaches zero. The canonical fan-out shape:

```go
var wg sync.WaitGroup
for _, url := range urls {
	wg.Add(1)
	go func() {
		defer wg.Done()
		download(url)
	}()
}
wg.Wait()
```

Read the choreography: before each launch the counter goes up; inside each
goroutine, `defer wg.Done()` guarantees the counter comes down on *every*
exit path — normal return, early return, even a panic (S1's defer semantics,
now load-bearing). `Wait` releases only when all launched work has counted
down. Two mechanical notes: a `WaitGroup` must not be copied after first use
(pass a pointer if a function needs it — `go vet` catches this), and
`Add(len(urls))` once before the loop is exactly as correct as `Add(1)` per
iteration.

`Wait` does more than block. It *synchronizes*: everything a goroutine wrote
before calling `Done` is guaranteed visible to the code after `Wait`
returns. The jargon is that `Done` **happens before** `Wait` returning. This
is what makes the pattern "goroutines write results into memory, `Wait`,
then read the results" correct rather than luck — and it is the shape of
your whole exercise.

## Add before go, not inside

Here is the same loop with one line moved, and it is broken:

```go
for _, url := range urls {
	go func() {
		wg.Add(1) // BROKEN: may run after Wait already returned
		defer wg.Done()
		download(url)
	}()
}
wg.Wait()
```

The `go` statement schedules the goroutine; it makes no promise about *when*
it first runs. Every launched goroutine may still be unstarted when the loop
ends — counter zero, nothing to wait for — so `Wait` sails straight through
and the function returns while the work runs on, or before it starts at all.
The poison is that it usually "works" on your machine: the goroutines often
do start in time, until one day under load they don't.

The rule: `Add` must run where `Wait` can see it — in the launching
goroutine, before the `go` statement. (Go 1.25 added `wg.Go(f)`, which
bundles Add/launch/Done correctly; this module pins Go 1.22, and the classic
form is what fills existing codebases, so practice it.)

## Closures, loops, and capture

Goroutine bodies capture surrounding variables by reference — the closures
lesson, verbatim. One historical trap you must recognize: before Go 1.22, a
`for` loop had *one* loop variable reused across iterations, so every
goroutine captured the same `i` and usually observed its final value. Since
Go 1.22 each iteration gets fresh variables, and the famous bug is dead in
modern modules — but you will still meet it in old code, old blog posts, and
interviews. The alternative style, passing data as an argument
(`go func(u string) { … }(url)`), still works and shows intent explicitly:
arguments are evaluated at the `go` statement, in the current goroutine.

## Data races and the detector

A **data race** is: two goroutines access the same memory location
concurrently, at least one access is a write, and nothing orders them. The
classic way to manufacture one:

```go
total := 0
for range 8 {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1000 {
			total++ // read, add, write — three steps, interleavable
		}
	}()
}
```

`total++` is not one operation: two goroutines can both read 41, both write
42, and an increment is gone. But "loses updates" undersells the problem. Go
defines program behavior only for race-free programs; the compiler and CPU
reorder memory operations on that assumption, so a racy program has no
meaning you can reason about. "I ran it and got 8000" proves nothing.

Because eyeballs are terrible at this, Go ships a **race detector**:

```sh
go test -race ./...
```

It instruments every memory access, and when two conflicting, unsynchronized
accesses actually occur it prints a report: `WARNING: DATA RACE`, the memory
address, then two stack traces — the access that tripped it and the previous
conflicting one — each tagged with which goroutine did it and where that
goroutine was created. Read a report as: *what variable* (the two stacks
point at the same line or the same field), *which two goroutines*, *who
forgot to synchronize*. Know its limits: it is a dynamic tool that only sees
races your run actually executes, so a clean run is evidence, not proof —
but every report is a real bug, never a false alarm. It costs roughly 2-20x
time and memory, which tests can afford: from this lesson to the end of the
stage, run every test with `-race`.

How do you *fix* a race with today's toolbox? Don't share. Give each
goroutine memory it exclusively owns — its own slice index, its own
accumulator cell; distinct elements of a slice are distinct memory — and
only read the results after the synchronization point (`Wait`). Channels
(next lesson) and mutexes (the Sync lesson) add tools for when sharing is
genuinely necessary; they change nothing about the definition.

## Goroutine leaks

A **goroutine leak** is a goroutine that can never proceed and never
terminate. Go gives you no way to kill a goroutine from outside, so a leaked
one keeps its stack — and everything it references — alive for the rest of
the process. One leak is invisible; a server leaking one per request dies
slowly, which is the worst way to die. Here is this lesson's flavor:

```go
for _, url := range urls {
	wg.Add(1)
	go func() {
		if err := fetch(url); err != nil {
			return // forgot Done — the counter never reaches zero
		}
		wg.Done()
	}()
}
wg.Wait() // if any fetch failed, blocks forever
```

Spot who leaks: not the failed worker — it returned and is gone. The leak is
the *waiter*: the goroutine stuck in `Wait` forever, plus everything it
holds. `defer wg.Done()` as the first line of the goroutine makes this bug
impossible — every exit path counts down. Most real-world leaks block on
channel operations instead; that variant is next lesson's. A crude probe you
can use today: `runtime.NumGoroutine()` before and after in a test. In
production the same census comes from pprof's goroutine profile — a sibling
of the CPU and heap profiles you will tour in the Tooling lesson.

## Exercise

Open [`exercise/`](exercise/) — a Go module with package `parallel`, a small
fan-out toolkit. `parallel.go` has three functions marked `TODO`;
`parallel_test.go` is the specification. Read the tests first, including the
`barrier` helper: it uses a second WaitGroup as a gate that opens only when
all goroutines are running *at once*, so a secretly sequential
implementation cannot pass (a watchdog fails loudly rather than hanging).

Acceptance criteria:

1. `RunAll(tasks)` runs every task in its own goroutine and returns only
   after all of them have finished. Empty input returns immediately.
2. `Map(in, f)` applies `f` concurrently — one goroutine per element — and
   returns the results in input order: `out[i] == f(in[i])`, length
   preserved, empty input included.
3. `Total(nums, workers)` sums `nums` using `workers` goroutines, each
   summing one contiguous chunk into its own subtotal cell, combined after
   `Wait`. Correct for uneven splits, `workers > len(nums)`, and empty
   input.
4. Every `Add` happens before its `go`; every goroutine defers its `Done`;
   no sleeps anywhere; no shared memory without a happens-before edge.
5. `go test -race ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`, race detector always on:

```sh
cd exercise
go test -race ./...
```

One piece of choreography, and don't skip it: when you reach `Total`, write
the tempting version first — a single shared `total` every worker adds to.
Run the tests with `-race` and *read the report*: find the two stack traces,
the shared variable, the goroutine creation sites. Then fix it by ownership
(one cell per worker), not by cleverness. Producing and dissecting one real
race report is the most valuable rep in this lesson.

## Further reading

- [Effective Go — Goroutines](https://go.dev/doc/effective_go#goroutines) —
  the canonical short statement of what a goroutine is.
- [Data Race Detector](https://go.dev/doc/articles/race_detector) — the
  user manual: report format, typical races, options.
- [The Go Memory Model](https://go.dev/ref/mem) — read the introduction and
  its advice; the happens-before guarantee `Wait` gave you today is defined
  here.
- [Concurrency is not parallelism](https://go.dev/blog/waza-talk) — Rob
  Pike's talk on the structure/execution distinction.
