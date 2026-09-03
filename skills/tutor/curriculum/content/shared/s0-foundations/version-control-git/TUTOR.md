# Tutor notes — Version Control with Git

## Where the learner is

Fourth lesson ever. They can navigate in a terminal, create files and folders,
and use an editor — nothing more. Git is their first *tool with internal
state*, and the three-place model (working tree / staging area / commit) is
the first invisible machinery they must trust without seeing. Expect the
staging area to feel like pointless ceremony at first; that objection is the
teaching moment, not a nuisance. Everything is local — remotes, GitHub, and
branches are the next lesson, so deflect those questions with "next lesson"
rather than answering early.

## Common misconceptions

- **"`git add` saves my file"** — saving happens in the editor (working
  tree); `add` stages; `commit` records. Learners who conflate these get
  "commits" missing their latest edits. Have them run `git status` between
  every command until the three places feel real.
- **"Commit uploads my code somewhere"** — everything in this lesson is on
  their disk. If they fear "publishing" broken code, remind them nothing
  leaves the machine yet.
- **"`git diff` is broken — it shows nothing"** — they staged the change;
  plain `diff` compares working tree to staging area. `git diff --staged` is
  the missing half. This one bites almost everyone.
- **"Adding a file to `.gitignore` removes it from Git"** — ignore rules only
  affect untracked files; a tracked file stays tracked, and old commits keep
  their contents. The check.sh secret-in-history failure is this
  misconception made visible.
- **"Deleting a file and committing erases it"** — history is append-only;
  the earlier snapshot still holds it. This is *the* reason secrets must
  never be committed.
- **`git init` in the wrong folder** — commonly the home directory, turning
  `~` into a repo that shadows everything. Symptom: `git status` lists
  thousands of untracked files. Fix: confirm with `pwd`, delete the stray
  `~/.git`, re-init inside `journal/`.
- **Identity not configured** — first commit fails with "Please tell me who
  you are". Point back to the One-time setup section rather than dictating
  the commands.

## Grilling points

- "A line you just typed in `notes.txt` — walk it through all three places
  until it's in history. Name the command that causes each hop."
- "Why does the staging area exist at all? Give me a concrete situation where
  committing straight from the working tree would produce a worse history."
- "What exactly is inside a commit?" (Snapshot, message, author, timestamp,
  link to the parent — the link is what makes history a chain.)
- "You committed a file containing a password, then deleted it and committed
  again. Is the password gone? Why not?"
- "Why is committing a compiled binary bad even though it's not secret?"
  (Regenerable, changes every build, noise that buries real edits.)
- "Run `git status` in your journal repo and read every line aloud — what is
  each section telling you to do?"

## Grading rubric

- **A** — check.sh fully green; commit messages genuinely descriptive (not
  10-character padding); learner narrates the three-place journey unprompted,
  explains `diff` vs `diff --staged`, and articulates *why* secrets are
  unrecoverable once committed.
- **B** — check.sh green; messages adequate; three-place model mostly solid
  but needs prompting on the staging area's purpose or on `--staged`.
- **C** — check.sh green only after heavy hinting, or learner drives by
  ritual ("I type add then commit") without the model. Pass only if a
  time-boxed re-explanation lands; otherwise one more iteration with a fresh
  folder.
- **Fail** — check.sh failing, or learner cannot explain what a commit is or
  why `.gitignore` exists. Rebuild the repo from scratch together, narrating
  every command.

## Remediation ladder

1. "Run `git status` inside `journal/` and read it aloud. Which of the three
   places is each file in right now?"
2. "check.sh told you exactly what's wrong — read the FAIL line's `fix:` hint
   and tell me which exercise step it maps to."
3. "The loop is always: `status` → edit → `diff` → `add` → `diff --staged` →
   `commit`. Run it once for `notes.txt` and tell me where your loop
   breaks."
4. Walk the failing step verbally — e.g. for the ignore step: "create the
   two files, run `git status`, write the two `.gitignore` lines, run
   `git status` again, then add and commit `.gitignore`" — but let them type
   every command.

If they committed `secret.txt`: don't teach history rewriting here. Have them
delete `journal/` entirely and rebuild — five minutes of practice, and the
lesson that prevention beats cleanup lands harder than any command would.

## After passing

Preview: "Your history lives only on this machine. Next lesson gives it wings
— branches to try ideas safely, and remotes to share the whole story with
other computers."
