# Storage brief — Ledgerly, two years on

The API you designed in the api-design lesson shipped and won.
Traffic is 4× and the single Postgres node underneath is running out
of runway. You are designing the storage layer that keeps the API's
promises for the next three years.

## The numbers

- 20,000,000 charges per day platform-wide; peaks at roughly 4× the
  daily average during flash sales.
- Reads dominate: dashboards, gets, and reconciliation produce about
  10 read requests for every charge created.
- A transaction row is on the order of 1 KB; indexes historically add
  about half again on top of table size.
- The `transactions` table currently holds ~9 billion rows (~9 TB
  before indexes) — four years of history, most of it added in the
  last two — and grows by the 20M/day above.
- The largest merchants hold around 100M rows each (the ~20M whale
  from your api-design brief, five times bigger). During a flash
  sale, a **single merchant** can account for ~30% of platform-wide
  writes for an hour.
- Fraud scoring makes ~3 feature lookups per charge (by card
  fingerprint or merchant id), with a p99 budget of 10 ms per lookup
  inside the 150 ms fraud budget. Features are recomputed by a
  pipeline; fraud tolerates features up to ~5 minutes stale.
- Idempotency keys: one per charge attempt, ~1 KB stored with the
  response, retained 24 h (your api-design scheme).

## Capacity anchors (given — cite them, don't re-derive)

- One well-run relational node: ~5,000 writes/s, ~15,000 simple
  indexed reads/s, comfortable up to **2-4 TB** of hot data. Beyond
  that, backups, failover, and schema changes become operational
  pain long before queries do.
- One in-memory key-value node: ~100,000 reads/s at sub-millisecond
  latency.
- Round trip within a datacenter: ~0.5 ms.

## The datasets

1. **Transactions ledger** — charges, refunds, and their linkage.
   Append-heavy, money math, multi-row invariants (a refund must
   reference and never exceed its charge). Support runs ad-hoc
   queries on it daily.
2. **Fraud feature store** — key → feature vector, read on every
   charge under the latency budget above, bulk-updated by the
   pipeline, staleness tolerated.
3. **Idempotency keys** — write-once records, exact-key lookup,
   24 h TTL. Remember what your api-design crash story required of
   them.

## Non-negotiable requirements

1. A charge is never executed twice — including across a database
   failover.
2. A reconciliation walk never silently skips or duplicates
   transactions (your cursor design depends on the indexes you choose
   here).
3. After support issues a refund, the dashboard list they are looking
   at shows it within ~1 second.
4. Regulators require 7 years of transaction history, queryable
   within 24 hours of a request; the last 90 days serve ~99% of all
   reads.

## The committed queries (from your API surface)

- Q1: fetch a charge by id.
- Q2: list a merchant's transactions, newest first, cursor-paginated
  on `(created_at, id)`.
- Q3: nightly reconciliation walk — Q2 repeated over a large slice of
  a merchant's history.
- Q4: support lookup of a charge by the merchant's own
  `external_ref` string.
- Q5: a merchant's *declined* charges in the last 7 days, newest
  first (dashboard widget).

## Explicitly out of scope

Caching layers, message queues, and multi-region deployment. One
region, one datacenter. Your job is the data.
