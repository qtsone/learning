# Version Control with Git

> `shared.foundations.version-control-git` · ~2-4h · Stage: Foundations

## Objectives

By the end of this lesson you can:

- Explain why version control exists and what problems it solves compared to
  copying folders around.
- Initialize a repository and record changes using `git init`, `git add`, and
  `git commit` with meaningful messages.
- Inspect history and pending changes using `git log`, `git status`, and
  `git diff`, and explain what each shows.
- Explain the difference between the working tree, the staging area, and a
  commit.
- Write a `.gitignore` that excludes build artifacts and secrets, and explain
  why they must not be committed.

## The folder-copying trap

You have probably already invented version control, badly. It looks like this:

```
report.docx
report_final.docx
report_final_v2.docx
report_final_v2_REALLY_final.docx
```

Copying files "just in case" works until it doesn't. Which copy has the fix
you made last Tuesday? What exactly changed between v2 and REALLY_final?
If two people edit copies at the same time, who wins? Folder copies answer
none of these questions — they only hoard.

A **version control system** (VCS) solves this properly. It keeps *one* folder
and, alongside it, a **history**: a chain of snapshots, each labeled with who
made it, when, and — crucially — *why*. You can compare any two snapshots,
rewind to an old one, and see the entire story of a project. **Git** is the
version control system essentially the whole software world uses.

## What a repository is

A Git **repository** ("repo") is an ordinary folder that Git watches. When you
run `git init` inside a folder, Git creates a hidden subfolder called `.git`
and stores the entire history in there. Two rules about `.git`:

- Never edit or delete anything inside it by hand. Delete `.git` and the
  history is gone — the files you see stay, but their story vanishes.
- Everything lives on *your* machine. Nothing in this lesson touches the
  internet; sharing history with other computers is the next lesson.

A snapshot in that history is called a **commit**. A commit records the state
of your files at one moment, plus a message you write, your name, and a link
to the commit that came before it. That link is what makes history a chain.

## One-time setup

Check that Git is installed (on macOS this may offer to install developer
tools — accept; on Linux, install the `git` package with your package
manager):

```sh
git --version
```

Then tell Git who you are. Every commit is labeled with this, and Git refuses
to commit until it's set:

```sh
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

`--global` means "for every repository on this machine" — you do this once.

## The three places

This is the mental model the whole lesson hangs on. A change travels through
three places on its way into history:

```
 working tree            staging area                history
 the files you   --add-->  the next     --commit-->  permanent
 actually edit             commit, being             snapshots
                           assembled                 (commits)
