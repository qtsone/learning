# WebSockets & SSE

> `focus.web.realtime` · ~3-4h · Stage: Focus: Web Services

## Objectives

By the end of this lesson you can:

- Compare WebSockets, Server-Sent Events and polling on directionality, proxy
  friendliness and reconnection, and choose the right transport for a feature.
- Implement an SSE endpoint in Go with correct headers, correct event framing
  and explicit flushing.
- Explain the WebSocket upgrade handshake, implement and validate it, and
  describe the read/write goroutine discipline and clean shutdown a socket
  needs.
- Implement a hub that fans events out to many clients without leaking
  goroutines and without letting one slow client stall the rest.
- Explain how heartbeats and client reconnection — including SSE's
  `Last-Event-ID` — keep long-lived connections healthy.

## The server cannot dial the client

Everything you have built so far is the same shape: a client asks, you answer,
the exchange is over. Now the requirement inverts. Something happened in your
system — an order shipped, a build finished, another user typed — and a browser
that is already sitting there needs to know. You cannot open a connection to it:
it is behind NAT, behind a firewall, possibly asleep. Every "push" technology on
the web is therefore the same trick: **the client opens a connection and the
server holds it open**. What the three options differ in is how long it is held,
what may travel over it, and what happens when it dies — because it will die.

## Polling, honestly

The dumbest thing works, and works more often than realtime enthusiasts admit:
`GET /api/notifications?since=<cursor>` every ten seconds. No state, no
connection to manage, no goroutine per client, every proxy on earth already
understands it, and a failed request is retried by the next tick. The arithmetic
that decides it: **latency you promise ÷ interval you can afford**. If "within a
minute" is fine, poll and go home. Rare updates and many clients waste almost
everything — 99% of responses say "nothing new" — but frequent updates make
polling *cheaper* than it looks, because every response carries a batch. **Long
polling** — hold the request open until there is news or a timeout — was the
bridge before browsers had anything better; it costs a held connection *and* a
full request cycle per message, reconnection storms are its signature failure,
and it survives only as a fallback.

## The three transports

| | Polling | SSE | WebSocket |
|---|---|---|---|
| Direction | client → server | server → client only | both |
| Protocol | plain HTTP | plain HTTP, one long response | HTTP upgrade to a framed protocol |
| Payload | anything | UTF-8 text | text or binary |
| Reconnect | next tick | automatic, with `Last-Event-ID` | you write it |
| Proxies / LBs | anything | almost anything (buffering is the risk) | needs upgrade support |
| Server cost | none between polls | goroutine + buffer per client | goroutine(s) + buffer per client |
| Browser API | `fetch` | `EventSource` | `WebSocket` |

The decision rule is nearly always: **does the client need to send messages over
the same connection?** Notifications, live prices, a build log tail, a progress
bar — those are one-way, and SSE is plain HTTP that your existing middleware,
auth cookies and observability already understand. Chat, collaborative editing,
multiplayer, anything where the client is a peer — WebSocket. A dashboard
refreshed once a minute — polling, and be pleased about it.

## SSE: the wire format is the whole protocol

An SSE response is one HTTP response that never ends. The body is a stream of
frames, each a block of `field: value` lines ended by a blank line:

```
id: 42
event: task.created
retry: 3000
data: {"id":42,"title":"ship it"}

```

Five rules cover the format:

- **`data:`** is the payload. Multi-line data is several `data:` lines; the
  client rejoins them with `\n`. Only text — UTF-8 — so binary is base64 or
  another transport.
- **`event:`** names the type, which browsers dispatch on
  (`es.addEventListener("task.created", …)`). Absent means `message`.
- **`id:`** labels the event; the client remembers the last one it saw.
- **`retry:`** sets the client's reconnection delay, in **milliseconds**.
- **A line starting with `:`** is a comment: clients parse and drop it, so it is
  invisible to application code — perfect for heartbeats, useless for telling
  the application anything, a point reconnection comes back to below.

The response headers are short but not optional:

```go
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("X-Accel-Buffering", "no")
```

`X-Accel-Buffering: no` is the one people learn the hard way: nginx and
friends buffer responses by default, and a buffered stream is a download that
arrives all at once, hours late. Do **not** set `Connection: keep-alive`;
net/http owns that header and it means nothing over HTTP/2.

