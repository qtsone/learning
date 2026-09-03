# Scenario brief — Cartwheel goes multi-region

Cartwheel is an e-commerce platform: buyers browse a catalog, fill carts,
check out; sellers manage stock through a dashboard. Today it runs in one
US datacenter on a stack you would recognize from S5: API services, a
leader-follower relational database, a cache layer, a message queue for
order processing.

The business is launching in Europe, and the board has signed off on a
second region. You are writing the distributed-systems section of the
design.

## The numbers

- 5M daily active buyers today, roughly 60% US / 40% EU after launch.
- Catalog reads dominate: ~50 page views per buyer per day. Checkouts are
  rare by comparison: ~2% of daily actives buy on a given day.
- Round trips: ~0.5 ms within a region, **~100 ms between the regions**.
- Twice a month Cartwheel runs a **limited drop**: a hyped product with
  fixed stock (say 5,000 units) that sells out in minutes, with buyers on
  both continents hammering the same inventory counter.

## What the business asked for

1. EU buyers get in-region page loads — the current transatlantic
   ~300 ms p95 is cited as the reason for poor EU conversion.
2. If a whole region goes down, the other keeps selling. "Down for
   maintenance because Virginia is on fire" is unacceptable.
3. Limited drops must **never oversell**. Refunding 300 buyers of a
   5,000-unit drop was a PR fire last year; legal wants zero.
4. For ordinary catalog items, modest overselling is fine — the warehouse
   holds buffer stock, and support can apologize. (Their words: "don't
   make ordinary items as hard as drops.")
5. A buyer who checks out must immediately see the order under "my
   orders" — the #1 support ticket is "I paid and my order vanished".

## The features on the table

| ID | Feature |
|----|---------|
| F1 | Catalog browse (product pages, search, prices) |
| F2 | Cart: add/remove items, view cart — from any of the buyer's devices |
| F3 | Checkout for **ordinary** items (payment + inventory decrement) |
| F4 | Checkout for **limited drops** (fixed stock, global contention) |
| F5 | "My orders" history for buyers |
| F6 | Seller dashboard: stock levels and sales totals |

## Explicitly out of scope

Payment processing internals, fraud, search ranking, CDN/static assets.
Assume the api-design lesson's idempotency-key machinery exists on every
mutating endpoint. Your job is the data: where it lives, how it
replicates, what readers may observe, and what happens when the ocean
link fails.
