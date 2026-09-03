# Capstone: Operations

> `go.capstone.operations` · ~6-10h · Stage: Expert Capstone (Go)

## Objectives

By the end of this lesson you can:

- Build a CI pipeline that runs tests, lint, and vulnerability checks on every
  push and blocks merge on failure.
- Deploy the service to a real target (a container platform, tying in
  focus-pack skills where you took them) with repeatable, scripted steps.
- Instrument the service with structured logs, metrics, and traces, and
  demonstrate answering "is it healthy?" from telemetry alone.
- Implement graceful shutdown and health/readiness endpoints, and show they
  behave correctly during a rollout.
- Write a short runbook covering deployment, rollback, and the top two failure
  modes.

## Software nobody can run is a private hobby

Your project builds, vets, races clean and survived a hardening pass. It also
runs in exactly one place, started by exactly one person, observed by reading
its output in a terminal. Everything in this lesson exists to fix that: to turn
a program that works on your machine into a system that somebody else can
deploy, watch, diagnose and switch off.

That "somebody else" is usually you, three months from now, at an inconvenient
hour, with none of today's context. Write for that person. Every artifact here
is a message to them: the image is "here is exactly what runs", the pipeline is
"here is what I refuse to ship", the telemetry is "here is what it is doing
right now", the runbook is "here is what to do about it".

The referee for this lesson is a bash script rather than a Go harness, because
what it reads is not Go: a Dockerfile, a pipeline definition, a runbook, your
docs. It needs no network, no docker daemon and no cluster — anything that
needs a tool you do not have is reported as skipped, never as a failure.

## First: what shape is your project?

The planning lesson let you pick, so some of you built a service, some a CLI,
some something in between. The operational questions are the same; the answers
differ. Read the row that is yours before the rest of the lesson.

| Question | Long-running service | CLI or batch tool | Library |
|---|---|---|---|
| What ships? | A container image | A container image *and* built binaries per platform | A tagged version — but the demo command still ships as an image |
| "Is it healthy?" | `/healthz` and `/readyz` | Exit codes, plus a `doctor`/self-check subcommand | The example command's self-check |
| Structured logs | One line per request | One line per run: inputs, counts, duration, outcome | From the demo command; the library takes a logger, never owns one |
| Metrics | Counters served over `/metrics` | Counters printed at the end of a run, or written where a scraper can read them | Exposed to the caller, not published |
| Graceful shutdown | Fail readiness, drain, exit | SIGINT mid-run leaves no half-written file and says what it did | The caller's problem — but cancellation must be respected |
| Rollback | Previous image, one command | The previous release, still downloadable | Consumers pin the old version |

Two things do not adapt. Everything in the table gets **written down**, and
everything gets **scripted**. A capability that exists only in your head is
worth nothing at 3am.

## The gate set: a pipeline that can say no

You built a pipeline in S4's ci-cd lesson. Now it guards your own repository,
and the question is which gates earn their runtime. Four:

- **`go build ./...`** — proof it compiles somewhere that is not your laptop,
  with no stale cache and no file you forgot to commit.
- **`go test -race ./...`** — the suite plus the race detector. CI's slower,
  differently-scheduled machine finds races your laptop hides.
- **`go vet ./...`** — the checks you run by hand, run when you forget.
- **`govulncheck ./...`** — you ran it by hand during hardening. Your
  dependencies grow vulnerabilities while you are not looking, so this one has
  to be a robot's job.

Add `test -z "$(gofmt -l .)"` too; the gate costs a second. Three properties
separate a gate from decoration:

- **It runs before the merge, not after.** Trigger on push *and* on proposed
  merges. A check that only runs on the main branch tells you what you already
  broke.
- **It blocks.** This is a repository setting — required status checks on the
  protected branch — not a line in the workflow file. No script can verify it
  for you, which is why you will be asked to show it in review. A gate you can
  merge past is a suggestion.
- **It is fast enough to obey.** Past ten minutes people stop waiting and start
  merging around it. If your suite is slow, that is a suite problem surfacing
  as a process problem.

