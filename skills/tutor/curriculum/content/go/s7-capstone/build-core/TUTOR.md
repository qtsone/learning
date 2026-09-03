# Tutor notes — Capstone: Core Build

## Where the learner is

They have finished the entire curriculum and specced their own project in the
planning lesson: PRD, non-goals, milestones, risks, at least three ADRs. This
is the longest single lesson they will ever do (20-30h, typically two to four
weeks of evenings), and it is the first time nobody has written the spec for
them. Two failure modes dominate, and neither is a knowledge gap:

- **Stalling.** Momentum dies somewhere around milestone two or three. The
  cause is almost never the code; it is that the next action stopped being
  obvious, or that the milestone was too big to finish in a session.
- **Drifting.** The project quietly becomes a different, larger project. The
  PRD's non-goals are the contract; hold them to it.

Guidance mode matters here. In `guided`, schedule reviews and chase them. In
`spartan`, they drive and you review what they bring — but still refuse to do a
single end-of-project review, because that is not what this lesson teaches.

## Milestone review protocol

**Review at each milestone, not at the end.** A single review of 25 hours of
work is a code-reading marathon that finds cosmetic issues and misses the
structural ones. Four reviews of five hours each catch a bad boundary while it
is still one afternoon of work to move.

Each milestone review runs the same short loop, 20-40 minutes:

1. **Demo first.** "Run it and show me the behaviour this milestone promised."
   No slides, no diff — the program. If it cannot be run, the milestone is not
   done, whatever the tests say.
2. **Read the diff for this milestone only.** Ask for the commit range. If
   there is one giant commit, that is the first finding.
3. **Two questions about the tests.** Which test pins this milestone's
   behaviour? What did you delete or rewrite, and why?
4. **One boundary question.** Which package owns this behaviour, and why not
   the neighbouring one? Draw the import direction if the answer is fuzzy.
5. **ADR check.** Did this milestone touch a decision you recorded? Held,
   revised, or superseded?
6. **Next milestone, out loud.** What is the first action next session, and
   what is the acceptance criterion? Make them state it before the call ends.

Record one line per review in the journal: milestone, verdict, one thing to fix
next time. That log is what the graduation review later in this stage reads.

Between reviews, respond to "I'm stuck" with process before code: how long has
this been stuck, what is the smallest version of it, can it be spiked in a
throwaway file, what did the last green commit look like.

## Common misconceptions

- **"The harness is the grade."** It is the floor. Eight green checks with a
  1,200-line `note.go` and no answer for "why this boundary" is a C.
- **"60% coverage is the target."** Watch for tests written to move the number:
  asserting `err == nil` and nothing else, testing getters, table tests with no
  assertions in the body. Ask what behaviour each test would catch breaking.
- **"I'll write the tests at the end."** Guarantees the design is untestable by
  the time they try, because nothing forced a seam.
- **"Refactoring means the earlier work was wasted."** The opposite: a design
  that survived contact and got revised is the thing they will defend in the
  graduation review. Deleting a superseded test is progress, not loss.
- **"Interfaces make it flexible."** One implementation, no test double, no
  seam — it is indirection. Ask who the second implementer is.
- **"Concurrency makes it fast."** Ask for the sizes in the PRD. Most capstone
  workloads are sequential-and-fine; a goroutine with no owner is a leak with
  ambition.
- **"main can hold the wiring *and* the rules, it's only a few functions."**
  This is how the 2,000-line `main.go` starts. Catch it at milestone one.

## Grilling points

- "Show me your walking skeleton commit. What integration risk did it retire?"
- "Which milestone was wrong when you got into it, and what did you do —
  extend it, split it, or push through?"
- "Draw the import graph. Now: which arrow would you have to reverse to swap
  your storage, and how long would it take?"
- "Pick your most-changed file. Why does it change so often — is it doing two
  jobs?"
