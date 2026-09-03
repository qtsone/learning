# API Design

> `shared.systems.api-design` · ~2-3h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Choose between REST and gRPC for a concrete service scenario and justify
  the choice by audience, tooling, and performance needs.
- Design a versioning strategy (URI, header, or schema evolution) and
  explain what breaking vs non-breaking changes it permits.
- Compare offset and cursor pagination and explain why cursors survive
  concurrent inserts while offsets do not.
- Explain idempotency, identify which HTTP methods must be idempotent, and
  design an idempotency-key scheme for unsafe operations.

## An API is a promise you can't take back

You have already *built* APIs — HTTP services in S3, gRPC and production
hardening in S5. This lesson is the other side of the table: designing the
contract before any code exists, for consumers you do not control.

That last part changes everything. Inside your own codebase, a bad function
signature costs one refactoring afternoon. A published API is different:
every observable behavior — field names, status codes, sort order, error
text — will be depended on by someone (this is Hyrum's law: with enough
consumers, your *implemented* behavior becomes your contract, whatever the
docs say). You can migrate a database schema quietly over a weekend; you
cannot silently change a response shape that a thousand integrations parse.

So API design is not about the first release. It is about **change over
time**: which protocol your audience can consume, which changes you may
make later, how a client walks a large collection, and what happens when a
request is sent twice. The design-intro discipline applies unchanged:
state your assumptions, put numbers on your load, and record every choice
as a trade-off with a named alternative.

## REST or gRPC: decide by audience first

This is not a religious question; it is three practical ones, in order:

1. **Who calls it?** Browsers, third-party developers, curl-wielding
   integrators → REST+JSON. Your own services on your own network → gRPC
   is on the table.
2. **What tooling can you demand of consumers?** gRPC requires consumers
   to take your `.proto` files and run code generation. Your own teams —
   fine, you did it in S5. Two thousand external merchants on every
   language and framework — a real adoption tax. REST asks only for an
   HTTP client and a JSON parser, which every runtime ships.
3. **What are the performance and streaming needs?** Recall the
   networking lesson: gRPC rides HTTP/2 — multiplexed streams, binary
   framing — and protobuf encodes far smaller and faster than JSON. It
   also gives you bidirectional streaming natively. On a hot internal hop
   at thousands of QPS this is real money; on a public endpoint the
   consumer's convenience usually dominates.

| | REST + JSON | gRPC + protobuf |
|---|---|---|
| Contract | Optional (OpenAPI, if disciplined) | Mandatory (`.proto` is the source of truth) |
| Payload | Text, human-readable, larger | Binary, compact, fast |
| Transport | Any HTTP version | HTTP/2 (multiplexing, streams) |
| Streaming | Bolt-on (SSE, WebSocket) | Native, both directions |
| Browser | Native | Needs a proxy (grpc-web) |
| Debugging | `curl` and eyeballs | `grpcurl`, generated clients |
| Codegen | Optional | Built-in, every major language |

A defensible default for a service with both faces: **REST at the edge,
gRPC inside** — public consumers get the universally consumable contract,
internal high-volume hops get the efficient one. Defaults are starting
points, not answers: in the review you will be asked *why* it holds (or
doesn't) for your concrete scenario, with the drivers named.

**In Go:** both cost you one import — `net/http` + `encoding/json` on
one side, `protoc`-generated stubs + `google.golang.org/grpc` on the
other. Implementation effort is a wash. The real cost of the choice
lands on your *consumers*, which is exactly why audience decides.

## Versioning: define "breaking" before choosing a mechanism

A versioning strategy is a promise about which changes you may make
without asking permission. So start with the taxonomy, not the mechanism.

**Non-breaking** (for well-behaved clients):

- Adding a new endpoint or RPC.
- Adding an *optional* response field.
- Adding an *optional* request parameter whose default preserves the old
  behavior.
- Loosening validation (accepting inputs you previously rejected).

**Breaking**:

- Removing or renaming a field, endpoint, or enum value.
- Changing a field's type, units, or meaning (`amount` in dollars →
  cents is breaking even though the type may not change).
- Making an optional request field required, or tightening validation.
- Changing status codes, error shapes, or default sort order that clients
  observably rely on (Hyrum again).