Two `EventSource` limits worth knowing before you promise a feature: it can only
issue a **GET** and cannot set headers, so authentication rides on the session
cookie from this pack's first lesson; and browsers cap **six** connections per
origin on HTTP/1.1, so a user with seven tabs open has one tab that hangs
forever — over HTTP/2 the cap is per-connection streams and the problem
disappears, which is a real reason to terminate TLS/HTTP2 in front of a stream.

## Flushing, or why nothing arrives

Go buffers your writes. Without an explicit flush your frames sit in a
`bufio.Writer` until it fills, which for a chat message is never:

```go
rc := http.NewResponseController(w)
_, _ = io.WriteString(w, frame)
if err := rc.Flush(); err != nil {
	return // the client is gone
}
```

`http.ResponseController` (Go 1.20+) is the modern door to the same
`http.Flusher` behaviour, and the one to use inside a middleware stack: S5's
logging middleware wraps `w` in its own type, and `w.(http.Flusher)` on that
wrapper fails unless the wrapper forwards the method. `ResponseController` walks
the `Unwrap()` chain instead, and when nothing in the chain can flush it reports
`http.ErrNotSupported` **without writing anything** — so a failed flush before
the first byte can still become a 500.

The same object gives you `SetWriteDeadline`, and you will need it: S5's
`http.Server.WriteTimeout` bounds *writing the response*, so on a stream — one
response that lasts hours — it kills every client on schedule. Exempt the route
or set a per-write deadline before each frame. Compression middleware has the
same flavour of bug: a gzip writer buffers, so flush it too, or keep it off.

## One client is a goroutine and a buffer

The cost model you are signing up for: each connected client is one goroutine
parked in a `select` (a few KB of stack), one file descriptor with its kernel
socket buffers (the expensive part — tens of KB, and `ulimit -n` is a real
ceiling), and one application-level buffer of pending events (yours to size).

Ten thousand clients is fine on one modern machine; it is also 10,000 things
that must each be cleaned up on exactly the right event. Two rules keep it
honest. **Every exit path unsubscribes** — `defer hub.Unsubscribe(sub)` on the
line after you subscribe, before any error branch exists to forget it, because a
handler that returns without unsubscribing leaks an entry in a map the publisher
walks on every event, and that leak grows the cost of every future publish. And
**no goroutine outlives its request**: `r.Context()` is cancelled when the client
disconnects, so it is the primary loop exit, and anything you spawn from the
handler takes that context and selects on it.

## The slow consumer

Now the failure that defines this lesson. A client on hotel wifi stops reading.
Its socket buffer fills. Your write blocks. If the publisher writes to clients
directly, one stalled client has stopped *every* client, and if you buffer
without a bound, one stalled client allocates until the process dies. So the
send is never unbounded and never blocking: each subscriber gets a bounded
channel, and the publisher does a non-blocking send:

```go
select {
case sub.events <- e:      // fits
default:                   // does not fit: the policy fires here
}
```

The `default` branch is a product decision with three honest options:

1. **Drop the newest** — fine for a "current temperature" gauge where only the
   latest matters; corrupting for anything the client accumulates.
2. **Drop the oldest** — a ring buffer of latest-N. Same silent gap, other end.
3. **Disconnect the subscriber.** The stream ends, the client notices, it
   reconnects with `Last-Event-ID`, and you replay what it missed.

Option 3 is what the exercise implements, because it is the only one where the
client *learns* that it fell behind; the others hide a gap inside data the user
is looking at. Note what makes it work: eviction is only humane because
reconnection is automatic and ids make replay possible. The transport's
recovery story is what lets you be ruthless about memory.

## The hub

Fan-out needs one owner of "who is connected" — one *per process*: two replicas
mean a client connected to A never sees an event published on B, which is a
broker's problem, not a hub's. Within a process, two shapes are idiomatic:

- **A mutex around a map** of subscribers. Publishing walks the map under the
  lock doing non-blocking sends; the lock is held only for a `select` each.
- **A hub goroutine** owning the map, fed by `register`, `unregister` and
  `broadcast` channels. No mutex, but every caller waits on a queue, and the
  hub goroutine becomes a thing to shut down.

Neither is wrong; the map is easier to read and easier to test, so that is what
you will write. Both must obey the one invariant that separates a working hub
from a panicking one. **The hub closes subscriber channels, and it does so under
the same lock it sends under.** A send on a closed channel panics, and the panic
happens in the *publisher's* goroutine — so a client disconnecting takes down an
unrelated request. Closing under the lock makes "closed" and "removed from the
map" a single atomic fact, and on the receiver's side a closed channel is the
signal to stop.

