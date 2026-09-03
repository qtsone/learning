# Code Review

> `shared.eng.code-review` · ~1-2h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Explain what code review is for beyond bug-catching: knowledge sharing,
  consistency, and design feedback.
- Review a provided diff and produce comments that distinguish blocking
  defects from optional suggestions.
- Phrase critical feedback about the code, not the author, and rewrite a
  hostile review comment constructively.
- Prepare a reviewable PR: a small focused diff, a descriptive title and
  description, and a self-review pass.
- Respond to review feedback on your own code, deciding when to push back
  with reasoning and when to concede.

## What review is actually for

Code review is the practice of a second person reading a change before it
merges. The obvious justification — catching bugs — is real but it is the
*weakest* of the returns. Your tests, your linter, and the pipeline you
built in the CI/CD lesson catch mechanical defects more reliably than a
tired human skimming a diff. What review buys that no machine can:

- **Knowledge sharing.** After review, two people understand the change.
  Multiply by every PR and no part of the system has exactly one person
  who can touch it — which is the difference between "Ana is on holiday"
  and "the payments code is frozen until Ana is back."
- **Consistency.** Reviewers hold the codebase to one set of conventions,
  so it reads as if one careful person wrote it. You felt the value of
  that in the clean-code lesson: uniform code is code you can skim.
- **Design feedback while change is cheap.** A wrong abstraction costs a
  comment to fix at review time and a migration to fix a year later.
  Review is the last cheap moment.
- **A written culture.** Review threads are where a team actually
  negotiates its standards — every argument settled in review becomes a
  precedent the next PR inherits.

One consequence worth internalizing now: review is the main way *your*
code gets better, too. Teams that review well are teams whose juniors
level up fast, because they get a senior's reading of their work every
few days.

The bar a reviewer holds a change to is **better, not perfect**: does the
codebase improve when this merges? Demanding perfection stalls every PR
in a swamp of taste; demanding improvement keeps the whole system moving
in one direction.

## Reading a diff like a reviewer

A diff is code with the context stripped away, so read it with the
surrounding files open — the exercise is set up exactly that way. Work
top-down through concerns, most expensive first:

1. **Scope and design.** Should this change exist, and in this shape? Is
   it one change or three in a trenchcoat?
2. **Correctness.** The bugs tests won't catch: handled errors, edge
   cases, security holes, lying doc comments. Your S4 lessons so far —
   security, SQL, HTTP clients — are precisely the reviewer's checklist.
3. **Tests.** New behavior with no new tests is a claim without evidence.
   Ask what the tests *prove*, not whether CI is green — a green pipeline
   only certifies what somebody thought to test.
4. **Readability.** Names, cohesion, comments — everything from the
   clean-code lesson, applied to someone else's work.
5. **Style nits.** Last, and mostly not at all: the formatter and linter
   already had this argument so you don't have to.

