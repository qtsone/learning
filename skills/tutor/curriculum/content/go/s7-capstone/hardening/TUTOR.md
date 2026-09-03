# Tutor notes — Capstone: Hardening

## Where the learner is

They have a finished core: milestones ticked, suite green under `-race`, real
package structure, a README a stranger can run. Eight harness checks passed in
the previous lesson, and they are pleased with it. This lesson takes the same
codebase and asks a harder question — does it hold when someone is trying to
break it — so the emotional shape is different from every lesson before it:
**the deliverable is a list of things wrong with their own work.**

That is the teaching moment. A learner who comes back with "I looked and found
nothing" has not done the exercise; a learner who comes back with four findings,
three fixes and one written-down accepted risk has done it exactly right, even
though the second report *feels* worse. Say that out loud early, or they will
optimise for the wrong report.

Two competing failure modes to watch for:

- **Security theatre.** A long SECURITY.md listing controls they did not
  implement, a fuzz target with one seed that was never run with `-fuzz`, a
  "finding" invented to fill the section. Everything is documented; nothing was
  discovered.
- **Hardening the whole world.** Rewriting the project to defend against
  threats their PRD explicitly excludes, and running out of the 8-12 hours with
  the actual defects untouched. Accepted risks exist precisely so this stops.

Guidance mode: in `guided`, do the first self-review pass *with* them on one
package, then let them do the rest. In `spartan`, they run the pass and bring
the findings; you attack the findings list, not the code.

## Common misconceptions

- **"The tests pass, so it is tested."** Coverage counts execution, not
  assertion. Push mutation thinking: name a line, name the change, name the test
  that goes red. Silence there is the finding.
- **"Fuzzing means running `go test -fuzz` once."** Without a *property* in the
  body, fuzzing only finds panics. Ask what the target asserts about accepted
  output; "it doesn't crash" is a start, not a target.
- **"The corpus is in the cache, that counts."** The generated corpus is
  machine-local and disposable; only `testdata/fuzz/<Target>/` is committed and
  rerun by everyone. This is the fuzzing misconception that survives S5.
- **"A timeout on the transport is a timeout."** Per-phase transport timeouts
  can all pass while the call still hangs. `Client.Timeout` bounds the whole
  exchange, body included.
- **"`context.Background()` is fine here, nothing cancels anyway."** Today.
  The point is that the function *cannot* be cancelled by its caller, and the
  day something needs to shut down gracefully, it is unreachable.
- **"govulncheck found nothing, so we're clean."** It found nothing *reachable*
  today, with that toolchain, against that database. Ask when they last ran it
  and what changes would invalidate the answer.
- **"Validation belongs everywhere, to be safe."** Validating at every layer
  means no layer owns it and all of them disagree. Boundary once, then trust.
- **"Accepted risk means I couldn't be bothered."** An accepted risk with an
  exposure, a reason and a reopening trigger is senior work. Grade it as such.

## Grilling points

- "Walk me to your trust boundaries. For each one: what is checked, where, and
  what happens to input that fails?"
- "Pick your most important function. Change one line in your head so it is
  subtly wrong. Which test catches it?"
- "Show me the fuzz target. What property does it assert beyond 'no panic'?
  What did the fuzzer find when you ran it — and if nothing, how long did you
  run it?"
- "Open the corpus directory. Where did each file come from, and what would
  break if I deleted them?"
- "Show me the regression test failing against the old code." (`git stash`,
  run, watch it go red. This is the single highest-value 60 seconds of the
  review.)
- "Every size limit in your project — name them. Now name an input with no
  limit." (There is almost always one.)
- "Where can a secret end up in a log line? Trace one error from the place it
  is created to the place it is printed."
- "What did govulncheck say, when did you run it, and what would make you run
  it again?"
- "Top three CPU entries and top three allocation entries. Which surprised you,
  and what is your hypothesis?" (Hypothesis only — no optimising yet.)
- "Name a risk you accepted. What is the blast radius, and what observation
  would make you reopen it?"
- "Which of your accepted risks would a reviewer argue with hardest, and what is
  your answer?"

## Grading rubric

- **A** — Nine harness checks green. The pass was systematic and produced real
  findings, at least one of which was a genuine defect fixed with a regression
  test they can show failing against the old code. The fuzz target asserts a
  property, was run with `-fuzz`, and its corpus is committed and explained.
  Every input surface has a size limit. Secrets cannot reach a log line, and
  they can trace why. govulncheck was run and read, with a date. Profiles
  captured and interpreted without optimising. Accepted risks name exposure,
  reason and trigger, and none contradicts the PRD.
- **B** — Harness green. Findings are real but the pass was ad hoc rather than
  checklist-driven; one input surface has no limit, or the fuzz property is
  "does not panic" only. Regression test exists but they cannot immediately show
  it red. Documentation solid, accepted risks a little thin on triggers.
- **C** — Harness green only after aiming at the checks: a fuzz target added at
  the end and never fuzzed, a hand-written corpus with no story, a SECURITY.md
  written to match the section list, a "finding" that was never a defect. They
  can explain the mechanics but not what they learned about their own code.
  Pass only with a written list carried into the next lesson.
- **Fail** — Harness red; or a security document that describes a program other
  than theirs; or a "fix" with no test; or they cannot say what their program
  trusts. Do not advance: the operations lesson deploys this code, and shipping
  something they have not examined is the wrong habit to install.

## Remediation ladder

1. **Narrow the surface.** "Forget the whole project. Take one function that
   touches outside data. What is the worst string I could hand it?" Most stalls
   are the size of the task, not the difficulty.
2. **Give them the checklist, not the answers.** Input surfaces → interpreters
   → secrets in output → unbounded allocations. Have them walk one package with
   it while you watch, then leave.
3. **Pair on one fuzz target.** Choose the boundary together, write the seeds
   together, and let them write the property. Run it live for 60 seconds — the
   moment it prints a crasher is the moment fuzzing stops being homework.
4. **If nothing is found, supply the hostile input yourself.** Hand them three
   strings for their format (empty, one past the limit, `../../etc/passwd`) and
   let the code fail in front of them. Then step back.
5. **If the risk is that they harden forever**, timebox: two hours on the top
   finding, then write the rest into accepted risks with triggers. Shipping with
   a written risk register is the lesson, not exhaustiveness.

## After passing

Preview: "It holds up under attack. Next you make it something other people can
run: containers, a pipeline that gates on these same checks, and the
observability that tells you what it is doing at 3am."
