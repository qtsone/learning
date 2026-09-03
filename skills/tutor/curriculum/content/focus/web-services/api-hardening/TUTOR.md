# Tutor notes — API Hardening

## Where the learner is

Fourth lesson of the web-services focus pack, after authentication,
authorization and realtime transports. They have S5's production HTTP server
in their hands: middleware chains, 1.22 mux patterns, graceful shutdown,
`log/slog`, fake clocks from advanced testing, and S4's SQL and
security/OWASP basics. They have *not* done S6 systems design — no SLO or
distributed-systems vocabulary; if the "per-instance is not global" point needs
expanding, expand it in plain terms (four processes, four separate counters).

The intellectual move of this lesson is a change of question. Authentication
and authorization asked "who is this" and "may they". Here the question is
"what does this request cost me, and who decides that number". Every technique
is a bound. If they leave able to say that sentence, the lesson landed.

Watch for two failure shapes. The first is cargo cult: they copy a header list
from a blog and cannot say what any of it does. Push on *why* until each header
has a reason or gets deleted. The second is security theatre by fear: they
start blocklisting quote characters "to prevent SQL injection". That is the
misconception the lesson exists to break.

## Common misconceptions

- **"CORS protects my API."** It does the opposite: the browser's default is
  already closed, and CORS is how you *open* it. It is enforced in the browser,
  it is invisible to `curl`, and an unlisted origin's request still reaches and
  runs your handler. If they think a CORS policy stops requests, go back to the
  same-origin policy and ask what the browser is actually preventing (reading
  the response, not sending the request).
- **"Reflecting the Origin header is a safe way to allow several origins."**
  It is a wildcard that also works with credentials — strictly worse than `*`.
  So is suffix matching: `https://app.example.com.evil.test`.
- **"Validation prevents SQL injection / XSS."** The load-bearing distinction
  of the lesson. Validation is a domain bound; injection is a property of the
  *sink*. `O'Brien` must be storable. Parameterized queries and contextual
  output encoding are not fallbacks for weak validation, they are the actual
  mechanism.
- **"`Content-Length` bounds the body."** It is a claim by the sender, and
  chunked encoding omits it. Only `MaxBytesReader` bounds anything.
- **"400 covers all bad input."** Push on 400 vs 422 vs 413 vs 415: they go to
  different people (serializer bug, user typo, oversized upload, wrong client
  configuration).
- **"`DisallowUnknownFields` should be on everywhere."** On the server, yes;
  in a client decoding someone else's API, no — new fields upstream would break
  you. Same function, opposite defaults, because the trust direction differs.
  (They met this from the client side in S3's JSON lesson.)
- **"A rate limiter makes me DoS-proof."** It is per process and per key. Four
  replicas multiply the limit by four; NAT puts thousands of users on one key;
  the bucket map is itself an allocation target.
- **"Sleep in the test to let tokens refill."** The whole point of the
  injected clock. A sleeping test is slow, flaky under `-race`, and asserts
  wall-clock behavior that is not the contract.
- **"HSTS on every response."** Ignored by browsers on plaintext by spec.
- **"`http.TimeoutHandler` kills the slow handler."** It answers the client.
  The goroutine runs until it notices its context.

## Grilling points

- "Walk me through your middleware order and defend each adjacency. What breaks
  if CORS moves inside the rate limiter? What breaks if the rate limiter moves
  inside the decoder?" (The two order arguments; they should reach "the browser
  reports a network error instead of my 429" and "I did the expensive work
  anyway".)
- "A client sends a 10 MB body to your endpoint. Trace what happens, line by
  line, with and without `MaxBytesReader`."
- "Why `MaxBytesReader` rather than `io.LimitReader`?" (Distinguishable error →
  413 instead of a misleading 400.)
- "Your validator rejects `'` in names to stop SQL injection. Convince me
  that's a good idea." (It is not: it breaks O'Brien and misses the real fix.
  Make them articulate sink-side escaping.)
- "You return `Retry-After: 0`. What does a client do?" (Retries immediately
  into another 429; round up, floor at 1.)
- "You run four replicas with a 100/s limiter. What is your actual limit? What
  would it take to make it 100/s globally, and what does that cost?"
- "Somebody deletes the bucket-eviction code as dead weight. What happens over
  a week?" (Unbounded map keyed by attacker-chosen values.)
- "Why is evicting a bucket after 1 second wrong when rate is 1/s and burst is
  60?" (Hands the client a full bucket; eviction must not be cheaper than
  waiting.)
- "Which of your security headers would you delete if an auditor asked you to
  justify every one, and which would you defend?"
- "A preflight arrives for `DELETE`. Your policy allows GET and POST. What
  status do you send and which headers?" (204, and no `Allow-Methods` — the
  denial is the absence.)

## Grading rubric

- **A** — All tests pass under `-race`. The limiter reads the clock once and
  holds the mutex only around bucket state; refusals round `Retry-After` up.
  `DecodeJSON` maps errors with `errors.As`/`errors.Is` and confines the
  unknown-field string match to one place. CORS precomputes its sets outside
  the request path and echoes origins exactly. The learner defends the stack
  order unprompted and states the validation-vs-encoding distinction cleanly.
- **B** — Tests pass; the design is sound but the seams are rough: time read
  inside the lock, error mapping done with `strings.Contains` where a typed
  error exists, CORS sets rebuilt per request, or `Retry-After` rounded down.
  Explanations mostly right, with one of the misconceptions above still live.
- **C** — Tests pass only with substantial hinting, or the learner treats the
  stack as a magic incantation and cannot justify an ordering. Pass only if a
  time-boxed remediation lands; otherwise iterate.
- **Fail** — Tests failing, or the learner asserts that boundary validation
  removes the need for parameterized queries or output encoding, or cannot
  explain what CORS protects. Remediate; both are load-bearing for the rest of
  the pack.

## Remediation ladder

1. "Run one failing test with `-run` and read the message aloud. What did it
   expect, what did it get?" (The failure text names the concept every time.)
2. Narrow to the concept, not the code: "Draw the bucket. It holds 3 tokens,
   the clock has not moved, four requests arrive. What happens to the fourth,
   and when does it become possible?"
3. Name the tool without the shape: "`errors.As` with `*http.MaxBytesError`
   tells you which of these failures happened"; "a preflight is an OPTIONS
   request carrying `Access-Control-Request-Method` — how do you recognise it,
   and why must it not reach the handler?"
4. Walk the algorithm verbally — read clock, lock, find-or-create bucket, add
   `elapsed * rate` capped at burst, spend or refuse — and let them type it.
   Only write code alongside them if step 3 stalls twice.

## After passing

Preview: "Next you meet GraphQL, where a single endpoint lets the *client*
choose the shape of the work — which makes half of what you just built
(per-endpoint limits, per-route sizing) stop applying in the obvious way."