Guard against the **flaky gate**: one test that fails one run in twenty teaches
everyone to press re-run, and the day it fails for a real reason, everyone
presses re-run. Quarantine or delete it the same day you notice it — a gate is
only worth its interruptions if red always means broken.

## Packaging: one artifact, built once

The unit you deploy is an image: your binary plus the smallest possible world
around it. Two stages, and the second one is nearly empty.

```dockerfile
FROM golang:1.22.12-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/app /usr/local/bin/app
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/app"]
```

Every line is load-bearing:

- **Multi-stage.** A single-stage image ships the compiler, the module cache
  and your source code to production. Attack surface and gigabytes, for
  nothing. The runtime stage carries one file.
- **`CGO_ENABLED=0`.** A static binary needs no libc at runtime, which is what
  lets the runtime stage be a base image with no operating system in it.
- **Pinned base tags.** `golang:1.22.12-bookworm`, never `golang` or
  `golang:latest`. `latest` means "whatever was pushed most recently", so an
  identical rebuild in six months produces a different image — and when you
  bisect a production problem you will not be able to trust the result. Pin a
  tag; pin a digest (`@sha256:…`) if you want it exact.
- **Non-root.** Containers run as root unless told otherwise, so a process that
  escapes its container starts life as root on the host. `USER` after the
  *final* `FROM` — each stage starts fresh, so a `USER` in the build stage does
  nothing for the runtime.
- **Copy `go.mod` and `go.sum` before the source.** Layer caching: dependencies
  re-download only when the module files change, not on every edit. `go.sum`
  belongs in the same layer because it is what makes `go mod download` a
  verification step rather than just a fetch — and because it puts your pinned
  hashes in that layer's cache key, so changing one actually rebuilds it.
  (No dependencies yet, so no `go.sum`? Commit an empty one rather than dropping
  it from the `COPY` — the day you add a dependency is not the day you want to
  remember this.)
- **A `.dockerignore`** (`.git`, docs, deploy scripts) keeps the build context
  small and stops anything secret wandering into a layer.

If you took the containers focus pack, this is `dockerizing-go` applied to your
own repository — reuse everything, including your compose file and manifests.
If you did not, the twenty lines above are the whole requirement; nothing later
in this lesson assumes more.

## Deploy is code, and rollback is a feature

"Deployment" is a sequence of commands; the only question is whether it lives
in a file or in your memory. In a file it can be reviewed, repeated, and run by
somebody who was not there when it was invented. Two properties matter more
than which tool you use:

- **Immutable versions.** Build the image once, tag it with a version, and
  promote *that artifact* onward. Never rebuild "the same" tag: two builds of
  one tag are two different images with one name, and you will eventually debug
  the wrong one.
- **Configuration comes from outside the image.** Same artifact in every
  environment; only the environment differs. An image that bakes in an endpoint
  is an image that cannot be promoted.

**Rollback is the feature you rehearse.** In an incident you roll back
*before* you understand the failure — the correlation between "we deployed" and
"it broke" is enough, and diagnosis is much calmer once users are fine again.
So the rollback path has to be a command you have actually run, not a paragraph
you believe. Run it once, on purpose, when nothing is wrong. Time it.

Then be honest about what rollback cannot undo: a schema migration that dropped
a column, a message consumed from a queue, a release published to strangers.
Where the old version cannot come back, the runbook needs the forward fix
instead, and knowing that in advance is the point.

## Configuration is an interface

Your configuration surface is an API you offer to whoever runs the program, and
it deserves the same care as your Go API.

- **Read the environment at startup, validate it there, and fail loudly.** A
  bad value should stop the process with a clear message — the rollout then
  fails while the old version is still serving. A default silently substituted
  for a typo is an outage that starts hours later.
- **Defaults that run.** With no configuration at all, the program should
  start. Every required-with-no-default variable is a step someone can miss.
- **Document every knob**: name, meaning, default, and whether it is a secret.
  The referee compares every `os.Getenv` in your code against your docs,
  because undocumented configuration gets discovered by whoever is on call.
