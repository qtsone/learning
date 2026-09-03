# Worksheet 1 — The consistency menu

For each feature in [`brief.md`](brief.md), choose the **weakest**
consistency model (or session guarantee) that still satisfies the
business asks. The menu, strongest first: linearizable · sequential ·
causal · session guarantees (read-your-writes, monotonic reads) ·
eventual.

Rules:

- The **Why this suffices** column must cite the brief (F-IDs, business
  asks 1-5, or a number).
- The **One step weaker** column must describe a *concrete user-visible
  anomaly* — a sentence a support ticket could contain, not "data might
  be inconsistent".
- Overshooting is a graded defect: every step above eventual costs
  latency or availability, so "linearizable, to be safe" with no costed
  reason gets sent back.

| Feature | Weakest sufficient model | Why this suffices (cite the brief) | One step weaker would mean… |
|---|---|---|---|
| F1 Catalog browse | … | … | … |
| F2 Cart (multi-device) | … | … | … |
| F3 Ordinary checkout: inventory decrement | … | … | … |
| F4 Limited-drop checkout: stock counter | … | … | … |
| F5 "My orders" after checkout | … | … | … |
| F6 Seller dashboard totals | … | … | … |

## Two probes

**P1.** For F5 you almost certainly did not need a *global* guarantee.
Whose reads need the promise, and what is the cheapest scope that
delivers it?

> …

**P2.** Name the one feature where you would pay for linearizability,
and state — in one sentence each — the two prices you pay for it during
normal operation and during a partition.

> …
