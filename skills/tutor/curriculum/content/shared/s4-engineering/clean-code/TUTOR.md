# Tutor notes — Clean Code

## Where the learner is

First lesson of S4 and their first *judgment* lesson: until now nearly every
exercise had a test deciding right from wrong; here the tests only guard
behavior, and quality is argued, not asserted. They come from S3 as an
idiomatic intermediate Go programmer (interfaces, generics, io, JSON, a full
concurrency arc) with two capstones behind them — so they can read and
refactor `report.go` easily; the work is in the *reasoning*. Verify is
discussion: grade from `NOTES.md`, the diff of `report.go`, and the
conversation. Expect capable coders who have never had to *defend* a rename;
push every "obviously better" to "better by which criterion?".

## Reference refactor shape

For grading — the destination, not the only acceptable one:

- `record` struct (`name`, `category`, `amount`); `parseRecord(line) (record,
  bool)` or `parseLines([]string) ([]record, int)` returning records plus a
  skipped count.
- An aggregation step producing a small stats struct (count, total, largest,
  per-category totals) — name like `summarize`.
- A formatting step rendering the stats — name like `format` / `render`.
- Exported entry renamed to something like `Build` or `Summarize`
  (`report.Build` — no stutter; `DoData` fails intent-revealing), reading as
  parse → summarize → format, with a contract doc comment covering blank
  lines (skipped silently), malformed lines (counted, reported in the
  `skipped:` line), and the no-valid-records error.
- Helpers stay unexported. The names that were fine: `k` (three-line loop
  scope) and `err` (fixed idiom). `sb` is a defensible third — it spans ~20
  lines, past the usual scope-length license, but it is the stock Go idiom
  for `strings.Builder`; accept it when they argue convention, push back if
  they only say "it's short". Accept `i`-class answers only with the
  scope-length argument.
- The one justified original comment is the trailing-newline "why" on the
  blank-line skip — it must survive (possibly reworded/moved). The
  commented-out `DoData` block and all narration must be gone.

## Common misconceptions

- **"Clean = short."** They golf the function down or split into six
  two-line helpers. Redirect to *reasons to change*: over-splitting is the
  smell part 4 explicitly asks them to reason about. A once-called helper
  whose name restates its body is a cost, not a win.
- **"gofmt made it clean."** The starter is deliberately gofmt-clean and
  still awful — use that as the wedge between uniform and clean.
- **"More comments = better documented."** They may *add* narration during
  the refactor out of diligence. Every comment must pass "does it say
  something the code cannot?".
- **"Short names are always bad"** after the naming section — watch for
  renaming loop-`k` to `categoryName` or `err` to `parseError`. That misses
  the scope-length criterion, which *licenses* short names in short scopes.
- **"Renaming the exported function breaks the tests, so I can't."** Updating
  call sites is part of a rename; the README says so explicitly. If they left
  `DoData` untouched for this reason, they dodged the API-design decision.
- **Behavior drift during the split** — e.g. reordering output lines,
  changing `skipped:` wording, or "fixing" the strict `>` in the max
  comparison (first-seen wins ties). The characterization test catches these;
  if they edited the *expected output* to make it pass, that is a fail-level
  misunderstanding of refactoring.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "Pick your favorite rename. Which criterion justifies it — and give me a
  context where the *old* name would have been acceptable." (Scope-length
  reasoning cuts both ways.)
- "You split the function three ways. Why not six? Show me the split you
  rejected and what it would have cost." (Parameter threading, unnameable
  fragments, once-called helpers.)
- "Which requirement change touches only `parseRecord` in your version? Which
  touches only the formatter? What did changing the input format cost in the
  *original*?" (Reasons-to-change made concrete.)
- "Defend keeping the trailing-newline comment while deleting `// parse the
  amount`. What's the test that separates them?" (Says something the code
  cannot.)
- "Why did the record struct beat the loose variables `n`, `grp`, `a`?"
  (Data that travels together, named as one concept; parameter lists shrink.)
- "The tests passed before you started. What were they *for*, then?"
  (Characterization: pinning behavior so cleanup can't silently change it.)

## Grading rubric

- **A** — Eight-plus real smells found, correctly tagged, plan order
  justified (e.g. tests-guarded renames first, structure last); rename table
  cites the *right* criterion per row; `k`/`err` spared with the scope
  argument (`sb` spared-with-convention also earns full marks); comment triage keeps exactly the why-comment, kills all
  narration and the dead block, and the new doc comment states edge-case
  behavior; split lands on the three-seam shape with a defensible rejected
  split; tests green, outputs untouched; they argue every call fluently.
- **B** — Cleanup lands and tests stay green, but reasoning has soft spots:
  criteria misattributed on a rename or two, one narration comment surviving
  or the why-comment deleted, the rejected-split answer thin, or the doc
  comment true but silent on edge cases. Solid in conversation once nudged.
- **C** — Code improved but notes are vibes ("clearer this way") rather than
  criteria; comment triage indiscriminate (kept everything or deleted
  everything, why-comment included); split done mechanically without being
  able to say what each piece's reason to change is. Pass only if live
  remediation lands — make them re-derive one rename and the comment test on
  the spot; otherwise another iteration on the notes.
- **Fail** — Expected test outputs edited to make the refactor "pass",
  exported API left as `DoData` to avoid touching tests, or they cannot
  distinguish narration from a why-comment even when prompted. Redo the
  relevant part together before re-discussing.

## Remediation ladder

1. "Cover the body of `DoData` and read only its name and signature. What do
   you *know* about what it does? That gap is the rename backlog."
2. "List the changes a product owner could ask for — new input column, new
   statistic, different layout. For each: which lines would you edit? Circle
   them. How many distinct regions did you just circle?"
3. "For each comment ask: if I delete it, is information *lost* that the code
   can't carry? Try it on `// loop over the lines`, then on the
   trailing-newline comment. Different answer? That's the whole discipline."
4. Sketch the three-function shape with them verbally — parse to a struct,
   summarize to a stats value, format to a string — then have them do the
   extraction alone with the tests running between each step.

## After passing

Preview: "You cleaned up the inside of one file. Next lesson zooms out — how
Go projects arrange packages, where module boundaries go, and why
dependencies must point one way — the same 'one reason to change' idea at
project scale."