- "Name a test you deleted. Why was deleting it the right call?"
- "Point at a place you decided *not* to abstract. What would make you change
  your mind?"
- "Where does a `context.Context` enter your program, and where does
  cancellation actually stop work?"
- "For each goroutine: what does it own, who waits for it, how does it stop?"
- "Which ADR was wrong? What did reality tell you that you did not know when
  you wrote it?"
- "What did you cut? Talk me through the non-goal you were most tempted to
  break."

## Reading an unfamiliar codebase

You will be reviewing a project you did not design. Read it in this order, and
narrate the order to the learner — it is a skill they need for the OSS lesson
later in this stage.

1. **README, then run it.** If you cannot get it running from the README
   alone, that is finding number one, before any code.
2. **`go list ./...` and the directory tree.** Names first: does the layout
   tell you what the program does?
3. **The domain package, types first.** Types before functions; the data model
   is the design. Rules and error vocabulary next.
4. **`main`.** The composition root shows the real dependency graph in twenty
   lines. Compare it to what step 2 implied.
5. **The test names.** `go test -run xxx -v ./...` lists them. Test names are
   the promised behaviour; gaps here are the interesting gaps.
6. **One end-to-end trace.** Follow a single user-visible operation from entry
   point to storage and back. Every seam you cross is a design decision to ask
   about.
7. **Only then, judgement.** Anything you flag before step 6 is a guess about
   code you have not understood.

Feed back in that order too: run/README problems, structure, domain modelling,
tests, then style. Do not open with naming nits — it teaches them that reviews
are about surfaces.

## Grading rubric

- **A** — All eight harness checks green. Milestones delivered as planned or
  changed with a stated reason. Layout defends itself: they can draw the import
  graph, justify each boundary, and name the seam that swaps storage. Errors
  wrapped and matchable, context flows, every goroutine has an owner and a stop
  condition, or a defensible "no concurrency needed, here is why". Tests pin
  behaviour at package boundaries; they can name one they deleted and why.
  ADRs current, with at least one revised or explicitly upheld. Coverage is a
  by-product of honest tests, not a target they chased.
- **B** — Harness green. Structure sound but with one soft spot they can name
  (a package doing two jobs, a fat interface, `main` holding one rule). Tests
  cover the core but lean on implementation details in places. Milestones
  delivered, though at least one was reviewed only after it was finished. ADRs
  exist but were not revisited without prompting.
- **C** — Harness green only after late scrambling: tests written to hit 60%,
  packages split at the end to satisfy the structure check, boxes ticked in one
  sitting. They can run the program and explain what it does, but the layout is
  post-hoc and the design cannot be defended. Pass only with a written list of
  what to fix, carried into the next lesson.
- **Fail** — Harness red; or green over a project that is not theirs, not
  finished, or whose milestones were rewritten to match what happened to get
  built. Also fail if they cannot explain code in their own repository. Do not
  advance — the rest of the stage builds directly on this codebase.

## Remediation ladder

1. **Process, not code.** "Which milestone are you on, when did it last run
   green, and what is the smallest next action?" Most stalls end here.
2. **Shrink the unit of work.** Take the stuck milestone apart on the call
   until one piece fits in a session, and have them state its acceptance
   criterion in one sentence.
3. **Fix the feedback loop.** Get `go test -race ./...` green — deleting or
   skipping the newest broken test if necessary — so they are working from a
   known-good baseline again. Green first, correct second.
4. **Pair on one seam.** Pick the single boundary causing the pain, move one
   type or one function across it together, and let them do the rest of the
   move alone.
5. **Renegotiate scope in writing.** If the project is genuinely too big, cut
   to a defensible core, write the ADR that records the cut, and update
   MILESTONES.md. Cutting scope with a reason is senior behaviour; grade it as
   such, not as failure.

## After passing

Preview: "The core works. Next you find out how it behaves when it is
attacked, mishandled, and pushed past its limits — the same codebase, held to a
harder standard."
