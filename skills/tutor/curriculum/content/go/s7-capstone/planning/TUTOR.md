# Tutor notes — Capstone: Planning

## Where the learner is

First lesson of the final stage, straight out of S6. They have designed a
system on paper (`shared.systems.design-capstone`) and built a service to
somebody else's specification (`go.advanced.project-service`). What is new is
that nobody hands them the brief, and that the thing they choose here is the
thing they will live with for roughly 60 more hours: built, hardened, operated,
profiled and defended.

Two failure modes dominate, and they are opposite. The ambitious learner picks
a distributed system with a web UI and a plugin API; the flat one picks a CLI
that renames files. Your job in this lesson is almost entirely calibration, and
you get exactly one cheap chance at it — after this, every correction costs
them build hours.

Verify is `discussion`. Grade from the four documents plus a 45-90 minute design
review. **Grade judgment, not ambition**: a modest project with sharp non-goals,
honest risks and a defensible cut order beats an impressive one with a plan that
assumes nothing goes wrong. The learner who says "I dropped the plugin system
because it serves a user I do not have" has demonstrated the thing this lesson
teaches.

One operational duty that is easy to skip and expensive to skip: before you
pass them, confirm `projects/capstone/` exists with a `go.mod` that builds, and
that they can state how a later lesson's harness finds it
(`TUTOR_CAPSTONE_DIR`, else `capstone.path` in the lesson's `exercise/`, else
`projects/capstone` at the workspace root, which the harness finds by walking
up). Every test-verified lesson in this stage fails loudly and confusingly if
the project is not there.

## Two touches, not one

**Touch 1 — selection check (10-15 minutes, before they write the PRD).** They
bring `SELECTION.md` filled for one or two candidates. This is the cheap
correction. Run the rubric rows, ask the four bring-your-own questions from
`briefs.md`, and give a verdict: approved, approved-if-cut (say exactly what),
or rejected (say exactly why, and what shape would work). Do not let them
proceed to a PRD on an unapproved project — three hours of paperwork on the
wrong project is three hours gone, and worse, it creates sunk cost that fights
you in the real review.

**Touch 2 — the design review (45-90 minutes).** The graded session, below.

## Design-review protocol

**Pre-read.** Read PRD, ADRs, MILESTONES, RISKS silently first. Check each is
materially filled — a PRD with three requirements and no non-goals, or two ADRs
where three were asked for, goes straight back with the remediation ladder.
Reviewing a stub teaches nothing. Then pick your three weakest points; they are
the agenda. Time-box: if you fall behind, protect steps 2, 4 and 6.

1. **The pitch (5 min).** They give the one-sentence pitch and the one flow.
   Do not interrupt. Listen for: do they lead with the problem or with a
   technology? Can they say the flow in one sentence without a diagram? A pitch
   that needs ninety seconds is a scope finding, and you now have evidence for
   step 2.

2. **Scope attack (15 min).** The centre of this review. Four moves, in order:

   - **Halve it.** "You have half the hours you planned. Which milestones
     survive, and is what remains still worth demoing?" A plan that does not
     degrade gracefully is a plan that will fail all at once.
   - **Delete the interesting part.** Name their favourite feature and remove
     it. If the project is now pointless, the ordering is wrong — that feature
     is the core and belongs in M1, not M4. If the project is still fine, they
     have just discovered a non-goal.
   - **Price the hidden work.** Ask for their hours estimate, then walk the
     list they left out: error handling, the second input format, config,
     tests for the concurrent path, the README, the ten minutes per commit.
     Learners routinely estimate the happy path and nothing else. Expect real
     totals of 1.5-2× their number and make them adjust the plan, not the
     estimate.
   - **Name the hard part.** "Where is the difficulty, and why is it hard?" If
     they cannot answer, the project is too small; if there are three answers,
     it is too big.

