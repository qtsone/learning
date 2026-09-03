# ADR-001 — Start with an in-memory store behind an interface

- **Status:** accepted
- **Date:** 2026-03-01

## Context

The core build needs storage from milestone M3, but the access patterns are
still moving: listing order and tag filtering changed twice during M2. Picking a
database now would freeze a schema around a design that is not settled, and the
capstone is graded on the core being finished, not on the storage being
durable.

## Decision

Store notes in memory behind a `Store` interface declared where it is consumed.
`Memory` guards its map with a `sync.RWMutex` so concurrent callers are safe.

## Consequences

- The core is finishable inside the milestone budget; no schema, no migrations.
- Every listing is O(n); acceptable at the sizes this project targets.
- Data does not survive a restart. That is a stated non-goal, not an oversight.
- Swapping in a durable store later is a new package plus one line in `main`,
  because nothing outside `store` names the concrete type.

## Revisited

M4 review: held. The interface stayed at three methods, which is evidence the
boundary was drawn in the right place.
