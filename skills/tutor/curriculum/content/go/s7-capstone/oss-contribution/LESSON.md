# OSS Contribution

> `go.capstone.oss-contribution` · ~10-15h · Stage: Expert Capstone (Go)

## Objectives

By the end of this lesson you can:

- Find and evaluate a real open-source issue suited to your skills, and explain
  why the project and the issue are a good fit.
- Navigate an unfamiliar codebase to locate the relevant code paths, and
  summarize how the affected subsystem works.
- Implement the fix or feature following the project's contribution guidelines,
  style, and test conventions.
- Open an upstream PR with a clear description, respond to maintainer review,
  and iterate until it is merged or meaningfully reviewed.
- Reflect on what differed from working in your own codebase, and what you
  would do differently next time.

## The first codebase that is not yours

Everything you have built in this stage is yours: you wrote the PRD, chose the
boundaries, decided what "done" meant, and changed any decision you disliked.
This lesson removes all of that at once.

Upstream, you own nothing. The style is settled, the design has history you
cannot see, and the person who decides whether your work lands is a volunteer
who owes you nothing — not a reply, not a review, not a merge. Your leverage is
not the quality of your code. It is how cheap you make it for them to say yes.

That reframing is the lesson. A correct patch that arrives as a 900-line diff
with three unrelated cleanups, no test and a one-line description costs a
maintainer an hour they did not ask for. A correct patch that arrives as sixty
lines, one concern, a failing-then-passing test and a description answering
every question before it is asked costs them ten minutes. The second one
merges. Often the second one is the *worse* patch.

You have also just spent this stage building the other side of the desk: a
README a stranger could start from, a diff you graded yourself, a design
attacked in review. Upstream you are now the stranger.

## Start the clock today

Most of the elapsed time in this lesson is not your time. Maintainers answer in
days; two weeks of silence on an open PR is ordinary, not rejection. Ten to
fifteen hours of *your* work spreads over three to six weeks of calendar —
roughly 2-3 hours on selection and triage, 1-3 reproducing and reading, 3-5
implementing in their style, one writing the PR, and 2-3 on review rounds that
trickle in over weeks.

So begin selection today and expect the review to run in the background
alongside the final lesson of the stage. The commonest way this lesson fails is
starting it in the last week and arriving with an unanswered PR and nothing to
discuss.

## Choosing a project: signals with dates on them

You are not looking for the most impressive project but for one that is
**alive, readable, and welcoming to strangers**, in that order. Every signal
below is checkable in ten minutes, and each wants a date or a number in your
worksheet, not an adjective.

| Signal | What good looks like | Where you check |
|---|---|---|
| Commit recency | Commits in the last month; not one burst a year ago | The default branch's history |
| Maintainer responsiveness | Recent issues have a maintainer reply within days | Sort issues by recently updated |
| PRs from outsiders merged | Several merged PRs from non-maintainers in the last few months, and open ones being reviewed rather than piling up | Merged PRs by author; the oldest open PRs |
| Labelled entry points | `good first issue` / `help wanted` labels that are *unassigned and recent* | The label filter |
| A real CONTRIBUTING.md | Describes the actual process: how to run tests, style, commit and PR conventions | Repo root or `.github/` |
| Licence and legal path | An OSI licence, and a stated DCO or CLA requirement you are willing to meet | `LICENSE`, CONTRIBUTING |
| CI on pull requests | PRs run tests automatically, so your patch gets objective feedback | Any recent PR's checks |
| Size you can hold | You can read the package your change touches in an afternoon | The directory tree |

The strongest single signal is **merged PRs from first-time contributors**: it
proves the path from stranger to merged commit exists and is walked. A project
with `good first issue` labels but no outside merges in a year has labels, not
a pipeline.

Anti-signals, in rough order of cost: an archived repo; dozens of open PRs
untouched for a year; a maintainer stepping back; outside patches answered
months later with "closing, stale"; a codebase whose relevant subsystem is
bigger than your capstone. Popularity is not a signal — the most starred
projects have the longest queues and the highest bar.

Two filters beat any checklist. **Contribute to something you use**: you know
what correct behaviour looks like and you will care when it merges. And
**prefer the mid-sized library or tool** over the ecosystem giant. Go's own
standard library takes contributions — you can even open a GitHub pull request
and a bot converts it into a Gerrit change — but review happens on Gerrit,
under a CLA, at a very high bar, on the core team's cadence; a fine thing to do
one day, a poor place to learn the loop. That mismatch between "where you file
it" and "where it is reviewed" is itself the lesson: read `CONTRIBUTING.md`
before you assume a project works the way its host does.