3. **Is it secretly three projects? (5-10 min).** Symptoms, in order of how
   often they show up: the pitch joins two nouns serving different users; more
   than one milestone reads like a walking skeleton; two ADRs cover components
   that never exchange data; the one-flow answer needed a paragraph. When you
   find it, do not say "too big" — that is unactionable. Do this instead: name
   the three projects out loud, ask which single one they would keep if the
   other two were impossible, and have them rewrite the other two as non-goals
   on the spot. Ten minutes of live surgery here saves a stalled build later.

4. **Requirements and non-goals (10 min).** Pick two functional requirements
   and ask how they would know each was met — if the answer is not observable,
   it is a wish. Check non-functional requirements are numbers with a stated
   origin. Then attack the non-goals: find the *missing* one by proposing an
   obvious extension ("surely it should also serve this over HTTP?") and see
   whether they defend the boundary or agree enthusiastically. Enthusiastic
   agreement is the finding.

5. **ADR attack (15 min).** Take the ADR they are least sure about and argue
   its rejected alternative as strongly as you honestly can. Then take one they
   are *confident* about and counterfactual it until something moves. Reject:
   straw-man alternatives, consequences sections that are really benefits, and
   flip conditions they could never observe. Finish with reversibility — "which
   of these three could you undo in an afternoon?" A record for a cheap
   decision is a smell (they are recording everything); a one-way door with no
   record is a gap.

6. **Milestones and risks (15 min).** Verify M0 is genuinely a skeleton — make
   them narrate it end to end and confirm it touches storage and the entry
   point. Confirm the ordering is by risk: "which milestone could prove your
   design wrong, and why is it not first?" Then delete a middle milestone and
   ask what breaks downstream; a plan where deleting M2 breaks M3 and M4 is
   horizontally sliced and needs re-cutting. On risks: pick the one with the
   highest cost and ask what they would see, this week, that tells them it is
   happening. Push for a spike with a hard timebox on the largest technical
   unknown.

7. **Close (5 min).** Fill any `quiz.json` gaps the conversation missed.
   Confirm the project directory and harness convention (see above). Give the
   grade plus the three specific things to change before they start M0, and
   have them state the cut order back to you.

### Calibration reference

Sizes, for a learner with 20-30 hours of core build. Their project will differ;
use this to place it, not to score it.

- **Right-sized:** 3-6 packages, one storage mechanism, one concurrency
  pattern, one interface with two implementations, 2-4 external-facing
  operations. The example briefs are all deliberately at this size.
- **Too big, common shapes:** anything with both a server and a UI; anything
  needing two storage systems; anything with a plugin or extension mechanism;
  anything distributed across processes; "and a web dashboard".
- **Too small, common shapes:** a wrapper over one library call; a formatter or
  converter with no persisted state; a single-file CLI with no concurrency; a
  reimplementation of a tutorial they have already followed.
- **Deceptive middles:** "a small database", "a tiny Docker", "a simple Git" —
  these sound bounded and are not, unless they have written aggressive
  non-goals naming exactly which one property they are implementing. Approve
  those only with the non-goals in hand.

## Common misconceptions

- **"Planning is overhead; I will figure it out as I build."** The counter is
  concrete, not moral: without a written problem statement and non-goals,
  nothing tells them what to cut at hour 40, so they cut whatever is in front
  of them, which is usually the tests.
- **"Ambitious is impressive."** Reverse it explicitly: the grade rewards a
  finished, defended, honest project. An unfinished distributed system grades
  below a complete, well-argued CLI. Say this out loud in touch 1.
- **"Non-goals are a way of admitting weakness."** They are the strongest
  signal of seniority in the document. Learners often write five trivial ones;
  push for the two that hurt.
- **"An ADR is documentation you write at the end."** A record written
  afterwards is a justification. It also has nothing to say when a later lesson
  asks which decisions held up.