One caveat carries the whole scheme: "adding a response field is safe"
holds **only if clients must ignore unknown fields**. Write that rule into
the contract explicitly (the *tolerant reader* rule). Protobuf enforces it
by design; JSON clients must be told.

The three mechanisms:

1. **URI versioning** — `/v1/charges`. Visible in every log line,
   trivially routable (an L7 balancer can send `/v2/` to a different
   deployment — networking lesson), cache-friendly, curl-friendly. Cost:
   the version is coarse — bumping it forks the *entire* surface, and
   every major version you ship runs in production until the last
   consumer leaves, which is usually years.
2. **Header versioning** — `Accept: application/vnd.ledgerly.v2+json` or
   a custom header. URLs stay stable and you can version per resource.
   Cost: invisible in URLs and logs unless you work at it, hostile to
   casual curl use, and caches must be told to `Vary` on the header.
3. **Schema evolution** — the protobuf discipline: never bump anything;
   make only additive changes, forever. Field *numbers* are the wire
   contract: never reuse or renumber one, `reserve` the numbers of
   deleted fields. Old clients ignore new fields; new servers tolerate
   old messages. This is the gRPC-native answer, and its REST equivalent
   is "additive-only JSON plus tolerant readers" — many mature public
   APIs run for a decade on `v1` this way.

Whichever you choose, a **deprecation policy is part of the design**:
announce, run old and new side by side, set a sunset date, and measure
who still calls the old surface before you turn it off. An API you can
never retire is a design failure with a long fuse.

## Pagination: walking a collection that won't stand still

First, the design-intro reflex: never return an unbounded list. Your
biggest consumer has millions of rows; serializing them in one response
is a self-inflicted outage. Every list endpoint gets a `limit` with a
server-enforced maximum. The question is how the client asks for *the
next page*.

**Offset pagination** — `?offset=200&limit=100`, mapping straight onto
SQL's `OFFSET`. Simple, and it supports "jump to page 7". It fails in two
ways:

1. **Drift under concurrent writes.** You read page 1 of a newest-first
   list (rows 1-100). While you read, two new charges land. You request
   `offset=100` — but everything shifted down by two, so rows 99-100 of
   the *old* page 1 are now at positions 101-102: you see them **again**
   (duplicates). If rows had been deleted instead, the list shifts the
   other way and you **skip** rows you never saw. Offsets count rows,
   and the rows moved. For a merchant reconciling payments, silently
   skipping a transaction is not a cosmetic bug.
2. **Deep offsets are O(n).** `OFFSET 1000000` forces the database to
   produce and discard a million rows before returning any (remember
   your index lessons: an offset is not a seek). Page 10,000 costs
   thousands of times more than page 1 — a tail latency cliff.

**Cursor pagination** — each response carries an opaque token naming the
position *after the last row returned*, in terms of a stable sort key:

```
GET /v1/transactions?limit=100&cursor=eyJjcmVhdGVkIjoi...
```

The token encodes the sort-key values of the last row — say
`(created_at, id)` — and the server resumes with an indexed seek,
in pseudocode-SQL:

```
WHERE (created_at, id) < (cursor.created_at, cursor.id)
ORDER BY created_at DESC, id DESC
LIMIT 100
```

Why it survives concurrent inserts: the cursor names a **position in the
data** ("strictly after this row"), not a **count of rows**. New rows
above the cursor change nothing about what comes after it; deleting the
anchor row itself is also fine, because the comparison is against its
*values*, not a lookup of the row. That is why the token encodes key
values rather than a row reference.

