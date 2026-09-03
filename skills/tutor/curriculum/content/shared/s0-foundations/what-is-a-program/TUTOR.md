# Tutor notes — What Is a Program?

## Where the learner is

The very first lesson of the whole roadmap. Assume zero programming knowledge,
zero terminal experience — ordinary computer literacy only (files, folders, a
browser). Nothing is installed yet and nothing needs to be. Define every term
on first use, go slower than feels necessary, and treat the paper-machine
trace as a confidence win, not a test. In `guided` mode, offer to do the first
loop pass of the trace together.

## Common misconceptions

- **"The computer understands my code"** — it only ever executes machine
  code; source text must be translated. If this is fuzzy, the whole lesson is
  fuzzy — revisit the CD-player/sheet-music analogy.
- **"The CPU is smart"** — it's fast, not smart. Each instruction is trivial;
  the magic is billions of trivial steps per second.
- **Memory vs disk confusion** — "memory" read as "where my files are". Files
  persist on disk; memory is the fast workspace wiped when the program ends.
- **"Interpreted code isn't translated"** — it is, just piecemeal and at run
  time, by the interpreter program the CPU is actually executing.
- **Loop off-by-one in the trace** — printing 32 (four doublings) or reaching
  step 7 with B at 1 or -1. The fix is discipline, not insight: one row per
  executed instruction, check step 3's condition every pass.
- **Treating the jump as "repeat"** — step 6 doesn't mean "loop"; it only
  moves the marker. The repetition *emerges* from 6 jumping back and 3
  deciding when to stop. This distinction is the seed of all control flow.

## Grilling points

Ask in the learner's own words (quiz.json has the core set; these go deeper):

- "Why can't the CPU execute the text you write, directly?"
- "A friend says Python 'just runs' without any translation. What's really
  happening when a Python program runs?"
- "In the paper machine, what decides which instruction runs next? Where did
  you feel that while tracing?"
- "Where is a program *before* you run it, and where is it *while* it runs?
  Why move it at all?"
- "You typo a word in a compiled language vs an interpreted one — when does
  each tell you, and why does that follow from the design?"
- "In the six-step run trace, what does the operating system do that the
  program couldn't do for itself?"

## Grading rubric

- **A** — Trace table correct (prints 64, step 4 ran five times); Part 2
  change is a single number proven by a second trace (either works: step 2
  putting 2 in B, or step 3's guard changed from 0 to 3 — both print 8);
  step-6-deleted question answered by tracing (prints 4); identifies
  themselves as an interpreter with a sound reason; explains
  compiler-vs-interpreter and the CPU/memory split unprompted and correctly.
- **B** — Trace correct, perhaps after one slip they caught themselves;
  Part 2 works but the reasoning wobbled (guess-then-fix is fine, untraced
  guessing is the wobble); core concepts explained with minor imprecision
  (e.g. vague on where the interpreter fits).
- **C** — Trace completed only with heavy hinting, or concept explanations
  lean on memorized phrases ("compilers translate ahead of time") that
  collapse under one follow-up. Pass only if time-boxed remediation lands;
  otherwise another iteration with a fresh paper program.
- **Fail** — Cannot trace the loop even with help, or believes the CPU runs
  source text directly after discussion. Remediate; do not advance — every
  later lesson stands on this model.

## Remediation ladder

1. "One row per instruction, no exceptions. What instruction is your marker
   on right now? Do only that, write the boxes, move the marker."
2. "Steps 3-6 are a cycle. Each time around: what happens to A, and what
   happens to B? Track B — it's counting something down."
3. "Let's do two passes together: A goes 2, then 4, then 8; B goes 5, 4, 3.
   Now you finish, and tell me what step 3 is waiting for."
4. Walk the full table with them narrating each move, then have them redo
   Part 2 (the print-8 change) entirely alone — that's the real check.

## After passing

Preview: "Next lesson you meet the terminal — the window where you'll type
real run commands instead of tracing them on paper."
