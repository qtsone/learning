# Exercise — three designs, three reviews

Nothing here compiles. You produce three written designs, and the
verification is a design interview in which your tutor plays a reviewer who
changes the constraints halfway through.

Work in this order:

1. Read [`brief.md`](brief.md) — Briefs A and B, with their numbers. Each
   brief twists the lesson's walkthrough deliberately; the twist is the
   exercise. Brief C is in [`brief-c.md`](brief-c.md); do not open it until
   your timer is running.
2. [`01-shortlink.md`](01-shortlink.md) — Trackly, a campaign-link service.
   Budget ~70 minutes.
3. [`02-chat.md`](02-chat.md) — Huddle, a support-desk chat. Budget ~70
   minutes.
4. [`03-speed-round.md`](03-speed-round.md) — a social feed, **30 minutes on
   a timer**, no scaffolding. Do this one last and do not overrun; running
   out of time with a phase missing is itself a finding.
5. Tell your tutor you are ready. Bring all three.

Rules of the game:

- **Phases in order, always.** Requirements, estimation, high-level design,
  deep dives, bottlenecks. If a deep dive changes a requirement, go back and
  amend worksheet section 1 rather than leaving it stale.
- **Every number carries its assumption.** Orders of magnitude are fine;
  bare numbers are not. Show the arithmetic, one line per step.
- **Every pick names its loser and its flip condition.** "Chose X over Y; Y
  wins when Z."
- **The twist must appear in your design.** If your Trackly answer would be
  identical without editable links, you designed the lesson's shortener, not
  Trackly.
- Your own words. Diagrams as indented text are fine — draw boxes and label
  the arrows with what flows over them.

Expect the review to push: your tutor will audit two of your calculations
live, argue the rejected side of a trade-off, replay a failure from the
brief against your design, and introduce at least one new constraint you did
not plan for. Rewriting part of a design in the room when a constraint lands
is a pass signal, not a failure.
