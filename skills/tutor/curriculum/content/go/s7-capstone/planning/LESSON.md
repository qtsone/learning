# Capstone: Planning

> `go.capstone.planning` · ~3-5h · Stage: Expert Capstone (Go)

## Objectives

By the end of this lesson you can:

- Select a capstone project and justify its scope against your skills, your
  timeline, and the graduation criteria.
- Write a PRD that states the problem, the target users, the functional
  requirements, and explicit non-goals.
- Record at least three ADRs covering key technical choices (storage, API
  style, deployment target, …) with alternatives considered and trade-offs.
- Decompose the project into graded milestones with acceptance criteria for
  each.
- Defend the plan under grilling: say what you would cut first if the timeline
  slips, and why.

## The last brief you will be handed

Every project you have built so far arrived pre-scoped. `go.advanced.project-service`
handed you a service specification and a test suite; S6's design capstone handed
you Codraft's traffic numbers and seven non-negotiables. Someone else decided
what "done" meant, and your job was to get there.

This stage hands you a budget instead: roughly 60 hours across six lessons, of
which this is the first. You pick the project, you decide what done means, and
then you build it, harden it, operate it, tune it and defend it — so the thing
you pick in the next three hours decides whether those later lessons are
productive or miserable. They all grade *whatever is in your project directory*.

This lesson's output is therefore paper: a PRD, three or more ADRs, a milestone
plan, a risk register. The only code you write is `go mod init`. Planning takes
a disproportionate share of a project's effort not because it is virtuous, but
because it is the one phase where changing your mind is free.

## The problem statement comes before the solution

Most self-directed projects start life as a solution: "I want to build a
Kubernetes operator", "I want to write a database". Those are technology
choices wearing a project's clothes. They cannot be cut down when time runs
short, because nothing in them says which parts matter.

A problem statement names who hurts, what it costs them now, and how you would
know it stopped:

> Today, `who` has to `current painful workflow`, which costs them
> `time, errors, money`. They tolerate it because `existing options and why
> each one misses`. We will know this worked when `observable change`.

The test is simple: state the problem using no technology nouns at all. If you
cannot, you picked an implementation and are reverse-engineering a
justification. That is allowed — engineers pick projects because the technology
interests them — but write both down separately. "I want to learn how
write-ahead logs behave under crash" is a *learning goal*; "restoring a backup
after a laptop dies takes me a weekend of manual copying" is the problem. The
learning goal picks the technique; the problem decides what ships.

You are sponsor, user and engineer at once, which removes the negotiation and
the honesty it forces. Write the statement as if sending it to someone who will
ask why that is worth 60 hours of your life.

## Non-goals are the load-bearing half

A requirement says what you will build. A non-goal is a **commitment not to
build something** — and it is the half that survives contact with a slipping
timeline, because it is the only part of the document that says no in advance.

Three kinds are worth writing:

- **Never** — outside the problem forever. "No multi-user accounts; this is a
  single-operator tool."
- **Not now** — plausible, deliberately deferred. "One node only; clustering is
  a later problem."
- **Someone else's** — assumed away. "Assumes an existing SMTP relay; we do not
  implement mail delivery."

A non-goal earns its place only if a reasonable person would otherwise expect
it; "this project will not cure cancer" is noise. The litmus test: the list
should mildly disappoint somebody — most likely you at hour 40, when the idea
comes back dressed as "it would only take an afternoon". It never takes an
afternoon.

## Choosing: the selection rubric

The project has to be big enough to exercise what you have learned, small
enough to finish in the hours you actually have, and interesting enough that
you still care at hour 35. Those pull against each other; the rubric is how you
check the pull is balanced.