```

- The **working tree** is the folder as you see it — the real files your
  editor saves.
- The **staging area** (also called the *index*) is a draft of your next
  commit. `git add <file>` copies a file's current state into the draft.
- A **commit** seals the draft into history, with your message attached.

Why the middle step? Because "everything I've touched lately" rarely makes
one coherent snapshot. The staging area lets you *choose* what goes into each
commit — fix and typo-cleanup as two commits, not one blur. Deliberate
snapshots are what make history readable later.

One trap worth naming now: `git add` does not "save" anything permanently,
and your editor's Save does not tell Git anything. Save puts your edit in the
working tree; `add` stages it; only `commit` records it.

## The recording loop

Day-to-day Git is a loop of four commands. First, the two that only *look*:

```sh
git status    # where is every changed file right now?
git diff      # exact line-by-line edits not yet staged
```

`git status` names each file's place in the three-place model: *untracked*
(Git has never been told about it), *modified* (changed in the working tree),
or *staged* (in the draft of the next commit). Read its output slowly — it
literally tells you what command to run next.

`git diff` shows what changed, line by line: `-` lines removed, `+` lines
added. After you stage a file, plain `git diff` shows nothing for it — the
edit has moved to the draft. To review the draft itself, run
`git diff --staged`. Then the two that *act*:

```sh
git add notes.txt        # stage this file's current state
git commit -m "Add opening summary to notes"
```

The text after `-m` is the **commit message**. Write it for a stranger — or
for yourself in six months, which is the same person. "fix", "stuff", and
"wip" tell that person nothing. A good message says what changed and, when it
isn't obvious, why: "Correct tax rate in March invoice totals". If you can't
summarize the change in one line, that's a hint it should be two commits.

## Reading history

```sh
git log            # every commit: author, date, full message
git log --oneline  # one line each: short id + message
```

Each commit is identified by a **hash** — a long string like `f4a1c9e…` that
Git computes from the commit's entire content. You'll use short prefixes of
these ids later to point at specific commits; for now, `--oneline` is the
quickest way to see the story of a repo. If `git log` fills more than a
screen it opens a pager: arrow keys scroll, `q` quits (as with `man` in the
terminal lesson).

## What never gets committed

Two kinds of files must stay out of history:

- **Build artifacts** — files a tool *generates* from your source (compiled
  programs, packaged output, logs). They're bulk without information: anyone
  with the source can regenerate them, and they change on every build,
  burying your real edits in noise.

  > **In Go:** once you reach the Go track, `go build` drops a compiled
  > binary named after the folder into your project — a classic artifact to
  > ignore.

- **Secrets** — passwords, API keys, private tokens. Here is the part that
  burns people: *commits are forever*. Deleting the file in a later commit
  does not remove it from the earlier one; anyone with a copy of the repo can
  read every old snapshot. And repos get shared — you'll do exactly that next
  lesson. The only safe secret is one that never entered history.

You keep such files out with a `.gitignore`: a plain text file, one pattern
per line, committed like any other file so the rules travel with the repo:

```
secret.txt
build/
```

`build/` (with the trailing slash) ignores that folder and everything in it.
Ignored files still exist on disk — Git simply stops suggesting them, so
`git status` stays quiet about them. Note the limit: `.gitignore` only
affects *untracked* files. If you've already committed a file, adding it to
`.gitignore` does not un-track it — and whatever it contained is already in
history.

## Exercise

Everything happens inside [`exercise/`](exercise/), in the terminal. You will
build a small journal repo; `check.sh` is the referee.

1. In `exercise/`, create a folder `journal`, `cd` into it, and run
   `git init`. (Git may print a hint about branch names — ignore it; branches
   are next lesson.)
2. Create `notes.txt` in your editor with a sentence or two about this
   lesson. Walk it through the loop: `git status`, `git add notes.txt`,
   `git status` again — notice the file move places — then commit with a real
   message.
3. Edit `notes.txt` again (add a few lines). Use `git diff` to see your edit,
   stage it, confirm with `git diff --staged`, and commit.
4. Create `secret.txt` containing a pretend password, and a folder `build`
   containing a file `output.txt` (pretend compiled output). Run
   `git status` — both appear. Write a `.gitignore` that ignores them, watch
   `git status` go quiet about them, then stage and commit `.gitignore`.
5. From `exercise/` (one level up from `journal`), run the checker until it
   passes: `bash check.sh`.

Acceptance criteria (exactly what `check.sh` verifies):

1. `journal/` is a Git repository with at least 3 commits.
2. Every commit message is at least 10 characters — no "wip"-grade messages.
3. `notes.txt` is tracked and appears in at least 2 commits.
4. `.gitignore` is committed; `secret.txt` and `build/` exist on disk but are
   ignored, untracked, and appear nowhere in history.
5. `git status` reports a clean working tree — nothing left half-done.

Expect `check.sh` to FAIL before you start — each failure tells you what to
fix. Run it after every step if you like; it never changes anything.

## Further reading

- [Pro Git — About Version Control](https://git-scm.com/book/en/v2/Getting-Started-About-Version-Control)
- [Pro Git — Recording Changes to the Repository](https://git-scm.com/book/en/v2/Git-Basics-Recording-Changes-to-the-Repository)
- [gitignore documentation](https://git-scm.com/docs/gitignore)
