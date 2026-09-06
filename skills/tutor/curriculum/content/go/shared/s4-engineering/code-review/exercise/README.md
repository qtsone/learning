# Exercise — Both chairs of a code review

You will sit in both chairs. First you review someone else's pull request
(parts 1-3); then you answer a review of your own code (part 4). Everything
you produce goes into `NOTES.md`, and your tutor grades it in conversation —
there is no test script for judgment.

The cast:

- `signup.go` + `signup_test.go` — the current `main` branch of a small
  account-signup package. Tests pass; `gofmt` and `go vet` are clean.
- `pr/PR.md` — pull request #14 as submitted: title, description, stats.
- `pr/change.diff` — the diff under review. Read it hunk by hunk with the
  `main` files open next to it.
- `pr/hostile-review.md` — an earlier review of the same PR, withdrawn for
  tone. Raw material for part 2.
- `author/your-pr.md` — a PR *you* wrote, with four reviewer comments
  waiting for your reply. Material for part 4.

Optional but recommended: apply the diff and watch CI lie to you —

```sh
patch -p1 < pr/change.diff
go test ./...        # green!
patch -Rp1 < pr/change.diff   # revert
```

Keep that green run in mind while you review: every defect you are about
to find sails straight through the pipeline you built in the CI/CD lesson.

## Part 1 — Review PR #14

Read `pr/PR.md`, then `pr/change.diff` against the `main` files. Write your
review in `NOTES.md`: one row per comment with its location, a severity
label — `blocking` / `suggestion` / `nit` / `question` — and the comment
text exactly as you would post it.

Bars to clear:

- At least three `blocking` comments, each saying what you see, why it
  matters, and what would resolve it.
- At least two non-blocking comments (`suggestion` or `nit`), explicitly
  marked as not holding up the merge.
- At least one `question` that is a genuine question — one whose answer
  could change your mind — not an accusation wearing a question mark.

Finish with a verdict (approve / request changes) and the two-or-three
sentence summary comment you would post on the PR as a whole.

## Part 2 — Rewrite the hostile review

Open `pr/hostile-review.md`. For each of the three comments, write the
version you would actually post. The technical point and the severity must
survive; the contempt must not. Comment 3 is the hard one: "rewrite this
properly" gives the author nothing to do — your rewrite has to.

## Part 3 — The PR that should have been

Critique the PR's packaging, not its code:

1. List what is wrong with the title and description as review artifacts.
2. This diff mixes concerns. Propose how it should have been split —
   list each resulting PR with a one-line title, and say which one you'd
   ask for first and why.
3. Write the full title and description the email PR deserved: what
   changed, why, and how a reviewer can verify it.
4. List three things a five-minute self-review pass (reading your own
   diff, top to bottom, as a stranger) would have caught before the PR
   was ever opened.

## Part 4 — The author's chair

Open `author/your-pr.md` and reply to all four comments in `NOTES.md`.
Label each reply concede / push back / answer, and give reasoning either
way — "done" is a concession, not reasoning, and "no" is not a push-back.
When you push back, cite something sturdier than preference: a criterion
from the clean-code lesson, a Go convention, a requirement.

Then write your personal rule, in one sentence, for when to concede.

## Part 5 — Bring it to the discussion

Re-read your part 1 review once more before the discussion and check it
against your own part 2 standard: would every comment survive the rewrite
test? Fix any that would not — then bring `NOTES.md` to your tutor.
