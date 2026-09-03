# Clean Code

> `shared.eng.clean-code` · ~2-3h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Rename a set of poorly named identifiers and justify each choice using
  intent-revealing naming criteria.
- Explain when a long function should be split and when splitting hurts
  cohesion, with a concrete example of each.
- Identify low-cohesion code in a sample and refactor it so each unit has a
  single reason to change.
- Classify comments in a code sample as justified (why, warning, contract) or
  deletable (what-narration), and defend each call.
- Critique a provided function against naming, size, and cohesion criteria and
  propose a prioritized cleanup.

## Code is read far more than it is written

Everything you built in the last three stages was judged by one question: does
it work? From this stage on, a second question carries equal weight: can the
next person work on it? That person is usually you, six months later, with no
memory of what `t` stood for.

Studies of professional codebases put the read/write ratio around ten to one —
for every line typed, ten are re-read while navigating, debugging, and
reviewing. So an hour "wasted" making code clearer pays back every time anyone
opens the file. "Clean" is not an aesthetic judgment; it is an engineering
trade-off, and this lesson gives you the three levers that matter most —
naming, function size, cohesion — plus the discipline for the one thing that
sits outside the code itself: comments.

One boundary up front: a formatter (you have run `gofmt` since your first
lesson) makes code *uniform*, not *clean*. Layout is mechanical; meaning is
not. Every problem in this lesson survives formatting untouched.

## Names that reveal intent

A name is the interface to a piece of code: readers use the name *instead of*
reading the body. A good name answers, at its call site, "what is this and why
does it exist?" — without a trip to the definition. Concrete criteria you can
justify a rename with:

- **Intent-revealing.** `elapsedDays` over `d`; `isRetryable` over `flag`.
  If the name needs a comment to explain it, the name has failed.
- **Not disinformative.** A name must not imply something false: `accounts`
  for something that isn't a collection, `list` for a map, `l` in a font where
  it reads as `1`. Wrong information is worse than no information.
- **One word per concept.** Pick `fetch` *or* `retrieve` *or* `get` for the
  same idea and use it everywhere. A codebase that has all three forces every
  reader to wonder whether the difference is meaningful.
- **Distinct where it matters.** `data` vs `data2`, `tmp`, `info` — noise
  words and number suffixes tell the reader "these differ" without saying how.
- **Searchable.** You will grep for names (S0's terminal lesson pays rent
  here). `MaxRetries` is findable; `m` is not.
- **Length proportional to scope.** This is the rule that reconciles the
  above with the short names you've seen everywhere in Go: a loop index alive
  for three lines can be `i` — the declaration is on screen, the name carries
  no load. The same `i` as a field on a long-lived struct is a defect. The
  further a reader can be from the declaration, the more the name must carry.

**In Go:** the conventions you've absorbed since S1 are these criteria
applied. `MixedCaps`, never underscores; short receiver names (`func (s
*Server)`) because the receiver's scope is one function and its type is a line
away; `err` and `ok` as fixed idioms; and no package stutter — a name is
always read *with* its package qualifier, so `report.Builder`, not
`report.ReportBuilder`. Effective Go's advice that a name's length should
match its scope is exactly the last criterion above.

## Function size: split on reasons, not line counts

You will hear rules like "functions must be under N lines." Ignore the number;
it is a proxy for two real signals.

**Split when abstraction levels mix.** A function should read as steps of one
story at one altitude. This shape:

```
process(lines):
    # -- parse --
    ...15 lines of field splitting and validation...
    # -- aggregate --
    ...15 lines of totals and maximums...
    # -- format --
    ...15 lines of output assembly...
```

is three functions wearing a trenchcoat — and the section comments are the
confession. Each block is a separate *reason to change*: a new input format
touches only parsing; a new statistic touches only aggregation; a layout
change touches only formatting. Split it, and each function becomes
independently understandable, testable, and changeable, while the parent
shrinks to a legible table of contents: `parse`, then `aggregate`, then
`format`.

**Don't split when the pieces can't stand alone.** Splitting has a price:
every extraction adds a name to invent, an indirection to follow, and a
parameter list to thread. A cohesive 30-line algorithm — a binary search from
S2, a partition step from your sorting lesson — is *one* idea; carving it into
`checkBounds`, `computeMid`, `adjustRange` leaves fragments that are
meaningless alone and force the reader to mentally re-inline them to
understand anything. The tell-tale signs you have over-split: helper functions
called exactly once whose names just restate their bodies, and fragments that
need five parameters or shared mutable state to communicate. If the extracted
piece doesn't have a name you'd stand behind and a job you can state without
saying "and", leave it inline.

The honest test is never the line count. It is: *how many distinct reasons to
change live in here?* One reason, thirty lines — fine. Three reasons, fifteen
lines — split it.

## Cohesion: one reason to change

That test generalizes beyond functions. A unit — function, type, module — is
**cohesive** when everything inside it serves one purpose, and it has, in the
classic phrasing, *a single reason to change*. "Reason to change" is a
requirement-shaped force: the input format, the tax rules, the output layout,
the retry policy. List the forces that would make you edit a unit; if you
count more than one, changes arrive tangled — you cannot touch the parsing
without re-reading (and risking) the formatting sitting next to it, and two
unrelated changes collide in the same function.

