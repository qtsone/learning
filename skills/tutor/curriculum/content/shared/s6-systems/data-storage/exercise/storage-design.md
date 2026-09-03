# Storage design — Ledgerly

Fill every section. Short and reasoned beats long and hedged. Lines
marked *assumption* or *cost* are graded content, not padding.

## 0 — Estimates first

Back-of-the-envelope, from the brief's numbers. Show the arithmetic;
round aggressively.

- Average write QPS (charges): <!-- number + one line of arithmetic -->
- Peak write QPS: <!-- number + assumption -->
- Peak read QPS (all read traffic): <!-- number + assumption -->
- Fraud lookup QPS, peak: <!-- number + assumption -->
- Storage growth per year, table + indexes: <!-- number + assumption -->
- Months until the current node leaves its 2-4 TB comfort zone:
  <!-- arithmetic from current size + growth -->
- The dominant axis (write QPS / read QPS / data size — which one
  actually forces action, per the anchors?): <!-- one sentence of evidence -->

## 1 — Store choice per dataset

For each dataset: the store *kind* (relational / key-value / document
/ wide-column — name a concrete example if you like), argued by the
three drivers, with the loser named.

### Transactions ledger

- Choice:
- Driver — consistency:
- Driver — query flexibility:
- Driver — scaling:
- Rejected alternative and what rejecting it costs us:

### Fraud feature store

- Choice:
- The driver that actually decided it here:
- What the 10 ms p99 budget and the 5-minute staleness tolerance
  each rule in or out:
- Rejected alternative and its cost:

### Idempotency keys

- Choice:
- The workload *looks* like TTL key-value. What requirement decides
  the store anyway, and how (walk the api-design crash story against
  your choice):
- How TTL cleanup works in your chosen store:
- Rejected alternative and its cost:

## 2 — Index plan and query-plan drills

### Index plan

For the five committed queries (brief): which indexes exist on
`transactions`, and which query each serves. Then the bill.

| Index (columns, in order) | Serves | Why this column order |
|---|---|---|
|  |  |  |
|  |  |  |
|  |  |  |

- Write-side cost at your peak write QPS (how many index writes per
  charge insert, and one sentence on whether that is acceptable):
- One index you deliberately did NOT create, and why:

### Drill A — diagnose this plan

Q2 (merchant list page) is timing out. The plan:

```text
EXPLAIN ANALYZE
SELECT * FROM transactions
WHERE merchant_id = 'm_1042'
ORDER BY created_at DESC, id DESC
LIMIT 100;

Limit  (cost=304901925.10..304901925.35 rows=100 width=1024)
       (actual time=603245.03..603245.08 rows=100 loops=1)
  -> Sort  (cost=304901925.10..304980885.36 rows=31584102 width=1024)
           (actual time=603245.02..603245.04 rows=100 loops=1)
        Sort Key: created_at DESC, id DESC
        Sort Method: top-N heapsort  Memory: 214kB
        -> Seq Scan on transactions
               (cost=0.00..303694412.00 rows=31584102 width=1024)
               (actual time=3.19..598214.40 rows=31584102 loops=1)
              Filter: (merchant_id = 'm_1042'::text)
              Rows Removed by Filter: 8968415898
Execution Time: 603245.221 ms
```

- What is the executor doing, in one sentence:
- The two lines that prove it (quote them):
- The fix (exact index, columns in order):
- What the plan should look like after the fix (scan type, and what
  happened to the Sort node — why):

### Drill B — diagnose this plan

Q5 (declined charges, last 7 days). The index from Drill A exists and
is being used — yet the widget takes 1.4 s. The plan:

```text
EXPLAIN ANALYZE
SELECT * FROM transactions
WHERE merchant_id = 'm_1042'
  AND status = 'declined'
  AND created_at > now() - interval '7 days'
ORDER BY created_at DESC
LIMIT 50;

Limit  (actual time=1408.98..1409.01 rows=50 loops=1)
  -> Index Scan Backward using
         transactions_merchant_created_id_idx on transactions
         (cost=0.71..91520.44 rows=812 width=1024)
         (actual time=2.77..1408.95 rows=50 loops=1)
        Index Cond: ((merchant_id = 'm_1042'::text) AND
                     (created_at > (now() - '7 days'::interval)))
        Filter: (status = 'declined'::text)
        Rows Removed by Filter: 183917
Execution Time: 1409.113 ms
```

- An index IS used — so why is it still slow (one sentence, citing a
  line from the plan):
- Two different fixes, and the cost of each:
- Which fix you'd ship, and why:

## 3 — Replication design

- Topology (how many nodes, who takes writes, who serves what reads):
- Sync / semi-sync / async — and what an "OK" to a charge-creating
  merchant means under your choice:
- Expected replication lag, normal and flash-sale (*assumption*):
- **Requirement 3, the refund story**: support issues a refund and
  the dashboard must show it within ~1 s. Walk the failure with a
  lagging replica, then your fix — name the mechanism:
- Which reads go to the leader, which to replicas (list the committed
  queries by name):
- **Requirement 1, the failover story**: the leader dies
  mid-flash-sale with N ms of writes not yet replicated. Walk what
  happens to (a) those charges, (b) their idempotency keys, (c) a
  merchant's retry of one of them. Does your design still guarantee
  "never executed twice"? If yes, what does that cost; if no, what do
  you change:

## 4 — Sharding plan

- The trigger: from section 0, *when* and *why* must you split (or
  archive) — which anchor is exceeded first:
- Does requirement 4 (7-year retention, 99% of reads in the last 90
  days) change the answer? What do you do with cold data, and how
  does that move the sharding date:
- Sharding key, and why (name what you rejected):
- Which committed queries stay single-shard under your key, and
  which become scatter-gather:
- **The whale**: one merchant is 30% of writes for an hour. What
  happens on your scheme, and your mitigation with its cost:
- Cross-shard hazard: name one invariant or transaction that your
  key choice keeps single-shard, and one operation that no longer
  has cross-shard atomicity — what do you do about it:
- Resharding strategy: how data moves when you add nodes, and why
  `hash mod N` is not your answer:
- What happens to the reconciliation cursor walk (requirement 2)
  under your sharding scheme:

## 5 — Trade-off log

At least three, in the form "Chose X over Y; it costs us Z." These
must be real losses, not straw men.

1.
2.
3.
