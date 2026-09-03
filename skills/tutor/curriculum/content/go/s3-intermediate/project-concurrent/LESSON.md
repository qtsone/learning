# Mini-Project: Concurrent Tool

> `go.intermediate.project-concurrent` · ~6-10h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement a complete concurrent CLI tool (a parallel link checker) combining
  goroutines, channels, and context.
- Implement bounded concurrency with per-request timeouts so the tool neither
  overwhelms targets nor hangs on slow ones.
- Implement clean shutdown on interrupt: cancel in-flight work and report
  partial results.
- Implement tests for concurrent code, including passing the race detector,
  and explain how you made the code testable.
- Justify your design choices (channels vs locks, pool size, error
  aggregation) in a code-review discussion.

## The brief

You are building `linkcheck`, a command-line tool that answers one question
fast: *which of these links are broken?* It reads URLs from standard input
(one per line; blank lines and `#` comments ignored), fetches them in
parallel, and prints a report:

```
$ linkcheck -c 8 -t 5s < urls.txt
ok 200 https://example.com/
fail https://example.com/old-page: status 404
fail https://example.com/slow: Get "…": context deadline exceeded
checked 3: 1 ok, 2 failed
```

The requirements that make it a *capstone* rather than a loop with `go` in
front of it:

- **Bounded concurrency.** `-c 8` means at most 8 requests in flight at once.
  Unbounded fan-out is how well-meaning tools become accidental
  denial-of-service attacks — and how they exhaust their own file descriptors.
- **Per-request timeouts.** `-t 5s` bounds each request separately. One dead
  server must not stall the whole run, and must not eat into other URLs'
  budgets.
- **Clean shutdown.** Ctrl-C cancels in-flight requests, skips unstarted
  ones, and still prints a full report — every URL accounted for, the
  unfinished ones marked canceled. Partial results beat discarded work.
- **Deterministic order.** Results print in input order regardless of which
  request finished first, so runs are comparable and diffable.

Everything you need was built in this stage: goroutines and `WaitGroup`,
channels and `select`, `context` cancellation and deadlines, the worker pool
from the patterns lesson, `io.Reader` composition, and closures wherever a
goroutine captures its job. This project is where they stop being lesson
topics and become one program.

## Just enough net/http

You'll study HTTP properly in the next stage; here you need only the client
surface, which is four calls:

```go
req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
if err != nil {
	return Result{URL: url, Err: err} // malformed URL — never sent
}
resp, err := client.Do(req)           // client is an *http.Client
if err != nil {
	return Result{URL: url, Err: err} // timeout, refused, DNS, canceled…
}
defer resp.Body.Close()
_, _ = io.Copy(io.Discard, resp.Body) // drain so the connection is reused
status := resp.StatusCode             // 200, 404, 500, …
```

Two contracts to respect:

- The request carries a `context.Context`. Cancel it (or let its deadline
  pass) and `Do` aborts mid-flight and returns an error — that error wraps
  `context.Canceled` or `context.DeadlineExceeded`, so `errors.Is` works on
  it. This is how everything you learned about context plugs into real I/O.
- Always close the body, and drain it first. An undrained connection can't go
  back into the client's connection pool, so each request pays for a fresh
  one. `io.Copy(io.Discard, resp.Body)` is the idiom.

Note that an HTTP *error status* like 404 is not a Go `error`: `Do` succeeded
— it delivered a response; the response just says "not found". Your `Result`
type keeps the two failure kinds separate, and `Summarize` folds them back
together for the verdict.

In tests, `httptest.NewServer` gives you a real HTTP server on a `127.0.0.1`
port, serving any handler you write, torn down with `Close`. Your tests spin
up local servers that return 200s, 404s, or block forever — **tests never
touch the real network**, so they pass on an airplane and in CI.

## Architecture: concurrency in one place

The tool is four functions, and only one of them is concurrent:

```
stdin ──▶ ParseURLs ──▶ Checker.Check ──▶ Summarize ──▶ report on stdout
          (pure)        (all the         (pure)
                         concurrency)
```

- `ParseURLs(r io.Reader) ([]string, error)` — reads the URL list. Pure
  transformation, trivially testable, no concurrency.
