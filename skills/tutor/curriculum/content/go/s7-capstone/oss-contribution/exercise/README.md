# Exercise — land one upstream contribution

Nothing compiles here and nothing runs. The work happens in a repository you do
not own, and these five worksheets are the record it happened well. Fill them
as you go, not afterwards: half of what they ask for (dates, the comment you
posted, what you believed before you read the code) cannot be reconstructed at
the end.

## 0. Start today

Most of the elapsed time in this lesson is not your time. Maintainers reply in
days, review in weeks, and go quiet without meaning anything by it. Begin
selection now and expect the review rounds to overlap the last lesson of the
stage.

| Phase | Your hours | Elapsed |
|---|---|---|
| Selection and triage | 2-3 | days |
| Reproduce and orient | 1-3 | days |
| Implement, test, match their style | 3-5 | days to a week |
| Write and open the PR | 1 | one sitting |
| Review rounds | 2-3 | one to four weeks |

## 1. Work in this order

1. **[`CANDIDATES.md`](CANDIDATES.md)** — score two projects on health signals.
   Every cell wants a date, a number or a link. **Clear your choice with your
   tutor before you triage issues**, and clear the issue before you write code:
   a rejected patch is a normal outcome, but a patch nobody ever wanted is an
   avoidable one.
2. **[`ISSUE-TRIAGE.md`](ISSUE-TRIAGE.md)** — triage three issues, choose one,
   reproduce it, then write the orientation summary *before* you edit anything.
3. Post your claim comment. Paste it into `ISSUE-TRIAGE.md` with its link and
   the date.
4. Do the work in their style, with a test that fails before and passes after.
5. **[`PRE-PR-CHECKLIST.md`](PRE-PR-CHECKLIST.md)** — the gate. Every box
   ticked, or waived with a reason on the line next to it.
6. Open the PR. Paste the description and the link into
   [`REVIEW-LOG.md`](REVIEW-LOG.md).
7. **[`REVIEW-LOG.md`](REVIEW-LOG.md)** — one row per review thread, with
   dates, while it happens.
8. **[`RETRO.md`](RETRO.md)** — written at review time, whatever the outcome.

## 2. Rules of the game

- **Read `CONTRIBUTING.md` first, and follow it over anything the lesson says.**
  Their process wins. Summarize it in `ISSUE-TRIAGE.md` so you cannot claim you
  read it without having read it.
- **Reproduce before you claim.** "It still happens on the current default
  branch, here is exactly what I ran" is the sentence that starts a good
  thread.
- **One issue at a time**, and say so publicly if you drop it.
- **One concern per PR.** A second bug is a second issue, not more diff.
- **No drive-by refactors, no reformatting, no new dependencies.** Anything the
  maintainer did not ask for is review cost you added to their week.
- **The test comes first**, and its before/after state is a claim you make in
  the PR description.
- **Log the conversation as it happens**, including the comment you disagreed
  with, and what you did about it.
- **No cosmetic volume.** Typo churn and unrequested "improvements" waste real
  volunteers' time and grade as a fail here, merged or not.

## 3. Before you say you are ready

- Why this project and not the other one — in health signals, not enthusiasm.
- How the affected subsystem works, in five to ten sentences, without notes.
- Why the diff is the size it is, and what you deliberately left out of it.
- The review comment you found hardest, and what you did with it.
- What you would do differently on the next one.

Then tell your tutor you are ready. There is no automated check and there
cannot be — the work is in someone else's repository. Your tutor will not open
a PR, push a branch, or post anything upstream for you; every public word is
yours. Bring the links, the diff, and the log, and expect to defend the choices
rather than the outcome. A well-argued closed PR grades above a merged typo
fix.