The costs you accept: you need a stable, unique, indexed sort key (a
timestamp alone is not unique — always tie-break with an id); there is
no "jump to page 7"; and the token must be **opaque** (base64 at minimum,
signed if you're serious) — expose readable structure and Hyrum's law
guarantees clients will construct their own.

## Idempotency: design for the retry

Recall S4's http-clients lesson: when a request times out, the client
cannot know whether the server processed it — the request may have been
lost, or the *response* may have been. Every production client retries
(yours did in S5). So the design question is never "will this arrive
twice?" but "what happens when it does?"

**Definition:** an operation is idempotent when performing it N times
leaves the system in the same state as performing it once. Note: same
*state*, not same *response* — a DELETE that returns `204` then `404` on
the retry is still idempotent; one deletion happened.

What HTTP promises (RFC 9110) — the same safe/idempotent table you wrote
clients against in S4, now read from the server's side, because these are
promises *you* are the one keeping:

- **GET, HEAD, OPTIONS, TRACE** — *safe* (read-only) and therefore
  idempotent.
- **PUT, DELETE** — idempotent **by specification**: `PUT` replaces the
  whole resource with the given state (replacing twice = replacing once),
  `DELETE` removes it. Clients and proxies are entitled to auto-retry
  them; if your PUT appends, you have broken a promise the protocol made
  on your behalf.
- **POST** — neither safe nor idempotent. Retrying is not automatically
  allowed, which is precisely why the dangerous operations live here.
- **PATCH** — not guaranteed. "Set field X to 5" is idempotent;
  "increment X by 5" is not. Design the patch semantics, don't assume.

The dangerous case is `POST /charges`: a timeout plus a naive retry is a
double charge and an angry customer. The fix is an **idempotency-key
scheme**, which makes an unsafe operation *effectively* idempotent by
contract — this is exactly how real payment APIs (Stripe et al.) work:

1. The client generates a unique key per *logical* operation (a UUID)
   and sends it: `Idempotency-Key: 07d0…`. Retries of the same operation
   reuse the same key; a genuinely new charge gets a new key.
2. The server records the key **atomically** before executing — a unique
   constraint, as in your SQL lessons; first writer wins. With the key it
   stores a fingerprint (hash) of the request body, and, once finished,
   the response (status + body).
3. A retry with a known, completed key returns the **stored response** —
   the operation does not execute again.
4. Same key, *different* body: reject (`409` or `422`). That client has
   a bug, and silently honoring either request hides it.
5. Same key while the first attempt is still executing: do not run it
   concurrently — return `409 retry-later` (or make the retry wait).
6. Keys expire (say, 24h) and are scoped per consumer and endpoint —
   unbounded key storage is a capacity problem you should be estimating,
   and cross-endpoint collisions must not link unrelated operations.

## The rest of the contract

If a client can observe it, it is contract. Two items worth designing on
day one, briefly: a **consistent error shape** — a machine-readable code
clients can branch on, a human message, and a request id for support —
because clients *will* parse your errors, so give them something stable
to parse; and **documented limits** — timeouts, rate limits, and max page
sizes are semantics, not trivia, and the S5 version of you that operated
services knows exactly why.

## Exercise

Open [`exercise/`](exercise/). You are the API designer for **Ledgerly**,
a payment platform with two faces: a public merchant API and an internal
service mesh. `brief.md` is the product brief with concrete numbers;
`api-sketch.md` is the design worksheet you fill in.

Acceptance criteria:

1. Estimates first: QPS (average and peak), read/write ratio, and
   idempotency-key storage, each with its assumption stated.
2. A protocol decision for *each* surface (merchant, internal), argued by
   audience, tooling, and performance — including the rejected
   alternative and what rejecting it costs you.
3. An endpoint table for the merchant surface (5-7 operations) with
   methods, paths, and status codes — methods chosen so the protocol's
   idempotency promises hold.
4. A versioning policy, plus all eight entries of the change drill
   classified breaking / non-breaking with a one-line justification each.
5. A cursor-pagination design for listing transactions: sort key,
   tie-break, token contents and encoding, and a two-sentence narrative
   of what happens when rows are inserted mid-walk.
6. An idempotency-key scheme for charge creation covering all six design
   points from the lesson.
7. A trade-off log with at least three entries of the form "chose X over
   Y; it costs us Z".

There is nothing to compile. When the worksheet is full, bring it to your
tutor — the verification is a design review, and you will defend every
box you filled.

## Further reading

- [RFC 9110 — Idempotent methods](https://www.rfc-editor.org/rfc/rfc9110#name-idempotent-methods)
- [Stripe — Designing robust and predictable APIs with idempotency](https://stripe.com/blog/idempotency)
- [Protocol Buffers — Updating a message type](https://protobuf.dev/programming-guides/proto3/#updating)
- [Slack Engineering — Evolving API pagination at Slack](https://slack.engineering/evolving-api-pagination-at-slack/)