- `(*Checker).Check(ctx, urls) []Result` — the concurrent core. Everything
  hard lives here, behind a boringly synchronous signature: slice in, slice
  out. Callers cannot tell it uses goroutines at all.
- `Summarize(results) Summary` — counts ok vs failed. Pure again.
- `run(ctx, args, in, out) error` — wires them together: parses flags, calls
  the other three, prints the report. It takes `io.Reader`/`io.Writer`
  instead of touching `os.Stdin`/`os.Stdout`, and a `flag.NewFlagSet`
  instead of the global `flag` package — both so tests can call it
  repeatedly with fake inputs. This is the same "thin `main`, testable
  `run`" shape as your S1 CLI capstone, now with a context threaded through.

`main` (provided, read it) is five lines: build a context with
`signal.NotifyContext(context.Background(), os.Interrupt)`, call `run` with
the real stdin/stdout/args, exit 1 on error. `signal.NotifyContext` is new
here: it returns a context canceled on the first Ctrl-C — the same
close-as-broadcast mechanism you know from the context lesson, wired to a
signal instead of a `cancel` call. That single cancellation propagates to
every request in flight — clean shutdown falls out of context propagation,
with no signal handling anywhere else.

Keeping the concurrency inside one function with a synchronous signature is
the design move to internalize. It bounds the blast radius: no goroutine,
channel, or lock leaks into the API, so callers (and tests) reason about a
plain function call.

## The worker pool, made concrete

`Check` is the worker pool from the patterns lesson, specialized:

```go
results := make([]Result, len(urls))
jobs := make(chan int, len(urls)) // indexes into urls
for i := range urls {
	jobs <- i
}
close(jobs)

var wg sync.WaitGroup
for range concurrency {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range jobs {
			// TODO: honor cancellation, then check urls[i]
			results[i] = ...
		}
	}()
}
wg.Wait()
return results
```

Design decisions worth being able to defend — a reviewer will ask:

- **Why a buffered jobs channel, filled and closed up front?** All jobs are
  known before any worker starts, so there is nothing to gain from a
  producer goroutine. `close` is the workers' shutdown signal: `range jobs`
  ends, `wg.Done` runs, `Wait` returns. No sentinel values, no quit channel.
- **Why `results[i] = …` instead of a results channel?** Each index is
  written by exactly one goroutine, and `wg.Wait()` orders every write
  before the return — so there is no data race and no mutex, *and* input
  order is preserved for free. A results channel is the right call when
  results must stream out as they arrive; here the caller wants the complete
  slice, so collecting-then-sorting through a channel would be extra moving
  parts for the same outcome. Channels vs locks vs neither is a per-problem
  decision, not a doctrine.
- **Why does cancellation still produce a full slice?** Workers keep
  draining `jobs` after cancellation but stop doing network work: for each
  remaining index they record `ctx.Err()` instead of making a request. Every
  URL gets a `Result`, so the report can show exactly what was and wasn't
  checked. The alternative — return early and drop the tail — throws away
  the user's mental map of what happened.

## Two contexts per request

There are two lifetimes in play and each gets its own context:

- **The run**: the context `main` passes in, canceled by Ctrl-C. It has no
  deadline — a long URL list may legitimately take minutes.
- **One request**: derived per URL with
  `context.WithTimeout(ctx, c.Timeout)`. Child contexts inherit
  cancellation, so Ctrl-C still aborts the request early; the deadline only
  adds a second way to die.

```go
ctx, cancel := context.WithTimeout(ctx, c.Timeout)
defer cancel() // always — releases the timer even on success
```

Per-request derivation is the point. A single `WithTimeout` around the whole
run would make the budget shared: with 200 URLs and one slow straggler
early in the list, everything after it inherits a nearly-spent deadline.
Deriving inside the worker gives every URL the same fresh budget. (Why not
`http.Client.Timeout`? That would also work for the timeout alone, but the
context version composes with cancellation, keeps the `Client` shareable
with different budgets, and is the shape you'll use for any I/O, not just
HTTP.)

## Testing code you can't schedule

