# Product brief — Ledgerly

Ledgerly is a payment platform. Merchants integrate it to charge their
customers, refund them, and reconcile their books. You are designing its
API surface — both of them.

## The two audiences

**Merchant surface (public).** ~2,000 merchant businesses integrate
against it from their own backends — every language and framework you can
imagine, mostly maintained by small teams. A browser-based merchant
dashboard (built by Ledgerly, but calling the same public API) shows
recent transactions and lets support staff issue refunds. Merchants also
run nightly reconciliation jobs that walk their full transaction history.

**Internal surface.** Ledgerly's own services, all written in Go, on one
network:

- A **fraud-scoring** service, called synchronously on *every* charge.
- A **settlement** service that reads ledger entries in bulk each night.

## The numbers

- 5,000,000 charges per day platform-wide; traffic peaks at roughly 4×
  the daily average during flash sales.
- Reads dominate: dashboards and reconciliation produce about 10 list or
  get requests for every charge created.
- The largest merchant has ~20,000,000 historical transactions and walks
  a large slice of them nightly.
- A transaction record is on the order of 1 KB.
- Charge creation has an end-to-end p99 budget of 500 ms, of which fraud
  scoring may spend at most 150 ms.

## Non-negotiable requirements

1. A charge is never executed twice, no matter how many times a merchant
   client retries — merchants' networks are as unreliable as anyone's.
2. A reconciliation walk must not silently skip or duplicate
   transactions, even while new charges are landing.
3. The public API will evolve for years without breaking existing
   merchant integrations; nobody wants to coordinate 2,000 upgrades.
4. Operations to support at minimum: create a charge, fetch a charge,
   list transactions, refund a charge, and whatever the internal
   services need.

## Explicitly out of scope

Authentication mechanics, currency conversion, and how the money
actually moves. Assume those exist. Your job is the contract.
