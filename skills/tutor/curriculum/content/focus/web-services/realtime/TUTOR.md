# Tutor notes — WebSockets & SSE

## Where the learner is

Third lesson of the web-services focus pack, after authentication and
authorization. They bring S5's production server (1.22 mux patterns, middleware
chains, graceful shutdown, `log/slog`), S5's concurrency and memory model, S5's
fake clocks, and S4's TDD. API hardening, GraphQL, background jobs and
performance are all still *ahead* of them: CORS, rate limiting and delivery
guarantees are not yet shared vocabulary, and the lesson explains in place the
one piece of each it needs. They have not done S6 either, so keep the two-
replica question concrete ("a client on A never sees an event published on B")
rather than reaching for systems-design terms.

The intellectual move is from *request/response* to *a connection you own for
hours*. Every handler they have written so far ends; this one does not, and
almost every rule in the lesson follows from that. Two facts must survive:
**a connected client is a goroutine plus a buffer, and both are finite**, and
**the slow client is the design problem** — bounded send, non-blocking publish,
and an eviction policy chosen deliberately rather than discovered in an outage.

Watch for the learner who reaches for WebSockets because they sound like the
serious answer. Make them say out loud what the client needs to *send*. When
the honest answer is "nothing", they have just talked themselves into SSE — and
the learner who talks themselves into polling has understood more, not less.

## Common misconceptions

- **"The server pushes to the client."** It cannot dial anything: the client
  opens a connection and the server holds it. All three transports are
  variations on how long it is held and what may travel over it.
- **"WebSockets are the modern one, so use them."** They cost a library, a
  protocol you cannot debug with `curl`, a second security model, and a
  reconnection loop you write yourself. Ask what the client sends back.
- **"Polling is amateur."** Do the arithmetic with them: latency promised over
  interval affordable, times clients. For rare updates with a promise measured
  in minutes, polling wins on every axis including on-call.
- **"I wrote the frame, so the client has it."** Not until it is flushed.
  Nothing arriving at all is the signature symptom, and it looks like a hang.
- **"`w.(http.Flusher)` is how you flush."** It is how you flush a bare
  `ResponseWriter`. Behind their own middleware it is a type assertion against
  a wrapper, and it fails. `http.NewResponseController(w)` walks `Unwrap()`.
- **"Bigger buffers mean nobody gets dropped."** A bigger buffer buys a longer
  stall before the same decision, at more memory per client. Unbounded means
  one stalled client can exhaust the process.
- **"A blocking send is fine, the client will catch up."** One blocked send
  inside the fan-out stops every other subscriber. That is a single hotel-wifi
  laptop taking out the dashboard for everyone.
- **"The reader closes the channel when it is done."** Then a publisher mid-
  send panics — in an unrelated request's goroutine. The hub owns the channel,
  closes it, and closes it under the same lock it sends under.
- **"`Last-Event-ID` replays what the client missed."** The transport gives you
  the round trip; the memory is yours. No backlog, no replay.
- **"When history is gone, send a comment saying so."** Comments are invisible
  to application code by design. A gap the page must act on has to be a real
  event — hence `ResyncEvent`.
- **"Replay first, then subscribe."** Anything published in the window belongs
  to neither and is lost silently. Subscribing first can duplicate an event the
  client can spot by id, which is a bug report rather than a mystery.
- **"`http.Server.Shutdown` will end my streams."** It waits for handlers to
  return, and a healthy stream never returns. Shut the hub down first.
- **"Heartbeats detect dead clients."** They do, but their first job is to stop
  a proxy or NAT from reaping a connection that is merely quiet.
- **"`Sec-WebSocket-Accept` proves who the client is."** It proves the
  *responder* speaks WebSocket. It is not authentication, and SHA-1's weakness
  is irrelevant to it.
- **"Sleep in the test until the heartbeat fires."** The clock is injected for
  exactly this. A sleeping test is slow, flaky under `-race`, and asserts wall-
  clock behaviour that is not the contract.

