# Tutor notes — Capstone: Operations

## Where the learner is

They have a built, tested, hardened project of their own, and until now it has
only ever run under their own hands. This lesson is the first time their code
has to be handed to someone else — a platform, a pipeline, a future on-call
version of themselves. Six to ten hours, mostly spent on artifacts rather than
Go: a Dockerfile, a pipeline, a runbook, telemetry wiring, a shutdown path.

Two populations, and they fail differently:

- **Took the containers focus pack.** Fast on packaging, and at risk of
  cargo-culting a manifest they already have. Push on *why* each line, and on
  the parts the pack did not cover: the runbook, the drill, the rollback
  rehearsal.
- **Did not take it.** Slower on the image and the deploy target, and prone to
  believing they cannot do the lesson without a cluster. They can: everything
  is checkable statically, and a single-VM or local container deploy is a real
  target. Do not let "I have no cloud account" become the blocker.

The lesson's real content is *rehearsal*. Learners will happily write a runbook
they have never followed; the whole value is in following it once.

**The reference is a second project, on purpose.** `solution/` here is
`tinysvc`, not the `notes` project the build-core, hardening and performance
references carry forward. The surface this lesson spends most of its words on —
readiness failing independently of liveness, a drain that lets in-flight
requests finish — has no CLI equivalent to show, so the reference demonstrates
the column a CLI learner cannot see from their own repository. A different
shape is not a lower bar: `tinysvc` still clears the build-core and hardening
harnesses — ticked milestones, the coverage floor, a validated trust boundary
with a fuzz corpus, `SECURITY.md` — because those grade properties, not
projects. Nothing in the referee prefers that shape. Name the CLI routes
through criteria 6-7 early,
before someone builds a fake HTTP server to satisfy a grep:

- **Health signal** — a `HEALTHCHECK` in the Dockerfile (exec form; a
  distroless runtime has no shell to run a string through), or a runbook
  heading for the self-check with the command in a fenced block. A `doctor`
  subcommand that prints its findings and exits non-zero is a health signal.
- **Metrics** — the lesson's CLI column says "counters printed at the end of a
  run", and the referee wants those counters to travel through `expvar`,
  `runtime/metrics`, a client library or a `/metrics` route. `expvar.NewInt`
  counters incremented during the run and dumped at exit satisfy both: the
  numbers stay named and typed instead of being formatted into a sentence,
  which is what makes them countable by something other than a human.
- **Drain** — the shutdown check *skips* when there is no `http.Server`, and a
  skip is not a failure. What replaces it is reviewed, not grepped: SIGINT
  mid-run leaves no half-written output and says what it did and did not
  finish.

The rest of the reference — Dockerfile, Makefile, CI workflow, runbook,
documented configuration — is shape-independent, so a CLI learner can read it
as their own worked example.

## Demos to ask for, not descriptions

Four things must be watched, not described. Keep each under ten minutes.

1. **Redeploy live.** They run their scripted deploy in front of you, from the
   file, without editing anything. Any hand-edit mid-demo is the finding.
2. **Roll back.** Deploy a version, roll it back, time it, and say aloud what
   rollback would *not* undo for their design.
3. **SIGTERM under load.** Start it, put a trickle of work through it, send
   SIGTERM. Watch for: readiness failing first, in-flight work finishing, a
   clean exit, and log lines proving each step. A drop of even one request is
   worth diagnosing together — that is where the lesson lands.
4. **The telemetry drill.** *You* choose the fault (bad config, dependency
   stopped, panicking handler, work queue stalled), they answer only from
   telemetry: up? serving? failing how much? since when? where? Any question
   they answer by reading source is a gap in their instrumentation, and the fix
   is instrumentation.

If a demo is impossible because there is nowhere to deploy, a local container
runtime or a single VM is an acceptable target. "It runs on my laptop with
`go run`" is not.

## Common misconceptions

- **"The referee passing means it is operable."** It reads files. It cannot see
  whether the deploy works, whether the rollback was ever run, or whether the
  runbook is true. Green plus a failed shutdown demo is not a pass.
- **"CI is green, so the gate blocks."** Blocking is a repository setting, not
  a workflow line. Ask to see required status checks, or a pull request that
  could not merge while red.
- **"Health and readiness are the same endpoint."** The commonest error, and
  the expensive one: a liveness probe that checks the database restarts every
  instance when the database blinks.