| The project must… | Why the stage needs it |
|---|---|
| Need at least two packages outside `main`, holding most of the code — real `internal/` boundaries, not everything in `package main` | Graded from the next lesson on, and the numbers are public: the harness wants ≥2 non-`main` packages and ≥60% of non-test lines outside `main` (S4's code-organization is the standard) |
| Have concurrency that earns its cost — a genuine reason for goroutines, not decoration | You will run the race detector across your own code repeatedly this stage |
| Persist state across a restart — files, an embedded store, or SQL | Storage is one of your first real ADRs, and later lessons assume state exists |
| Define at least one interface with more than one real implementation (a production one and a test fake counts) | It is what lets your suite run without the network, which every later lesson depends on |
| Start from one command and show its value in under a minute | You will demo it at every milestone review; a project you cannot demo cannot be graded |
| Run entirely on hardware, data and credentials you already have | A dependency you cannot obtain is a risk you cannot mitigate |

Then the two shapes that go wrong. **Secretly three projects** is the
commonest, and it hides well because each third is reasonable. Symptoms:

- The problem statement joins two nouns serving different users.
- Your milestone list contains more than one walking skeleton.
- Two of your three ADRs concern parts that never touch.
- You cannot name the single flow that makes the thing worth having in one
  sentence.

The fix is never "work faster". Pick the third with the most interesting hard
axis, make the other two non-goals, and check the design does not actively
prevent them later.

**The wrong kind of small.** A thin wrapper over one library call, or a CLI
that reformats a file: nothing to design, harden or profile — sixty hours of
polish on a single `main.go` teaches you polish. If the whole thing fits in an
evening, it is a warm-up, not a capstone.

The criterion not on the rubric, because it cannot be measured: **you have to
want it.** Motivation decays, and at hour 35 it is the only budget left. A less
impressive project you are curious about beats one you have to force.

## Four briefs, for inspiration only

[`exercise/briefs.md`](exercise/briefs.md) carries four worked briefs written
the way a sponsor would hand them over: problem, hard axis, why each clears the
rubric, and how each typically over-scopes.

| Brief | The hard axis it forces |
|---|---|
| **Courier** — backup/sync daemon with a content-addressed local store | Crash safety and resumable work: what is on disk after a kill -9 mid-run |
| **Digest** — self-hosted feed aggregator producing a daily digest | Politeness under concurrency: per-host limits, conditional fetches, backoff |
| **Ledger** — structured log ingester with a small query API | Parsing, retention and compaction, and queries that stay fast as data grows |
| **Relay** — durable job queue with leases, retries and a dead-letter path | Delivery semantics: leases, crash recovery, and what "at least once" costs |

They are examples, not a menu: a project of your own that clears the rubric is
strictly better, because you will care about it more.

## Milestones that keep the thing runnable

The single most useful property of a plan is that the project *works at every
point in it*. A build broken for nine days is a build where you cannot tell
progress from wishful thinking, and where every bug has nine days of changes to
hide in.

**M0 is a walking skeleton**: the thinnest path touching every layer end to
end. One real input enters through the real entry point, passes through a real
(if stupid) core, produces real persisted output, and one test proves it. For
Courier: copy one file into the store, record it, read it back byte-identical.
It should feel embarrassingly small. What it buys is that every later milestone
changes a working program instead of stepping toward a hypothetical one.

Rules for the rest:

- **Slice vertically, not horizontally.** Never "M1: the storage layer". A
  milestone with no observable behavior cannot be demoed or accepted, and turns
  out half-wrong when something finally uses it.
- **Every milestone ends green.** `go build ./...`, `go vet ./...` and
  `go test ./...` clean, the program still runs, the work committed.
- **Acceptance criteria before you start.** Two to four per milestone, phrased
  so somebody else can check them without asking what you meant.
- **Order by risk, not by comfort.** The milestone that could invalidate the
  design goes early, while changing it is cheap; the polish you look forward to
  goes last, where it can be cut.
- **Four to seven milestones**, three to six hours each. A milestone bigger
  than a day is one where you cannot tell whether you are stuck.

Write them in `MILESTONES.md` at your project root, as a task list:

```markdown
## M0 — walking skeleton
- [ ] `courier put <file>` stores one file and prints its content hash
- [ ] `courier get <hash>` writes the bytes back out, byte-identical
- [ ] one round-trip test passes; `go vet ./...` clean
```

Use exactly this checkbox syntax (`- [ ]` unticked, `- [x]` done): the next
lesson checks the file mechanically and requires every box ticked, so the
format is not cosmetic. Reorder and rewrite milestones freely as you work —
plans are supposed to move — but a milestone you **cut** moves to a `## Cut`
section as a plain bullet with a one-line reason, never a box left unticked.
Deleting the record of a decision is the one edit not allowed.

## The risk register

A risk is an uncertainty with a cost — "the code might have bugs" is neither,
being certain and unpriced. A useful entry has five parts: what could happen,
what it costs in hours, how likely it is, the **early warning sign** that tells
you it is happening, and the mitigation you can do *now*.

The trigger is the part people skip and the part that pays: "if a file is not
round-tripping by the end of M1, the chunking design is wrong" turns a vague
worry into an observation you will actually notice.

For technical unknowns the mitigation is almost always a **spike**: a
timeboxed, throwaway experiment answering one specific question, scheduled
*before* the milestone that depends on the answer. Two hours to find out
whether the library can stream results is cheap; finding out at hour 40, with
three milestones built on the assumption, is the most expensive event that
happens to capstones. Spikes are throwaway by definition — keeping the code
means you skipped the experiment rather than ran it.

Non-technical risks belong in the register too, and are usually the ones that
fire: hours disappearing into work, a week away, plain boredom. Their
mitigations are structural — "M4 is the first cut; M0-M3 is a complete, useful
tool without it" is written once and thanked for later.

## ADRs: record the decisions that are expensive to unmake

You met Architecture Decision Records in S4's documentation lesson — context,
options, decision, consequences — and wrote decision records against a design
in S6. New here is choosing *which* decisions deserve one, because a project
where every choice gets a record is a project where none get read.

The test is reversibility. If undoing the decision costs an afternoon, just
decide and move on. If undoing it means rewriting several packages, migrating
data already on disk, or changing a contract something else depends on, it is a
one-way door and it gets a record. You need at least three; the usual
candidates at this size:

- **Storage** — flat files, an embedded store or SQL, and the shape of the data
  in it.
- **Interface** — CLI, HTTP API, both; and if HTTP, what the contract looks
  like (S6's api-design vocabulary applies directly).
- **Concurrency model** — worker pool, goroutine per connection, or a single
  owner goroutine per entity that everything else sends messages to.
- **Deployment target** — where this runs at the end of the stage.
- **Dependency policy** — standard library only, or which third-party modules
  and why. Worth writing even when boring: it is the decision most often
  violated by accident at 11pm.

Each record carries at least two *real* alternatives — options a competent
engineer would actually weigh, with an honest cost for each including the one
you chose — a decision citing the requirement or non-goal that settles it, the
consequences (what you now live with; the bill, not the benefit), and a **flip
condition**: the concrete observation that would make you revisit. "Nothing
would ever change my mind" is dogma, not a decision.

ADRs are immutable once accepted: when you change your mind, write a new one
and mark the old `Superseded by ADR-007`. Showing what you believed *at the
time* is the whole reason the file is worth reading a year later. Date them and
do not backfill — later in this stage you will be asked which held up under
contact with the code and which you revised, and a backfilled ADR has nothing
to say about that.

## Where the work lives

Your capstone is a real Go module in your workspace at `projects/capstone/` —
beside `lessons/`, not inside them. It stays there for the rest of the stage,
and every later lesson grades what it finds there. Create it now:

```sh
cd <your workspace root>              # the directory holding lessons/ and projects/
mkdir -p projects/capstone
cd projects/capstone
go mod init example.com/courier       # any module path; use your own name
git init                              # the capstone keeps its own history
```

The layout the rest of the stage expects:

```
projects/capstone/
├── go.mod
├── README.md            # what it is and how to run it — graded from the next lesson on
├── MILESTONES.md        # your checklist, checked mechanically later
├── cmd/…                # main package(s): thin shells
├── internal/…           # the real packages
└── docs/
    ├── PRD.md
    ├── RISKS.md
    └── adr/ADR-001-storage-is-a-content-addressed-file-tree.md …
```

**How the later lessons find your project.** The test-verified lessons in this
stage cannot ship a fixed exercise — the code being graded is yours. They ship a
*conformance harness* instead: tests that shell out to your project's toolchain
and inspect your files. Each resolves the project directory in this order:

1. the `TUTOR_CAPSTONE_DIR` environment variable, if set;
2. otherwise a single-line path in a file named `capstone.path` in that
   lesson's `exercise/` directory (a relative path resolves against the
   exercise directory);
3. otherwise the built-in fallback: `projects/capstone` at your workspace
   root, which each harness finds by walking up from its own exercise
   directory. From a scaffolded exercise that is
   `../../../../projects/capstone`, but nothing depends on the depth.

Create `projects/capstone` as above and the fallback finds it on its own —
that is the whole point of the convention. If none of the three resolves to a
directory containing a `go.mod`, every test in that harness fails immediately
and tells you how to fix it; running a harness *before* the project exists
fails that way expectedly, not because anything is broken. The environment
variable is the override, set once per shell:

```sh
export TUTOR_CAPSTONE_DIR="$HOME/path/to/workspace/projects/capstone"
```

The `capstone.path` file is per-lesson and survives new shells:

```sh
cd lessons/<the-s7-lesson>/exercise
echo '../../../../projects/capstone' > capstone.path
ls "$(cat capstone.path)/go.mod"      # verify before trusting it
```

Count the `../` from the exercise directory to your workspace root yourself;
that last `ls` is the check, and it takes five seconds.

## Exercise

Open [`exercise/`](exercise/) and fill the templates in this order:
[`SELECTION.md`](exercise/SELECTION.md) (score your candidate — two candidates
beats one), [`PRD.md`](exercise/PRD.md),
[`ADR-000-template.md`](exercise/ADR-000-template.md) copied once per decision,
[`MILESTONES.md`](exercise/MILESTONES.md), then
[`RISKS.md`](exercise/RISKS.md). When they are filled in, move them into the
project as [`exercise/README.md`](exercise/README.md) describes. The templates
are scaffolding; the project directory is the deliverable.

Acceptance criteria:

1. `projects/capstone/` exists, contains a `go.mod`, and `go build ./...`
   succeeds inside it. A single `main.go` that prints the project name is
   enough — this is the module, not the milestone.
2. `docs/PRD.md` states a problem with no technology nouns in it, names the
   target user, lists 5-9 functional requirements with stable IDs (F1, F2, …),
   gives non-functional requirements as numbers rather than adjectives, and
   lists at least five non-goals — of which at least two are things a
   reasonable person would expect you to build.
3. At least three ADRs exist in `docs/adr/`, numbered and dated, each with at
   least two real alternatives and an honest cost for each, a decision citing a
   requirement or non-goal ID, a consequences section stating what you now live
   with, and a concrete flip condition.
4. `MILESTONES.md` sits at the project root with four to seven milestones. The
   first one is a walking skeleton touching every layer (number them from M0
   or from M1 — pick one and keep it; the reference projects you will be
   graded against start at M1). Each milestone has two to four
   acceptance criteria written as `- [ ]` task-list items, and the list is
   ordered by risk.
5. `docs/RISKS.md` holds at least five risks, each with a cost in hours, a
   likelihood, an early-warning trigger, and a mitigation you can act on now.
   At least one mitigation is a timeboxed spike scheduled before the milestone
   it protects, and at least one risk is non-technical.
6. `SELECTION.md` is complete: every rubric row answered for your chosen
   project, and the one flow that makes the thing worth having, in one
   sentence.
7. You can say aloud which milestone you would cut first if you lost a third of
   your time, what the project still does without it, and which non-goal stops
   the scope growing back.

There is nothing to run. This lesson is verified by a 45-90 minute design
review in which your tutor argues your project is too big, deletes milestones
to see what breaks, and takes the losing side of every ADR you wrote. Come with
the documents finished; a plan defended is worth more than a plan admired.

## Further reading

- [go.dev — Organizing a Go module](https://go.dev/doc/modules/layout) — the
  official word on `cmd/`, `internal/`, and when a package earns existence.
- [Michael Nygard — Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
  — the original ADR format, still the best two pages on the subject.
- [Alistair Cockburn — Walking Skeleton](https://wiki.c2.com/?WalkingSkeleton)
  — where M0 comes from, and why thin-but-complete beats thick-but-partial.
- [Basecamp — Shape Up, "Fixed time, variable scope"](https://basecamp.com/shapeup/1.5-chapter-06)
  — planning against a budget you cannot move, which is exactly your situation.