- **Secrets are not configuration you print.** They arrive by the same
  mechanism and then never appear in a log line, an error message, a metric
  label or a crash dump. If your config type has a `String` method, make it
  redact them.

## Lifecycle: starting, being ready, stopping on purpose

A rollout is a controlled kill. The platform starts a new instance, waits for
it to say it is ready, sends it traffic, then sends the old one SIGTERM and
waits a fixed grace period before SIGKILL. Every part of that sentence is
something your program participates in.

**Health is not readiness** — you met this in S5's observability lesson, and
now it has consequences.

- **Liveness** (`/healthz`): "is this process fundamentally broken?" A failure
  means *restart me*. Keep it dumb: if it checks a dependency, one slow
  database restarts every instance you have.
- **Readiness** (`/readyz`): "can I serve traffic right now?" A failure means
  *stop sending me work*, without a restart. Warming a cache, waiting on a
  dependency, or draining are all not-ready-but-alive.

**Graceful shutdown**, in order:

1. Catch the signal:
   `ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`.
2. Fail readiness *first*. The load balancer needs a few seconds to notice; if
   you skip this, the requests it routes to you during those seconds are the
   ones you drop.
3. `srv.Shutdown(ctx)` with a bounded timeout, so in-flight requests finish.
4. Close what you own — flush buffers, cancel workers, wait for them.
5. Exit with a status code that says which of those happened.

The two numbers have to agree: your shutdown timeout must be **shorter** than
the platform's termination grace period. Longer, and SIGKILL lands mid-drain,
which is exactly the connection reset you were trying to avoid.

For a CLI the same discipline reads differently: SIGINT during a long run must
leave no half-written output file and must print what it did and did not
finish. "Interrupted after 412 of 1000 records; rerun with `--resume`" is the
graceful shutdown of a batch tool.

## Telemetry that answers the only question

The demonstration this lesson asks for is narrow: **answer "is it healthy?"
without reading the code, attaching a debugger, or logging into the box.**
Three signals, three jobs, all from S5 — metrics tell you *something* is wrong,
traces tell you *where*, logs tell you *what*.

What that requires of your project:

- **Structured logs on stdout.** `log/slog`, key/value attributes, fixed keys
  (`method`, `route`, `status`, `duration_ms`), one line per unit of work.
  Fixed keys are what make a log searchable and countable rather than readable;
  free text is a wall. Log level from configuration, so you can turn on debug
  in production without a deploy. Do not log secrets, tokens or whole request
  bodies.
- **Counters someone else can read.** Stdlib `expvar` over `GET /metrics` is
  enough; a Prometheus client is nicer if you already have one. Publish the
  RED trio for request work — rate, errors, duration — and one saturation
  gauge if you have a queue or pool. Watch cardinality: label by *route
  pattern*, never by request path, user id or error string.
- **Traces where they earn their weight.** A trace is worth instrumenting when
  a request crosses processes. A single-process capstone gets most of the value
  from a request id propagated through the logs, which is the same idea with a
  smaller bill. If your project does make outbound calls, propagate the trace
  context and say so in review.

Then rehearse the drill, because it is what you will be asked to do:

1. Start your system. Break it deliberately — bad config, dependency down, a
   handler that panics, a queue you stop draining.
2. From telemetry alone: is it up? is it serving? what fraction is failing?
   since when? which part?
3. Anything you could not answer is a gap in your instrumentation. Fix the
   instrumentation, not the anecdote.

## The runbook

A runbook is the page you open at 3am. Written well it turns an incident into a
procedure; written as prose it is help nobody can use under stress.

Four sections earn their place:

- **How to deploy** — the literal commands, plus how you know it worked.
- **How to roll back** — one command if possible, safe to run before you
  understand anything.
- **What to check when it is down** — an ordered list, cheapest and most likely
  first, each with the command *and* how to read its answer. This section is
  the one that gets used and the one that rots fastest.
- **Who to page** — who owns this, how they are reached, when it is fair to
  wake them, who to escalate to after how long. "Me, and nobody else" is a
  legitimate answer, if you also write what happens when you are unreachable.

