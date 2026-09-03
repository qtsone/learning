# Milestones — <project name>

> Copy to `projects/capstone/MILESTONES.md` (project **root**, not `docs/`).
> Later lessons in this stage read this file mechanically: they require the
> `- [ ]` task-list syntax and, at the end of the core build, zero unticked
> boxes. Keep it current as you work — it is the progress record, not a plan
> you wrote once.
>
> Delete this quoted block in your copy.

**Total estimate:** … h **Hours available:** … h **Started:** YYYY-MM-DD

## Definition of done — applies to every milestone

A milestone is done when all of the following hold. This list is not repeated
per milestone; it is assumed by all of them.

- Its acceptance criteria are ticked and demonstrable — you can show them
  working from a clean checkout.
- `go build ./...`, `go vet ./...` and `go test -race ./...` are clean, and
  the program still runs. The race detector is the bar from the next lesson
  on, so tick boxes against it now rather than meeting it at the end.
- New behavior has tests. Behavior you changed has updated tests.
- `README.md` still tells the truth about how to run it.
- The work is committed, with the milestone named in the message.

---

## M0 — walking skeleton  ·  est. …h

The thinnest path that touches every layer end to end: real entry point, real
core, real persisted output, one test. Deliberately stupid. If it takes more
than a few hours, it is not a skeleton.

- [ ] …
- [ ] …
- [ ] one end-to-end test covers the whole path

## M1 — <name>  ·  est. …h

*Riskiest thing first: this should be the milestone most likely to invalidate
your design, while changing the design is still cheap.*

**Delivers:** … (cite the requirement IDs: F1, F3 …)

- [ ] …
- [ ] …
- [ ] …

## M2 — <name>  ·  est. …h

**Delivers:** …

- [ ] …
- [ ] …

## M3 — <name>  ·  est. …h

**Delivers:** …

- [ ] …
- [ ] …

## M4 — <name>  ·  est. …h

*The first candidate for the axe. If this milestone disappearing would leave
you with something you would not want to demo, your ordering is wrong.*

**Delivers:** …

- [ ] …
- [ ] …

---

## Cut

Milestones you decided not to build, with the reason and the date. Plain
bullets — **no checkboxes here**, so the mechanical check does not see an
unticked box for work you deliberately dropped.

- M5 <name> — cut YYYY-MM-DD: … (which non-goal or requirement made it
  optional)

## Log

One line per milestone as you finish it: date, actual hours against the
estimate, and anything the milestone taught you that changed the plan. The next
lesson's reviews start from this log.

| Milestone | Done | Est. h | Actual h | What it changed |
|---|---|---|---|---|
| M0 | | | | |
