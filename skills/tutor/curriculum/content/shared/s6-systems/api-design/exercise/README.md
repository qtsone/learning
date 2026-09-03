# Exercise — design the Ledgerly API

You are the API designer. Nothing here compiles; the deliverable is a
defensible design, and the verification is a design review with your
tutor.

Work in this order:

1. Read [`brief.md`](brief.md) — the product brief. Every number in it
   exists to be used.
2. Fill [`api-sketch.md`](api-sketch.md) section by section, top to
   bottom — estimates first, exactly as in the design-intro lesson.
   Where the sheet asks for an assumption or a trade-off, that line is
   graded, not decoration.
3. When every section is full, tell your tutor you're ready for review.

Rules of the game:

- **Numbers over adjectives.** "High traffic" is not an estimate;
  "~230 QPS at peak, because …" is. Order-of-magnitude is fine —
  stating the assumption is the point.
- **Every choice names its loser.** A decision without a rejected
  alternative and its cost will be challenged first in review.
- Prose is fine, bullets are fine. Copy-pasted lesson text is not — the
  review is conducted in your own words.

Expect the review to push back: your tutor will inject new requirements
("a merchant now wants X — what breaks?") and attack your idempotency
scheme with crash scenarios. That is the exercise.
