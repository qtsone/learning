# Exercise — design Ledgerly's storage layer

You are the storage designer. Nothing here compiles; the deliverable
is a defensible design, and the verification is a design review with
your tutor.

Work in this order:

1. Read [`brief.md`](brief.md) — the numbers and the non-negotiables.
   Every number in it exists to be used.
2. Fill [`storage-design.md`](storage-design.md) section by section,
   top to bottom — estimates first, exactly as in design-intro.
   Section 2 contains two real query plans; diagnosing them is graded
   content, not decoration.
3. When every section is full, tell your tutor you're ready for
   review.

Rules of the game:

- **Numbers over adjectives.** "The table gets big" is not an
  estimate; "~7 TB/year, so a 2-4 TB comfort limit is crossed in
  month N, because …" is. Order-of-magnitude is fine — stating the
  assumption is the point.
- **Every choice names its loser.** A store choice without a rejected
  alternative and its cost will be challenged first in review.
- **Guarantees chain downward.** The api-design lesson promised
  merchants a cursor that never skips and a charge that never
  executes twice. Wherever a storage decision could break one of
  those promises, say so explicitly — finding those places is most
  of this exercise.
- Prose is fine, bullets are fine. Copy-pasted lesson text is not —
  the review is conducted in your own words.

Expect the review to push back: your tutor will kill your leader
mid-flash-sale, lag your replicas, and hand your biggest merchant a
flash sale. That is the exercise.
