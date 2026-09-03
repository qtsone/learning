# Sync Primitives

> `go.intermediate.sync` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement safe shared-state access with `sync.Mutex` and explain what
  happens without it under the race detector.
- Choose between `Mutex` and `RWMutex` for a given read/write ratio and
  justify the choice.
- Implement one-time initialization with `sync.Once` and lock-free counters
  with `sync/atomic`.
- Choose between channels and locks for a given concurrency problem and
  justify the choice (share memory by communicating vs protect state).
- Explain why copying a struct containing a mutex is a bug and how `go vet`
  catches it.

## Sometimes state really is shared

The last two lessons preached *share memory by communicating*: hand values
from goroutine to goroutine through channels so that only one goroutine
touches a piece of data at a time. That is the right default for moving work
through a program — pipelines, fan-out, signaling.

But some data isn't *moving* anywhere. A request counter, a cache, a
configuration loaded at startup — many goroutines need the *same* piece of
state, forever. You *could* dedicate a goroutine to own it and serve
`get`/`set` requests over channels, and sometimes that design is right. But
for "several goroutines read and write one map", it's ceremony: two channels,
a request struct, a serving loop — to guard a two-line critical section. Go
ships the `sync` package because locks are the simplest correct tool for
protecting shared state in place. The proverb has a second half people forget:
channels for *communication*, mutexes for *serialization*.

## The race, one more time

You met the race detector in the goroutines lesson. Here is the enemy at its
smallest:

```go
var n int
var wg sync.WaitGroup
for range 2 {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1000 {
			n++ // read n, add 1, write n — three steps, not one
		}
	}()
}
wg.Wait()
fmt.Println(n) // 2000? Often 1347, or 1998, or…
```

`n++` compiles to *load, add, store*. Two goroutines can both load `1000`,
both add, both store `1001` — one increment vanishes. Worse, the Go memory
model says a program with a data race has no defined behavior at all; you're
not guaranteed even a "mostly right" answer. Maps are stricter still: the
runtime detects unsynchronized map access and kills the program outright
(`fatal error: concurrent map writes`) — that one isn't even a recoverable
panic.

`go test -race` catches this class of bug and names both sides:

```
WARNING: DATA RACE
Read at 0x00c000014088 by goroutine 8:
  tutor.local/sync.(*Counter).Value()
      counter.go:24 +0x3c
Previous write at 0x00c000014088 by goroutine 7:
  tutor.local/sync.(*Counter).Inc()
      counter.go:18 +0x44
```

Read it as an accusation with two exhibits: this line read the memory, that
line wrote it, and nothing ordered the two. One caveat to carry with you: the
detector only sees races that actually *happen* during the run. A clean run
is evidence, not proof. A report, however, is always a real bug — never
"flaky test", always "fix the synchronization".

## sync.Mutex: one goroutine at a time

A `Mutex` (mutual exclusion lock) has two methods. `Lock` blocks until the
lock is free, then takes it; `Unlock` releases it. Everything between the two
is a **critical section** — at most one goroutine runs it at a time.

The idiom is a struct that owns both the lock and the data it guards, with
the mutex declared directly above the fields it protects:

```go
type Counter struct {
	mu     sync.Mutex
	counts map[string]int
}

func (c *Counter) Inc(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[name]++
}
```

`defer c.mu.Unlock()` right after the `Lock` is the standard shape: the lock
is released however the function exits, and the pairing is visible at a
glance. Three rules to internalize:

- **Reads need the lock too.** A read racing a write is still a data race —
  the reader can observe a half-updated value, and for maps it can crash.
  "It's just a read" is the most common way to lose to `-race`.
- **Keep critical sections small.** Lock, touch the data, unlock. Don't hold
  a lock across I/O, channel operations, or calls into code you don't control
  — that's how contention (and deadlock) happens.
- **A `Mutex` is not reentrant.** If a locked method calls another method
  that locks the same mutex, the goroutine deadlocks *on itself*. Structure
  helpers as unexported methods that assume the lock is already held.

One more consequence of "the struct owns its lock": once data is guarded,
never hand out references to it. If `Snapshot` returned the internal map
itself, callers could mutate it after the lock was released — perfectly
race-y despite every method locking correctly. Return a copy made while
holding the lock (the `maps.Clone` function from Go 1.21's `maps` package
does it in one line).

