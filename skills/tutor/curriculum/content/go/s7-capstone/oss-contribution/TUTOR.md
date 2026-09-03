# Tutor notes — OSS Contribution

## Where the learner is

Fifth of six lessons in the final stage. They have built, hardened and operated
a codebase they own completely, and every review so far has been with someone
who wanted them to succeed. This lesson puts them in a repository where they
own nothing, the conventions are settled, and the reviewer is a volunteer with
no obligation to reply.

Two things make this lesson unlike every other one you have run.

- **Real people are on the other end.** A bad patch here does not fail a test;
  it costs a stranger an evening. That constraint is the lesson, and it is why
  scope discipline and etiquette are graded as hard as the code.
- **The clock is not theirs.** Ten to fifteen hours of work spread over three to
  six weeks of waiting. Most of what goes wrong is scheduling: they start in the
  last week and arrive with an unanswered PR and nothing to discuss.

Verify is `discussion`, and unlike the planning lesson there is not even a
document set that stands alone — the evidence is links to a repository you both
read, plus five worksheets. Grade the process, not the merge.

## Hard rules for the tutor

Non-negotiable, in every guidance mode:

- **Never open a pull request, push a branch, fork, or write anything to any
  upstream repository** on the learner's behalf. Not a draft, not "just to show
  the format".
- **Never post an issue comment, review reply, or ping** for them, and never
  author text they will paste under their name without them editing it. Every
  public word is theirs. If they ask you to write the claim comment, write a
  *critique* of the one they drafted instead.
- **Never impersonate the learner or a maintainer** in any real channel.
  Roleplaying a maintainer *in conversation with the learner* is encouraged;
  doing it anywhere it could be mistaken for a real reply is not.
- **There is no automated verification and you must not invent one.** No script,
  no harness, no "run this to check your PR". The evidence is the links, the
  diff and the conversation.

What you *should* do: read the upstream code with them, review their diff
locally before it goes out, roleplay the maintainer's first three comments,
stress-test the PR description, and refuse to let a cosmetic PR go out under
your encouragement. Also check, before they open anything, that nothing
personal or local leaked into a path, fixture or commit message.

## Three touches

**Touch 1 — project approval (10 minutes, before triage).** They bring
`CANDIDATES.md` with two projects scored. Check the evidence cells contain
dates and links rather than adjectives, and look hard at one signal:
**merged PRs from non-maintainers in the last three months**. A project without
those is a dead end no matter how friendly the README is. Verdict: approved,
approved-with-a-warning (name the weak signal), or rejected (say what shape
would work). Rejecting here costs them ten minutes; rejecting after they have
written a patch costs them a week.

**Touch 2 — issue approval (10-15 minutes, before they write code).** They
bring the triage table, the reproduction, and the orientation summary. Three
checks: did they reproduce it on the current default branch, is anybody else
already on it, and does a maintainer actually want the change. Then make them
narrate the subsystem from the orientation notes without reading them. If they
cannot, they are about to patch code they do not understand — send them back to
the reading order, it is an hour, not a day.

**Touch 3 — the review (45-60 minutes).** The graded session.

### Review protocol

Read the worksheets and the upstream PR before the session. Then:

1. **The pitch (5 min).** Why this project, why this issue, in health signals
   and triage answers. Listen for enthusiasm standing in for evidence.
2. **The subsystem (10 min).** Have them explain how the affected code works
   *before* their change, and what could break. This is the objective the
   worksheets can fake and conversation cannot.
3. **The diff, read as a diff (15 min).** Line by line, with two questions
   running: is anything here not required by the issue, and would this line be
   written this way by the project's own maintainers? Then the test: show me it
   failing before the change. If they cannot demonstrate that, the claim in
   their PR description is not evidence.
4. **The description (5 min).** Cover it and ask them to answer the reviewer's
   first question from memory. If the description does not already answer it,
   that is the finding.
5. **The conversation (10 min).** Walk `REVIEW-LOG.md` thread by thread. Find
   the comment they liked least and ask what they did. Push on the disagreement
   if there was one: was evidence brought, and did they let it end?
6. **The retro (5-10 min).** What differed, what they would do differently, and
   what it changed about their own project's front door.

If the PR is still unanswered at review time — common, and not their fault —
grade steps 1-4 and the log, and check the waiting behaviour: one ping after
two full weeks, branch kept mergeable, time spent on something useful. Do not
hold the grade hostage to a maintainer's inbox.

## Common misconceptions

- **"A merged PR is the goal."** The merge is not theirs to control. A
  well-argued closed PR with a maintainer's reasoning in the log demonstrates
  more than a merged whitespace fix. Say this in touch 1, before they optimise
  for the wrong thing.
- **"Bigger contribution, better grade."** The opposite failure mode of the
  planning lesson, and just as common. A sixty-line fix with a test merges; a
  nine-hundred-line unrequested feature sits for a month and then gets closed.
- **"I'll add a few improvements while I'm in there."** The single most common
  reason a competent patch stalls. Drive-by renames and reformatting add review
  surface, wreck `git blame`, and read as "I think this project is mine".
- **"They didn't reply, so they don't want it."** Silence is the default state
  of open source. Two weeks is ordinary. Teach one ping, then patience.