- **"A milestone is a chunk of the codebase."** Horizontal slicing ("the
  storage layer") is the default instinct and produces a project that does not
  run for a week. Vertical slices only.
- **"Risks are technical."** The register that predicts reality usually
  contains "I will lose two evenings a week" and "I will get bored during the
  plumbing milestone".
- **"Estimates are commitments."** They are instruments. The point of the
  number is that being at 2× is a fact they notice at hour 6 rather than
  hour 30.
- **"The capstone can live anywhere."** It lives in `projects/capstone/`. Every
  later harness looks for it there or where they point it.

## Grilling points

Beyond the quiz set:

- "Read me your problem statement. Now read it again with every technology noun
  removed — is there anything left?"
- "Who is the user, and what do they do today instead? Why is that not good
  enough?"
- "Which non-goal will you personally violate first, and what stops you?"
- "You are at hour 45 with one milestone left and it is not going well. What
  ships?"
- "Your storage ADR: argue the option you rejected. Convince me."
- "Which of your three ADRs could you reverse in an afternoon? Why does that
  one have a record?"
- "Show me the milestone that could prove your design wrong. Why is it not
  first?"
- "Delete M2. What still works, and what does M3 do now?"
- "Your riskiest assumption — what experiment would settle it, and how long
  would you give it before you changed the plan?"
- "How do the later lessons in this stage find your project? Talk me through
  the three ways."

## Grading rubric

Grade the judgment on display, not the ambition of the pitch.

- **A** — Problem stated without technology; five or more non-goals including
  at least two that hurt; requirements with IDs that the ADRs and milestones
  actually cite; three or more ADRs with genuine alternatives, honest costs,
  and flip conditions they could observe; milestones vertically sliced, ordered
  by risk, M0 a real skeleton; risks priced with triggers and at least one
  scheduled spike; and — the signature of the grade — under attack they revise
  coherently: they can say what moves, what does not, and why, including a cut
  order they can defend. Project directory created and the harness convention
  understood.
- **B** — All documents present and materially complete; scope defensible; but
  one dimension is thin — non-goals soft, one ADR with a straw-man alternative,
  a milestone or two sliced horizontally, or a risk register with no triggers.
  They fix it in the room when pushed.
- **C** — Documents present but formal: requirements nothing references,
  non-goals nobody would have expected anyway, ADRs recording preferences
  rather than one-way doors, or a milestone plan that is a table of contents
  for the codebase. Scope survived only because you cut it for them. Pass only
  with the fixes made before M0 starts, and record what you required.
- **Fail** — Missing documents; a project that fails the rubric (no storage, no
  concurrency, nothing to design, or three projects in a trench coat); no
  capstone module; or a plan they cannot defend at all — cannot say what to cut,
  what is hard, or why they chose this. Remediate; do not let them start
  building. Starting the wrong project is the single most expensive mistake
  available in this stage.

Two calls worth making explicitly. A learner who *reduces* scope during the
review has not lost points — that is the lesson landing, and it should read as
evidence toward A. A learner who defends an oversized project fluently is not
demonstrating judgment; fluency is not calibration, and the grade should say so.

## Remediation ladder

1. "Which single sentence in your PRD would settle an argument about whether a
   feature is in scope? If none, that is the gap."
2. "Name the three projects hiding in this one. Which would you keep if the
   other two were impossible?"
3. "Take your favourite milestone and write down what someone else would type
   to check it is done. Now do that for M0."
4. "Pick your riskiest assumption. Write the two-hour experiment that settles
   it, and the milestone it must happen before."
5. Only then: work one ADR through with them out loud — context, two real
   options with costs, decision, consequences, flip condition — and have them
   write the other two alone.

## After passing

Preview: "Next you build the core: milestone by milestone, reviewed as you go,
with a harness that grades the engineering properties of whatever you built —
that it compiles and vets clean, that your own tests pass under the race
detector, that the package structure is real. Your MILESTONES.md becomes the
progress record, and by the end every box has to be ticked."
