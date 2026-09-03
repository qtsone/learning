# Tutor notes — Channels

## Where the learner is

One lesson into the concurrency arc. From the goroutines lesson they have the
`go` keyword, `WaitGroup`, and — importantly — the race detector habit and
the "no sleeps as synchronization" rule. Channels, `select`, `close`, and nil
channels are all first contact. No mutexes yet (next lesson), no context (two
ahead), no worker pools or errgroup (patterns lesson) — don't reach for those
when explaining; the done-channel idiom is as far as cancellation goes here.
Their mental model of "waiting" is `wg.Wait()`; the conceptual leap is that a
channel synchronizes *and* transfers a value in one operation.

## Common misconceptions

- **"A buffer fixes the deadlock"** — it delays it until the buffer fills.
  Probe with: what capacity would you pick, and what happens at capacity+1?
  Default is unbuffered; a buffer needs a nameable reason.
- **"Every channel must be closed or it leaks"** — channels are garbage
  collected regardless. `close` is a broadcast signal for receivers (`range`,
  done-channels), not cleanup.
- **"The receiver can close when it has had enough"** — only the sender
  closes; a receiver closing makes the sender's next send panic.
- **"Receiving from a closed channel blocks or panics"** — it *never blocks*:
  buffered leftovers first, then zero values with `ok == false`. Only *send*
  and *re-close* panic. The 3×3 table in the lesson is the reference; make
  them reproduce it.
- **"select tries cases top to bottom"** — uniformly random among ready
  cases. Anyone encoding priority as case order has this wrong.
- **Sends outside the goroutine** — writing `Generate` with the loop before
  `return`: blocks forever on the first unbuffered send. The
  `returnsPromptly` helper fails with a message naming exactly this.
- **Leaving an exhausted channel in the select** — a closed channel is
  *always* ready, so the loop spins on zero values (or forwards them). The
  nil-out move exists precisely because closed and nil are opposites.
- **Expecting merge order** — `MergeTwo`'s output order is scheduler
  business; the tests sort before comparing. If the learner asks why, that's
  a teachable moment about nondeterminism being normal.
- **Sleeps creeping back in** — any `time.Sleep` in their solution is an
  automatic conversation: what were you waiting for, and which channel
  operation already waits for it?

## Grilling points

- "Walk me through `ch := make(chan int); ch <- 1; fmt.Println(<-ch)` in
  `main`. What happens, and why does the runtime *know* it can crash?"
  (Every goroutine asleep — no one can ever wake anyone.)
- "Delete the `defer close(out)` from `Square`. Which test fails, what does
  the failure say, and what would happen in a real program instead?"
  (Timeout message here; silent goroutine leak in production.)
- "The interleaved-producer test deadlocks a drain-a-then-b MergeTwo.
  Trace the deadlock chain." (Merge waits on `a`; producer is stuck sending
  `b <- 2`; nobody moves.)
- "After `close(done)`, could a consumer still receive one more value from
  `Counter`? Why is that acceptable?" (Both cases ready → random pick;
  cancellation is asynchronous. The guarantee is *eventual* close.)
- "Why does `TryRecv(nil)` work with no nil check?" (A nil case is never
  ready, so `default` runs — the same select covers all four situations.)
- "You send a `*Report` through a channel and the receiver mutates it. Data
  race?" (No — send happens-before receive — *provided* the sender stops
  touching it. Ownership transfer, and the race detector confirms.)
- "When is a buffered channel the right call? Give a concrete example with a
  capacity." (Known burst size, decoupling rates; seed for the semaphore
  pattern in the patterns lesson — don't unpack it yet.)

## Grading rubric

- **A** — All tests pass under `-race`; every stage owns its output and
  closes via `defer close(out)` at the top of its goroutine; `MergeTwo` is a
  single select loop using comma-ok and nil-ing (or an alternative they can
  defend); `TryRecv` is one select, no nil special-case; no sleeps anywhere.
  Learner reproduces the nil/open/closed table and explains random select
  choice and the interleaved-producer deadlock unprompted.
- **B** — Tests pass but with roughness: manual close on each exit path
  instead of defer, boolean `aDone`/`bDone` flags alongside correct nil-ing,
  or a shaky-but-recoverable account of buffered blocking semantics.
- **C** — Tests pass only after heavy hinting, or the learner cannot explain
  why the interleaved test exists or why a closed channel's receive never
  blocks. Pass only if remediation lands within the session; otherwise
  iterate.
- **Fail** — Tests failing or hanging; a `time.Sleep` doing synchronization
  work; or `MergeTwo` passing by accident while the learner sees nothing
  wrong with sequential draining after the deadlock-chain question.
  Remediate, don't advance.

## Remediation ladder

1. "Run `go test -race -run TestGenerate -v`. Read the failure aloud — did
   Generate *block*, or did it emit the *wrong values*? The message tells
   you which mistake you made."
2. "In your `Generate`, which goroutine executes `out <- n`? At that moment,
   who is receiving? An unbuffered send needs both sides to exist."
3. "For `MergeTwo`: in a select, is a *closed* channel's receive case ready
   or not ready? And a *nil* channel's? Which behavior do you want from an
   input that is finished?"
4. Sketch the shape verbally — goroutine, `defer close(out)`, loop while
   `a != nil || b != nil`, comma-ok per case, nil on `!ok` — and let them
   type it. Then point the same close-discipline at `Counter`'s select on
   `done`.

## After passing

Preview: "Channels are not Go's only synchronization tool. Next lesson:
mutexes, RWMutex, and atomics — and the judgment call of when a channel is
the wrong choice."