- **"Graceful shutdown means catching the signal."** Catching it and then
  exiting immediately is an abrupt shutdown with extra steps. The order is
  fail readiness → drain → close what you own → exit.
- **"`latest` is fine, it's my project."** It means the image they debug is not
  the image they shipped, and it is unfalsifiable after the fact.
- **"Non-root is a nice-to-have."** Ask what the blast radius of a container
  escape is when the process is root. Also watch for `USER` in the build stage
  only — a real and common mistake the referee catches.
- **"More logs is more observability."** Free-text lines at debug level are not
  a signal. Ask them to *count* something from their logs; if they cannot, the
  keys are wrong.
- **"Metrics for everything."** Watch for labels carrying user ids, paths or
  error strings. Recall S5's cardinality bomb: one label with unbounded values
  takes down the metrics backend, not the service.
- **"Nobody pages me, so the paging section is silly."** Then the answer is
  "me, and here is what happens when I am unreachable" — written down. The
  section exists to expose single points of failure, including human ones.

## Grilling points

- "Your deploy script — run it. Now tell me which step would break if I ran it
  from a fresh checkout on a different machine."
- "Roll back. How long did it take? What would rollback *not* undo?"
- "Why that base image tag, and what happens to your build in a year?"
- "Your process gets SIGTERM one second into a five-second request. Walk me
  through what happens, in order, and where the request ends up."
- "What is your shutdown timeout, and what is the platform's grace period?
  Which is larger, and why does that matter?"
- "Your liveness probe: what does it check, and what would happen to your fleet
  if a dependency it touches got slow?"
- "Show me one log line. What can I count with it? Now show me a line I could
  not count."
- "Which metric label has the most possible values in your system? What happens
  when a user controls it?"
- "I am on call and you are unreachable. Open your runbook and tell me the
  first three commands I run."
- "Name the two ways this actually breaks. Which one have you seen?"
- "What in your image would an attacker use if they got a shell in it?"
- "Where does a secret enter your process, and name every place it could leak."

## Grading rubric

- **A** — Referee green. Deploy and rollback demonstrated live from scripts,
  with a timed rollback and an honest account of what it cannot undo. Shutdown
  demo clean: readiness fails first, in-flight work completes, exit code
  correct, log lines prove it. Telemetry drill answered entirely from
  telemetry, including "since when". Runbook has been followed and fixed;
  failure modes are specific to their system. They can defend every Dockerfile
  line and name their configuration surface without opening the file.
- **B** — Referee green, deploy and rollback work, but one demo is shaky: a
  dropped request during shutdown they can explain but not yet fix, or a
  telemetry question they answer by reading logs by hand where a metric should
  exist. Runbook correct but visibly unused. Blocking gate configured but never
  tested against a red build.
- **C** — Referee green only after the artifacts were written to satisfy it:
  a runbook of headings, a deploy script never run, health endpoints that
  return 200 unconditionally, `/metrics` exposing nothing anyone would look at.
  They can run the program but cannot operate it. Pass only with a written list
  of what to fix, carried into the remaining lessons.
- **Fail** — Referee red; or artifacts that describe a deployment that never
  happened; or a shutdown demo that loses requests with no idea why; or a
  runbook whose rollback command does not work when run. Do not advance on a
  rollback that has never been executed.

## Remediation ladder

1. **Shrink the target.** If they are stuck on where to deploy, drop to a local
   container runtime or one VM and get the *scripted* path working end to end.
   The target is negotiable; the script is not.
2. **One artifact at a time, in this order**: image → pipeline → lifecycle →
   telemetry → runbook. Each one is independently useful, and the order means
   every step is demoable.
3. **Read the referee's fix line aloud.** Each failure names the missing
   property and what to add. Have them say what the check is protecting against
   before they fix it — the point is the risk, not the string.
4. **Pair on the shutdown path only.** It is the piece most likely to be
   subtly wrong, and watching one `curl` get dropped teaches more than any
   explanation. Then have them re-run the demo alone.
5. **Make the runbook true by using it.** Sit with them while they follow their
   own triage steps against a fault you introduce, and have them edit the file
   at every improvisation. This converts a document into a tool in twenty
   minutes.

## After passing

Preview: "Your own system is deployable, observable and documented. Next you
leave your repository entirely: find a real issue in somebody else's project
and land a patch there, on their terms."