```go
case e, ok := <-sub.Events():
	if !ok {
		return // evicted, or the hub is shutting down
	}
```

## Shutdown, heartbeats and the idle timeout

Two operational facts about connections you hold for hours. **Graceful shutdown
deadlocks on streams.** `http.Server.Shutdown` waits for active handlers to
return, and a streaming handler returns when the client leaves — never, on a
healthy stream — so shutdown blocks until its deadline and your deploy takes 30
seconds per instance. Give the hub a shutdown that closes every subscriber and
call it *before* `Shutdown`: the handlers see closed channels, return, and the
server drains immediately. (`http.Server.BaseContext` is the same lever, via a
context every request inherits.)

**A quiet connection is a dead connection.** Load balancers and proxies close
connections with no bytes for some idle period — 60 seconds on many defaults,
less on mobile carrier NAT — and usually without telling either end: your server
thinks it has a client, the client thinks it has a server, nobody is talking.
The answer on both transports is a heartbeat well under the shortest idle
timeout in the path. In SSE it is a comment frame:

```go
case <-ticks:
	write(Comment("heartbeat " + clock.Now().UTC().Format(time.RFC3339)))
```

It resets every idle timer on the path and doubles as your liveness check: the
write fails when the connection is really gone, which is often the only way you
find out. The ticker comes from the injected `Clock` — the S5 pattern — so tests
fire heartbeats instead of sleeping.

## Reconnection is the protocol's job (SSE)

This is where SSE earns its keep. A browser's `EventSource` reconnects on its
own after a connection ends, waiting the delay you sent in `retry:`, and it
sends the last id it saw back to you:

```
GET /events HTTP/1.1
Last-Event-ID: 42
```

Your handler reads that header, asks the hub for everything after event 42,
writes it, and continues live: the client gets a hiccup instead of a gap. That
only works if you assign ids and keep a backlog — the transport gives you the
round trip, not the memory. The interesting case is when you *cannot* fill the
gap: the client was away an hour and your backlog holds a hundred events. Do not
replay a partial range, which is a silent lie; send an application-visible event
— the exercise calls it `resync` — telling the page to refetch its state and
carry on. A comment would not do: the page never sees comments.

One ordering subtlety: subscribe **first**, then replay. The other order leaves
a window where an event published between the replay and the subscription is
lost by both; subscribing first can send an event twice instead, which the
client spots by id. A duplicate is a bug report; a gap is a mystery.

## WebSockets

When the client must send too, you need a real bidirectional connection, and
that means leaving HTTP behind after one handshake.

