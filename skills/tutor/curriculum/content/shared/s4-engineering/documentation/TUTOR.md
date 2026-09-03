# Tutor notes — Documentation

## Where the learner is

Final lesson of S4 — the last shared engineering lesson before S5 returns to
pure Go. They arrive straight from code review, so lean on that bridge:
review threads evaporate, documentation is the durable residue. They are a
capable intermediate Go programmer; nothing in the exercise code will
challenge them, and that is deliberate — the `snippets/` code is *good* so
that all the work is documentation judgment. If they start refactoring it,
redirect immediately: behavior and structure are off-limits, this is a
documentation pass. Verify is discussion: grade from the four rewritten
artifacts (README, doc comments, ADR, changelog entry), `NOTES.md`, and the
conversation.

## Reference answers

The grading source of truth — check the learner's *claims* against these
facts, not against their prose style.

**Contract facts for `package store`** (every doc comment must be true of
this, and the required edge cases covered):

- `Open`: a missing file is not an error — returns an empty store; the file
  is only created by the first `Add`. Errors: unreadable file, invalid JSON
  (both wrapped). It does not create directories.
- `Add`: empty name → error, nothing written; an existing name is
  **silently overwritten**; every successful call rewrites the whole file to
  disk immediately. Subtle bonus point: on a save error the in-memory map
  was already updated, so memory and file can diverge — credit anyone who
  documents or even spots this.
- `Get`: missing name → error wrapping the `ErrNotFound` sentinel; callers
  match with `errors.Is`. The returned error includes the name.
- `List`: names sorted alphabetically; empty (non-nil) slice for an empty
  store.
- `Store` / package: **not** safe for concurrent use — no locking anywhere.
  This must be documented somewhere sensible (type or package comment).
- Form: comment above the declaration, complete sentences, first sentence
  starts with the identifier (`// Open reads…`, `// ErrNotFound is
  returned…` — "is returned when" is the accepted idiom for sentinels).

**README must-haves**: one-paragraph purpose (what + who for); Go toolchain
requirement; real install commands (`go build .` / `go install`); a
quickstart whose commands and output match reality (`add`, `get`, `list`);
the `-path` flag with its default `snippets.json`; where data lives. The
stale `-dir` sentence and "see main.go" must be gone. The one true, useful
old sentence: the data file is plain JSON you can edit by hand — it should
survive in some form. Commands must have actually been run; spot-check by
asking them to run the quickstart cold.

**Changelog answer key** (audience = CLI users *and* importers of `store`):

- `9f3c2a1` -dir→-path — Changed, **breaking** for CLI users (flag gone,
  and its meaning shifted from folder to file); needs a migration line.
- `41d0b77` Get/ErrNotFound — Changed, **breaking** for importers (callers
  checking `== ""` now get an error instead).
- `b2e91c4` sorted List — Changed (user-visible ordering; accept "breaking
  for order-dependent importers" if argued, but don't require it).
- `7ac00d3` get command — Added.
- `c31f9e0` newline corruption — Fixed.
- `88aa412` refactor, `5b7ca29` comment typo, `e0d51f2` CI — **omit**: no
  user of the binary or importer behaves differently.

**Placement drill key**: install commands → README; `Get` missing-name
behavior → doc comment; JSON-vs-SQLite → ADR; `-dir` renamed → changelog;
concurrency safety → doc comment (type or package); worked first-snippet
example → README quickstart; future tag search → none of the four (issue
tracker — aspirational content is not documentation).

**ADR quality bar**: context names forces — single-user CLI, tiny data
(dozens of snippets), zero-dependency install, offline, hand-inspectable —
without smuggling in the conclusion. Options: SQLite via `database/sql` +
`modernc.org/sqlite` (real queries, concurrent-safe, survives partial
writes — but a dependency and an opaque file for a toy-sized dataset);
one-file-per-snippet directory (no parsing, easy per-snippet edits — but
listing/metadata scattered, odd filenames); single JSON file (wins on
simplicity and transparency). Costs that must appear (any two): whole-file
rewrite on every `Add`, everything in memory (O(n)), no safe concurrent
writers, a corrupt file loses everything, migration needed if data grows.
Superseding trigger (NOTES): multi-user/concurrent access, data too big for
memory, or query needs — then a new ADR supersedes 0001, never an edit.

## Common misconceptions

- **Aspirational docs.** Documenting the code they wish existed — claiming
  `Add` rejects duplicates, or "fixing" concurrency by writing "safe for
  concurrent use". Docs record what *is*; wanted changes go in NOTES part 5
  as proposed code changes. If they *edited behavior* to match nicer docs,
  that's a ground-rule violation, not initiative.
- **Longer echo = contract.** "Get returns the snippet stored under the
  given name from the store" is still an echo, just wordier. Apply the
  falsifiability test sentence by sentence.
- **Changelog = reformatted git log.** All nine commits, refactor and CI
  included. Run the filter with them: does any user behave differently?
- **Breaking changes buried** mid-list or phrased from the inside
  ("refactored flag handling") instead of first, loud, with a migration.
- **The ADR as advertisement.** Consequences listing only benefits, or
  context written backwards from the winner ("JSON is simplest, so…").
  Forces first, and at least two honest costs.
- **Editing ADRs when decisions change.** The log is append-only:
  supersede, don't rewrite history.
- **README as full reference** — every flag, every function, duplicated
  from the doc comments. That's a routing failure: one home per fact; the
  README shows one example and points onward.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go
deeper):

