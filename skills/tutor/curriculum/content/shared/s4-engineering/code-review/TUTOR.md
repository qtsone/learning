# Tutor notes — Code Review

## Where the learner is

Ninth lesson of S4, second discussion-verify lesson (clean-code was the
first, so the "quality is argued, not asserted" mode is familiar). They
arrive with the whole stage behind them — clean code, TDD, debugging, SQL,
security, HTTP clients, CI/CD — and the exercise deliberately cashes those
lessons in: the planted defects are all things an earlier S4 lesson taught
them to see. Verify is discussion: grade from `NOTES.md` and conversation.
They have almost certainly never written a review or replied to one; expect
competent defect-*spotting* and unpracticed severity, phrasing, and
push-back judgment — that judgment is the lesson.

## Reference review of PR #14 (for grading part 1)

The planted defects, with expected severity:

- **SQL injection — blocking, non-negotiable.** `fmt.Sprintf` builds the
  INSERT from raw user input; email is unconstrained (`'); DROP TABLE
  users;--` in an email passes `validEmail`). Resolution: parameterized
  `?` placeholders as in the original. A learner who labels this
  `suggestion` or misses it has failed the security lesson's rent check —
  probe hard.
- **Dropped `db.Exec(q)` error — blocking.** Failed inserts return `nil`;
  callers believe the account exists. Bonus catch (A-signal): Register's
  doc comment now *lies* — it still promises wrapped database errors, a
  stale contract per clean-code.
- **No tests for the new behavior — blocking.** `validEmail` and the new
  Register path are untested; the only test-file change is rename churn.
  Ties to TDD ("new behavior with no new tests is a claim without
  evidence").
- **`validEmail` is naive — question (or blocking with reasoning).**
  `strings.Contains(e, "@")` accepts `"@"`, `"a@"`, spaces. Best form is
  a genuine question about requirements ("what should we accept here? bare
  `@` currently passes"); accept blocking if argued via the missing spec +
  missing tests. This is the natural home for their required `question`.
- **`ValidUsername` → `IsValidUsername` rename — split-this-out.** Unrelated
  to email, breaks the exported API for any other importer, and doubles
  the diff with churn (`ErrInvalidUsername`'s doc referencing it had to
  change too). Expected: request it be pulled into its own PR (blocking on
  scope grounds, or a firmly-marked suggestion). Key signal: they see it
  as *scope*, not naming taste.
- **`// check the email` — nit.** Narration per clean-code. `e` as the
  param is *fine* (scope-length criterion) — a learner nitting `e` has
  regressed on clean-code; nudge.

Verdict: request changes, with a summary that leads with the blockers and
acknowledges the feature is worth having. Part 3 expectations: title and
description say nothing ("misc", "various"); split ≈ (1) rename-only PR
(mechanical, reviewable in a minute — most argue it first *if it's judged
worth the API break at all*), (2) email feature with validation + tests;
any coherent 2-3-way split defended is fine. Self-review catches: the
Sprintf query, the dropped error, missing email tests, the accidental
scope creep, the stale doc comment — any three.

## Reference replies for part 4 (the author's chair)

- **A (ErrNotFound sentinel) — concede.** Real caller-facing defect; the
  reviewer's reasoning (`errors.Is` support) is exactly S1/S3 error
  doctrine. "Done, added `ErrNotFound`" plus a test is the A-answer.
- **B (rename `t`, add narration comment) — push back, with citation.**
  Scope-length licenses `t` two lines from its declaration;
  `// decode the response body` is narration by the clean-code test.
  Grade the *grounding*: criterion or convention cited, tone civil, and
  ideally an offer ("happy to rename if team style differs").
- **C (retries) — agree but defer.** Legitimate idea, out of scope for a
  one-method PR; belongs in the client/transport layer anyway (their
  http-clients lesson). Expected: file a follow-up, keep the PR focused.
  Cramming retries in = missed the scope discipline they just demanded of
  jbx in part 3; point at the mirror.
- **D (Decoder vs ReadAll) — answer.** Streams instead of buffering the
  whole body, matches the io philosophy from S3; honest answers admit
  either is fine for small payloads. Any reasoned answer passes; "that's
  just how I do it" does not.

Concede rule should compress to something like: "concede on correctness,
convention, and consistency; push back only when I hold reasoning — not
preference — the reviewer lacks."

## Common misconceptions

- **"Review = bug hunt."** Their part 1 is all defects, no scope/design
  comment, and the summary ignores the PR packaging. Push: which comment
  spreads knowledge? Which enforces a convention?
- **Severity inversion.** The rename tagged blocking-as-naming while the
  injection sits at `suggestion`. Worse than missing a defect — it
  misleads. Make them rank their own comments by cost-if-ignored.
- **"Kind = soft."** Hostile rewrites that downgrade blocking to
  "maybe consider…". Tone is the delivery; severity is the content; the
  rewrite must keep the block. Comment 2's rewrite must also drop the
  "you always" generalization entirely, not soften it to "you sometimes".
- **"Every comment must be obeyed."** Part 4 concedes all four — including
  B, adding narration comments they'd have deleted in clean-code. The
  push-back is not optional; it's an objective.
- **"Push back = defend everything."** The mirror failure: contesting A
  with vibes. Concede/push-back is decided by who holds the reasoning.
- **"CI was green, so the code is fine."** The optional `patch` step
  exists to break this: the injection passes `go vet` and `go test`.
  Ask directly what green proved and didn't.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go
deeper):

- "Your team's tests are excellent and CI is strict. What does review
  still buy? Rank the returns for *this* PR." (Knowledge, consistency,
  design; the injection proof that green ≠ good.)
