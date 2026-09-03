# Tutor notes — Branches & Remotes

## Where the learner is

Fresh from version-control-git: they can init/add/commit/log/diff a single
straight line of history on one branch, on their own machine. Terminal habits
exist but are still slow and literal. This is the biggest conceptual jump in
S0 — history becomes a graph and the repo stops being alone in the world. No
GitHub account is assumed; the exercise fakes the remote with a local bare
repo, so keep the GitHub parts conceptual and optional to try for real.

## Common misconceptions

- **"A branch is a copy of the project"** — it's a movable name pointing at a
  commit; nothing is copied. If this model is missing, everything downstream
  (merging, remote-tracking) turns to magic. Fix it first, with drawings.
- **"A conflict means I broke something"** — it's git *asking a question* it
  cannot answer itself. Learners panic at the markers; walk the calm
  edit → add → commit procedure and mention `git merge --abort`.
- **"origin/main is live"** — they assume git always knows the server's
  state. `origin/main` updates only on push/pull/fetch and goes stale the
  moment anyone else pushes.
- **"Pull request is a git command"** — it's a hosting-site feature layered
  on branch + push. Untangle which steps of the flow are git and which are
  the website.
- **"push sends my files"** — push sends *commits*; uncommitted edits never
  leave the machine. Pairs with switching-with-dirty-tree confusion: teach
  the commit-before-switch habit rather than stashing (out of scope).
- **Being on the wrong branch** — half of all exercise failures. The reflex
  to build: read `git status`'s first line before doing anything.

## Grilling points

- "Draw your exercise repo's final history on paper. Where do `main`,
  `add-milk`, and `weekend-plans` point? How many parents does the last
  commit have?"
- "Why did merging `add-milk` create no commit, while merging
  `weekend-plans` did?" (Fast-forward vs true divergence.)
- "You commit on `main` but don't push. Where is your `main`? Where is
  `origin/main`? What does the remote have?"
- "In the PR flow, list which steps are git commands and which happen on
  GitHub."
- "Mid-conflict you decide it was all a mistake. What gets you out, and what
  state are you in afterwards?"
- "Why is the server's copy bare — what would go wrong if people edited
  files directly on it?"

## Grading rubric

- **A** — `check.sh` fully green; learner draws the final commit graph
  correctly, explains fast-forward vs merge commit, resolves the conflict
  deliberately (can say why the fenced block became that line), and
  articulates the local / remote / remote-tracking distinction unprompted.
- **B** — Green checker; branches and merging are solid, but the three-mains
  distinction needs prompting, or the conflict was resolved by trial and
  error rather than understanding the markers.
- **C** — Green only after heavy hinting, or the remote model is still magic
  ("push puts my folder on the internet"). Pass only if remediation of the
  branch-pointer drawing lands; otherwise iterate.
- **Fail** — Checker failing, or steps were pasted without understanding —
  cannot say what a branch is, what the merge commit joined, or what push
  moved. Remediate; do not advance.

## Remediation ladder

1. "Inside `repo/`, run `git log --oneline --graph --all` and read it aloud.
   What does each name on the right point at?"
2. "Run `git status`. Which branch are you on, and is the tree clean? Most
   checker failures start with one of those two."
3. "Read the first `FAIL` line from `check.sh` — its fix line names the
   command. Which numbered exercise step does it belong to?"
4. For the conflict: "Open `list.md`. The `<<<<<<<` block is your side, the
   `>>>>>>>` block is theirs. Delete all three marker lines, write the one
   line the criteria ask for, then `git add list.md && git commit`."

## After passing

Preview: "Next you zoom out from your machine: how the internet actually
works — so 'a website that hosts repositories' stops being hand-waving."
