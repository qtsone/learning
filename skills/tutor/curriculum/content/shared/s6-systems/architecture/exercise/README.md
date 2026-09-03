# Exercise — the Larder architecture review

You are the architect Larder brought in. Nothing here compiles; the
deliverable is a defensible architecture proposal, and the verification
is a design review with your tutor.

Work in this order:

1. Read [`brief.md`](brief.md) — the codebase description and the
   numbers. Every number in it exists to be used; several exist to be
   found *unimpressive*, which is also a finding.
2. Fill [`architecture-review.md`](architecture-review.md) section by
   section, top to bottom — estimates first, exactly as in every design
   lesson this stage. Lines marked *assumption*, *cost*, or *evidence*
   are graded content, not decoration.
3. When every section is full, tell your tutor you're ready for review.

Rules of the game:

- **Numbers over adjectives.** "The monolith doesn't scale" is not a
  finding; "search runs at ~100× order traffic and is the only genuine
  capacity axis" is. Order-of-magnitude is fine — stating the
  assumption is the point.
- **Evidence over vibes.** Your diagnosis cites the brief: who touches
  whose tables, where transactions commit, what fails together.
- **Every choice names its loser.** A decision without a rejected
  alternative and its cost gets challenged first in review.
- Prose is fine, bullets are fine. Copy-pasted lesson text is not — the
  review is conducted in your own words.

Expect the review to push back: your tutor will replay the brief's
email-provider outage against your design, crash your outbox relay,
double your headcount, and ask what breaks and what flips. That is the
exercise.
