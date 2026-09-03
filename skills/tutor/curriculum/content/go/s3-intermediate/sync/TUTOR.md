# Tutor notes — Sync Primitives

## Where the learner is

Eleventh lesson of S3, straight after goroutines and channels. They can spawn
goroutines, wait with `WaitGroup`, run `-race`, and build channel pipelines
with `select` and `close`. Two lessons of "share memory by communicating" may
make locks feel like a regression — frame this lesson as completing the
toolbox, not retracting the advice. `context` and worker pools are next; if
they ask about cancellation or bounding goroutines, park it: "next lesson".
Their S2 hash-table work means the map-behind-a-lock shape should feel
familiar from the data side; the synchronization is the only new part.

## Common misconceptions

- **"Reads don't need the lock"** — the big one. They'll lock `Inc` and
  leave `Value` bare. Have them run `go test -race` and read the report:
  it names the unlocked read and the locked write. Reinforce: a data race
  means *undefined behavior*, not "slightly stale value"; unsynchronized map
  access can kill the process outright.
- **"A clean `-race` run proves the code is race-free"** — the detector only
  flags races that occur in the interleavings it observed. Clean run =
  evidence; report = certain bug. Never let them dismiss a report as flaky.
- **"RWMutex is the better Mutex"** — upgrading everything to `RWMutex` "for
  performance". It costs more per operation; the win exists only when reads
  heavily dominate with real reader concurrency. Default is `Mutex`.
- **Writing under `RLock`** — `Set` with `RLock`/`RUnlock` compiles fine and
  is a data race. The store test under `-race` catches it; make them explain
  *why* the compiler can't.
- **Check-then-act with `if l.cfg == nil`** — looks correct, races twice
  (the unsynchronized read of `cfg`, and the double load). The once test's
  call counter catches double loads; `-race` catches the read. This is the
  moment to name the pattern so they recognize it forever.
- **"Atomics fix everything"** — trying to guard the map with "atomic"
  thinking, or believing two `atomic.Int64`s preserve a joint invariant.
  Atomics protect one word; compound invariants need the lock.
- **Value receivers on locked types** — `func (c Counter) Inc` copies the
  mutex; the map is shared but the lock is not. `go vet` (copylocks) flags
  it; the concurrent tests usually fail too. Connect to S1's value-vs-pointer
  receiver lesson: sync types make the choice load-bearing.
- **Returning the guarded map** — `Snapshot` returning `c.counts` directly.
  Every method locks, yet callers mutate shared state lock-free. The
  snapshot test catches it; the lesson's phrase is "never hand out references
  to guarded data".

## Grilling points

- "Delete the lock from `Value` and run `go test -race`. Read the report
  aloud — which two lines does it accuse, and of what?"
- "`n++` lost updates in the lesson's demo. Walk me through the exact
  interleaving of load/add/store that loses one."
- "Why does `Snapshot` copy the map while *holding* the lock? What breaks if
  you unlock first, then copy?"
- "When would you demote `Store`'s `RWMutex` to a plain `Mutex`? What would
  you measure to decide?" (Read/write ratio, contention, critical-section
  cost — benchmarks, not vibes.)
- "Your `Loader` — what goes wrong with `if l.cfg == nil { l.cfg = l.load() }`
  under `-race`? Two distinct answers." (Racy read of `cfg`; duplicate
  loads.)
- "Product now wants `Hits` and `Misses` with a hit-*ratio* readable at any
  instant. Still two atomics? Why not?" (Reader can observe between the two
  updates; joint invariant needs a mutex.)
- "Change `Inc` to a value receiver. What does `go vet` print, and what
  actually happens at runtime — is the map corrupted, unprotected, or both?"
- "You built `Counter` with a mutex. Sketch the channel version — owner
  goroutine, request channels. When would that design win?" (E.g. when
  access must compose with `select`, or ownership/lifecycle is the point —
  preview of patterns lesson.)

## Grading rubric

- **A** — All tests pass under `-race`; `go vet` clean; every guarded access
  is inside the lock with `defer` unlocks; `Get`/`Len` use `RLock`;
  `Snapshot` copies under the lock; `Loader` uses `once.Do` (no
  check-then-act); `Hits` is pure atomic. Learner explains a race report,
  the Mutex-vs-RWMutex tradeoff, and channels-vs-locks unprompted.
- **B** — Tests pass under `-race` but with design misses: `Lock` instead of
  `RLock` in `Store` reads, manual copy loop plus an unneeded lock re-take,
  or `Unlock` without `defer` in multi-return functions. Explanations mostly
  solid; RWMutex reasoning fuzzy.
- **C** — Tests pass only after ladder hints, or the learner can't say why
  reads need the lock / what the race report means. Pass only if a
  re-explanation in their own words lands; otherwise iterate.
- **Fail** — Tests or `-race` failing; a mutex copied (vet report ignored);
  `Loader` "fixed" with sleeps or a busy-wait; or the solution works but the
  learner can't explain what the lock protects. Remediate — this lesson's
  concepts are load-bearing for everything after it.

## Remediation ladder

1. "Run `go test -race ./...` and read the *first* report or failure aloud.
   Which function, which line, read or write?"
2. "List every line that touches `c.counts` (or `s.vals`). For each: which
   lock is held at that moment? Every touch must sit between Lock and
   Unlock — no exceptions for reads."
3. For `Loader`: "`Config` must do three things: run `load` once, remember
   the result, hand the same result to everyone. Which of the three does
   `once.Do` give you for free, and where must the result live so the other
   callers can see it?"
4. Sketch `Inc`'s body shape — `c.mu.Lock()`, `defer c.mu.Unlock()`, one
   line of work — and have them mirror that shape across the remaining
   methods themselves, choosing `RLock` where the method only reads.

## After passing

Preview: "Next is `context` — how you tell goroutines to stop: cancellation,
deadlines, and threading both through call chains. Locks protect state;
context manages lifetime."
