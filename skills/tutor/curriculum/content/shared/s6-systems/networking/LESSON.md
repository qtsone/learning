# Networking Deep Dive

> `shared.systems.networking` · ~3-4h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Explain the TCP handshake, flow control, and head-of-line blocking, and
  choose between TCP and UDP for a given workload with justification.
- Describe what TLS provides — confidentiality, integrity, authenticity —
  and walk through the handshake, including certificate verification.
- Contrast HTTP/1.1, HTTP/2, and HTTP/3 in terms of multiplexing,
  head-of-line blocking, and transport, and pick one for a given service.
- Compare L4 and L7 load balancing and explain when each is appropriate.
- Implement a small program exercising raw TCP connections and TLS
  configuration, and observe connection behavior.

## The arrows between the boxes

Design-intro had you drawing boxes and arrows. This lesson is about the
arrows. You have been living on top of them since S3: `net/http` handed you
request and response objects, gRPC in S5 gave you typed calls, and both
stand on the same three layers — TCP carries bytes, TLS protects them,
HTTP gives them meaning. When a design review asks "what happens to your
latency across an ocean?" or "why is one gRPC backend hot while the others
idle?", the answer lives below the libraries, and this lesson takes you
there.

S0's internet lesson gave you the map: IP addresses, DNS, ports, packets.
The one-line recap that matters here: **IP delivers packets best-effort**.
Packets can be lost, reordered, duplicated, or delayed, and IP does not
care. Everything below is machinery built to live with that.

## TCP: a reliable byte stream over an unreliable network

TCP's contract: whatever bytes you write arrive at the other end complete,
in order, exactly once — or the connection errors out. It earns that
contract with three mechanisms worth knowing by name.

**The handshake.** Before any data moves, both sides agree to talk and pick
starting sequence numbers:

```text
client                                server
  | ── SYN (seq=x) ─────────────────▶ |   "let's talk; my bytes start at x"
  | ◀──────── SYN-ACK (seq=y, ack=x+1) |   "yes; mine start at y; got yours"
  | ── ACK (ack=y+1) ───────────────▶ |   "got yours; sending data"
```

Three packets, one full round trip, before the first byte of application
data. Recall design-intro's latency table: a cross-ocean round trip is
~100-150 ms, so a fresh connection costs you that much before hello — and
TLS adds another round trip on top. This is why long-lived connections and
pooling matter: you pay the handshake once and amortize it over thousands
of requests. Your S5 HTTP client did this for you (`http.Transport` keeps
connections alive); now you know what it was saving.

**Reliability and flow control.** Every byte is numbered. The receiver
acknowledges what arrived; the sender retransmits what wasn't acknowledged
in time. Two separate brakes slow a sender down, protecting two different
parties:

- **Flow control** protects the *receiver*: it advertises a receive window
  — "I have room for this many more bytes" — and the sender never exceeds
  it. A slow consumer shrinks the window, possibly to zero, and the sender
  stalls. This is backpressure at the transport layer.
- **Congestion control** protects the *network*: the sender probes for
  available bandwidth (starting small — slow start), and on packet loss
  backs off sharply, treating loss as a congestion signal. On a long,
  lossy link this is what actually limits your throughput.

**A stream, not messages.** This is the property that bites every engineer
once, so we make it bite you in a test instead of in production. TCP
preserves byte order but **not write boundaries**. Three `Write` calls may
arrive as one `Read`; one `Write` may arrive split across five `Read`s.
The kernel and every middlebox between you are free to split and merge
segments however they like. So any protocol built on TCP must define its
own **framing** — a rule for finding message boundaries in the stream.
The two classic designs:

- **Delimiters**: HTTP/1.1 headers end at a blank line (`\r\n\r\n`).
  Human-readable, but you must scan every byte and escape the delimiter.
- **Length prefixes**: send "N bytes follow", then N bytes. HTTP/2 frames
  and gRPC messages work this way, and so will your exercise.

In Go: `conn.Read(buf)` returning fewer bytes than you asked for is not an
error — it is the normal case. When you need exactly N bytes, loop, or use
`io.ReadFull`, which does the loop for you and returns
`io.ErrUnexpectedEOF` if the stream dies partway.

