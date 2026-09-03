# Worksheet 2 — Alert triage and design

Two parts: clean up the alert inventory you inherited, then design the
pages your worksheet-1 SLOs actually deserve.

## Part A — triage the inherited alerts

For each existing alert: **page** (wakes a human, now), **ticket** (queue
for working hours), or **delete**. One line of reasoning each; classify by
the symptom-vs-cause test, not by how scary the alert sounds. At least one
of your reasons should cite Tuesday's incident as evidence.

| # | Alert | page / ticket / delete | Why |
|---|-------|------------------------|-----|
| A1 | db-replica CPU > 90% for 5 min | … | … |
| A2 | feed error-budget burn rate > 14.4× over 1 h | … | … |
| A3 | any pod restarted | … | … |
| A4 | DB primary disk > 80% full | … | … |
| A5 | upload success ratio < 99% over 10 min | … | … |
| A6 | feed cache hit ratio < 70% | … | … |
| A7 | push-provider 5xx > 1% | … | … |
| A8 | TLS certificate expires in 14 days | … | … |

## Part B — design the pages

Define **two** page alerts guarding your own SLOs (S1 and S3). For each:
the burn-rate condition (long window + short confirmation window), and the
arithmetic showing what it catches.

### Page 1 — guards S…

**Condition:**

```text
burn rate > …  over  … h    AND    burn rate > …  over the last … min
```

**Arithmetic** — show both:

```text
budget spent if it fires at the threshold: … × … = … % of the 30-day budget
time for a 100% outage to page:  100% / S1 budget = burn …×  → fires in ≈ …
time for Tuesday (12% errors) to page: 12% / … = burn …×     → fires in ≈ …
```

### Page 2 — guards S…

**Condition:**

```text
…
```

**Arithmetic:**

```text
…
```

## Part C — the leftovers

Where do the causes you demoted (cache hit ratio, replica CPU, …) live
now, so they still help during an incident? One or two sentences.

> …
