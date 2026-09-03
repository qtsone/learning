# Tutor notes — Networking Deep Dive

## Where the learner is

Second lesson of S6, straight after design-intro's estimation drill. They
have shipped real HTTP services (S3), gRPC (S5), and know `net/http`,
`http.Transport`, and `context` from the outside — but they have used those
libraries without ever seeing the byte stream underneath. S0 gave them the
vocabulary (IP, DNS, ports, packets) and nothing more.

This is the stage's first implementation lesson, and the first code since
S5, so expect enthusiasm plus rust. The theory is broad (TCP, TLS, HTTP
generations, load balancing) and the exercise is deliberately narrow:
framing and a TLS config. Keep the discussion anchored to the two things
they build — every acceptance criterion maps to a concept from the lesson,
and the fastest way to make head-of-line blocking or certificate
verification concrete is to point at their own test output.

The exercise never touches a real network: `net.Pipe` and an in-memory test
CA. If they ask "is this real?", the answer is yes — a `*tls.Conn` over a
pipe runs exactly the handshake a socket would.

## Common misconceptions

- **"TCP delivers messages."** The single most valuable correction in this
  lesson. They wrote `conn.Write(a); conn.Write(b)` and expect two reads.
  Their fragmented-stream test failing is the teachable moment; make them
  say the rule aloud: *order is preserved, boundaries are not*.
- **"A short `Read` is an error."** Reading fewer bytes than requested is
  the normal case. If they wrap `Read` in a retry-on-error loop instead of
  reaching for `io.ReadFull`, they have the mental model backwards.
- **"HTTPS means secure."** Probe for the authenticity leg: encryption to
  an unverified peer is encryption to the attacker. Anyone who has ever
  "fixed" a certificate error with `InsecureSkipVerify: true` needs to say
  out loud what that turned off.
- **"TLS encrypts, so the CA is a formality."** The chain is what makes the
  key mean a name. Ask what stops them from minting a certificate for
  `api.stripe.com` right now (nothing — no trusted root will sign it).
- **"HTTP/2 removed head-of-line blocking."** It removed the *application*
  one and inherited the *transport* one. If they cannot name which layer
  each lives at, they cannot explain why HTTP/3 needed a new transport.
- **"HTTP/3 is faster, so use it everywhere."** Inside a datacenter, loss
  is rare and HTTP/2 is the workhorse (and gRPC's mandatory transport). The
  gain is on lossy, high-RTT last miles.
- **"UDP is the fast option."** UDP is the *no-guarantees* option. Choosing
  it means owning ordering, retransmission, and congestion control. Make
  them answer "what do you build back?" before they may pick it.
- **"L7 is just a better L4."** L7 costs CPU per request, terminates TLS
  (so it must be inside the trust boundary), and adds a hop. L4 stays right
  for TLS passthrough, non-HTTP protocols, and the outermost tier.
- **"The 4-byte length is just a number."** It is input from a stranger.
  Anyone who allocates before checking it against a ceiling has written the
  memory-exhaustion bug for real.

## Grilling points

- "Your `ReadFrame` gets a header claiming 3 GB. Walk me through what your
  code does, in order, and what a version without the ceiling would do to
  the process."
- "Why is `io.EOF` a different answer from `io.ErrUnexpectedEOF` here? Give
  me the operational consequence — what does each mean at 3 a.m.?"
- "Delete the length prefix. Design the same protocol with a delimiter
  instead. What did you just make harder?" (Escaping, scanning every byte;
  the HTTP/1.1 vs HTTP/2 trade in miniature.)
- "You set `ServerName`. What exactly would go wrong without it, given the
  certificate still chains to a trusted root?"
- "Your tests use a private CA. What is the real-world equivalent of that
  pool, and how does mutual TLS change the picture inside a platform?"
- "A user in Sydney reports slow first page loads; your servers are in
  Frankfurt. Estimate the setup cost before a single byte of HTML moves —
  handshakes included." (Design-intro's numbers: ~250 ms RTT, TCP + TLS
  ≈ 2 RTTs ≈ half a second.)
- "Your gRPC fleet has one backend at 90% CPU and five idle, behind an L4
  balancer. Explain it, then fix it."
- "A video call and a bank transfer. Which transport for each, and what do
  you build yourself in the UDP case?"
- "You move a chatty service from HTTP/1.1 to HTTP/2 and p99 gets *worse*
  on a mobile network. What is your first hypothesis?"

## Grading rubric

- **A** — All tests pass under `-race`. `ReadFrame` uses `io.ReadFull` (or
  an equivalent loop), checks the length before allocating, and translates
  the body's `io.EOF` into `io.ErrUnexpectedEOF` deliberately, not by
  accident. `WriteFrame` writes nothing when it rejects. `ServeEcho` maps a
  clean end-of-stream to `nil` and returns everything else. TLS configs set
  roots, name, and a version floor with no `InsecureSkipVerify` anywhere.
  In discussion they can place head-of-line blocking at the right layer per
  HTTP version, justify a TCP/UDP choice with a workload, and explain the
  L4 gRPC imbalance unprompted.
- **B** — Tests pass, but with a wart they cannot defend: a hand-rolled
  read loop that duplicates `io.ReadFull`, an allocation before the size
  check, or EOF handling that works because the test happened to hit the
  path they coded. Theory solid on TCP and TLS, hand-wavy on HTTP/2 vs
  HTTP/3 or on load-balancer layers.
- **C** — Tests pass only after heavy hinting, or the TLS config was
  filled in field-by-field from failure messages without understanding what
  verification does. Time-box remediation on the two big stories
  (stream-not-messages, authenticity) before passing.
- **Fail** — Tests failing, `InsecureSkipVerify` used to get green, or the
  learner cannot explain why framing exists at all. Do not advance: caching
  and message-queues assume this comfort with byte-level protocols, and the
  capstone will ask them to choose a transport and defend it.

## Remediation ladder

1. "Run `go test -race -run TestReadFrameReassembles`. Read the failure
   aloud: how many bytes did that reader hand you per call, and how many
   did your code assume it would get?"
2. "What in the standard library already means 'read exactly N bytes or
   tell me why not'? Read its documentation on which error it returns when
   zero bytes arrived versus some bytes arrived." (`io.ReadFull`.)
3. "Your header promised five bytes and the stream ended with none of them
   delivered. `io.ReadFull` calls that `io.EOF`. Is that a clean shutdown?
   What must you return instead, and where in your code does that
   translation belong?"
4. For TLS: "Open `tlsconfig_test.go` and list the four fields the tests
   assert on. For each, say in one sentence what an attacker gains if it is
   missing." Then let them write the config — do not dictate the struct
   literal.

## After passing

Preview: "You now own the bytes and the trust between two machines. Next:
API design — what you put *inside* those bytes, and the contracts that
survive a version bump."