## sync.RWMutex: many readers, one writer

Some state is read constantly and written rarely — configuration, routing
tables, caches. A plain mutex makes fifty concurrent readers queue up
single-file for no reason: readers don't conflict with each other, only with
writers. `sync.RWMutex` encodes exactly that:

- `RLock` / `RUnlock` — the *read* lock. Any number of goroutines may hold
  it simultaneously.
- `Lock` / `Unlock` — the *write* lock. Exclusive: it waits for all readers
  to leave and blocks new ones.

So: read-only methods take `RLock`, mutating methods take `Lock`. How do you
choose between `Mutex` and `RWMutex`? By the read/write ratio and the cost of
the critical section. `RWMutex` does more bookkeeping per operation than
`Mutex`, so for tiny critical sections with modest concurrency the plain
mutex often wins outright. The honest rule: **default to `Mutex`; reach for
`RWMutex` when reads heavily dominate** (think hundreds of reads per write)
**and there's real reader concurrency** — and if it matters, measure, using
the benchmark skills from your S2 timing probes. Be careful never to mix the
pairs up: writing under `RLock` is a data race the detector will happily
demonstrate.

## sync.Once: exactly once, no matter who asks first

Lazy initialization has a trap. This looks reasonable and is wrong:

```go
if l.cfg == nil {       // two goroutines both see nil…
	l.cfg = l.load()    // …and both load. Also: racy read of l.cfg.
}
```

This is **check-then-act**: between the check and the act, another goroutine
can do the same thing. `sync.Once` solves it:

```go
type Loader struct {
	once sync.Once
	cfg  *Config
}

func (l *Loader) Config() *Config {
	l.once.Do(func() { l.cfg = l.load() })
	return l.cfg
}
```

`Do` guarantees the function runs exactly once per `Once` value, and — the
subtle, load-bearing part — every caller *waits* until that one run has
completed and then *sees its effects*. The second goroutine doesn't skip
ahead and read a half-built `cfg`; the `Once` establishes the
happens-before edge, which is why the race detector blesses this code. Since
Go 1.21 the stdlib also offers `sync.OnceValue(f)`, which wraps a function
and returns a cached-result version of it — the same machine, prepackaged.
Use `Once` for init that must be lazy (expensive, or dependent on runtime
information); for init that can just happen at startup, plain package
initialization is simpler.

Note what `Once` does *not* do: it remembers that `Do` ran, not that it
*succeeded*. If the function can fail, a failed first attempt is permanent —
you'd cache the error too, or rethink the design.

## sync/atomic: lock-free, for one word at a time

For a single integer counter, a mutex is more machinery than the job needs.
The `sync/atomic` package provides operations the CPU performs indivisibly —
no lock, no critical section, no lost updates. Since Go 1.19 the typed API is
the one to use:

```go
var hits atomic.Int64

hits.Add(1)        // atomic increment
n := hits.Load()   // atomic read
```

`atomic.Int64`, `atomic.Bool`, `atomic.Pointer[T]` and friends are structs
you embed in your own types; their zero values are ready to use. The race
detector understands them — concurrent `Add` and `Load` are ordered and
race-free.

The limit arrives fast: atomics protect **one word**. The moment an invariant
spans two values — "increment `total` and `errors` together", "check the
count, then append" — separate atomics don't compose into one atomic step.
Another goroutine can observe the state *between* your two operations, and
check-then-act is back. Compound invariants want a mutex. The honest scope of
`sync/atomic` in application code: simple counters, gauges, and flags. When
in doubt, take the lock — a mutex you can reason about beats an atomic you
can't.

## Copying a mutex is a bug

A `sync.Mutex` is an ordinary struct value, and Go copies struct values
freely — assignment, function arguments, value receivers. That's precisely
the problem: **a copy of a mutex is a new, independent mutex** that happens
to start in whatever state the original was in. Two copies means two locks
"guarding" the same logical data — which is to say, no lock at all.

The classic way to write this bug is a value receiver:

```go
func (c Counter) Inc(name string) { // receiver copies c — and c.mu
	c.mu.Lock()                     // locks the copy's mutex
	c.counts[name]++                // map is shared; the lock is not
	c.mu.Unlock()
}
```

