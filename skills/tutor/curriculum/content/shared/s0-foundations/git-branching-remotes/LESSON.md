# Branches & Remotes

> `shared.foundations.git-branching-remotes` · ~2-4h · Stage: Foundations

## Objectives

By the end of this lesson you can:

- Explain what a branch is and why work happens on branches instead of directly on `main`.
- Create, switch between, and merge branches, and resolve a simple merge conflict.
- Connect a local repository to a GitHub remote and synchronize with `push` and `pull`.
- Walk through the pull-request flow: branch, push, open PR, review, merge.
- Explain the difference between a local branch, a remote branch, and a remote-tracking branch.

## Commits form a chain — branches are name tags on it

In the last lesson every `git commit` saved a snapshot, and each snapshot
remembered the one before it. Drawn on paper, your history was a chain:

```
A --- B --- C
```

So what is `main`, exactly? Just a **name tag stuck on one commit** — the
latest one, `C`. When you commit, git creates the new snapshot and *moves the
tag forward*. That's the whole secret: a **branch is a movable name pointing
at a commit**. It is not a copy of your project, not a folder, not anything
heavy — creating one writes a few bytes.

One more name to know: **HEAD** is git's note-to-self about *which branch you
are currently on*. `git status` prints it on its first line
(`On branch main`).

## Why not just commit to main?

Because `main` has a job: it should always hold the version that *works*.
The moment you start an experiment — half-finished, maybe broken — you don't
want it living on `main`, where you (or, later, teammates) expect solid
ground. So you put the experiment on its own branch:

```
A --- B --- C          ← main   (untouched, still works)
             \
              D --- E  ← add-milk   (your work in progress)
```

Both lines of history exist at once. If the experiment works out, you merge
it in. If it's a dead end, you delete the tag and `main` never knew.

## Creating and switching

```sh
git branch               # list branches; * marks the one you're on
git switch -c add-milk   # create a branch AND switch to it
git switch main          # go back
```

(Older tutorials use `git checkout` for switching — it works, but `switch`
was added because `checkout` did too many unrelated jobs. Use `switch`.)

Switching does two things: it moves HEAD to the other branch, and it rewrites
the files in your folder to match that branch's snapshot. Yes — files change
on disk when you switch. That's normal and safe *if you committed first*;
make "commit, then switch" your habit. To see the whole picture at any time:

```sh
git log --oneline --graph --all
```

## Merging

To bring a branch's work into `main`, stand on `main` and merge:

```sh
git switch main
git merge add-milk
```

Two things can happen:

- **Fast-forward** — `main` hasn't moved since you branched, so git simply
  slides the `main` tag forward to where `add-milk` is. No new commit; the
  chain stays straight.
- **Merge commit** — *both* branches gained commits, so the histories
  genuinely diverged. Git creates a new commit with **two parents**, one on
  each line, tying the fork back together:

```
A --- B --- C --------- M   ← main
             \         /
              D --- E ·     ← add-milk
```

## Merge conflicts

When the two sides changed *the same lines of the same file*, git cannot pick
for you — how would it know which version is right? So it stops mid-merge and
writes both versions into the file, fenced with markers:

```
<<<<<<< HEAD
# Family shopping list
=======
# Weekend shopping list
>>>>>>> weekend-plans
```

