# Brief — Larder

Larder is a seven-year-old online marketplace connecting independent
food producers with buyers. The product works, the company is past
product-market fit, and engineering is at an inflection point: everyone
agrees "the architecture" is the problem, and nobody agrees on what to
do. You were brought in to decide.

## The system today

- A single application, deployed as **one unit**, backed by **one
  relational database** with one shared schema.
- Folder structure suggests modules — `checkout`, `catalog`, `users`,
  `fulfillment`, `growth` — but the boundaries are not enforced:
  checkout queries catalog's tables directly, `growth` runs analytics
  SQL against production, and cross-module joins are routine. Roughly
  a third of database transactions write tables belonging to more than
  one "module".
- All communication between modules is synchronous, in-process calls.

## The people

26 engineers in 4 teams, each owning a slice of the codebase:

| Team | Owns | Headcount |
|---|---|---|
| Checkout & Payments | checkout, payment-provider integration | 7 |
| Catalog & Search | product catalog, search and browse | 7 |
| Fulfillment | seller tools, orders-to-shipment | 6 |
| Growth | notifications, email, analytics | 6 |

The company expects ~3× order growth and ~40 engineers within 18
months. There is no appetite for a feature freeze.

## The numbers

- **Orders:** 500,000 per day; peaks at roughly 3× the daily average.
- **Search and browse:** 60,000,000 requests per day, same peak factor.
  The Catalog & Search team wants a dedicated search datastore (inverted
  index) instead of SQL text queries, and wants to scale reads
  independently.
- **Order placement** is one synchronous request that performs, in
  order (p99 contribution per step, from tracing):

  | Step | p99 |
  |---|---|
  | Reserve inventory + record order (one DB transaction) | 250 ms |
  | Charge via payment provider | 350 ms |
  | Send confirmation email (external provider) | 1,200 ms |
  | Notify seller (push + email) | 300 ms |
  | Update search index | 400 ms |
  | Write analytics events | 150 ms |

  Checkout p99 lands around 2.7 s. The business target is **≤ 800 ms**.

- **Deploys:** the whole application ships on a shared release train,
  3 times a week, with a median of **35 merged changes per release**.
  About 1 release in 12 is rolled back — reverting everyone's work at
  once. Each team wants to deploy independently.
- **Tests:** the full suite takes 45 minutes, and teams routinely break
  each other's tests.
- **Last quarter's worst incident:** the email provider degraded for
  90 minutes; because confirmation email is an inline checkout step
  (with retries), checkout error rate hit 40%. The postmortem's only
  action item was "decouple this", unassigned.

## Stated tolerances (from the business, non-negotiable)

1. Checkout p99 ≤ 800 ms.
2. An order is never lost and never charged twice, regardless of
   retries anywhere.
3. Confirmation email within 2 minutes of the order.
4. Search index staleness up to 60 seconds is acceptable.
5. Analytics may lag by a day.
6. Migration must be incremental — the product keeps shipping
   throughout.

## Explicitly out of scope

Infrastructure choices (cloud, orchestrator, broker vendor, search
vendor), authentication, and the payment provider's internals. Assume a
production-grade message broker with at-least-once delivery is
available the day you need it. Your job is the architecture and the
path to it.
