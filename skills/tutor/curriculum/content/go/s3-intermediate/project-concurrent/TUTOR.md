# Tutor notes — Mini-Project: Concurrent Tool

## Where the learner is

End of the intermediate stage: they have done the full concurrency arc
(goroutines → channels → sync → context → patterns) plus interfaces, io, and
closures, but this is the first time everything must coexist in one program.
Expect the individual pieces to be solid and the *joints* to wobble: wiring
cancellation through a pool, deciding where results are aggregated, keeping
the concurrency out of the API. This is also their first serious contact
with net/http as a client — keep the HTTP surface shallow (the lesson gives
them exactly what they need) and steer energy toward the concurrency design.
Budget several sessions; it is a 6-10h project.

## Common misconceptions

- **"I'll sleep until the goroutines are done."** Any `time.Sleep` used as
  synchronization is a bug here, in their code or their reasoning. The pool
  ends with `wg.Wait`; the tests end with gates and watchdogs. If a sleep
  appears, revisit the sync lesson's happens-before framing.
- **One goroutine per URL.** It passes the status tests and fails
  `TestCheckBoundsConcurrency`. The bound is the point — connect it to the
  semaphore/worker-pool discussion from the patterns lesson.
- **`wg.Add(1)` inside the goroutine.** A race between `Add` and `Wait`;
  `Check` can return before workers start. `Add` before `go`, always.
- **Appending to a shared results slice.** `results = append(results, …)`
  from workers is a data race the race detector flags immediately. Contrast
  with `results[i] = …`: distinct elements, one writer each, `wg.Wait`
  publishes. If they reach for a mutex around append, ask what it costs them
  (order) before offering the index trick.
- **Believing `results[i] = r` from multiple goroutines must be a race.**
  The inverse confusion — some learners add a pointless mutex. Distinct
  indexes are distinct memory; the race detector agrees.
- **One `WithTimeout` around the whole run.** Passes nothing loudly but
  fails `TestCheckAppliesPerRequestTimeout` semantics eventually (the fast
  URLs after a slow one inherit a spent budget). The timeout must be derived
  per request, inside the worker.
- **Forgetting `defer cancel()`** on the per-request context — leaks a
  timer per URL; `go vet` catches it (lostcancel).
- **Returning early on cancellation** with a short slice, or `nil`. The
  contract is a full slice with `ctx.Err()` recorded for unstarted URLs —
  partial *results*, not partial *slices*.
- **Treating 404 as a Go error.** `Do` succeeded; the response is the
  answer. `Result` separates `Status` from `Err` deliberately — probe
  whether they can articulate why.
- **Global `flag.Parse` in `run`.** Panics/fails on the second test call.
  `flag.NewFlagSet` per invocation is the testability move.
- **Forgetting to drain/close the response body.** Works in tests, leaks
  connections in real runs. Low stakes here, but worth one review comment.

## Grilling points

- "Walk me through Ctrl-C, from the keypress to the summary line." (Signal →
  `NotifyContext` cancels ctx → in-flight `Do` aborts with wrapped
  `context.Canceled` → workers drain remaining jobs recording `ctx.Err()` →
  `wg.Wait` → full report.) They wrote only the middle; they must own the
  whole chain.
- "Why `results[i] = …` with no mutex? Convince me it's not a race." Then:
  "when would you switch to a results channel?" (Streaming results as they
  arrive, unknown result count, fan-in from heterogeneous sources.)
- "Why is the jobs channel buffered and closed before workers start? What
  would an unbuffered channel change?" (Needs a producer goroutine;
  otherwise `Check` deadlocks filling it. Good learners will have hit this.)
- "Your timeout is per request. Argue for a whole-run deadline instead —
  when would that be the right design?" (CI budget, SLA on the whole job —
  make them defend a position, both are legitimate designs.)
- "How did the tests pin down 'at most 3 in flight' without sleeping?" They
  should explain the gate handler and the in-flight counter. If they only
  shrug at the test file, the testing objective isn't met.
- "What does `-race` actually detect, and what does it prove when it stays
  silent?" (Unsynchronized access pairs on exercised code paths; silence is
  evidence, not proof.)
- "Why does `run` take `io.Reader`/`io.Writer` and args instead of using
  `os.Stdin` and the global flag package?"

## Grading rubric

- **A** — All tests pass under `-race`; pool matches the contract (bounded,
  order-preserving, full slice on cancel); per-request `WithTimeout` with
  `defer cancel()`; body drained and closed; `run` is a clean composition of
  the three pieces; learner defends channels-vs-index aggregation, pool
  sizing, and the cancellation path fluently and can explain how the gate
  tests work.
- **B** — Tests pass under `-race` but with rough edges: pointless mutex
  around indexed writes, missed body drain, clunky-but-correct cancellation
  handling, or design answers that are right but shallow (e.g. can't say
  when a results channel would win).
- **C** — Tests pass only after heavy hinting, or pass without `-race`
  having been run, or the learner cannot explain the shutdown chain or why
  the results writes are race-free. Time-boxed remediation on the weak
  objective before advancing.
- **Fail** — Race reports, sleeps as synchronization, unbounded fan-out, or
  a solution they cannot walk through line by line. Remediate via the
  relevant prior lesson (sync for races, patterns for the pool), then retry.

## Remediation ladder

1. "Run `go test -race -run TestCheckBoundsConcurrency ./...` and read the
   failure top to bottom. What never happened, according to the message?"
2. "Forget URLs — you have 8 envelopes and 3 clerks. Describe the desk
   setup: where do envelopes wait, how does a clerk pick the next one, how
   do you know everyone's finished?" (Rebuild the pool picture from the
   patterns lesson.)
3. "Where is the one place a Result is written? Who writes index 4, and
   what guarantees the write is visible when `Check` returns?" (Points at
   jobs-carry-indexes and `wg.Wait`.)
4. "Inside the worker, before the request: what should `ctx.Err()` tell
   you, and what do you record if it's non-nil? After that, wrap the request
   in `context.WithTimeout(ctx, c.Timeout)` with a `defer cancel()` — now
   re-run the timeout and cancel tests." Stop short of dictating `checkOne`;
   let them assemble it.

## After passing

Stage complete — next is Engineering Practice: turning working code into
professional code (clean code, TDD, debugging, SQL, security, and HTTP
clients done properly — the depth this project's four `net/http` calls
deliberately skipped).
