# Documentation

> `shared.eng.documentation` · ~1-2h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Write a README that lets a stranger install, run, and understand the purpose
  of a project without asking questions.
- Write doc comments for a public API that state the contract, not a
  restatement of the signature.
- Write an ADR capturing a real decision: context, options considered,
  decision, and consequences.
- Maintain a changelog entry for a release, distinguishing added, changed,
  fixed, and breaking items.
- Choose which documentation form (README, doc comment, ADR, changelog) fits a
  given piece of information and justify it.

## Four documents, four readers

The code review lesson ended with something uncomfortable: the best arguments
about your code happen in review threads, and review threads evaporate. The
reviewer moves on, the thread scrolls away, and six months later nobody —
including you — can answer the questions the thread once settled.
Documentation is the durable residue: the set of artifacts that answer a
reader's question when nobody is around to ask.

The word "documentation" hides four different documents, and the way to tell
them apart is not their format but their **reader** — who they are, what they
are asking, and at what moment they are asking it:

| form | reader | their question | the moment |
|------|--------|----------------|------------|
| README | a stranger meeting the project | what is this, and how do I run it? | first contact |
| doc comment | a caller of your API | what may I rely on here? | mid-code, at the call site |
| ADR | a maintainer (often future you) | why is it built this way? | questioning a decision |
| changelog | an existing user | what changed since I last looked? | deciding whether to upgrade |

They also fail differently. A missing README costs you users; a wrong doc
comment costs a caller correctness; a missing ADR costs the team a re-fought
argument; a missing changelog makes every upgrade a leap of faith. Keep the
table in mind — the whole lesson is learning to route information to the
right row.

## The README: pass the stranger test

The README is the project's front door, and its author is the worst-placed
person on earth to judge it. You know which directory to run commands from,
which tool needs installing first, what the project is *for* — so your draft
silently assumes all of it. The fix is a concrete test. Imagine a competent
stranger: skilled, but with zero context and a clean machine. From the README
alone they must be able to:

1. Say in one sentence what the project does and whether it is for them.
2. Install it, copy-pasting commands that work.
3. Run one real example and see it work.

— all **without asking a human**. That's the stranger test, and it dictates
the minimum spine: a one-paragraph purpose (what it does, who it's for, why
it exists), requirements, install steps, a quickstart with real commands and
real output, and pointers to deeper docs. Short is fine. Unanswerable
questions are not.

Four anti-patterns account for most bad READMEs:

- **Written for yourself.** "Build it the usual way." Whose usual?
- **Stale commands.** A README describing a flag renamed months ago is worse
  than no README — clean code's lesson about stale comments applies with
  interest: it is disinformation with authority. The defense is mechanical:
  before committing, run every command in the README exactly as written, from
  a clean directory. (Your CI lesson suggests the next step — a workflow can
  run the quickstart on every change.)
- **"See the source for details."** The reader at the front door is exactly
  the person who shouldn't need to read the source yet.
- **Aspirational docs.** Documenting the roadmap as if it existed.
  Documentation records what *is*; what you wish existed belongs in an issue
  tracker.

## Doc comments: the contract at the call site

In clean code you triaged comments into *why*, *warning*, and *contract*, and
learned that `// GetUser gets a user.` is narration wearing a contract's
clothes. This section is that third category grown up: what does a real
contract say?

Start from what the reader already has. The signature shows the name, the
parameters, the types. A doc comment that repeats those adds nothing — an
**echo**. What the signature *cannot* show is behavior, and behavior at the
edges is precisely what a caller must know before relying on you:

- What happens on empty, zero, or missing input?
- What do the errors mean — which sentinel, wrapped how, what should the
  caller match with?
- What invariants hold on the result — sorted? never nil? safe to mutate?
- Is the zero value usable? Is there hidden persistence or another side
  effect per call?
- May two goroutines (or threads) use it at once?

A useful discipline: **every sentence in a doc comment is a promise a test
could falsify.** "Returns names in sorted order" — a test could catch that
lie. "List lists the snippets" — no test could ever fail because of that
sentence, so it says nothing; delete it. And promises must be true *today*:
you write a doc comment by reading the body, not by remembering the intent.
This is also why contracts live glued to the declaration rather than in a
separate manual — the same diff that changes behavior must change the
promise, and your reviewer (last lesson) sees the pair together or flags the
mismatch.

