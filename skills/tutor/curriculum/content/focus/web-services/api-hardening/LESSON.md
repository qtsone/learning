# API Hardening

> `focus.web.api-hardening` · ~3-4h · Stage: Focus: Web Services

## Objectives

By the end of this lesson you can:

- Implement per-client rate limiting in Go middleware with a token bucket and
  return correct 429 responses.
- Explain what CORS actually protects, how preflight requests work, and
  implement a correct (non-wildcard) CORS policy.
- Set security response headers and explain the attack each one blocks.
- Implement deep request validation: body size limits, strict JSON decoding
  with `DisallowUnknownFields`, and field-level constraint checks.
- Explain why input validation at the boundary does not replace output
  encoding and parameterized queries downstream.

## The edge is where you meet strangers

Your service so far assumes a client that behaves. The internet contains no
such client: it contains crawlers, a misconfigured retry loop in someone's
mobile app, a scanner walking your paths, and occasionally somebody
deliberately hunting for the request that costs you the most to answer.

The authentication and authorization lessons asked *who is this* and *may they
do this*. This one
assumes both are answered and asks a third: **what does answering this request
cost me, and what is the worst a caller can make that number?** Every technique
here is a bound — on bytes, rate, time, and what a browser may do with your
answer. None makes a service secure alone; together they mean a bad client
degrades your service instead of ending it.

## Bound the bytes

Here is the whole vulnerability in one line — `body, err := io.ReadAll(r.Body)`.
How big is `body`? You do not know. The client decides. `Content-Length` is a
claim, not a limit, and chunked encoding omits it entirely. A handful of
concurrent requests streaming a body forever takes the process down, and it
costs the attacker almost nothing: they are sending, you are allocating.
`json.NewDecoder(r.Body).Decode(&v)` has the same hole — it decodes until the
value ends or the reader does.

The fix is one wrapper:

```go
r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
```

Past the limit it returns an `*http.MaxBytesError` and tells the server to stop
keeping a connection alive for a body it will never read. Why not
`io.LimitReader`? Because `LimitReader` reports a clean `io.EOF` at the
boundary: a truncated 10 MB body arrives as "malformed JSON", you return 400,
and the caller spends an afternoon debugging JSON that is perfectly valid. A
distinguishable error is what lets you answer **413 Content Too Large**.

Headers get the same treatment through `Server.MaxHeaderBytes` (1 MB by
default — a default, not a decision).

## Malformed, or merely wrong?

Two different failures hide behind "bad request":

- **Malformed** — I could not parse this: a brace is missing, a string sits
  where a number belongs, there is a field I do not know. → **400 Bad Request**.
- **Semantically invalid** — I parsed it perfectly and it does not mean
  anything: priority 9 on a 1-5 scale, an empty title. → **422 Unprocessable
  Content**.

Not pedantry: a 400 tells a client developer "your serializer is wrong", a 422
tells them "your user typed something wrong", and those go to different people.

Strict decoding is the first half:

```go
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
```

Remember S3's JSON lesson: `encoding/json` ignores unknown fields by default,
and for a *client* reading someone else's API that forgiveness is right — new
fields upstream must not break you. On the *server* side it is backwards. A
client that sends `{"titel": "..."}` gets a 201 and an empty title and finds
out in production; a client that sends `{"role":"admin"}` to an endpoint that
ignores `role` cannot tell whether it was ignored or honoured. Reject what you
did not agree to parse.

One more check most people skip:

```go
if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
	return badRequest("request body must contain a single JSON object")
}
```

`Decode` stops at the end of the first value. Everything after it — a second
object, trailing garbage — is silently discarded unless you look.

Mapping decoder errors is the fiddly part, and `errors.As` does most of it:
`*json.SyntaxError` carries the byte offset, `*json.UnmarshalTypeError` carries
the *field name* — exactly what a client needs. The one ugly case is unknown
fields, where `encoding/json` returns a plain error reading
``json: unknown field "colour"`` and no typed error exists. String matching is
genuinely the only option, so confine it to one function with the prefix in a
named constant and never let it spread.

## Field-level errors

`{"error":"invalid input"}` makes a user fix one thing per round trip. Report
every failure at once, named:

