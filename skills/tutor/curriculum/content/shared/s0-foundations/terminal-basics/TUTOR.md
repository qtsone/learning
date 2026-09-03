# Tutor notes — The Terminal

## Where the learner is

Second lesson ever, and the first hands-on one. They understood programs
conceptually in `what-is-a-program` but may never have opened a terminal in
their life. They have no editor yet (next lesson) and no git — every file
operation must happen through the shell, which is exactly the point. In
`guided` mode, celebrate the first successful `ls` like it matters, because
it does. Nudge toward `man` and tab completion instead of dictating commands;
dictating defeats the lesson.

## Common misconceptions

- **Terminal and shell conflated** — "the terminal ran my command". The window
  draws text; the shell (zsh/bash) interprets and runs. If fuzzy, re-anchor
  with the telephone analogy from LESSON.md.
- **"Where my files are" vs working directory** — they don't realize each
  shell sits at exactly one place, and two windows can sit in different
  places. Most early errors are location errors; train the `pwd` reflex.
- **Relative paths treated as global** — expecting `cd trip` to work from
  anywhere. Have them say the resolved absolute path out loud.
- **rm sends files to Trash** — it does not. Make them repeat this before they
  run their first `rm`.
- **cp vs mv blur** — after `mv`, they look for the file in the old place; or
  they `cp` when asked to move and are surprised check.sh complains about
  leftovers.
- **"command not found" = "my computer is broken"** — walk the PATH story:
  the shell searched a list of directories and found nothing; typo, not
  installed, or installed off-PATH.
- **Stuck in the `man` pager** — they don't know `q`. Everyone hits this once;
  make it a joke, not a failure.
- **`>` appends** — it truncates. If they clobber `location.txt` while
  experimenting, that's a teachable moment, not a setback.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "You have two terminal windows open. Can they be in different working
  directories? What does that tell you about where 'current directory'
  lives?"
- "Walk me through exactly what the shell does when you press Enter on
  `ls trip` — starting from how it finds `ls` at all."
- "Why did `rmdir mess` refuse until mess/ was empty, and why is that a
  feature rather than an annoyance?"
- "A Linux tutorial's `ls` flags error on your Mac. What's going on, and
  where's the authoritative answer?" (BSD vs GNU userland; local `man`.)
- "If you had run `pwd > location.txt` from `exercise/` instead of `trip/`,
  what would the file contain, and how did check.sh catch it?"

## Grading rubric

- **A** — check.sh fully green; learner navigated with relative paths and tab
  completion, found the hidden-files flag via `man ls` themselves, and can
  explain every command they ran, including why the photos must be absent
  from `mess/`.
- **B** — Green, but they leaned on copy-pasting command shapes from the
  lesson or needed the `-a` flag revealed; explanation mostly solid, minor
  confusion on absolute vs relative.
- **C** — Green only after heavy hinting, or explanation shows commands are
  incantations (can't say what `mv` did vs `cp`). Pass only if a short
  remediation pass lands; otherwise iterate.
- **Fail** — check.sh not green; or they rearranged files in Finder/a file
  manager instead of the shell (ask casually how they did it — the checks
  can't tell); or they cannot distinguish absolute from relative paths.

## Remediation ladder

1. "Run `bash ./check.sh` and read only the first FAIL aloud. In your own
   words, what is it asking for?"
2. "Before running anything: where are you right now? What does `pwd` say,
   and where do the paths in your command start from?"
3. "Open `man <command>` for the step you're stuck on and read me the first
   sentence of its description. What are its two arguments here?"
4. Give the exact command shape for the current step only (e.g.
   `mv mess/draft.txt trip/notes/`), have them type it — never paste it for
   them — then re-run check.sh together.

## After passing

Preview: "Next lesson you get a real editor — no more `echo >` to write
files. The terminal doesn't go away; it lives inside the editor from now on."