**In Go:** step 5 nearly vanishes. `gofmt` ends layout debates, `go vet`
catches a class of mistakes, and the
[Code Review Comments wiki](https://go.dev/wiki/CodeReviewComments) gives
reviewers a shared, citable vocabulary for the rest. That page exists so
Go reviews can say "see the wiki on error strings" instead of relitigating
taste per PR — citing a convention beats asserting a preference.

## Severity: say which comments block

The single highest-leverage review habit is labeling every comment with
what you expect to happen to it:

- **blocking** — a defect or standard violation; the PR does not merge
  until it is addressed. Say what you see, why it matters, and what would
  resolve it.
- **suggestion** — would improve the change; author's call. Mark it
  explicitly: "not blocking".
- **nit** — trivial polish. Batch them, and never let them outnumber your
  substantive comments.
- **question** — you are missing context and the answer could change your
  mind. A real question, not an accusation with a question mark: "what
  happens if the name is empty here?" is a question; "you didn't think
  about empty names, did you?" is an attack in costume.

Unlabeled comments force the author to guess your intent, and both guesses
are bad: treat everything as blocking and the PR crawls through three
rounds of nit-polishing; treat everything as optional and the injection
you flagged ships to production. Severity inversion — a `nit` label on a
security hole, `blocking` on a variable name — is a worse review failure
than missing the defect entirely, because it actively misleads.

## Comment craft: about the code, not the author

Review comments are criticism delivered in writing, in public, with your
name on it — the tone you set is the tone your team inherits. The
mechanics of a comment that lands:

- **Address the code.** "This query concatenates user input" — not "you
  concatenated user input". The author *is* going to feel reviewed either
  way; pointing at the code gives them somewhere to stand that isn't
  self-defense. A defect is a property of the diff, not a verdict on a
  person.
- **Be specific and actionable.** Name the line, name the problem, name
  an acceptable exit. "Rewrite this properly" contains zero information;
  the author who receives it can only guess what you want and resent you
  while guessing.
- **Give the why.** "Use `?` placeholders" teaches one fix. "User input
  in the SQL string means a crafted username runs as SQL — placeholders
  keep data out of the statement" teaches the category. Comments with
  reasons are the knowledge-sharing channel actually operating.
- **Never generalize about the person.** "You always ignore errors" is an
  attack on identity and unanswerable — the only reply is a defense of
  self. The same point about the code — "this `Exec` error is dropped;
  callers get `nil` on a failed insert" — is checkable and fixable.

Hostility is not rigor. A hostile comment that is technically right still
fails at its only job, because the author's energy goes into swallowing
the insult instead of absorbing the point — and the next author, having
watched, will structure their work to avoid your review rather than to
pass it. Praise, meanwhile, is allowed and useful: "this table test is
exactly the pattern we want" costs one line and reinforces what a good
change looks like.

## The author's chair, part 1: a reviewable PR

Review quality is set before the reviewer arrives, by the shape of what
you send them. Reviewers do their best work on small, single-purpose
diffs; past a few hundred changed lines, review degrades into skimming
and "LGTM" — the big PR gets a *worse* review precisely because there is
more to check.

- **One reason to change per PR.** The clean-code cohesion test at PR
  scale. Refactors ride separately from behavior changes — mixed
  together, the reviewer cannot tell which lines are *supposed* to change
  behavior, so the mechanical rename hunks bury the three lines that
  matter. "Misc improvements + email support" is a confession.
- **Title and description do the reviewer's setup.** Title: one specific
  imperative line — "signup: add email validation and storage", not
  "misc fixes". Description: what changed, *why* (the ticket, the
  motivation), and how to verify it. The description is also what `git
  log` archaeology finds in two years, long after the review threads are
  forgotten.
- **Self-review before requesting.** Read your own diff top to bottom,
  as a stranger, before anyone else does. This pass reliably catches the
  embarrassing tier — leftover debug prints, an accidental rename that
  doubled the diff, the test you meant to write — and every one you catch
  yourself is a review round you don't spend a colleague on. Five
  minutes; highest ratio in this lesson.

**In Go:** the mechanical pre-review is your CI/CD lesson verbatim —
`gofmt`, `go vet`, `go test ./...` locally before pushing. A PR that
fails the robot's checks wastes the human's attention on what the robot
would have said for free.

## The author's chair, part 2: receiving review

A review of your code arrives and some of it stings. The professional
skill is deciding, comment by comment, what happens next — and neither
"fight everything" nor "concede everything" is a strategy. Silent
concession on a point you believe is wrong ships worse code; reflexive
defense of every line teaches reviewers to stop reading carefully.

**Concede when:**

- the reviewer found a defect — correctness ends the discussion;
- they point at a written convention or the surrounding code's
  consistency — the standard outranks both of you;
- your honest best counter-argument is "I prefer mine". Preference ties
  go to the existing style, not to whoever types longer paragraphs.

**Push back when you have reasoning the reviewer lacks** — a requirement
they haven't seen, a measurement, a convention that supports your shape —
and push back by *showing it*, not asserting it. "Effective Go's getters
section covers this — no `Get` prefix on getter names" is a push-back
that can end a thread; "I like it better this way" never ends anything. A reviewer
suggestion that is good but out of scope gets the honest middle answer:
agree, file it as a follow-up, keep this PR focused — the same discipline
you want from authors, exercised from the author's side.

Two mechanics that keep threads healthy: answer every comment — with a
fix, a reason, or a follow-up reference — because silence reads as either
contempt or an oversight; and when a disagreement survives two written
rounds, stop typing and talk, then record the outcome on the thread.
Escalating to a conversation is not a failure of review; ten-reply
threads are.

## Exercise

Open [`exercise/`](exercise/) — you sit in both chairs. The `main` branch
of a small signup package is in the root; `pr/` holds PR #14 ("Misc
improvements + email support"), its diff, and an earlier review of it that
was withdrawn for hostility; `author/` holds a PR of yours with four
reviewer comments awaiting replies. `README.md` walks you through five
parts; everything you write goes in `NOTES.md`. There is no grading
script — your tutor reviews your review.

Acceptance criteria:

1. Your part 1 review finds the real defects: at least three `blocking`
   comments each stating what, why, and what would resolve it; at least
   two explicitly non-blocking comments; at least one genuine question —
   plus a verdict and summary comment. Severity must not be inverted.
2. All three hostile comments are rewritten with the technical point and
   severity intact — specific, actionable, aimed at the code; the vague
   summary comment becomes something the author can act on.
3. Part 3 names what is wrong with the PR's packaging, proposes a
   defensible split with one PR ordered first for a stated reason, writes
   the title and description the email PR deserved, and lists three
   self-review catches.
4. Part 4 replies to all four comments, each labeled concede / push back /
   answer with real reasoning — at least one concession and at least one
   push-back grounded in a criterion or convention, not preference — and
   ends with your one-sentence concede rule.
5. `NOTES.md` is complete and you can defend every call in discussion.

## Further reading

- [Google Engineering Practices — Code Review](https://google.github.io/eng-practices/review/)
  — both halves: the reviewer's standard and the author's guide
- [Go Wiki — Code Review Comments](https://go.dev/wiki/CodeReviewComments)
  — the citable convention list for Go reviews
- [Conventional Comments](https://conventionalcomments.org/) — one
  well-known labeling scheme for comment severity
- [How to Do Code Reviews Like a Human](https://mtlynch.io/human-code-reviews-1/)
  — the comment-craft half of this lesson, expanded