Plus the top **two failure modes**: the ways this system actually breaks, not a
catalogue of everything that could. For each one: the symptom you see first,
the check that confirms it, the cause in a sentence, and the action. If your
project has never broken, predict the two your design makes most likely and
label them predicted — writing them down is what makes you notice you were
right.

Two habits keep a runbook true. **Commands, not descriptions**: a step you
cannot paste while half awake is not a step. And **use it**: run your own
runbook during the shutdown and rollback rehearsals. Every place you improvised
is a place it was wrong.

## Exercise

Make your capstone operable. The referee in [`exercise/`](exercise/) reads your
project, resolved in the same order as the core-build harness:

1. `TUTOR_CAPSTONE_DIR` in the environment;
2. the first line of `exercise/capstone.path` (relative paths resolve against
   the exercise directory);
3. `projects/capstone` at your workspace root, which the referee finds by
   walking up from its exercise directory.

If none of them is a directory containing a `go.mod`, the first check fails and
tells you how to fix it. [`exercise/RUNBOOK-template.md`](exercise/RUNBOOK-template.md)
gives you the headings with prompts under each; the prompts are not answers.

Acceptance criteria — 1-8 are exactly what the referee checks:

1. **The image is built properly.** A `Dockerfile` (root, `docker/`, `build/`
   or `deploy/`) that is multi-stage, pins every base image to a tag or digest
   (never `latest`, never untagged), and runs as a non-root user in the *final*
   stage.
2. **Deployment is scripted.** A `deploy`/`release` target in a `Makefile`, a
   `deploy.sh`, a compose file, or manifests under `deploy/`, `k8s/` or
   `manifests/`. Steps that live in your head do not count.
3. **CI exists and runs on both events**: on push and on proposed merges.
4. **CI runs four gates**: `go build`, `go test -race`, `go vet`,
   `govulncheck`. If your workflow calls `make`, the referee reads your
   `Makefile` too.
5. **A runbook** — `RUNBOOK.md` at the project root (or under `docs/`) — with
   headings for deploy, rollback, what to check when it is down, and who to
   page; with **real commands in fenced blocks** under deploy and rollback; and
   with a failure-modes section naming at least two.
6. **Telemetry**: a structured logger (`log/slog` or equivalent) actually used
   at call sites; a health signal (`/healthz` + `/readyz`, or a `HEALTHCHECK`,
   or a documented self-check command for a CLI); and a metrics surface —
   `expvar`, `/metrics` or a client library for a service, or documented
   end-of-run counters for a CLI.
7. **Lifecycle**: the process handles SIGTERM (`signal.NotifyContext` or
   `signal.Notify`), and — if it serves HTTP — drains with `srv.Shutdown(ctx)`.
8. **Configuration is documented**: a Configuration section listing your knobs,
   and *every* variable read via `os.Getenv` documented somewhere in your docs.

The rest is graded in review, because no script can see it:

9. **The gate actually blocks.** Show the branch protection / required-checks
   setting, or the pull request that could not merge while red.
10. **You deployed it somewhere real** and can redeploy it live, from the
    scripted steps, without editing anything by hand.
11. **You have rehearsed the rollback**, know how long it takes, and can name
    what it cannot undo.
12. **The shutdown demo**: send SIGTERM under load and show that readiness
    fails first, in-flight work finishes, and the exit is clean.
13. **The telemetry drill**: with a fault you introduced, answer "is it
    healthy?" from telemetry alone — up? serving? failing how much? since
    when? where?
14. **The runbook is true.** You followed your own rollback and triage steps
    and fixed everything you had to improvise.

Run the referee from the exercise directory, as often as you like:

```sh
cd exercise
bash check.sh                             # static checks, no network needed
TUTOR_OPS_ALLOW_NETWORK=1 bash check.sh   # also try docker build / kubectl dry-run
```

## Further reading

- [Docker docs — Multi-stage builds](https://docs.docker.com/build/building/multi-stage/)
- [go.dev — govulncheck](https://go.dev/blog/govulncheck)
- [The Twelve-Factor App — Config](https://12factor.net/config)
- [Google SRE book — Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/)