**Head-of-line blocking.** In-order delivery has a price: if segment 17 is
lost, segments 18-40 sit in the receiver's kernel buffer, undeliverable,
until 17 is retransmitted — even if your application multiplexes many
independent logical streams over the connection and 18-40 belong to other
streams. One lost packet stalls everything behind it. File this away; it
decides the HTTP/2 vs HTTP/3 story below.

## UDP, and how to choose

UDP adds almost nothing to IP: port numbers and a checksum. Datagrams
preserve message boundaries (one send is one receive, if it arrives), but
there is no handshake, no ordering, no retransmission, no flow or
congestion control. Whatever reliability you need, you build.

That sounds strictly worse — until reliability itself is the problem.
TCP's retransmission means *waiting for stale data*. For live audio, video,
or game state, a retransmitted packet arrives too late to be useful; you
would rather skip ahead than stall. That inversion — **when stale data is
worthless, retransmission is harmful** — is the core UDP use case. The
other two: tiny request/response exchanges where a retry is cheaper than a
handshake (DNS), and building a better transport in userspace (QUIC, which
carries HTTP/3).

Choosing, with justification — ask three questions:

1. **Must every byte arrive?** Files, APIs, money: yes → TCP. Live media
   and telemetry where the next sample supersedes the last: no → UDP.
2. **Is latency worth more than completeness?** If a late message is a
   useless message, TCP's stalls hurt more than loss does.
3. **Can you afford to build what you drop?** Choosing UDP means owning
   ordering, loss recovery, and congestion behavior yourself. "We'll just
   use UDP for speed" without that plan is how you flood a network.

Default to TCP; reach for UDP when question 1 or 2 flips and you have an
answer for question 3.

## TLS: three guarantees and a handshake

TLS wraps a connection and provides exactly three guarantees. Learn them as
a unit, because losing any one quietly voids the other two:

- **Confidentiality** — an observer on the path sees ciphertext.
- **Integrity** — any tampering with the bytes is detected.
- **Authenticity** — you know *who* you are talking to, via certificates.

Authenticity is the one engineers discard by accident. An encrypted
connection to an unverified peer is worthless: a machine-in-the-middle
terminates TLS with you (encrypted!), reads everything, and re-encrypts to
the real server. Encryption without authentication is encrypting *to the
attacker*.

**The handshake (TLS 1.3).** One round trip, three acts:

1. **ClientHello** — supported versions and ciphers, the server name it
   wants (SNI), and a fresh ephemeral public key share.
2. **ServerHello** — the server's choice of cipher plus its own key share.
   Both sides now derive the same shared secret (ephemeral Diffie-Hellman);
   everything from here on is encrypted. The server sends its
   **certificate** and a signature proving it holds the private key.
3. **Verification and Finished** — the client verifies the certificate
   (below), both sides exchange Finished messages proving they derived the
   same keys, and application data flows.

Because the key shares are ephemeral — generated per connection and thrown
away — recording traffic today and stealing the server's key tomorrow
decrypts nothing. That property is **forward secrecy**. Resumed
connections can even send data in the first flight (0-RTT), at the cost of
replay risk — a trade-off you name, not ignore.