```
GET /ws HTTP/1.1
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Version: 13
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==

HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

The accept value is `base64(SHA-1(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))`,
a fixed GUID from RFC 6455. It is not authentication and not integrity — SHA-1
being broken changes nothing — its job is to prove the responder really
understands WebSockets, so an intermediary that mistook the handshake for a
normal request cannot be tricked into "completing" it. After the 101 the
connection is no longer HTTP: it carries binary **frames** with an opcode, a
length, and — from browsers, always — a masking key.

Three things about running one, which the exercise deliberately does *not* have
you implement. **One reader goroutine and one writer goroutine, and no more** —
concurrent writes interleave frames and corrupt the stream, so several producers
go through a channel one writer owns. **Ping/pong is a control frame, not your
protocol**: the server pings on a timer, browsers answer automatically, and the
real point is the **read deadline** — set one a little longer than the ping
interval and push it forward on every pong, or a half-open connection (the peer
is gone but no FIN arrived) stays invisible. **Closing is a handshake**: send a
close frame with a status, wait briefly for the peer's, then close the socket;
on shutdown that plus cancelling the read context stops the pair cleanly.

**Security differs from HTTP in one dangerous way.** The same-origin policy does
not apply to WebSockets: any page the user visits can open a socket to your
host, and the browser attaches this pack's session cookie. A cross-origin
`fetch` at least triggers a browser-enforced negotiation first (CORS — the next
lesson takes it apart); a socket handshake has no preflight and nothing to
withhold. The defence is yours: compare `Origin` against an allowlist and reject
what does not match, or you have built **cross-site WebSocket hijacking**.

Use a library for the frame layer: **`github.com/coder/websocket`** (the
maintained continuation of `nhooyr.io/websocket`; context-aware, small API) or
`github.com/gorilla/websocket` (older, everywhere). The standard library has
none, and `golang.org/x/net/websocket` is frozen. Framing, masking,
fragmentation, close codes and compression are a lot of protocol to re-derive,
and getting them wrong is a memory-safety bug rather than a rendering glitch.
The handshake is yours, though — auth, origin checking and subprotocol
negotiation live there — so it is the part you implement here.

Named so you know they exist, and left out: WebTransport and HTTP/3 datagrams;
WebRTC data channels for peer-to-peer and anything latency-critical enough to
prefer UDP; presence and "who is online"; and delivery guarantees, which the
background-jobs lesson takes seriously.

## Exercise

Open [`exercise/`](exercise/) — one `realtime` package. `clock.go` is provided;
four things are yours:

```
event.go       SSE framing: Event.Frame and Comment
hub.go         subscribe / unsubscribe / publish, eviction, backlog, shutdown
sse.go         the streaming handler
websocket.go   the upgrade handshake and its origin check
```

The `_test.go` files are the specification; `support_test.go` holds the fake
clock and a `ResponseWriter` that records each flush separately — read that one
first, because it shows you what "flushed" means to a test.

Acceptance criteria:

1. `Event.Frame` renders `id`, `event`, `retry`, `data` in that order, one
   `data:` line per line of `Data`, `retry` in whole milliseconds and omitted
   below 1 ms, a blank line to end the frame, and `""` for an empty event.
2. `Frame` returns `ErrInvalidField` (matched with `errors.Is`) and no frame
   when `ID` or `Name` holds a newline, or any field holds a carriage return or
   NUL. `Comment("heartbeat")` is `": heartbeat\n\n"`.
3. `Hub.Subscribe` registers a subscriber with a bounded buffer;
   `Hub.Unsubscribe` removes it and closes its channel exactly once, and is
   safe to call twice, with an unknown subscriber, or with `nil`.
4. `Hub.Publish` delivers to every subscriber without blocking and reports
   `(delivered, evicted)`. A subscriber whose buffer is full is evicted:
   removed from the hub and its channel closed.
5. `Hub.Publish` records identified events in a backlog of the configured size;
   `Hub.Since(id)` returns the events after `id` and whether `id` was found at
   all. An empty id is never found.
6. `Hub.Shutdown` closes every subscriber, leaves `Subscribers()` at 0, makes
   `Subscribe` return `ErrHubClosed`, makes `Publish` a no-op, and is safe to
   call twice.
7. All of the hub is safe under `-race` with subscribers and publishers on many
   goroutines — in particular, no send on a closed channel.
8. `Handler.ServeHTTP` subscribes *before* writing anything, answers **503**
   when the hub is closed, sets `Content-Type: text/event-stream`,
   `Cache-Control: no-cache` and `X-Accel-Buffering: no`, sets no `Connection`
   header, and flushes so a client sees each frame as it is written.
9. A `ResponseWriter` that cannot be flushed gets **500** and no subscription
   is left behind.
10. The handler sends `retry:` first when configured, then — for a request with
    `Last-Event-ID` — the events the hub still has, or `ResyncEvent` when it
    does not, then streams live events.
11. A heartbeat comment carrying the injected clock's time in RFC 3339 UTC is
    written on every tick of a ticker obtained from `Clock.NewTicker`.
12. The handler returns when the request context is cancelled or its
    subscriber channel closes, and leaves no subscription behind either way.
13. `AcceptKey` reproduces RFC 6455's example, and `CheckUpgrade` returns
    `ErrBadMethod`, `ErrNotUpgrade`, `ErrBadVersion` or `ErrBadKey` for each
    way a handshake can be wrong.
14. `Upgrader.Handshake` answers 101 with the three upgrade headers on success;
    405 + `Allow`, 426 + `Sec-WebSocket-Version`, 403 for a disallowed origin
    (empty allowlist denies every browser), and 400 otherwise — never leaking
    `Sec-WebSocket-Accept` on a rejection.

```sh
cd exercise
go test -race ./...
```

No test sleeps, and none may: time moves only when the fake clock is told to. If
you reach for `time.Sleep` to make a test pass, the design is wrong, not the
test.

## Further reading

- [WHATWG HTML — Server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [pkg.go.dev — net/http.ResponseController](https://pkg.go.dev/net/http#ResponseController)
- [RFC 6455 — The WebSocket Protocol](https://www.rfc-editor.org/rfc/rfc6455)
- [pkg.go.dev — github.com/coder/websocket](https://pkg.go.dev/github.com/coder/websocket)
- [OWASP — Testing WebSockets (cross-site hijacking)](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/11-Client-side_Testing/10-Testing_WebSockets)
