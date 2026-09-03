# Worksheet 2 — Back-of-the-envelope estimates

Numbers before boxes. Use the brief's confirmed facts (10M registered,
~3 MB originals, ~300 KB feed size) plus your own stated assumptions.
Round aggressively; show every step. A day is ~10⁵ seconds.

## Assumptions

Every estimate below must trace back to a row here. One-line justification
each — "typical for consumer apps", "matches the brief's launch-campaign
spike", your own reasoning. Add rows as needed.

| # | Assumption | Value | Justification |
|---|------------|-------|---------------|
| A1 | Daily active users (share of 10M registered) | … | … |
| A2 | Photos uploaded per active user per day | … | … |
| A3 | Feed photos viewed per active user per day | … | … |
| A4 | Peak-to-average traffic factor | … | … |
| A5 | … | … | … |

## Write path

```text
uploads/day  = … × …            = …
write QPS    = … / 10⁵          ≈ … /s average
peak         = … × A4           ≈ … /s
```

## Read path

```text
views/day    = … × …            = …
read QPS     = … / 10⁵          ≈ … /s average
peak         = … × A4           ≈ … /s
read:write ratio ≈ …
```

## Storage

Count what is stored per upload (original? resized copy? metadata?) —
list the pieces, then total.

```text
bytes per upload  = … + … + …   ≈ …
storage/day       = uploads/day × bytes/upload ≈ …
storage/year      ≈ …            (before any replication)
```

## Bandwidth

Feed serving dominates — use the resized size for views.

```text
egress/day   = views/day × …    ≈ …
egress rate  = … / 10⁵          ≈ … /s average, … /s peak
ingress/day  = uploads/day × …  ≈ …
```

## Sanity checks

Two checks, in prose. Compare a result against an anchor number from the
lesson (or your own S5 measurements): does the read QPS need one server or
a fleet? Is the storage figure laughable or expected for a photo product?

1. …
2. …

## The dominant axis

One sentence: which quantity (read QPS / write QPS / storage / bandwidth)
will dominate this system's design, and which line above proves it?

> …