## Grilling points

- "A live-price dashboard, a chat feature, and a report that takes five minutes
  to build. Pick a transport for each and defend it in one sentence."
- "Ten thousand connected clients. What is that in goroutines, file descriptors
  and memory — and which of the three runs out first?"
- "A client on hotel wifi stops reading. Walk me through what happens with (a)
  the publisher writing to sockets directly, (b) an unbounded buffer, (c) what
  you built."
- "You evicted a subscriber. What does that user actually see, and what has to
  be true for the answer to be 'a hiccup'?"
- "Why is dropping the newest event acceptable for a temperature gauge and
  corrupting for a chat log?"
- "Write me the interleaving that panics if the hub closes a channel outside
  the lock it publishes under. Whose goroutine dies?"
- "Deploys now take 30 seconds per instance. Why, and what do you call before
  `Shutdown`?"
- "Draw the window where an event is lost if you replay before subscribing.
  Now tell me what subscribing first costs instead, and why you prefer it."
- "You are behind nginx and a cloud load balancer. Name three ways a working
  stream breaks in production that never show up on your laptop." (Response
  buffering; idle timeout with no heartbeat; `WriteTimeout` on the server;
  gzip middleware buffering; the six-connection-per-origin cap on HTTP/1.1.)
- "Two replicas, one hub each. A user is connected to A, the event is published
  on B. What does the user see, and what are your options?"
- "Why does `Frame` refuse a newline in `ID` but allow one in `Data`?"
- "Your socket endpoint has no `Origin` check and your app uses session
  cookies. Write me the attack, step by step."

## Grading rubric

- **A** — All tests pass under `-race`. `Publish` is non-blocking and evicts
  under the lock; `remove` is the single place a channel is closed and it is
  idempotent; the handler subscribes before writing a byte, defers
  `Unsubscribe` on the next line, uses `http.NewResponseController` rather than
  a type assertion, and its loop selects on the request context, the subscriber
  channel and the injected ticker. `Since` copies what it returns. The learner
  explains the eviction policy as a product decision, and can say why
  reconnection is what makes eviction humane.
- **B** — Tests pass and the design is sound, but the seams are rough: the
  backlog aliased instead of copied, the flush probe done as
  `w.(http.Flusher)` with a fallback, `Unsubscribe` deferred after the header
  block, or a validation rule reimplemented in two places. Explanations mostly
  right with one misconception still live.
- **C** — Tests pass only after substantial hinting, or the learner cannot say
  what the bounded channel is for beyond "the test wanted it". Pass only if a
  time-boxed remediation lands; otherwise iterate.
- **Fail** — Tests failing; or the learner still believes a blocking send is
  acceptable, that the reader may close the channel, or that `Last-Event-ID`
  replays by itself. All three are load-bearing for the capstone.

## Remediation ladder

1. "Run the one failing test with `-run` and read the message aloud — what did
   it expect, what did it get?" The failure text names the concept each time.
2. Move to the state, not the code: "Draw the hub with two subscribers, one
   buffer full. Now publish. Which map entry changes, which channel closes, and
   who is holding the lock while it happens?"
3. Name the tool without the shape: "one `select` with a `default` branch is a
   non-blocking send"; "`http.NewResponseController(w).Flush()` reports
   `http.ErrNotSupported` *before writing anything*"; "a nil channel blocks
   forever in a `select`, which is what you want when heartbeats are off".
4. Walk one path verbally end to end — subscribe, headers, probe flush, retry,
   replay, ticker, loop — and let them type it. Only write code beside them if
   step 3 stalls twice.

If they are stuck across the board, order the work: `event.go` first (pure
functions, fast feedback), then `hub.go` (no HTTP in it at all), and only then
`sse.go`, which is the two of them wired to a request. `websocket.go` is
independent and can be done any time.

## After passing

Preview: "Next is API hardening — the edge of the service. Body limits, strict
decoding, rate limiting, CORS, and the timeout table that explains why your
stream was being cut off at thirty seconds."