## Choosing an issue: triage before enthusiasm

Finding an interesting issue is easy. Finding one that is *still wanted, still
open, in scope, and the size you think it is* takes a triage pass. Ask, in this
order, and stop at the first no:

1. **Is it still real?** Can you reproduce it on the current default branch,
   with the exact commands in the issue? Half of old bug reports were fixed
   incidentally, and "I cannot reproduce this on main today, here is what I
   ran" is itself a useful comment.
2. **Is it claimed?** Read the whole thread and search open PRs referencing it.
   Somebody quietly working on it for a week is the commonest wasted evening in
   this lesson.
3. **Does a maintainer want it?** A bug report from a user is not agreement
   that it should be fixed, and definitely not agreement on *how*. The gold
   standard is a maintainer having written what should happen.
4. **Is it in scope?** Check the README's non-goals and any closed PRs
   proposing something similar. Closed-with-explanation PRs are the cheapest
   map of a project's boundaries you will ever find.
5. **Can you size it?** Your estimate is the diff you imagine, plus the test,
   plus the docs, plus two review rounds. Multiply the diff by three and see
   whether you still want it.

The types of contribution, ranked by how well they suit a first patch:

- **A bug with a reproducer.** Best starting point: done is objective, and you
  can write the failing test before the fix — the discipline from the hardening
  lesson, now applied to somebody else's bug.
- **Documentation that is wrong.** Undervalued and genuinely welcome: wrong
  docs are bugs, and a fixed example that used to fail is a real contribution.
- **A missing test for existing behaviour.** Low risk, high welcome, and it
  forces you to read the subsystem properly.
- **A small feature a maintainer explicitly asked for**, in writing, with the
  shape described.
- **An unrequested feature.** The most commonly rejected PR in open source: it
  arrives as a surprise, adds maintenance burden forever, and asks a stranger's
  design to be relitigated. Do not start here.

## Claiming: the comment that gets a reply

Read `CONTRIBUTING.md` before you type anything and follow it over anything
written here — some projects want an issue first, some a design sketch, some
ask you not to claim at all and just send the patch.

"Can I work on this?" is a weak comment: it asks the maintainer to do work
(reply, assign) and offers nothing. A comment that gets a reply shows effort
already spent:

> I can reproduce this on `main` (`go test ./internal/parse -run TestQuoted`
> fails with …). It looks like `parseQuoted` in `internal/parse/quote.go`
> returns before the closing-quote check when the input ends mid-token. I would
> fix it there and add a table case to the existing test. Two questions before
> I start: is that the right layer, and do you want the error wrapped as
> `ErrSyntax` or a new sentinel? I can have a PR up this week.

Four properties: evidence you reproduced it, the code path you found, an
approach small enough to argue with, and one or two narrow questions. Add a
rough timeline so silence from you later is not a mystery.

Claim one issue at a time, and say so in the thread if you drop it. After a
week of silence on a small, clearly-scoped bug it is usually fine to send the
patch anyway, offering to close it if the direction is wrong — a diff is easier
to react to than a proposal. On anything larger, silence means do not build it
yet.

## Reading a codebase you did not write

You cannot read it all, and you do not need to. You need one path: from the
observable symptom to the line that causes it, and enough of the surrounding
subsystem to know your change is safe.

1. **Get it building and its tests green first**, the way `CONTRIBUTING.md`
   says. You now know what green looks like on your machine, so any red later
   is yours.
2. **Grep for the string.** The exact text of an error message, log line or CLI
   output from your reproduction finds the relevant file more often than
   everything else combined.
3. **Start from the failing test, or write one.** It is both your entry point
   for reading and, later, the proof your fix works
   (`go test -run TestName ./path/...`).
4. **Read types before functions.** The data model is the design, and the error
   vocabulary tells you what the subsystem thinks can go wrong.
5. **Follow one call chain end to end**, entry point to output, writing down
   the seams you cross. `go doc ./internal/pkg` gives you a package's exported
   surface in one screen.
6. **Ask git why.** `git log -p <file>` and `git blame` on the lines you are
   about to change give you the intent, and often the issue that put them
   there. Code that looks wrong is sometimes load-bearing.

Only then form an opinion; anything you flag earlier is a guess about code you
have not understood. Write the summary while it is fresh — five to ten
sentences on how the subsystem works, what your change touches, and what could
break. If you cannot write it, you have not read enough to be trusted with the
patch, and you need it for the PR description anyway.

## The diff a reviewer can say yes to