- "Order your part 1 comments by cost-if-ignored. Does your labeling
  match that order?" (Severity calibration, live.)
- "Take your rewrite of hostile comment 1. What did you keep from the
  original and what did you delete? Why is 'do you even read your own
  code' unanswerable?" (Identity vs code; specific/actionable/why.)
- "jbx says: 'the rename is one keystroke to review, why a separate PR?'
  Convince them." (Churn buries signal; API break needs its own decision;
  reviewer attention economics.)
- "You pushed back on B. The reviewer replies 'still prefer todoResult'.
  Next move?" (Two written rounds → talk; consistency beats preference;
  who decides and why.)
- "When is 'agree, follow-up issue' the *wrong* answer to a review
  comment?" (When the comment is a blocker on this change — deferral is
  for out-of-scope improvements, not defects.)

## Grading rubric

- **A** — All three blockers found and labeled blocking, injection
  resolution correct (placeholders); question is genuine; rename treated
  as scope; nit calibrated; rewrites keep severity and gain specificity;
  part 3 split defensible with real title/description; part 4 has the
  concession, a cited push-back, the deferral, a reasoned answer, and a
  crisp concede rule; defends every call fluently, and bonus catches
  (stale doc comment, `e` is fine) appear.
- **B** — Blockers found but a severity wobble or two (e.g. missing-tests
  as suggestion), one rewrite that softens severity or stays vague, or a
  part 4 reply labeled right but reasoned thin. Corrects cleanly when
  challenged in discussion.
- **C** — Defects spotted but review craft absent: no labels or all-blocking,
  rewrites merely polite rather than actionable, part 4 all-concede or
  all-defend. Pass only if live remediation lands — make them relabel and
  re-reply on the spot; otherwise another iteration on `NOTES.md`.
- **Fail** — Injection missed or non-blocking, hostile rewrites that stay
  hostile ("politely, do you even read your code"), or they cannot say
  why "you always ignore errors" fails as a comment even when prompted.
  Redo the relevant part together before re-discussing.

## Remediation ladder

1. "Apply the diff and run the tests — green. Now register the username
   `x'); DROP TABLE users;--`@evil as an email in your head. What happens?
   Which of your labels does that change?"
2. "For each comment you wrote, ask: what happens if the author ignores
   it? If the answer is 'production incident', it's blocking; if 'mild
   regret', it isn't. Relabel."
3. "Rewrite test: does your comment name the line, the consequence, and
   an exit? Run it on hostile comment 3, then on your own weakest
   comment."
4. "For part 4: for each comment, who holds more reasoning — you or the
   reviewer? Correctness → concede. Convention → whoever cites it.
   Preference → existing style wins. Apply the ladder out loud, comment
   by comment."

## After passing

Preview: "You've now argued about code in writing. The last lesson of the
stage is the other written channel — READMEs, doc comments, ADRs — the
documents that answer questions before anyone has to ask them in review."