Low cohesion rarely announces itself; you detect it with questions:

- Which requirement changes would touch this unit? (More than one force —
  low cohesion.)
- Do these lines share data and purpose, or just a location? A function whose
  first half never exchanges data with its second half is two functions.
- Does part of this code keep reaching into *another* unit's data to do its
  work? Then it probably belongs there — move it to where the data lives.

The refactor is the one from the previous section, applied deliberately:
separate along the reason-to-change seams, name each piece for its single
purpose, and let data flow between them through parameters and results
instead of shared mutable state. You proved in S1 and S3 that functions
returning values beat functions with side effects for testability — cohesion
is the same force at design scale.

**In Go:** cohesion also has a project-scale form — which code shares a
package, and which direction imports may point (Go refuses import cycles
outright, as the interfaces lesson hinted). That scale is exactly the next
lesson; here you practice cohesion inside a single file first.

## Comment discipline

Comments are the only part of a codebase nothing checks: the compiler ignores
them, tests can't fail on them, and when the code changes, nothing forces the
comment to follow. A stale comment is disinformation with authority. So the
bar is: **a comment must say something the code cannot.** Three kinds clear it:

- **Why.** Rationale, trade-offs, links to the bug that forced this shape.
  `// Retry once: the vendor API drops ~1% of first attempts.` The code shows
  *that* you retry; only the comment can say *why once*.
- **Warning.** Code that looks wrong or removable but is load-bearing.
  `// Do not reorder: Close must run before the final flush.`
- **Contract.** Documentation on a public boundary, stating what callers may
  rely on — including edge cases the signature can't express.

And one kind never clears it: **narration** — comments that restate the code
in English. `// loop over the lines`, `// increment the counter`. Narration
is written for the author's comfort and read by no one; worse, it rots. Every
urge to narrate is a design signal: rename or extract until the code says it,
then delete the comment. Commented-out code is narration's zombie cousin —
version control (S0) remembers every deleted line; the graveyard in the file
just scares readers who can't tell whether it's coming back.

**In Go:** contracts have a concrete form — doc comments. Every exported
identifier gets a `// Name ...` comment, a complete sentence starting with the
name, and `pkg.go.dev` renders it as the package's documentation; you have
been reading these since S1. Note what the convention does *not* license: a
doc comment that merely echoes the signature (`// GetUser gets a user.`) is
narration in a contract's clothing. State what the caller can rely on —
behavior at the edges, what happens on empty input, what errors mean.

## Refactoring without breaking things

The exercise asks you to change working code, which raises the question every
professional refactor answers first: how do you know you didn't break it? With
tests — you've written table-driven tests since S1. The exercise ships a
**characterization test** that pins the current observable behavior. Your
loop: make one small change, run `go test ./...`, stay green, commit mentally,
repeat. A rename is only complete when every reference is updated — including
call sites in tests; that's renaming, not cheating. What you may never do is
change the *expected outputs*: clean code is a refactor, and a refactor
preserves behavior by definition.

## Exercise

Open [`exercise/`](exercise/) — a small Go module containing `report.go`, a
sales-report builder that works and is `gofmt`-clean, yet is a museum of this
lesson's smells. `README.md` walks you through five parts: critique it and
write a prioritized cleanup plan, then execute the plan — renames, comment
triage, cohesion split — with the characterization tests green after every
step. You record decisions in `NOTES.md`; there is no grading script — your
tutor reviews the notes, the diff, and your reasoning in conversation.

Acceptance criteria:

1. `NOTES.md` part 1 lists at least eight distinct smells with line
   references, each tagged naming / size / cohesion / comments, ordered into
   a prioritized cleanup plan you can defend.
2. Every identifier you rename appears in the part 2 table with the criterion
   that justifies it; no cryptic wide-scope names survive, and short names
   that were *fine* (loop-local ones) are left alone — with a note saying why.
3. Every comment in the original is classified (why / warning / contract /
   narration / dead code) with a verdict; narration and commented-out code are
   gone; the surviving why-comment stays; the exported function has a real
   contract doc comment.
4. `report.go` is split so parsing, aggregating, and formatting each have a
   single reason to change, and `NOTES.md` names one further split you
   deliberately did *not* make and why.
5. `go test ./...` passes on your final version with unchanged expected
   outputs, and the code is `gofmt`-clean.

Run the tests from inside the exercise folder — they pass *before* you start,
and must still pass when you finish:

```sh
cd exercise
go test ./...
```

## Further reading

- [Effective Go — Names](https://go.dev/doc/effective_go#names)
- [Go Wiki — Code Review Comments](https://go.dev/wiki/CodeReviewComments)
  (the checklist Go reviewers apply; half of it is this lesson)
- [Go Doc Comments](https://go.dev/doc/comment) — the contract format for
  exported identifiers
- [Martin Fowler — Code Smell](https://martinfowler.com/bliki/CodeSmell.html)