Scope discipline is the whole craft here, and it is mostly about what you
*leave out*.

- **One concern per PR.** If you find a second bug, open a second issue. A PR
  that fixes two things cannot be partially merged, so it waits for agreement
  on both.
- **No drive-by refactors, no reformatting.** Renaming a variable you dislike
  adds review surface, wrecks `git blame`, and signals that you think the
  project is yours. If something genuinely should change, raise it in an issue
  after your patch lands.
- **Match the local style even where you would do it differently** — their test
  helper, error phrasing, table shape, naming. A reviewer should not be able to
  tell which lines are yours by looking.
- **Add the regression test that fails before and passes after**, and say so in
  the description: that is the sentence converting a claim into evidence. If
  the area has no tests, follow the closest pattern and say what you did.
- **Update the docs and changelog the project keeps.** An unreleased section in
  `CHANGELOG.md` wants your entry.
- **Commit hygiene their way.** One squashed commit or a clean series;
  imperative subject lines under about 70 characters saying *why*; often a
  `Signed-off-by` line (`git commit -s`, the Developer Certificate of Origin).
  Check the recent history to see what they actually merge.

Size is the review-latency knob you control: a sixty-line PR gets reviewed this
weekend, a nine-hundred-line one when someone finds an afternoon.

## The PR description a reviewer needs

The description exists to make review cheap. Assume the reviewer has not read
the issue recently, does not remember the subsystem, and has fifteen minutes.

```markdown
Fixes #1234.

## What
`parseQuoted` returned early when input ended mid-token, so a trailing
unterminated quote parsed as a valid empty value instead of an error.

## Why this approach
The check belongs in `parseQuoted` rather than the caller, because every caller
would otherwise need it; #1101 rejected the caller-side variant for that
reason. I used the existing `ErrSyntax` rather than a new sentinel, to keep the
exported error set unchanged.

## Testing
Added `TestQuotedUnterminated` to the existing table in `quote_test.go`; it
fails before the change and passes after. `go test ./...` and `go vet ./...`
are clean on my machine.

## Not included
The same early-return shape exists in `parseBrace`; it needs a different fix
and I opened #1240 for it rather than growing this PR.
```

What each part buys: linking the issue closes it and supplies context; *why
this approach* pre-empts the first review question and shows you read the
history; *testing* is how a reviewer trusts the change without running it;
*not included* proves the scope was chosen rather than accidental. Never open a
PR you have not first read as a diff yourself.

## Go-specific etiquette

- **`gofmt` is not negotiable.** Unformatted Go marks a patch as careless in
  one glance. Run `gofmt -l .` and their linter config before pushing.
- **Do not add dependencies.** Many Go projects are standard-library-only or
  run a strict dependency policy; a PR that grows `go.mod` is a policy
  decision, not a bug fix. If one is genuinely needed, ask in the issue first.
- **Exported API is forever.** A new exported function, method or field is a
  maintenance commitment under Go's compatibility norms. Prefer the unexported
  fix; if the API must change, say so at the top of the description.
- **Errors and tests their way.** If the package uses sentinel errors and
  `errors.Is`, extend that vocabulary; table-driven is the norm, but the table
  shape, helper names and `t.Run` naming come from the file you are editing.
- **Legal step done early.** DCO sign-off is a commit flag, a CLA a one-time
  signature. Both are trivial, and both block a merge indefinitely when
  discovered after review has finished.

## Review without defensiveness

Review comments are about the code, and the maintainer usually knows something
you do not: a bug from 2019, a downstream consumer, a platform where your
approach breaks. Assume that first.

Three legitimate responses, and nothing else:

- **Agree and fix.** The common case. Make the change, reply on the thread
  saying what you did, do not argue.
- **Ask, concretely.** "Do you mean I should move the check into the caller, or
  keep it here and rename?" — a narrow question with a proposal beats "what do
  you mean?".
- **Disagree with evidence, once.** State the reason, show the test or the
  benchmark or the linked issue, and then accept the answer. Repeating an
  argument after a maintainer has decided is how a mergeable PR becomes an
  unpleasant thread.

Mechanics: reply to every thread rather than silently pushing changes, push
follow-up commits rather than force-pushing over a review in progress (unless
they ask for a rebase), re-request review once when you are done, and never
take the review to another channel. Implementing a comment you disagree with is
a normal day; say so plainly and move on.

## Waiting, pinging, and the outcomes that are not "merged"

Silence is the default state of open source and almost never about you. Wait a
full two weeks, then post one short, friendly ping referencing the issue and
offering to close if it is not wanted. One. A weekly ping costs you the
maintainer's goodwill.

