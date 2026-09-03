# The brief

Framepost has been in production for six months. You are taking over
operations. Everything below is real data from the running system — use it;
do not re-derive it.

## Production facts

Traffic (30-day averages unless noted):

- ~10M registered users, ~2M daily actives, one primary region.
- Feed loads: **~1,000 requests/s average, ~3,000/s peak** (evenings).
- Photo uploads: **~20/s average, ~50/s peak**.
- Feed latency today: p50 ~120 ms, p99 ~350 ms.
- Measured feed success ratio last quarter: **99.95%** (all failures
  counted at the load balancer).
- Metrics exist at the load balancer, in every service, and nowhere else —
  there is no client-side (in-app) telemetry yet.

Architecture, as built (all pieces you know from this stage):

- Clients → load balancer → **feed service** and **upload service**.
- **Relational database**: one primary (all writes), two read replicas.
  Automated failover promotes a replica in **60-90 s**; writes fail during
  promotion, reads keep working.
- **Feed cache** in front of the replicas: ~92% hit ratio on feed reads.
  The replicas alone sustain roughly **a third** of peak feed read load.
- **Resize workers** consume an internal queue (at-least-once, from the
  message-queues lesson): originals (~3 MB) arrive in object storage, a
  ~300 KB feed rendition is written back. Typical lag: seconds.
- **Push provider** (third-party): notifies followers of new photos.
  Currently called **synchronously** by the upload service after a
  successful upload, with a 10 s timeout. Its historical uptime: ~99.5%.

Support can post to a public status page; on-call rotates weekly.

## The user journeys that matter

The product team considers these the two journeys worth guarding:

- **J1 — Load the feed**: a follower opens the app and sees photos.
- **J2 — Post a photo**: a user uploads; the app says "posted" only after
  the original is durably stored and the metadata row exists.

## Raw incident log — last Tuesday

Assembled afterwards from chat scrollback and dashboards, verbatim quality:

```text
14:02  deploy feed-service v417 (release note: "new feed cache key format")
14:04  feed cache hit ratio drops 92% -> 11% (seen on dashboard AFTER the
       incident; nobody was looking at the time)
14:04  feed error rate starts climbing; settles around 12% (timeouts),
       feed p99 ~9 s
14:06  alert "db-replica-2 CPU > 90% for 5 min" fires into #alerts.
       It fires most nights during the batch import; on-call has it muted.
14:21  support escalates in #eng: burst of "feed won't load" tickets
14:23  on-call (Dana) opens dashboards, sees 12% feed errors, p99 9 s
14:31  Dana restarts db-replica-2 (acting on the CPU alert). Error rate
       briefly rises to ~17% (one fewer replica while it restarts).
14:41  Priya spots the 14:02 deploy in #deploys, suggests rollback
14:44  rollback of v417 starts
14:49  rollback complete
14:53  feed error rate back under 0.1%; p99 back to ~350 ms as the cache
       re-warms
15:40  engineering manager in #eng: "Who approved the v417 deploy? This
       can't happen again."
```

The status page was never updated; support answered tickets with "we are
not aware of an issue" throughout. Uploads (J2) were unaffected.
