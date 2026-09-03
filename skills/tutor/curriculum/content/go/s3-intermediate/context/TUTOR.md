# Tutor notes — Context

## Where the learner is

Twelfth lesson of S3, three lessons into the concurrency arc. They can start
goroutines, coordinate with WaitGroups and channels (including `select`,
closing, and close-as-broadcast), and choose between channels and
Mutex/atomic/Once. They wrap errors with `%w` (S1) and compare with
`errors.Is` (type-assertions lesson). They have **not** seen worker pools,
errgroup, semaphores, or graceful shutdown as named patterns — that is the
next lesson, so keep the exercise's single worker a single worker. All test
runs in this lesson and beyond should be `go test -race ./...`; if they run
without `-race`, ask them to make it a habit before grading.

## Common misconceptions

- **"cancel() kills the goroutine"** — nothing in Go kills goroutines.
  Cancellation closes a channel; code that never selects on `ctx.Done()`
  runs to completion (or blocks forever). If this is shaky, revisit
  close-as-broadcast from the Channels lesson.
- **cancel treated as error-path-only** — they call `cancel()` only when
  something fails, or not at all. It must be `defer cancel()` on every
  path: deadline contexts own a timer, derived contexts stay linked into
  the parent's tree, and blocked goroutines wait on that `Done` forever.
  `go vet ./...` (`lostcancel`) catches the escaped-uncalled cases.
- **Guarding only the receive side** — the classic. Their worker selects
  `Done` vs `jobs` but sends `results <- r` bare. The send-side test exists
  precisely for this; the watchdog message names the fix. Ask: "which
  operations in your worker can block? Cancellation must cover each one."
- **`==` against ctx.Err()** — works on bare sentinels, breaks the moment a
  layer wraps with `%w`. The wrapped test cases force `errors.Is`. Also
  worth surfacing: `Canceled` and `DeadlineExceeded` are sentinel *values*,
  not types — this is the errors.Is half of the Is/As pair.
- **Returning invented errors on cancellation** — `return errors.New("canceled")`
  destroys the caller's ability to branch on *why*. The convention is
  `return ctx.Err()`, wrapped at most.
- **Storing ctx in a struct** — usually surfaces as "why can't I just put
  it on my client type?" Lifetime mismatch: a context lives one call tree,
  the object serves many. Point at the contexts-and-structs blog post in
  Further reading.
- **Values as a parameter bus** — smuggling a config or DB handle through
  `context.WithValue`. The test: can the function still do its job without
  the value? Metadata yes, dependency no.
- **String keys for values** — collides across packages and staticcheck
  flags it (SA1029). The collision test fails any implementation another
  package could reach; unexported key type is the fix.
- **`WithTimeout` re-derived at every hop** — each hop granting itself "3
  more seconds" quietly extends the total budget; an absolute deadline
  propagates intact. This is the Timeout-vs-Deadline choice in one line.

## Grilling points

- "Walk me through what happens, channel by channel, when `cancel()` is
  called while your `Square` worker is blocked sending a result."
- "Why does `Square` return `ctx.Err()` instead of `nil` or its own error
  on cancellation? Who upstream depends on that choice?"
- "A request must finish by 02:00 and passes through three services. Timeout
  or Deadline, and what goes wrong with the other choice?"
- "Why `defer cancel()` even when the call succeeded? Name the two things
  that stay alive if you skip it."
- "Your `Retryable` returns true for `DeadlineExceeded`. Give a concrete
  scenario where retrying a `Canceled` error does damage." (Duplicate side
  effects; the caller already gave up.)
- "Why is the values key an unexported struct type instead of the string
  `\"request-id\"`? What exactly does the type system prevent?"
- "`ctx.Done()` on `context.Background()` returns nil. Why does your
  `Await` still work when given Background?" (A nil channel in a select
  blocks forever — that case simply never fires.)

## Grading rubric

- **A** — All tests green under `-race`; worker guards both receive and
  send with `ctx.Done()` and returns `ctx.Err()` verbatim; `Retryable`
  uses `errors.Is` (the one-liner or an explicit two-sentinel switch both
  fine — but they must be able to say why `Canceled` maps to false);
  values use an unexported key type with the `, ok` assertion; gofmt-clean;
  can explain defer-cancel and timeout-vs-deadline unprompted.
- **B** — Tests pass but with rough edges: `Retryable` as an `if/else`
  chain with a redundant nil check, worker structured awkwardly (duplicated
  return paths), or the defer-cancel rationale needs one prompt. Concepts
  solid.
- **C** — Tests pass only after heavy hinting, or the code fights the
  point: send-side guard added by trial and error without being able to
  say what blocked, `errors.Is` used but `==` defended as equivalent,
  values key changed from string only because the test failed. Works, but
  the model is missing — remediate before advancing.
- **Fail** — Tests failing or hanging, or they cannot explain what closing
  `Done` does to a selecting goroutine. Reteach from the Channels lesson's
  close-as-broadcast; the patterns lesson ahead assumes all of this.

## Remediation ladder

1. "Run `go test -race ./...` and read the first failure aloud — which
   function, and did it return the wrong value or time out?"
2. For worker hangs: "List every line in `Square` that can block. For each,
   what unblocks it if no job or reader ever comes?"
3. "Your worker needs to race two events in two different places: 'work
   arrived' vs 'told to stop', and 'result accepted' vs 'told to stop'.
   Which Go statement races channel operations?" (They know `select` — let
   them place the second one themselves.)
4. Talk the worker through shape by shape — outer select with `Done` and
   comma-ok receive, `return nil` on closed jobs, inner select around the
   send, `return ctx.Err()` in both `Done` arms — and let them type every
   line. Same ladder for `Retryable`: "which lesson gave you a way to match
   sentinel errors through wrapping?"

## After passing

Preview: "Next is Concurrency Patterns — worker pools, errgroup, and
graceful shutdown. Everything composes from today's pieces: every pattern
takes a ctx, and every goroutine honors it exactly the way your `Square`
does."