**In Go:** the contract has an official format —
[doc comments](https://go.dev/doc/comment). The comment sits immediately
above the declaration, is made of complete sentences, and its first sentence
begins with the identifier's name: `// Open reads the snippet file at path…`.
A `// Package store …` comment introduces the package as a whole. The reward
for the convention is tooling: `go doc`, your editor's hover, and pkg.go.dev
all render these comments as the package's reference documentation — you
have been reading them since S1. Every exported identifier deserves one;
`// Deprecated:` marks the ones callers should migrate away from.

## ADRs: write down the why before it evaporates

Of everything a project knows, *why* rots fastest. The code shows what it
does; the tests pin what it must do; but the reasons — why a JSON file and
not a database, why polling and not a queue — live in heads and chat
scrollback. Six months later a newcomer hits the strange-looking choice and
either re-fights the whole argument (expensive) or "fixes" it without
knowing what it was protecting against (worse).

An **Architecture Decision Record** is the antidote: a short numbered file,
one per decision, with four sections.

- **Context** — the forces at the time of deciding: constraints,
  requirements, what you knew and didn't. Forces, not conclusions: "single
  user, tiny data, must work offline" — not "JSON is best".
- **Options considered** — the real alternatives, each with honest
  trade-offs. The losers matter most: they prove the winner was chosen, not
  defaulted into, and they save the next person from re-proposing them cold.
- **Decision** — one sentence, active voice: "We store snippets in a single
  JSON file."
- **Consequences** — what follows, good *and bad*. An ADR listing only
  benefits is advertising, not a record; the costs are what the next reader
  needs to check whether the decision still holds.

Two properties make the format work. ADRs are **small** — a page, one
decision each. And they are **immutable**: when a decision changes, you
don't edit ADR 1, you write ADR 7 saying it supersedes ADR 1 and why. The
result is an append-only log of the project's thinking — version control for
decisions, matching what S0's git lesson gave you for code. Write one
whenever a decision would be expensive to reverse, or when you catch
yourself settling a real argument in a review thread — promote the argument
to an ADR before the thread evaporates.

## The changelog: the release seen from outside

`git log` is a diary for maintainers. A **changelog** is a letter to users.
The user upgrading from 0.1.0 asks three things: will my setup still work,
what do I get, and what must I change? Answering them takes two acts of
judgment per commit:

- **Curate.** Only user-visible changes enter. Refactors, CI tweaks, comment
  typos — if no user of the binary and no importer of the package behaves
  differently, it doesn't belong. This is why a nine-commit release can be a
  five-line entry.
- **Classify.** The Keep a Changelog convention names the buckets: *Added*,
  *Changed*, *Fixed* (plus *Deprecated*, *Removed*, *Security* when needed).
  **Breaking** changes are flagged loudest — first in the entry, clearly
  marked, with what the user must do about it.

Write each line from the outside in the reader's vocabulary: "the `-dir`
flag is now `-path`", not "refactored flag handling". And don't do
archaeology at release time — keep an *Unreleased* section at the top and
add the line in the same change that earns it, then stamp a version and date
when you ship.

Version numbers are the changelog's headline. Semantic versioning makes them
a contract: breaking change → major, new feature → minor, fix → patch
(pre-1.0 projects conventionally signal breakage with minor bumps).

**In Go:** modules take that contract literally — versions are git tags like
`v0.2.0`, and from v2 onward the major version becomes part of the module
path (`example.com/mod/v2`), which is how the toolchain can promise that
upgrading within a major version never breaks your build's import contract.

## Routing: every fact has one home

The final skill is placement, and three principles decide it:

1. **Closest to what it describes.** A function's contract lives on the
   function; a project-shaped decision lives in the project's ADR log.
   Distance is rot: the further a doc lives from the thing it documents, the
   less likely the same diff updates both.
2. **One home per fact.** Duplicate a fact and the copies *will* diverge —
   then a reader must decide which one is lying. The README shows one worked
   example and points at the reference; it does not restate every flag.
3. **Written for the reader's moment.** A caller mid-edit will not open the
   README; a stranger at first contact will not read doc comments; an
   upgrader reads the changelog, not the ADR log.

So the routing test for any piece of information is: *who needs this, and at
what moment?* The answer names the form — and "no one, at no particular
moment" is an answer too: that fact belongs in the issue tracker or nowhere,
not smeared across docs where it will rot. Information with no home ends up
in someone's head, which is where projects go to become unmaintainable.

## Exercise

Open [`exercise/`](exercise/) — you are inheriting `snippets/`, a small
working CLI (it would survive your clean-code review) whose documentation is
a disaster. `README.md` in the exercise folder walks you through five parts:
stranger-test the project README and rewrite it, replace the echo doc
comments in `store/` with real contracts, write ADR 0001 for the
JSON-file-versus-SQLite decision, curate the commit history into a `0.2.0`
changelog entry, and finish with a placement drill. You record every
decision in `NOTES.md`; there is no grading script — your tutor reviews the
four rewritten artifacts, the notes, and your reasoning in conversation.

Acceptance criteria:

1. `NOTES.md` part 1 lists at least six questions a competent stranger could
   not answer from the old `snippets/README.md`; your rewritten README
   answers all of them, and you have run every command in it exactly as
   written, from a clean directory.
2. Every exported identifier in `package store` (the package itself,
   `ErrNotFound`, `Store`, `Open`, `Add`, `Get`, `List`) has a doc comment
   in Go's form whose every sentence is true of the code as-is — covering at
   minimum: `Open` on a missing file, `Add` on an existing name and an empty
   name plus its persistence behavior, `Get`'s error semantics, `List`'s
   ordering, and concurrency safety. The `NOTES.md` table records what each
   old comment failed to say.
3. `docs/adr/0001-single-json-file-storage.md` is complete: context stated
   as forces (not conclusions), at least two rejected options with honest
   trade-offs, a one-sentence decision, and consequences including at least
   two genuine costs.
4. Every commit in `HISTORY.txt` is classified in `NOTES.md` — Added,
   Changed, Fixed, breaking, or omitted, each with a reason — and
   `CHANGELOG.md` gains a `0.2.0` entry with breaking changes flagged first,
   written from the user's perspective.
5. The placement drill table in `NOTES.md` names a justified home (or
   justified homelessness) for every item.
6. The code still builds — doc comments are code, so from `snippets/`:
   `go build ./...` and `gofmt -l .` are both clean. Behavior is off-limits:
   this is a documentation pass, not a refactor.

## Further reading

- [Go Doc Comments](https://go.dev/doc/comment) — the official contract
  format, with good and bad examples
- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) — the changelog
  convention the exercise uses
- [Michael Nygard — Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
  — the original ADR essay; the four-section form comes from here
- [Semantic Versioning](https://semver.org/) — the version-number contract
  changelogs headline
