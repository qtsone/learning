# Worksheet 3 — Failure-mode audit

## Part A — the catalogue, applied

For each failure mode, invent a **concrete Cartwheel scenario** (name
the services/regions involved), the defense you would design in, and
the cost or residual risk of that defense. Generic textbook answers
("use timeouts") score nothing without the Cartwheel specifics.

| Failure mode | Concrete Cartwheel scenario | Defense (specific) | Cost / residual risk |
|---|---|---|---|
| Partial failure (dependency slow or dead — indistinguishable) | … | … | … |
| Network partition (beyond worksheet 2: pick an *intra*-region split or a partial one) | … | … | … |
| Clock skew | … | … | … |
| Split brain | … | … | … |

**Clock-skew probe.** A teammate proposes: "when the two regions'
carts disagree after a partition, keep the item with the latest
timestamp." Explain precisely how this loses a buyer's cart item with
no error logged anywhere, and propose an ordering that does not involve
comparing wall clocks.

> …

## Part B — fallacies to bug reports

Pick **three** of the eight fallacies. For each, write a miniature
post-incident story for Cartwheel: symptom (what paged you or what the
ticket said), root cause (the fallacy, cashed out), and the fix. At
least one story must involve **retry-induced duplication** and name the
mechanism that should have prevented it.

### Fallacy 1: "…"

- **Symptom:** …
- **Root cause:** …
- **Fix:** …

### Fallacy 2: "…"

- **Symptom:** …
- **Root cause:** …
- **Fix:** …

### Fallacy 3: "…"

- **Symptom:** …
- **Root cause:** …
- **Fix:** …