```json
{"error":{"message":"validation failed","fields":[
  {"field":"title","message":"must not be empty"},
  {"field":"priority","message":"must be between 1 and 5"}
]}}
```

Those field names are API contract — a client renders them next to form inputs
— so renaming one is a breaking change, and a stable order keeps both your
tests and your UI still. The counterweight: error messages are output too.
Never echo the decoder's raw text (it can quote the body back), and remember
that `"user 4471 not found"` confirms which ids exist.

## Validation is not the last defense

This is the point people get wrong most often, so state it plainly:
**validating input at the boundary does not make the data safe downstream.**

Validation answers "is this input acceptable *to my domain*". Injection is a
different question: "is this value being pasted into a language that will
interpret it?" Each such language has its own escaping rule, and the boundary
cannot know them — SQL needs parameterized queries (S4's SQL lesson), HTML
needs contextual output encoding at render time (`html/template` escapes
differently inside an attribute, inside a `<script>`, inside a URL), and a
shell, a log line, an LDAP filter and a header each need their own.

Input validation is a *bound* (title is at most 80 characters); output encoding
is a *correctness requirement* at each sink. Blocklisting "dangerous
characters" at the door tries to do the sink's job from a place that cannot see
the sink, and fails in both directions: it breaks O'Brien and it misses the
next encoding. Accept the character. Escape it where it is used.

## Rate limiting: the token bucket

A rate limiter answers "how often may this caller ask". The standard shape is a
**token bucket**: each client has a bucket holding at most `burst` tokens, it
refills at `rate` tokens per second, and each request spends one. Empty bucket,
no request. It absorbs a short spike (a page loading eight resources) while
holding the long-run average at `rate` — which a naive "N per fixed minute"
counter does not, since a client can spend 2N across a window boundary.

You do not need a goroutine per client dripping tokens in. Store the balance
and the time it was taken, and compute the refill on read:

```go
elapsed := now.Sub(b.updated)
b.tokens = math.Min(burst, b.tokens+elapsed.Seconds()*rate)
b.updated = now
```

Two things follow. `now` must be **injected** — the `Clock` interface from S5's
testing lesson — or your tests are sleeps, and sleeps under `-race` are flaky
by construction. And the map of buckets is shared mutable state touched from
every request goroutine, so it is guarded by a mutex.

Three honest caveats:

1. **The map is itself a target.** One entry per distinct key means anyone with
   a /64 of IPv6 can make you allocate until you die. Evict idle buckets — but
   only after `burst/rate`, the time a drained bucket takes to refill, or
   eviction hands its owner a full bucket and undoes the limit.
2. **What is a "client"?** The peer address is the weakest useful key: NAT puts
   thousands of people behind one, and `X-Forwarded-For` is client-controlled
   text anyone can set — read it only when you know how many proxies you own.
   An authenticated account id is far better when you have one.
3. **Per-instance is not global.** This limiter lives in one process; four
   replicas make your "100/s" 400/s. A truly global limit needs shared counters
   and a network hop on the hot path, so the usual answer is a per-instance
   limit sized so `replicas × limit` is survivable, plus a gateway limit if you
   have one.

The refusal is `429 Too Many Requests` with `Retry-After`, in whole seconds.
Round *up*: rounding down invites the client straight back into another
refusal.

`golang.org/x/time/rate` is the production-grade version of this algorithm, and
its `AllowN(now time.Time, n int)` takes the time explicitly for the same
testability reason. You hand-roll it here because the ten lines of arithmetic
are the lesson; reach for the library afterwards.

## CORS, correctly

Start from what the browser is doing. The **same-origin policy** is a browser
rule: script running on `https://app.example.com` may *send* a request to
`https://api.other.com` but may not *read* the response. That protects users,
because the browser attaches their cookies to that request automatically. CORS
is how your API says "actually, this origin may read my answers."

Three consequences that surprise people:

- **CORS is enforced in the browser, not in your server.** `curl` ignores it,
  and so does every non-browser client. It is not access control: a request
  from an unlisted origin still reaches your handler and still runs. What stops
  it is authentication and authorization, earlier in this pack.
- **CORS makes your API *more* reachable, not less.** The default is closed
  already; a permissive policy is you opening a door.
- **The absence of a header is the denial.** There is no "CORS reject" status
  code. You answer, you simply grant no permission, and the browser refuses to
  hand the response to the page.

A **preflight** is the browser asking permission *before* sending a request it
considers non-simple (anything but GET/HEAD/POST, or a POST with a
`Content-Type` other than the three form-ish ones — which means every JSON POST
you have ever written). It looks like:

```
OPTIONS /tasks
Origin: https://app.example.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: content-type, authorization
```

It is a question — "may this origin send a POST with these headers?" — that
carries no body and must not reach your handler. Answer 204 with the
permissions granted: `Access-Control-Allow-Origin`, `-Allow-Methods`,
`-Allow-Headers`, and `-Max-Age` so the browser stops asking for a while.

Two bugs to name:

**Wildcard plus credentials.** `Access-Control-Allow-Origin: *` with
`Access-Control-Allow-Credentials: true` would let *any* website make
authenticated requests as your logged-in user and read the results. Browsers
refuse the combination outright — which people "fix" by reflecting whatever
`Origin` arrived, and that is strictly worse: a wildcard that also passes
credentials. Match against an explicit list, echo the exact string, and never
prefix- or suffix-match: `https://app.example.com.evil.test` ends with your
domain and is not your domain.

**Missing `Vary: Origin`.** If you echo the origin, the response depends on the
`Origin` header, and a cache that does not know that will hand one origin's
`Access-Control-Allow-Origin` to another. Set it on every response, including
the ones carrying no CORS headers at all.

One more thing granting an origin does *not* grant. A page may read only the
seven safelisted response headers — `Cache-Control`, `Content-Language`,
`Content-Length`, `Content-Type`, `Expires`, `Last-Modified`, `Pragma` — and
nothing else, however carefully you set it. The `Retry-After` your limiter
computes to the second is invisible to the browser client this whole argument
is about; so is an `ETag` a client needs for its next conditional request, and
so is the `X-Request-Id` you ask users to quote in a bug report. Every header a
client must *act on* has to be named in `Access-Control-Expose-Headers`.

## Headers that matter, and headers that are theatre

For a JSON API, these earn their bytes:

- `X-Content-Type-Options: nosniff` — stops the browser second-guessing your
  declared `Content-Type`. An endpoint that reflects caller text and gets
  sniffed as HTML executes it.
- `Content-Security-Policy: default-src 'none'; frame-ancestors 'none';
  base-uri 'none'` — if a response is ever rendered as a document it may load
  nothing and nobody may frame it. `frame-ancestors` is the modern replacement
  for `X-Frame-Options`.
- `Strict-Transport-Security` — the browser refuses plaintext to this host for
  `max-age` seconds, killing the "downgrade the first request" attack. Send it
  **only over TLS**: on a plaintext response it is ignored by spec, and sending
  it anyway hides the question of whether you are behind TLS at all. Learn what
  `includeSubDomains` and `preload` commit you to before using them; both are
  hard to undo.
- `Referrer-Policy: no-referrer` — cheap, and keeps ids out of the `Referer`
  header on the next page your user visits.

And theatre: `X-XSS-Protection` is dead — no current browser implements it, and
the filter it once enabled introduced its own vulnerabilities. `X-Frame-Options`
is redundant next to `frame-ancestors`. A long CSP of `script-src` and
`style-src` rules on an endpoint that only returns `application/json` is noise
you must justify in every audit that follows. Saying *why* each header is there
is the actual deliverable.

## Timeouts, layer by layer

Recall S5: `http.Server`'s timeout fields all default to zero, and zero means
*no limit*. Four of them matter, and each ends a specific attack:

| Field | Bounds | Ends |
|---|---|---|
| `ReadHeaderTimeout` | headers | slow-header connections (Slowloris) |
| `ReadTimeout` | headers + body | a body dribbled one byte at a time |
| `WriteTimeout` | writing the response | a client that stops reading |
| `IdleTimeout` | keep-alive gaps | parked connections never reused |

Above those sit two more layers. `http.TimeoutHandler` bounds a single handler:
it runs the handler on its own goroutine with a deadline on the request context
and answers 503 when the deadline wins. It does *not* kill the goroutine —
nothing in Go can — which is why every call beneath it takes a
`context.Context` and checks it. That deadline rides the request context
downward, so your database call, HTTP client call and queue publish inherit it
and a caller who gave up does not leave you working.

One constraint people discover in production: **`WriteTimeout` must exceed the
handler timeout**, or the connection dies before the 503 explaining the failure
can be written and clients see a reset instead of a diagnosis.

## Order is part of the design

A middleware stack is not a set. Two arguments from this lesson's stack:

- **CORS outside the rate limiter.** A 429 with no `Access-Control-Allow-Origin`
  is, to the browser, an unreadable cross-origin response — the frontend
  developer sees "network error" and cannot tell you were rate limiting. The
  cost of this choice is that preflights bypass the limiter; they are small and
  bodyless, and if their volume ever matters you limit them separately.
- **Rate limiting outside anything expensive.** If parsing runs first, a client
  over its budget still made you read a body and decode it. The refusal must be
  a map lookup.

Security headers go outermost, so every response carries them — including ones
produced by middleware that never reached your handler.

And one kind of route does not belong inside this stack at all. To be able to
throw a late response away and write a 503 instead, `http.TimeoutHandler`
buffers everything the handler writes — and the writer it hands down implements
neither `Flush` nor `Unwrap`, so `http.NewResponseController(w).Flush()` finds
nothing to flush and reports `http.ErrNotSupported`. Put the realtime lesson's
SSE handler behind it and the first flush fails, which that handler correctly
treats as fatal: a 500 instead of a stream. Register streaming routes outside
`Timeout` — their own mux branch, wrapped by everything except this one — and
bound them per frame with `ResponseController.SetWriteDeadline` instead.

## Exercise

Open [`exercise/`](exercise/) — a Go module, package `harden`, with a working
middleware chain, error envelope and `Clock` already in place. The tests are
the specification; read them first.

Acceptance criteria:

1. `Limiter.Allow` implements a per-key token bucket over the injected
   `Clock`: a fresh key starts with a full burst, tokens refill at `rate` per
   second capped at `burst`, keys are independent, and a refusal reports how
   long until the next token. It is safe under `-race`.
2. `Limiter.Cleanup(idle)` removes every bucket untouched for at least `idle`
   and returns the count.
3. `RateLimit` answers a refusal with 429, a whole-second `Retry-After` of at
   least 1, and the standard error envelope, and never calls the next handler.
4. `DecodeJSON` enforces, in order: `application/json` (415), the byte limit
   via `http.MaxBytesReader` (413), strict decoding with unknown fields and
   type mismatches reported as 400 *naming the field*, a single JSON value
   (400), and `Validate` (422 with every failing field).
5. `CreateTaskRequest.Validate` reports every broken rule in declaration
   order: non-blank title of at most 80 *runes*, priority 1-5, at most 5 tags.
6. `NewCORS` rejects wildcard-plus-credentials with `ErrCredentialedWildcard`;
   the middleware sets `Vary: Origin` on every response, echoes an allowed
   origin exactly, names `ExposedHeaders` in `Access-Control-Expose-Headers` on
   an allowed non-preflight response, answers preflights with 204 without
   calling the handler, and grants nothing to an unlisted origin or method.
7. `SecurityHeaders` sets `nosniff`, the CSP and `Referrer-Policy` always,
   HSTS only when `r.TLS != nil`, and never sets `X-XSS-Protection`.
8. `NewServer` sets all four timeouts plus `MaxHeaderBytes`, with
   `ReadHeaderTimeout <= ReadTimeout` and `WriteTimeout > HandlerTimeout`.
9. `Harden` composes `SecurityHeaders -> CORS -> RateLimit -> Timeout -> h`,
   so a 429 still carries CORS and security headers and never touches the
   request body.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -race ./...
```

They fail before you start. Make them green, then be ready to defend the
*order* of your stack out loud — that question is coming.

## Further reading

- [pkg.go.dev — net/http (MaxBytesReader, TimeoutHandler, Server)](https://pkg.go.dev/net/http)
- [MDN — Cross-Origin Resource Sharing (CORS)](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [pkg.go.dev — golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
- [RFC 9110 §10.2.3 — Retry-After](https://www.rfc-editor.org/rfc/rfc9110#field.retry-after)
- [RFC 6585 §4 — 429 Too Many Requests](https://www.rfc-editor.org/rfc/rfc6585#section-4)