You cannot tell the scheduler "now run worker 2". The provided tests get
determinism anyway, with two techniques worth stealing for the rest of your
career:

- **Gate handlers.** A test server whose handler signals arrival on one
  channel and blocks on another until the test releases it. That lets a test
  *hold* the pool in a known state: "exactly 3 requests in flight, nothing
  released" is a fact established by channel operations, not by sleeping and
  hoping. The bounded-concurrency test counts in-flight handlers under a
  mutex and fails if the count ever exceeds `Concurrency`; the cancellation
  test holds both workers at the gate, cancels, and then asserts the server
  never saw a third request. No `time.Sleep` anywhere — a sleep-based test
  is wrong twice, flaky on slow machines and slow on fast ones.
- **Watchdogs, not sleeps.** Where a buggy implementation would block
  forever (a deadlocked pool, a `Check` that never returns), the tests wrap
  the wait in a `select` with a generous `time.After` and a message telling
  you what never happened. A watchdog never makes a correct implementation
  flaky — it only converts "hangs for 10 minutes" into a readable failure.

Read `checker_test.go` before coding — it is the precise spec, and reading
gate tests is half this lesson's value. Then run everything under the race
detector, every time:

```sh
go test -race ./...
```

`-race` instruments memory accesses and reports unsynchronized read/write
pairs even when the test still passes. For this project treat a race report
as a failing test: a "passing" concurrent program with a data race is a
program you haven't caught lying yet.

## Exercise

Open [`exercise/`](exercise/) — a module with:

- `main.go` — provided and complete: the signal wiring. Read it.
- `checker.go` — `Result`, `Checker`, and the concurrent core. Your main
  work site.
- `report.go` — `ParseURLs`, `Summarize`, `run`. Your other work site.
- `checker_test.go`, `report_test.go` — the spec. Read them first.

Acceptance criteria:

1. `ParseURLs` returns one URL per line, whitespace-trimmed, skipping blank
   lines and lines starting with `#`; it propagates the reader's error.
2. `Check` returns exactly one `Result` per URL, **in input order**, with
   `Result.URL` set. Completed requests record `Status`; failures (bad URL,
   connection refused, timeout, canceled) record `Err`.
3. At most `Concurrency` requests are in flight at any moment, and with more
   jobs than workers the pool genuinely runs `Concurrency` requests
   simultaneously. `Concurrency <= 0` behaves as 1. A nil `Client` means
   `http.DefaultClient`.
4. Each request gets its own `context.WithTimeout` of `Timeout` (when
   `Timeout > 0`) derived from `Check`'s context. A server that never
   responds yields a `Result.Err` matching
   `errors.Is(err, context.DeadlineExceeded)` while other URLs in the same
   run still succeed.
5. Canceling `Check`'s context aborts in-flight requests and starts no new
   ones; `Check` still returns promptly with a full slice in which
   affected URLs have `Err` matching `errors.Is(err, context.Canceled)`.
6. `Summarize` counts `Checked` (all), `OK` (`Err == nil` and
   `Status < 400`), and `Failed` (everything else).
7. `run` parses `-c` (default 4) and `-t` (default 10s) from `args` using a
   `flag.NewFlagSet`, reads URLs from `in`, and prints to `out`: one line
   per URL in input order — `ok <status> <url>` on success,
   `fail <url>: …` otherwise — then exactly
   `checked <n>: <ok> ok, <failed> failed`. It returns a non-nil error if
   parsing/reading fails or any link failed.
8. `go test -race ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -race ./...
```

They fail on the starter — build it up piece by piece (`ParseURLs` and
`Summarize` are gentle warm-ups; then the pool, then timeouts, then
cancellation, then `run`). When everything is green, play with the real
thing: `go run . -c 8 < urls.txt` with a file of URLs you care about, and
press Ctrl-C mid-run to watch your clean shutdown report partial results.

## Further reading

- [Go Blog — Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines)
- [Go Blog — Contexts and structs](https://go.dev/blog/context-and-structs)
- [pkg.go.dev — net/http/httptest](https://pkg.go.dev/net/http/httptest)
- [Go Blog — Data Race Detector](https://go.dev/doc/articles/race_detector)
