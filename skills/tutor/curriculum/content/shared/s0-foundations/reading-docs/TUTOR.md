# Tutor notes — Reading Docs & Error Messages

## Where the learner is

Seventh lesson of S0, still zero programming. They can navigate a terminal,
use man pages mechanically, run the basic git workflow, and have a rough
client/server picture of the web. Go is NOT installed yet (dev-environment is
next) — the Go compiler error in this lesson is read-only, a preview. This is
a meta-skill lesson; the win condition is a changed relationship with errors:
from "scary red text" to "the tool telling me exactly what to do". Verify is
discussion — grade from their `NOTES.md` and the conversation around it.

## Common misconceptions

- **"An error means I broke something"** — beginners fear errors as damage.
  Nothing in this exercise can harm their machine; errors are output, not
  consequences. If they hesitated to run part 1, address this first.
- **"Errors are unreadable expert-speak"** — they skip to searching or asking
  without reading. Make them read `cat: recipes.txt: No such file or
  directory` aloud; every word is plain English once split into who/what/where.
- **"First hit on a search = the docs"** — many land on SEO tutorial farms or
  five-year-old blog posts and treat them as ground truth. Push on *domain*
  and *who maintains it*.
- **"More errors = more problems"** — they try to fix a whole cascade at
  once. Reinforce first-error-first, re-run after each fix.
- **Stack trace read upside down** — blaming `main` because it's listed, or
  fixating on the deepest frame even when it's library code. The heuristic is
  "deepest frame that is *your* code", not "top" or "bottom".
- **Paraphrasing errors from memory** when asking or searching — insist on
  copy-paste verbatim; paraphrases destroy the exact strings search engines
  and helpers need.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "In part 1, git and cat both failed to find something. How did each *tell*
  you what it looked for, and where?" (git names `.git` and the parent-dir
  search; cat names the file.)
- "For error A, why not start reading at `main`, line 8? When *would* the
  calling frames matter?" (When the deepest frame is correct code being
  handed bad input from above.)
- "You need to know whether `git log` has a flag to show one line per commit.
  Tutorial, how-to, reference, or changelog — and why?"
- "A blog post's git instructions don't work on your machine. Give me two
  different explanations, and which document you'd check for each." (Post is
  wrong → reference; version drift → changelog/release notes.)
- "Why does the exact error text go in quotes when searching, and why strip
  your own file names out of it?"
- "What happened when you wrote the part 4 question? Why does that work?"

## Grading rubric

- **A** — All NOTES.md slots filled with specifics (actual message text, real
  URLs); part 1 errors reproduced *and* fixed; stack trace read correctly with
  the "deepest frame of my code" reasoning; scavenger sources genuinely
  official and correctly typed; part 4 question has all four ingredients plus
  minimal steps, and they spotted the `git add` fix while writing it (or can
  explain the rubber-duck effect regardless).
- **B** — Everything attempted and mostly right, but with a soft spot: a
  vague classification in part 3, a question missing one ingredient, or
  trace-reading that needed one nudge. Solid in conversation.
- **C** — Notes filled by going through the motions: paraphrased errors,
  "google" cited as a source, question written as a plea ("git broken, help").
  Pass only if live remediation lands — have them re-dissect one error and
  rewrite the question on the spot.
- **Fail** — Didn't run part 1 (fear or skipped), can't split any error into
  who/what/where even with prompting, or sees no difference between a random
  blog and the reference. Redo the exercise together, then re-discuss.

## Remediation ladder

1. "Read the error out loud, word by word. Which word is a program's name?
   Which words name a thing on your disk?"
2. "The message says what the tool *looked for*. Where did it look? Is the
   thing actually there — check with `ls`."
3. "For the trace: tell it as a story starting from `main` — who called
   whom? Now, where did the story *end*?"
4. For part 4: dictate the four ingredient headings into their draft and have
   them fill each one from their terminal scrollback — don't write the
   sentences for them.

## After passing

Preview: "Next lesson you build your dev environment — install the Go
toolchain, verify it with `go version`, and learn the project hygiene the
whole Go track sits on. Those go.dev docs you found? You're about to use them."