While you wait, keep the branch mergeable — rebase when it conflicts, keep CI
green — and do something useful instead of refreshing the tab: leave a genuinely
helpful comment on someone else's PR, pick up a second small issue, or improve
the docs you had to reverse-engineer while orienting.

The outcomes, all of which are normal:

- **Merged**, sometimes squashed, reworked or partly rewritten by a maintainer.
  Read the rewrite closely; it is free senior review of your instincts.
- **Still open** at the end of this lesson. Expected. You are graded on the
  process and the review conversation, not on a merge you do not control.
- **Closed.** Scope, direction, maintenance burden, timing, or someone got
  there first. Reply once, thank them, and ask the only question that pays:
  "what would have made this acceptable?" That answer is the most valuable
  thing you take away from this lesson. Do not argue, do not reopen, do not
  repost it elsewhere.

One thing that is never acceptable: volume for its own sake. Cosmetic PRs —
typo-only churn, whitespace, "improvements" nobody asked for — waste real
volunteers' time, and maintainers recognise them instantly. One patch a
maintainer was glad to receive beats ten they had to close.

## Exercise

Land — or genuinely attempt to land — one upstream contribution, and keep the
paper trail. The worksheets in [`exercise/`](exercise/) are what you are graded
from: [`CANDIDATES.md`](exercise/CANDIDATES.md) (project health),
[`ISSUE-TRIAGE.md`](exercise/ISSUE-TRIAGE.md) (issues, reproduction,
orientation), [`PRE-PR-CHECKLIST.md`](exercise/PRE-PR-CHECKLIST.md) (the gate),
[`REVIEW-LOG.md`](exercise/REVIEW-LOG.md) (the conversation, with dates) and
[`RETRO.md`](exercise/RETRO.md) (the reflection).
[`exercise/README.md`](exercise/README.md) gives the order and the timeline.

Acceptance criteria:

1. **Two candidate projects scored** in `CANDIDATES.md`, every health signal
   answered with a date, a number or a link — no adjectives — and a decision
   saying why one and not the other.
2. **`CONTRIBUTING.md` read and summarized**: how to run the tests, the style
   and lint requirements, commit and PR conventions, and whether a DCO
   sign-off or CLA is needed. If the project has no contribution docs, say what
   you inferred from recent merged PRs instead.
3. **Three issues triaged, one chosen** in `ISSUE-TRIAGE.md`, each run through
   the five triage questions, with the reproduction recorded as an exact
   command plus observed and expected output.
4. **Orientation written**: five to ten sentences on how the affected subsystem
   works, the files and functions you expect to change, and what could break —
   written before you start editing.
5. **Claimed in public before you wrote code**: the comment text and its link,
   showing evidence, a proposed approach, and a narrow question.
6. **The change follows their conventions**: their test style, `gofmt` and lint
   clean by their config, no unrelated edits in the diff, no new dependency,
   and a test that fails before your change and passes after (or, for a docs
   contribution, the before/after evidence that the example was wrong).
7. **`PRE-PR-CHECKLIST.md` complete**, every box ticked or explicitly waived
   with a one-line reason.
8. **A PR opened upstream**: the link, plus a description carrying what, why
   this approach, how it was tested, and what you deliberately left out.
9. **`REVIEW-LOG.md` kept with dates**: every review thread, what was asked,
   what you did, and whether you agreed, asked, or disagreed with evidence. If
   the PR is still unanswered, log the ping date and what you did while
   waiting.
10. **`RETRO.md` written**: what differed from working in your own codebase,
    what the maintainer knew that you did not, what you would do differently,
    and the honest outcome — merged, open, or closed.

There is no automated check for this lesson and there cannot be: the work
happens in someone else's repository. Your tutor will not open a pull request,
push a branch, or write anything to an upstream project on your behalf — every
public word is yours, under your name. Verification is a conversation in which
you show the links and defend the choices: why that project, why that issue,
why the diff is that size, and what you did with the review you got.

## Further reading

- [go.dev — Contributing to the Go project](https://go.dev/doc/contribute) —
  the canonical process for Go itself: CLA, Gerrit, review culture. Read it
  even if you contribute elsewhere; it is a model of a documented process.
- [Open Source Guides — How to contribute](https://opensource.guide/how-to-contribute/)
  — the etiquette in more depth, from the people who host most of it.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) — the
  checklist Go reviewers have in their heads. Run your diff past it first.
- [Developer Certificate of Origin](https://developercertificate.org/) — the
  whole text of what `git commit -s` asserts. Two minutes, once.
