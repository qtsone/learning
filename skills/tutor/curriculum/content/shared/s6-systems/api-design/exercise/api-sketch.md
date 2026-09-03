# API design sketch — Ledgerly

Fill every section. Short and reasoned beats long and hedged. Lines
marked *assumption* or *cost* are graded content, not padding.

## 0 — Estimates first

Back-of-the-envelope, from the brief's numbers. Show the arithmetic;
round aggressively.

- Average charge-creation QPS: <!-- number + one line of arithmetic -->
- Peak charge-creation QPS: <!-- number + assumption -->
- Read QPS (lists + gets), peak: <!-- number + assumption -->
- Transaction storage growth per year: <!-- number + assumption -->
- Idempotency-key storage at your chosen retention: <!-- number + assumption -->
- One estimate the brief does NOT ask for but your design depends on:
  <!-- name it, estimate it, state the assumption -->

## 1 — Protocol decisions

### Merchant surface

- Choice (REST / gRPC / other):
- Driver 1 — audience:
- Driver 2 — tooling:
- Driver 3 — performance:
- Rejected alternative and what rejecting it costs us:

### Internal surface (fraud, settlement)

- Choice:
- Drivers (audience / tooling / performance — the ones that actually
  decided it here):
- Rejected alternative and its cost:
- Does the 150 ms fraud budget influence this choice? How?

## 2 — Merchant surface: resources and operations

5-7 operations. Choose methods so the protocol's idempotency promises
hold — you will defend each row.

| Operation | Method + path | Success status | Failure statuses you commit to | Idempotent as designed? |
|---|---|---|---|---|
|  |  |  |  |  |
|  |  |  |  |  |
|  |  |  |  |  |
|  |  |  |  |  |
|  |  |  |  |  |

Naming/consistency rules you are imposing on the whole surface (plural
nouns? id format? timestamp format?):

-

## 3 — Versioning policy

- Strategy (URI / header / additive-only schema evolution / mix):
- What "breaking" means under your policy, in one sentence:
- The tolerant-reader rule you write into the contract (exact wording):
- Deprecation policy in 2-3 lines (announce → dual-run → sunset — with
  rough durations and how you know who's still on the old surface):

### Change drill

Classify each proposed change under YOUR policy: **NB** (non-breaking,
ship it), **B** (breaking — say what you'd do instead or how you'd roll
it out). One line of justification each.

| # | Proposed change | NB / B | Why — and if B, the escape route |
|---|---|---|---|
| 1 | Add an optional `metadata` map to the Charge response |  |  |
| 2 | Rename response field `amount` to `amount_minor_units` |  |  |
| 3 | Keep the name `amount` but change its meaning from decimal dollars to integer cents |  |  |
| 4 | Add a required `currency` field to the create-charge request |  |  |
| 5 | Add a new `GET /payouts` endpoint |  |  |
| 6 | Charge `status` gains a new possible value: `disputed` |  |  |
| 7 | Start rejecting amounts above 1,000,000 (previously accepted) |  |  |
| 8 | Internal proto: field `risk_notes` (number 4) was deleted last quarter; reuse number 4 for a new `risk_flags` field |  |  |

## 4 — Pagination: `list transactions`

Design for the biggest merchant (20M rows, nightly full walk) and for
requirement 2 of the brief.

- Scheme and why (name what you rejected):
- Sort key + tie-break:
- What the cursor token contains, exactly:
- Token encoding, and why opaque:
- Default and maximum `limit`, with the reasoning:
- Narrative (2-3 sentences): a client is mid-walk, 200 new charges land.
  What does the client see, and why is nothing skipped or duplicated?
- What happens if the row the cursor points at is deleted?

## 5 — Idempotency: `create charge`

The brief's requirement 1. Design the full scheme:

- Who generates the key, and what makes a "logical operation" (when does
  a client reuse a key vs mint a new one)?
- Where the server records it, and what guarantees first-writer-wins
  (name the mechanism):
- What is stored alongside the key:
- Replay of a *completed* key returns:
- Same key, different request body →
- Same key while the first attempt is still executing →
- Retention period and scope, with the storage estimate from section 0:
- One failure story, walked through: the server executes the charge,
  then crashes before sending the response. The client retries. Step
  through what your scheme does.

## 6 — Error contract

- Error body shape (sketch it — the fields and what each is for):
- Three concrete errors from your surface (code, status, when):

| Code | HTTP status | When |
|---|---|---|
|  |  |  |
|  |  |  |
|  |  |  |

## 7 — Trade-off log

At least three, in the form "Chose X over Y; it costs us Z." These must
be real losses, not straw men.

1.
2.
3.
