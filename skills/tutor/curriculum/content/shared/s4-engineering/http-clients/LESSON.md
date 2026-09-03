# Consuming HTTP APIs

> `shared.eng.http-clients` · ~3-4h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Explain HTTP method and status-code semantics and choose the correct
  handling for 2xx, 4xx, and 5xx responses.
- Implement an API client that sets timeouts, closes response bodies, and
  decodes JSON responses safely.
- Explain why a client without timeouts is a production incident waiting to
  happen.
- Implement retries with exponential backoff and jitter, retrying only
  idempotent or safe-to-repeat requests.
- Choose which failures to retry versus surface immediately, and justify the
  policy.

## Talking to someone else's computer

Almost every service you will ever write is also a *client*: it calls a
payment API, a weather API, an internal user service. Calling over the
network is nothing like calling a function. The other machine may be slow,
overloaded, rebooting, or gone; the network between you may drop, duplicate,
or delay packets (remember S0's how-the-internet-works). A function call has
two outcomes — return or panic. A network call has a third, nastier one:
**you don't know what happened**. The request may have been processed even
though you never saw the response.

Everything in this lesson — status handling, timeouts, retries, idempotency —
is a consequence of that third outcome.

## Methods and status codes are a contract

HTTP methods carry meaning beyond "send these bytes":

| Method | Meaning | Safe? | Idempotent? |
|--------|-------------------------|-------|-------------|
| GET | read a resource | yes | yes |
| HEAD | read headers only | yes | yes |
| PUT | replace a resource | no | yes |
| DELETE | remove a resource | no | yes |
| POST | create / do something | no | **no** |
| PATCH | partial update | no | **no** (in general) |

**Safe** means the request doesn't change server state — you can fire it
freely. **Idempotent** means doing it twice has the same effect as doing it
once: deleting an already-deleted record is still deleted; replacing a
document with the same content is the same document. POST is neither: two
POSTs to `/payments` can charge a card twice. Keep that table in your head —
it decides, later in this lesson, what you are allowed to retry.

Status codes come in classes, and each class demands a *different reaction
from you, the client*:

- **2xx — success.** Proceed: read and decode the body.
- **3xx — redirect.** Most client libraries follow these for you.
- **4xx — your fault.** The request itself is wrong: bad path (404), bad
  payload (400), missing credentials (401/403). Retrying the identical
  request will produce the identical failure. Fix the request; don't retry.
  The one exception is **429 Too Many Requests** — "slow down" — which is
  explicitly an invitation to retry later.
- **5xx — their fault.** The server broke (500) or a proxy couldn't reach it
  (502/503/504). Nothing was wrong with your request, so trying again later
  can genuinely succeed. These are the *transient* failures retries exist for.

A status code is data, not an exception: your client should turn "the server
said 404" into a value the caller can inspect, distinct from "I couldn't
reach the server at all."

## The lifecycle of a request

Every HTTP call, in any language, walks the same path:

1. **Build** the request: method, URL, headers, optional body.
2. **Send** it and wait for a response.
3. **Check the status** before touching the body.
4. **Read** the body (with a size limit if you don't trust the sender).
5. **Release** the connection — in most libraries, by closing the body.

Step 5 is where clients silently rot. HTTP clients keep a **connection pool**:
finished connections are reused for the next request instead of paying TCP
and TLS setup again. A response body you never close is a connection the pool
never gets back. Each leaked body pins a socket and its buffers; under load
you run out of file descriptors and every request starts dialing from
scratch — or failing.

In Go:

```go
resp, err := client.Do(req)
if err != nil {
	return err // transport failure: DNS, refused, timeout — there is no response
}
defer resp.Body.Close() // ALWAYS, on every path, before touching the status
```

Note the split: `err != nil` means *no HTTP conversation happened*. A 404 is
a successful conversation with a disappointing answer — `err` is nil and
`resp.StatusCode` is 404. Confusing these two is the classic first bug in
every Go HTTP client.

## Timeouts: the incident you schedule in advance

The default in most HTTP libraries — including Go's — is **no timeout**.
Think about what that means: your program sends a request and is prepared to
wait *forever*. Now put that client inside a web service handling 500
requests per second, and let the API it calls stop answering (not refuse —
just hang, as overloaded services do). Every in-flight call parks a goroutine,
a socket, and memory, and none of them ever come back. Requests pile up
behind them, your service stops answering *its* callers, and their
no-timeout clients pile up too. One hung dependency, and the outage
propagates upstream through every client that was "prepared to wait". That
is why "client without timeouts" is not a code smell — it's a production
incident with a delayed fuse.

Timeouts come in layers, and you usually want two:

- A **client-wide ceiling**: no single request may ever take longer than
  this, period. A blunt safety net.
- A **per-request deadline** carried by the call's context, so a caller with
  200ms of budget can pass that budget down.

In Go:

```go
client := &http.Client{Timeout: 10 * time.Second} // ceiling for everything

ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
defer cancel()
req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
```

You met `context` in the concurrency arc — this is its natural habitat.
`http.NewRequestWithContext` ties the request to the context: when the
deadline passes or the caller cancels, the transport abandons the request and
`Do` returns an error wrapping `context.DeadlineExceeded` or
`context.Canceled`. And never reach for `http.DefaultClient` in production
code: it is the shared, zero-timeout client.

## Decoding responses safely

Order matters: **check the status before decoding**. A 500 response's body is
an HTML error page or a stack trace, not the JSON you expect — decoding it
first yields a confusing parse error that hides the real story. And when you
do read an error body (they often contain useful messages), read it through a
size limit: you control your request, but the *response* is someone else's
output, and "someone else" occasionally sends you 2GB of stack trace.

Turn non-2xx responses into a **typed error** carrying the status code, so
callers can react to *what the server said* rather than string-matching an
error message. You built this muscle in S1's error-handling lesson; here it
earns its keep.

In Go:

```go
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error: status %d", e.StatusCode)
}
```

Callers use `errors.As(err, &apiErr)` to fish it out of a wrapped chain, then
branch on `apiErr.StatusCode`. For the happy path, decode straight from the
body stream (`json.NewDecoder(resp.Body).Decode(&v)` in Go) into the struct
you defined — the JSON techniques from S3 apply unchanged.

## Retries: backoff, jitter, and knowing when to stop

Transient failures — a 503 during a deploy, a dropped connection — heal
themselves. Retrying converts a blip into a non-event. But naive retries are
how clients finish off a struggling server:

- **Retry immediately** and you hammer a service that is already drowning.
- **Retry on a fixed schedule** and every client that failed together retries
  together — synchronized waves, each one a fresh spike. This is the
  **thundering herd**.

The standard remedy has two parts:

1. **Exponential backoff**: wait longer after each failure — base delay,
   then 2×, 4×, 8×, capped at some maximum. You give the server room to
   recover, and you give up escalating at the cap.
2. **Jitter**: randomize each wait (a uniform pick from `[0, delay]` — "full
   jitter" — works well). Randomness desynchronizes the herd, spreading the
   retry load flat instead of in waves.

Just as important is the **decision of what to retry**:

- Transport failures (couldn't connect, connection reset) — retry: the
  request may never have arrived.
- 5xx and 429 — retry: the server told you it's (temporarily) unable.
- Other 4xx — never: the request is wrong and will be wrong next time too.
- Cancelled or expired context — never: the *caller* gave up; retrying past
  your caller's deadline is work nobody will collect.

And the method matters more than the status: retrying is only safe when
repeating the request is safe — the GET/PUT/DELETE column of the table
above. A timed-out POST is the "you don't know what happened" case in its
sharpest form: the charge may have gone through. Blindly retrying
non-idempotent requests is how customers get billed twice. (Real payment
APIs solve this with idempotency keys — a client-chosen request ID the
server deduplicates on — worth recognizing when you see it.) Cap the total
attempts: a request that failed five times is telling you something no sixth
attempt will fix; surface the error and let it reach your logs and alerts.

## Testing a client without a network

Your tests must not depend on someone else's uptime, so they never touch the
real network. The trick is to run a tiny real HTTP server *inside the test*,
on a loopback port, whose handler you script: "answer 503 twice, then
succeed." Your client code is exercised end to end — real sockets, real
headers — against a server you fully control.

In Go this is `net/http/httptest`:

```go
ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, `{"name":"go"}`)
}))
defer ts.Close()
// ts.URL is the server's address — point your client at it
```

You'll write handlers properly in the next stage; here you only need to read
them. The exercise tests also inject a fake *sleep* function so retry waits
are recorded instead of actually slept — the same "make time injectable"
move you saw in the TDD lesson, because a test suite that sleeps its way
through backoff schedules is a test suite nobody runs.

## Exercise

Open [`exercise/`](exercise/) — a module with two work files
(`client.go`, `retry.go`) and their tests. You'll build `Client`, a small,
production-shaped JSON API client. Read the tests first; they are the
specification.

Acceptance criteria:

1. `New(baseURL)` returns a `Client` whose underlying `http.Client` has a
   10-second timeout.
2. `GetJSON(ctx, path, v)` issues a GET for `BaseURL+path` built with
   `http.NewRequestWithContext`, sends `Accept: application/json`, and goes
   through `c.HTTP`.
3. On 2xx, `GetJSON` decodes the JSON body into `v`. On any other status it
   returns an `*APIError` (findable with `errors.As`) carrying the status
   code and at most `maxErrorBody` bytes of the body; `Error()` mentions the
   status code.
4. The response body is closed on every path — success and error.
5. `Backoff(p, attempt)` doubles `BaseDelay` per attempt, capped at
   `MaxDelay`. `Jitter(d)` returns a uniformly random duration in `[0, d]`.
6. `ShouldRetry` says yes to transport errors, 429, and 5xx `APIError`s; no
   to other 4xx, context cancellation/expiry, and nil.
7. `GetJSONRetry(ctx, path, v, p)` makes up to `p.MaxAttempts` calls,
   waiting `Jitter(Backoff(...))` **via `c.sleep`** between attempts (no
   wait after the last), stops early on success or a non-retryable error,
   and makes no request once `ctx` is cancelled.
8. `go test ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green, one function at a time
(`Backoff` and `Jitter` are a gentle warm-up; `GetJSON` is the heart).

## Further reading

- [MDN — HTTP response status codes](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status)
- [AWS Builders' Library — Timeouts, retries, and backoff with jitter](https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/)
- [pkg.go.dev — net/http Client](https://pkg.go.dev/net/http#Client)
- [pkg.go.dev — net/http/httptest](https://pkg.go.dev/net/http/httptest)
