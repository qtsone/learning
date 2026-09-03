# Tutor notes — Consuming HTTP APIs

## Where the learner is

Deep into S4, after security fundamentals. They have the full Go toolkit this
exercise needs — structs, methods, interfaces, JSON, error wrapping, and
context from the concurrency arc — but have never assembled it into a
production-shaped network client. httptest appears here for the first time as
a *reading* skill (they write servers in S5); make sure the test files don't
feel like magic. The retry loop is the integration point where everything
meets — expect the most friction there.

## Common misconceptions

- **"err != nil covers HTTP errors."** `Do` returns an error only for
  transport failures; a 404 arrives with `err == nil`. If their `GetJSON`
  never constructs an `APIError`, this is the gap.
- **"http.DefaultClient is fine."** It has no timeout. Connect this to the
  cascade story: hung dependency → parked goroutines/sockets → their service
  hangs → its callers hang.
- **"Client.Timeout and context deadlines are the same thing."** One is a
  client-wide ceiling, the other a per-call budget the caller controls; you
  want both.
- **Forgetting `defer resp.Body.Close()`** or closing only on the happy
  path. The close-tracking transport test catches it; make them explain *why*
  it matters (connection pool starvation), not just add the defer.
- **Decoding before checking the status** — a 500's HTML body becomes a
  baffling JSON error that hides the real failure.
- **"Retry everything."** Push on 4xx (same request, same failure), POST
  (double charge), and cancelled contexts (the caller already gave up).
- **Jitter as decoration.** If they see jitter as optional polish, replay
  the thundering herd: 1000 clients failing together retry together, in
  synchronized waves, unless randomized.
- **Sleeping with `time.Sleep` instead of `c.sleep`** — the waits test fails
  with an empty slice; the lesson's point about injectable time is the fix.

## Grilling points

- "Your GetJSON gets back a 404. Walk me through exactly which branch runs
  and what the caller receives. Now make it a connection refused instead."
- "Why does the client-wide timeout exist if every request already carries a
  context deadline?" (Safety net for callers who forget; belt and braces.)
- "The server times out on a POST /payments. Your retry loop fires again.
  What just happened, and what do real payment APIs do about it?"
  (Idempotency keys — recognition, not implementation.)
- "Why full jitter — `[0, delay]` — rather than, say, delay ± 10%?" (Spread
  matters more than the average; ±10% still bunches the herd.)
- "GetJSON returns a decode error for a broken 200 body. Your ShouldRetry
  retries it. Defend or change that policy." (Defensible either way — the
  point is that they *have* a justification.)
- "Why does the exercise route waits through `c.sleep`? What would the test
  suite look like without it?" (Ties back to the TDD lesson's seam thinking.)

## Grading rubric

- **A** — All tests pass; GetJSON reads as the lifecycle (build, send, defer
  close, status check, limited error read, decode); retry loop is clean with
  no sleep-after-last-attempt off-by-one; can justify every line of the
  retry policy and explain the 429 exception unprompted.
- **B** — Tests pass but with rough edges: error wrapping absent or
  inconsistent, backoff loop convoluted, or the ShouldRetry justification
  wobbles on context cancellation. Solid on timeouts and body-closing.
- **C** — Tests pass after heavy hinting, or they cannot explain the
  transport-error/status-error split without prompting. Remediate before
  advancing: that split is the load-bearing idea.
- **Fail** — Tests failing, retry logic copied without being able to trace
  one iteration aloud, or they believe a 404 shows up as `err != nil`.

## Remediation ladder

1. "Read the failing test's message and its handler — what does the fake
   server do, and what did your client do in response?"
2. For GetJSON: "List the five lifecycle steps from the lesson. Which line
   of your function is each one? Which step is missing?"
3. For the retry loop: "Trace attempt by attempt for two 503s then a 200,
   MaxAttempts=5. Say aloud: check what, wait how long, call what, decide
   what. Now compare with your loop."
4. Sketch the loop's skeleton in words — check context, sleep-if-not-first,
   attempt, return on success or non-retryable, remember the last error —
   and let them turn each clause into code.

## After passing

Preview: "Your client now survives other people's outages. Next: CI/CD —
making a pipeline run your tests and linters so nothing broken ships, which
is where the discipline you've built this stage becomes automatic."
