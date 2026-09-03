# Exercise — write a design doc and defend it

This is the stage capstone. Nothing compiles; the deliverable is a
document a reviewer can attack, and the verification is a 60-90 minute
design review in which they will.

Work in this order:

1. Read [`brief.md`](brief.md) and choose your system: the default
   (**Codraft**, a collaborative document service), one of the three
   alternates, or your own if it clears the bar. If you bring your own,
   clear it with your tutor **before** you start writing — a brief that
   cannot be estimated cannot be reviewed.
2. Fill [`01-design-doc.md`](01-design-doc.md) top to bottom. Requirements
   before estimates, estimates before architecture — the order is the
   method, and skipping ahead is the failure mode this stage has been
   drilling since design-intro.
3. Fill [`02-decisions.md`](02-decisions.md) as you make the calls, not
   afterwards. A record written after the fact is a justification, not a
   decision.
4. Fill [`03-failure-and-scale.md`](03-failure-and-scale.md) by walking
   *your own* diagram — every box and every arrow on it, including the
   boring ones.
5. Fill [`04-review-prep.md`](04-review-prep.md) last. Attack your design
   before your tutor does; the gap between what you find and what they
   find is the most useful signal in this lesson.

Rules of the game:

- **Arithmetic is visible.** `400k × 3 × 11 min / 1440 ≈ 9k concurrent`
  earns credit; a bare `9k` does not.
- **Every assumed number gets a one-line justification.** No silent
  numbers anywhere in the document.
- **Every box names the requirement that pays for it.** Expect to be
  asked, box by box, and expect a component with no answer to be deleted
  live.
- **Every decision names its loser and its bill.** Alternatives with
  costs, and a consequences line that says what you now live with.
- **Round aggressively.** Orders of magnitude decide architectures;
  three significant figures decide nothing.
- Where the brief is silent, state an assumption and continue. Never
  stall, and never invent a fact and present it as given.
- Write in your own words. Copy-pasted lesson text will be discovered in
  the first five minutes of the review.

Budget roughly 3-4 hours for the document. When it is complete, tell your
tutor you are ready. In the review they will re-derive your numbers with
you, argue for the alternatives you rejected, take your dependencies down
one at a time, push you to 10× — and introduce **new constraints partway
through**. Your job is not to be unmoved by them; it is to say precisely
what moves, and why.
