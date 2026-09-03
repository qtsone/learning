# Worksheet 1 — SLIs, SLOs, and error budgets

Source: [`brief.md`](brief.md), journeys J1 (load the feed) and J2 (post a
photo). Numbers before promises: every target must be defensible against
the measured baselines.

## SLIs

One availability SLI per journey, plus one latency SLI for J1 (the journey
users feel). Write each as a **good/valid ratio**, name the measurement
point, and note what that point is blind to.

| # | Journey | SLI (good events / valid events) | Measured where | Blind to |
|---|---------|----------------------------------|----------------|----------|
| I1 | J1 availability | … / … | … | … |
| I2 | J1 latency | requests faster than … ms / … | … | … |
| I3 | J2 availability | … / … | … | … |

## SLOs

A target and a window for each SLI. Justify the target from the brief's
baselines and the user's actual tolerance — and for I1, explicitly argue
why not one nine *higher* and why not one nine *lower*.

| # | SLI | Target | Window | Why this target (cite baseline numbers) |
|---|-----|--------|--------|------------------------------------------|
| S1 | I1 | … % | rolling … days | … |
| S2 | I2 | … % | … | … |
| S3 | I3 | … % | … | … |

**Why not one nine higher / lower for S1:**

> …

## Error budgets

For S1 and S3, compute the budget both ways. Show every step; a 30-day
window is 43,200 minutes, a day is ~10⁵ seconds.

```text
S1 budget fraction   = 1 − …            = … %
S1 as outage minutes = … × 43,200       ≈ … min / 30 days
S1 requests/30 days  = 1,000/s × …      ≈ …
S1 as failed requests= … × …            ≈ …

S3 budget fraction   = …
S3 as outage minutes = …
S3 as failed requests= …  (uploads run at ~20/s)
```

## Sanity check

One prose paragraph: your S1 target, the measured 99.95% baseline, and
last Tuesday's incident — are all three consistent? (If your SLO would have
been comfortably met during a 50-minute 12%-error incident, or could never
be met by the system as measured, something is mis-set.)

> …