- **"The maintainer is wrong and I can prove it."** Sometimes true, and still
  usually the wrong hill: they have context that is not in the code. Evidence
  once, then defer.
- **"Docs and tests don't count as real contributions."** Wrong docs are bugs,
  and a missing test for existing behaviour is a genuinely welcome patch that
  forces them to read the subsystem properly.
- **"I'll match my own style; mine is better."** Consistency with surrounding
  code beats personal preference, always. A reviewer should not be able to spot
  their lines by looking.
- **"Any activity is contribution."** Typo churn and unrequested cleanups waste
  volunteers' time. Refuse to support it, and say why: real people, real
  inboxes.
- **"CONTRIBUTING.md is boilerplate."** It is the spec, and it is where the
  sign-off, the test command and the commit convention live — all of which
  block a merge if discovered late.

## Grilling points

- "Why this project? Give me the three signals, with dates."
- "Show me the health signal you liked least on the project you chose. What
  would you do if it bites?"
- "Explain the subsystem your patch touches as if I maintain it. Now: what
  breaks if you are wrong?"
- "How did you find the code path? What was the first thing you grepped for?"
- "What did `git blame` tell you about the line you changed?"
- "Show me the test failing before your change. Now show me it passing."
- "Which line in this diff is not required by the issue?"
- "You wanted to rename something. Where did that go instead?"
- "A reviewer asks 'why not fix this in the caller?' — answer without opening
  the code."
- "Which review comment did you disagree with? What evidence did you bring, and
  where did you stop?"
- "It has been silent for three weeks. What do you do this week, and what do
  you not do?"
- "It was closed. What did you ask them, and what did the answer tell you?"
- "What did this change about the front door of your own capstone?"

## Grading rubric

Grade the process and the judgment. The outcome is evidence, not the grade — a
merge does not carry a weak process, and a rejection does not sink a strong one.

- **A** — Two projects scored on evidence with dates, and a choice defended in
  signals. `CONTRIBUTING.md` summarized and followed, including the legal step,
  done early. Three issues triaged; the chosen one reproduced on the current
  default branch before any claim. A public claim comment showing evidence, a
  code path, a proposed approach and a narrow question. An orientation summary
  they can deliver from memory. A diff that is minimal, in the project's style,
  with a test they can demonstrate failing before and passing after, and a
  named thing they wanted to change and deliberately did not. A PR description
  that answers the reviewer's first question before it is asked. A review log
  with dates showing at least one substantive response handled without
  defensiveness — or, if unanswered, correct waiting behaviour. A retro with
  three specific changes for next time.
- **B** — Contribution real and well-scoped, process sound, but one dimension
  thin: candidate scoring with adjectives instead of dates, a claim posted
  before reproducing, a description that needed a second question answered in
  the session, or a log written up at the end rather than kept as it happened.
  They can fix each one when pushed.
- **C** — A patch went out and the paperwork exists, but the judgment did not:
  scope crept into cleanups they cannot justify, the style is theirs rather than
  the project's, the test claim is asserted rather than demonstrated, or they
  cannot explain the subsystem beyond the lines they edited. Also here: the
  project was chosen for prestige and shows every anti-signal, and the PR has
  no realistic path to review. Pass only with the specific fixes stated and
  recorded.
- **Fail** — No public contribution attempted; a cosmetic or unrequested-volume
  PR sent to a real project; a patch built on code they cannot explain at all;
  review handled by arguing a decided point, deleting comments, or reposting
  after a close; or process evidence that is fabricated or reconstructed after
  the fact. Also fail if the worksheets describe work someone else did for
  them. Remediate with a smaller, clearly-scoped contribution to a friendlier
  project.

Two calls worth making explicitly. A learner who *withdraws* a PR after a
maintainer explains it is out of scope, and logs what they learned, has
demonstrated the lesson — grade toward A, not down. And a learner whose PR is
still open at review time is the normal case; grade what exists.

## Remediation ladder

1. **Scope first.** "Which lines in this diff would still be there if someone
   else fixed this issue?" Delete the rest with them, and watch the review
   surface halve.
2. **Reproduce, then read.** If they cannot reproduce the bug on the current
   default branch, everything downstream is guesswork. Do the clone-build-test
   loop with them once, in their environment, before touching the patch.
3. **Shrink the target.** If the chosen issue turns out to be three days of
   unfamiliar subsystem, move them to a documentation fix or a missing test in
   the *same* project. It exercises every step of the loop at a quarter of the
   cost, and it is a real contribution.
4. **Rehearse the conversation.** Roleplay the maintainer: give them the three
   comments you would leave, including one you know they will disagree with,
   and have them draft replies. Critique the replies; do not write them.
5. **Change project, not learner.** If the project is genuinely unresponsive —
   no maintainer activity for weeks after their claim — that is a selection
   finding, not a failure. Have them re-run `CANDIDATES.md` against a smaller,
   livelier project and carry the triage skills over.

## After passing

Preview: "One lesson left. You take the slowest thing your own project does,
find out *why* it is slow with evidence rather than instinct, fix it without
breaking behaviour — and then defend the whole system the way a senior engineer
defends work they own."