Maps are reference-like (S1), so `c.counts` still aliases the real data — but
each call locks a throwaway copy of the mutex, so the "protection" protects
nothing. The same bug hides in ranging over a slice of structs that contain
mutexes, or passing such a struct by value.

You don't have to catch this by eye. `go vet` ships a `copylocks` check:

```
$ go vet ./...
./counter.go:17:9: Inc passes lock by value: tutor.local/sync.Counter contains sync.Mutex
```

This is why every `sync` type's documentation says "must not be copied after
first use", and why types containing sync primitives use pointer receivers
and get passed as pointers, always. Make `go vet ./...` part of your reflex;
the tooling lesson later this stage builds it into your workflow properly.

## Channels or locks?

Both tools make concurrent access safe. Choosing well is a design skill, and
the razor is short: **what is the data doing?**

- **Moving between goroutines** — work items to workers, results back,
  "we're done" signals: channels. You transfer *ownership*; one goroutine
  touches the value at a time, no lock needed. This is the pipeline world of
  the last lesson.
- **Sitting in one place, accessed by many** — counters, caches, registries,
  config: a mutex (or `RWMutex`). The state stays put; you serialize access
  to it.
- **A single word** — one counter, one flag: `sync/atomic`.
- **Init that must happen once** — `sync.Once`.
- **Waiting for goroutines to finish** — `sync.WaitGroup`, as you've been
  doing since the goroutines lesson.

Test your judgment: could you build this lesson's `Counter` as a goroutine
owning the map, with `Inc` and `Value` as channel round-trips? Absolutely —
and it would be triple the code to express "protect two lines from
interleaving". Could you build the previous lesson's pipeline with a shared
slice and a mutex? Also yes — and you'd reinvent coordination that channels
give you for free, badly. Neither tool is more "idiomatic Go" than the other;
*matching the tool to the shape of the problem* is the idiom.

## Exercise

Open [`exercise/`](exercise/) — a Go module with three files to complete and
their tests:

- `counter.go` — `Counter`, a mutex-guarded map of named counts (`Inc`,
  `Value`, `Snapshot`), and `Hits`, a lock-free `atomic.Int64` counter.
- `store.go` — `Store`, a read-mostly key/value store built on
  `sync.RWMutex`: `Get` and `Len` take the read lock, `Set` the write lock.
- `once.go` — `Loader`, which uses `sync.Once` so an expensive `load`
  function runs exactly once no matter how many goroutines call `Config`.

The tests hammer every type from multiple goroutines (with `WaitGroup`s and
exact-count assertions — no sleeps), so an unsynchronized "solution" fails
loudly, especially under `-race`.

Acceptance criteria:

1. `Counter.Inc`/`Value`/`Snapshot` are safe under concurrent use: 8
   goroutines x 200 `Inc`s yield exactly 1600, and reads racing writes are
   clean under `-race`.
2. `Snapshot` returns a *copy*: mutating the returned map does not affect the
   counter.
3. `Store.Get`/`Set`/`Len` are concurrency-safe; `Get` and `Len` use the read
   lock (`RLock`), `Set` the write lock — be ready to justify the `RWMutex`
   choice for this read-mostly store.
4. `Loader.Config` runs `load` exactly once under 16 concurrent callers, and
   every caller gets the same `*Config`. No `if cfg == nil` check-then-act.
5. `Hits` uses `sync/atomic` (no mutex): concurrent `Inc`s are never lost.
6. `go test -race ./...` passes, `go vet ./...` reports nothing (pointer
   receivers everywhere — a value receiver would copy the lock), and the code
   is `gofmt`-clean.

Run the checks from inside `exercise/`:

```sh
cd exercise
go test -race ./...
go vet ./...
```

Run with `-race` every time on this and every future concurrency exercise —
the flag is the whole point. Expect failures on the starter; read them first.
While you work, ask of every line that touches shared data: *which lock is
held right now, and who else can reach this?*

## Further reading

- [pkg.go.dev — sync](https://pkg.go.dev/sync)
- [pkg.go.dev — sync/atomic](https://pkg.go.dev/sync/atomic)
- [Go wiki — MutexOrChannel](https://go.dev/wiki/MutexOrChannel)
- [go.dev — Data Race Detector](https://go.dev/doc/articles/race_detector)