`HEAD` marks your side (the branch you're on); the other name marks theirs.
A conflict is **a question, not an error** — nothing is broken. The calm
procedure:

1. Open the file in your editor. Decide what the text *should* say.
2. Replace the whole fenced block — markers included — with the final text.
3. `git add <file>` to declare "resolved", then `git commit` to finish the
   merge (git pre-fills a fine message; just save it).

Lost mid-merge? `git status` says exactly where you are, and
`git merge --abort` backs out and returns you to the pre-merge state.

## Remotes: a copy of the repo somewhere else

Everything so far lives in one place: the `.git` folder on your machine. A
**remote** is *another complete copy* of the repository that your repo knows
by a nickname. Usually that copy sits on **GitHub** — a website that hosts
git repositories so people can back up and share them. But git honestly does
not care where the copy lives: a remote can be a URL on the internet *or a
plain folder on your own disk*, and every command works identically. (The
exercise uses a folder, so you need no account and no network.)

By convention the first remote is nicknamed `origin`:

```sh
git remote add origin <url-or-path>
```

The copy on the server side is usually a **bare** repository — all history,
no working files (nobody edits code *on* the server). You can make one
yourself with `git init --bare`.

## push, pull, and the three "mains"

```sh
git push -u origin main   # send your commits to origin; -u explained below
git pull                  # bring origin's new commits to you
```

`push` uploads *commits* — never your uncommitted edits. Now, carefully,
because this trips everyone: after that push there are **three** different
things people casually call "main":

- **Your local branch `main`** — in your repo. Moves when *you* commit.
- **The remote branch `main`** — inside origin's repo. Moves when *anyone
  pushes* to it.
- **`origin/main`** — a **remote-tracking branch** in *your* repo: a
  read-only bookmark recording where origin's `main` was *the last time you
  talked to it*. It updates only when you push, pull, or fetch — so it can be
  stale. If a teammate pushed an hour ago and you've done nothing since, your
  `origin/main` still points at the old commit.

`git pull` is really two steps: *fetch* (download origin's new commits and
update `origin/main`) then *merge* (`origin/main` into your `main`). The `-u`
on your first push links your `main` to `origin/main` as its **upstream**, so
from then on plain `git push` and `git pull` know where to go.

## The pull-request flow

A **pull request** (PR) is not a git command — it's a GitHub feature layered
on top of branches and pushing: a request that says *"please merge my branch
into main"*, plus a web page where the change is discussed. The rhythm every
team uses:

1. **Branch** — `git switch -c fix-typo`, commit your work there.
2. **Push the branch** — `git push -u origin fix-typo`.
3. **Open the PR** on the website: pick your branch, describe the change.
4. **Review** — teammates read the diff and comment; you push more commits
   to the same branch and the PR updates itself.
5. **Merge** — a button on the PR merges the branch into `main` *on the
   server*, exactly like your local `git merge`.
6. Everyone runs `git pull` on `main` and receives the change.

Why the ceremony? Review catches mistakes, the discussion becomes a written
record of *why*, and `main` stays the branch that always works.

## Exercise

You'll run the whole story locally: a repo, branches, one real conflict, and
a folder named `remote.git` standing in for GitHub. Work **inside this
lesson's `exercise/` folder**. Names and file contents must match exactly —
the checker looks for them. (If `git commit` complains "Please tell me who
you are", redo the identity setup from the previous lesson.)

1. In `exercise/`, create the repo: `mkdir repo`, `cd repo`,
   `git init -b main`. Create `list.md` containing exactly:

   ```
   # Shopping list

   - bread
   ```

   Commit it.
2. Create the stand-in GitHub and connect it:
   `git init --bare ../remote.git`, then
   `git remote add origin ../remote.git`, then `git push -u origin main`.
3. Branch like a PR: `git switch -c add-milk`, add the line `- milk` to the
   end of the list, commit, and push the branch with
   `git push -u origin add-milk`.
4. Merge it: `git switch main`, `git merge add-milk` (watch it fast-forward),
   `git push`.
5. Manufacture a conflict: on a new branch `weekend-plans`, change the first
   line to `# Weekend shopping list` and commit. Back on `main`, change the
   first line to `# Family shopping list` and commit.
6. `git merge weekend-plans` — it conflicts. Resolve so the first line reads
   exactly `# Family weekend shopping list`, finish the merge, `git push`.

Acceptance criteria (what `check.sh` verifies):

1. `repo/` is a git repository, you finish on `main`, and the working tree
   is clean.
2. Branches `add-milk` and `weekend-plans` exist and are both merged into
   `main`, and `main` contains a merge commit (the conflict merge).
3. `list.md` on `main` starts with `# Family weekend shopping list`, still
   lists `- bread` and `- milk`, and has no leftover conflict markers.
4. `remote.git` is a bare repository, `origin` points at it, your latest
   `main` and the `add-milk` branch are pushed, and local `main` tracks
   `origin/main`.

Run the checker from the `exercise/` folder, as often as you like:

```sh
cd exercise
bash check.sh
```

It fails before you start — each `FAIL` line tells you what to fix next.

## Further reading

- [Pro Git — Branches in a Nutshell](https://git-scm.com/book/en/v2/Git-Branching-Branches-in-a-Nutshell)
- [Pro Git — Working with Remotes](https://git-scm.com/book/en/v2/Git-Basics-Working-with-Remotes)
- [GitHub Docs — About pull requests](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests)
- [Learn Git Branching](https://learngitbranching.js.org/) — interactive visual practice