**Certificate verification** is where authenticity actually happens. A
certificate binds a public key to a set of names ("this key speaks for
`api.example.com`"), signed by a certificate authority, whose own
certificate is signed by another, chaining up to a root your machine
already trusts. The client checks three things:

1. The **chain of signatures** reaches a trusted root.
2. The **name matches** — the certificate covers the host the client
   *intended* to reach. A perfectly valid certificate for `evil.example`
   must not pass for `api.example`. This is the check `ServerName` drives.
3. The **validity window** — not expired, not yet-to-begin (plus
   revocation checks in serious deployments).

In Go: `crypto/tls` performs the whole verification by default — you
configure *what to trust* and *what to require*, and never turn checks off:

```go
// Client: which roots to trust, which name to demand, floor the version.
cfg := &tls.Config{
    RootCAs:    pool,             // nil means the system trust store
    ServerName: "api.internal",   // the name the certificate must match
    MinVersion: tls.VersionTLS13,
}

// Server: present a certificate, floor the version.
cfg := &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS13,
}
```

Which floor? TLS 1.3 is the one to demand when you control both ends — it
drops the legacy ciphers and cuts a round trip from the handshake. TLS 1.2
is the floor you pick when you must still reach peers you don't control;
everything below it is broken and non-negotiable. The exercise floors at
1.2 for exactly that reason, and the rule to carry away is *set the floor
deliberately* — a zero `MinVersion` means the library, not you, decided.

`InsecureSkipVerify: true` disables steps 1 and 2 entirely — authenticity
gone, machine-in-the-middle trivial — while the connection still *looks*
encrypted. It exists for tests and debugging; treat it in review the way
you treat `panic()` in a handler. (Servers can also demand client
certificates — mutual TLS — the standard way services authenticate each
other inside a platform.)

## HTTP/1.1, HTTP/2, HTTP/3

Three generations, each attacking the previous one's head-of-line problem:

**HTTP/1.1** — one request at a time per connection. (Pipelining existed
and died; responses had to return in order, so one slow response blocked
all — application-level head-of-line blocking.) The workaround: open ~6
parallel connections per host, paying 6 handshakes and 6 congestion-control
ramp-ups. Text headers, re-sent in full on every request.

**HTTP/2** — many concurrent **streams** multiplexed over *one* TCP
connection, as binary, length-prefixed frames (your exercise's framing,
industrialized), plus header compression. Application-level HOL: solved.
But all streams still share one TCP byte stream, so one lost segment
stalls *every* stream — the transport-level HOL from earlier. On clean
datacenter networks that is rare and HTTP/2 is a huge win; on lossy paths
it can genuinely underperform 1.1's six independent connections.

**HTTP/3** — keeps HTTP/2's model but swaps the transport: **QUIC**, built
on UDP in userspace. QUIC makes streams first-class *in the transport*:
each stream is retransmitted and delivered independently, so a lost packet
stalls only its own stream. TLS 1.3 is built in (transport + crypto
handshake in a single round trip; 0-RTT on resumption), and connections
are identified by a connection ID rather than the IP/port 4-tuple, so a
phone hopping from Wi-Fi to cellular keeps its connections.

| | HTTP/1.1 | HTTP/2 | HTTP/3 |
|---|---|---|---|
| Transport | TCP | TCP | QUIC (UDP) |
| Requests per connection | 1 at a time | many streams | many streams |
| HOL blocking | per request | at transport (TCP) | none across streams |
| Setup RTTs (new conn, incl. TLS 1.3) | 2 | 2 | 1 (0-RTT resumed) |
| Headers | text, repeated | HPACK compressed | QPACK compressed |

Picking one: **inside a datacenter**, loss is rare and RTT is sub-ms —
HTTP/2 is the workhorse, and it is gRPC's mandatory transport, so your S5
services already made this choice. **Public, browser- and mobile-facing
traffic** benefits most from HTTP/3: lossy last miles, long RTTs, network
hopping. In practice you rarely pick in application code — you enable
HTTP/3 at the CDN or load balancer and often keep 1.1/2 from the edge to
your origin, where the network is clean.

## Load balancing: L4 and L7

One box cannot serve everyone (and a single box is a single point of
failure), so a balancer spreads traffic across replicas. The design
question is *which layer it understands* — named for OSI layers 4
(transport) and 7 (application):

**L4** balances **connections**. It looks only at IP addresses and ports —
typically hashing the 4-tuple to pick a backend — and shovels packets
without parsing payloads. Consequences: wire-speed and cheap; works for
any TCP/UDP protocol, not just HTTP; can pass TLS through untouched so the
backend terminates it (the balancer never sees plaintext). But it cannot
see requests: no path-based routing, no per-request retries, and every
connection is pinned to one backend for its lifetime.

**L7** balances **requests**. It terminates TLS and parses HTTP, so it can
route `/api/*` and `/static/*` to different fleets, retry an idempotent
request against another replica, split 1% of traffic to a canary, attach
auth, cache. The price: real CPU per request, certificates and their
rotation live at the balancer, one more hop, and the balancer sees
plaintext — it must be inside your trust boundary.

The pinning consequence deserves a war story you are now equipped to
predict. Your S5 gRPC service used long-lived HTTP/2 connections. Put an
L4 balancer in front of it: each client's connection pins to one backend,
and *all* of that client's multiplexed calls follow it. Two chatty clients
hash onto the same backend and it burns while others idle — and
connection-count balancing can never fix a load imbalance it cannot see.
gRPC fleets want an L7 balancer that spreads individual requests, or
client-side balancing where the client holds connections to every backend.

When each is appropriate: **L4** for non-HTTP protocols, TLS-passthrough
requirements, extreme throughput at the outer edge, or dumb-and-reliable
spreading of short-lived connections. **L7** for HTTP APIs — which is to
say, most services — whenever you need per-request routing, retries, or
gRPC fairness. Real systems layer them: an L4 tier at the edge fans into
an L7 fleet, with health checks at both layers ejecting dead backends.

## Exercise

Open [`exercise/`](exercise/) — a Go module for a package `netlab`. You
will build the two layers this lesson dissected: framing over a raw
connection, and a TLS configuration that actually verifies who it talks
to. No real network is involved: tests drive your code over `net.Pipe`,
an in-memory, synchronous `net.Conn` pair — which also makes TCP's
"stream, not messages" property brutally concrete, because the tests
deliver your frames one byte at a time.

**Part A — framing** (`frame.go`): a length-prefixed protocol.

- `WriteFrame(w, payload)` — 4-byte big-endian length, then the payload.
- `ReadFrame(r)` — read exactly one frame back, however the bytes arrive.
- `ServeEcho(conn)` — loop: read a frame, write it back, until the peer
  hangs up cleanly.

**Part B — TLS configuration** (`tlsconfig.go`): build the client and
server `tls.Config` values that make an in-memory handshake succeed
against a test certificate authority — and *fail* when the name is wrong
or the CA is untrusted. The tests then run your echo server over the TLS
connection: a `*tls.Conn` is just another `net.Conn`.

Acceptance criteria:

1. `WriteFrame` writes the 4-byte big-endian length then the payload, and
   rejects payloads over `MaxFrameSize` with `ErrFrameTooLarge`, writing
   nothing.
2. `ReadFrame` returns exactly one payload no matter how the stream was
   fragmented or coalesced; a claimed length over `MaxFrameSize` fails
   with `ErrFrameTooLarge` *before* allocating or reading the body (the
   length field is attacker-controlled input).
3. End-of-stream is precise: `io.EOF` between frames (clean),
   `io.ErrUnexpectedEOF` inside a frame (truncation). `ServeEcho` returns
   `nil` on a clean close and the error otherwise.
4. `ClientTLSConfig` trusts exactly the given pool, demands the given
   server name, and floors `MinVersion` at TLS 1.2 or newer;
   `ServerTLSConfig` presents the given certificate with the same floor;
   `InsecureSkipVerify` appears nowhere. Handshakes succeed with the right
   name and trusted CA, and fail with a wrong name or an empty pool.
5. `go test -race ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -race ./...
```

They fail on the starter — read the failure messages; they are written to
teach. Start with Part A: `frame_test.go` is the specification.

## Further reading

- [High Performance Browser Networking](https://hpbn.co/) — free online;
  the TCP, TLS, and HTTP/2 chapters are this lesson with the volume
  turned up.
- [The Illustrated TLS 1.3 Connection](https://tls13.xargs.org/) — every
  byte of a real handshake, annotated.
- [Cloudflare Learning — What is HTTP/3?](https://www.cloudflare.com/learning/performance/what-is-http3/)
  — a clear QUIC and head-of-line-blocking explainer.
- [pkg.go.dev — crypto/tls](https://pkg.go.dev/crypto/tls) — the `Config`
  documentation rewards a full read; you now know what every field is for.