- "Your README says a stranger can install this. Prove it: from an empty
  directory, run only what the README says. Where's the first guess?"
- "Pick one sentence from your `Open` comment. What test would catch it if
  it were a lie?" (Falsifiability applied — then ask for a sentence they
  *cut* because nothing could falsify it.)
- "Both `41d0b77` and `88aa412` changed `store.go`. One is in your
  changelog, one isn't — defend the line between them."
- "A teammate reads ADR 0001, disagrees, and opens a PR moving to SQLite.
  What happens to ADR 0001 — and what should the PR description cite?"
- "Where did you document that `Store` isn't goroutine-safe, and why there
  rather than the README?" (Reader's moment: the caller is mid-code.)
- "Which of your four artifacts rots fastest, and what mechanism keeps each
  honest?" (README — run its commands, CI can too; doc comments — same-diff
  adjacency plus review; changelog — written at merge time; ADR — immutable
  by design.)

## Grading rubric

- **A** — README passes a live stranger test (commands run verbatim, purpose
  says what *and* who for, stale `-dir` gone, hand-editable-JSON fact kept);
  all seven doc comments in Go form, every required contract point present
  and true, no echo sentences; ADR has honest forces, two real rejected
  options, and two genuine costs; changelog matches the answer key with both
  breaking items first and migration lines, noise omitted with reasons;
  placement drill fully justified including the trick item; defends every
  call fluently.
- **B** — All four artifacts land but with soft spots: one contract point
  missing (usually concurrency or the silent overwrite), a README command
  they clearly never ran, only one real cost in the ADR, one noise commit in
  the changelog or the `b2e91c4` classification unargued, or a placement
  answer by vibes. Solid once nudged in conversation.
- **C** — Artifacts written but judgment thin: doc comments wordier echoes;
  README a wall duplicating the reference; changelog a reformatted git log
  or breaking changes unremarked; ADR argues for the decision instead of
  recording it. Pass only if live remediation lands — have them re-derive
  `Get`'s contract from the body and re-classify two commits on the spot;
  otherwise another iteration.
- **Fail** — Documentation makes false claims about the code (aspirational
  or unverified), behavior was edited to match the docs, or an artifact is
  missing. Redo the relevant part together before re-discussing.

## Remediation ladder

1. "Close the code. Using only your README, write down the exact commands
   from a fresh checkout to your first `get` printing its snippet. Circle
   every place you guessed — each circle is a README gap."
2. "Take your `Add` comment. For each sentence, point at the line in the
   body that makes it true. A sentence with no line is either an echo (cut)
   or a hope (verify or cut)."
3. "For each commit in HISTORY.txt ask one question: does someone running
   the binary, or importing `store`, behave differently because of this?
   Sort the nine commits into yes/no out loud."
4. Build the ADR verbally with them: name the forces *before* naming any
   option; for each loser finish 'we rejected X because…', for the winner
   finish 'and we accept the cost of…'. Then they write it alone.

## After passing

Preview: "That closes Engineering Practice — S4 done. Next stage is Go again
and starts with HTTP servers: `net/http` from the serving side, the mirror
image of the clients you hardened two lessons ago."
