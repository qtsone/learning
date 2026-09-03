# Worksheet 2 — Partition drill

## The scenario

The transatlantic link fails for **90 seconds**. Both regions are
internally healthy; buyers on both continents are mid-session. Nothing
about this is hypothetical — inter-region links flap.

For each feature, make the CAP call **now**, in the design, so nobody
improvises it during an incident. Rules:

- **C or A** is per feature, not one answer for the whole system.
- **What the user sees** must be concrete UI-level behavior: "add to
  cart succeeds, banner says prices may be delayed" — never "degrades
  gracefully".
- Every **A** row must say what happens *after* the partition heals:
  what could have diverged, and how it reconciles (merge rule, repair
  job, apology email — name it).
- Every **C** row must say what is refused and for whom (one region?
  both?).

| Feature | C or A | What the user sees during the 90 s | After healing: divergence + reconciliation (A) / what was refused (C) |
|---|---|---|---|
| F1 Catalog browse | … | … | … |
| F2 Cart | … | … | … |
| F3 Ordinary checkout | … | … | … |
| F4 Limited-drop checkout (a drop is live!) | … | … | … |
| F5 "My orders" | … | … | … |
| F6 Seller dashboard | … | … | … |

## Duration sensitivity

Pick the one feature whose answer above is most fragile. At what
partition duration does your call flip (or does it?), and what changes?

> Feature: … · Flip point: … · Because: …

## The else-case (PACELC)

No partition today — just the permanent ~100 ms ocean. For the two
features below, choose how a write replicates to the other region and
show the latency arithmetic from the brief's RTTs.

| Feature | Sync or async cross-region replication | Write latency the buyer pays (show arithmetic) | What you gave up |
|---|---|---|---|
| F2 Cart add | … | … | … |
| F4 Drop stock decrement | … | … | … |

**Connect the dots:** does your else-case choice for each feature match
your partition choice for it (latency-leaning pairs naturally with A,
consistency-leaning with C)? If any pair mismatches, defend it.

> …
