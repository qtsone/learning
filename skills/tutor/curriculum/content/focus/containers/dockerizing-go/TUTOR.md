# Tutor notes — Dockerizing Go Apps

## Where the learner is

Second lesson of the containers pack, straight after Docker Fundamentals, so
images, layers, the build cache, tags-versus-digests and the basic command
vocabulary are one lesson old and still fragile. They have S0-S4: intermediate
Go including the concurrency arc, interfaces, generics, JSON and io, plus
engineering practice — packages, TDD, debugging, `database/sql` with
`modernc.org/sqlite`, security basics, HTTP clients over `httptest`, and
GitHub Actions.

They have **not** done HTTP servers as a subject. `exercise/main.go` is given
to them finished for exactly that reason. If they start asking about
`ServeMux` patterns, graceful shutdown or middleware, answer in one sentence
and steer back: this lesson is about the image.

Many learners will have no Docker daemon. That is planned for — every graded
check is static. Encourage installing Docker (the size numbers are far more
convincing when they measure them), but never let its absence become an
excuse or a blocker.

## Common misconceptions

- **"Multi-stage means the image is built twice, so it's slower."** Both
  stages were always going to be built; multi-stage only changes which one is
  kept. The build cache still applies to the builder stage.
- **"I'll just `RUN rm -rf` the toolchain at the end."** Layers are additive;
  a deletion in a later layer adds a whiteout entry and *grows* the image.
  This is the single most common wrong instinct here.
- **"`CGO_ENABLED=0` is a size optimization."** It is a linking decision. The
  size effect is incidental; the runtime-base compatibility effect is the
  point.
- **"`exec …: no such file or directory` means my COPY path is wrong."** It
  usually means the dynamic loader is missing. Make them articulate *which*
  file the kernel could not find.
- **"scratch is free."** It costs CA certificates, tzdata, `/etc/passwd`, a
  shell and `/tmp`. Ask which of the four they have handled.
- **"Alpine is small, so Alpine is the safe default for Go."** Alpine is musl.
  Fine while cgo is off; a trap the moment it is not.
- **"`EXPOSE` opens the port."** It is metadata. `-p` publishes.
- **"USER app is non-root, done."** Names need `/etc/passwd`. On scratch that
  is a startup failure, not a warning.
- **Reordering `COPY` for cache without understanding it** — cargo-culted from
  a blog post. Ask them to predict which layers rebuild after a one-character
  edit to `main.go`.

## Grilling points

- "Your final image is 10 MB and the naive one is 900 MB. Name the three
  heaviest things that did not come across, and say why none of them are
  needed at runtime."
- "Walk me through what the kernel does when it starts your binary. Where
  does a dynamically linked binary go looking, and what happens on scratch?"
- "You need to add a dependency that requires cgo. What changes in your
  Dockerfile — every line that has to move?"
- "Which of your layers rebuild when you change one character in `main.go`?
  Which rebuild when you add a dependency to `go.mod`?"
- "Your service starts failing HTTPS calls to a payment API after you move to
  scratch. Diagnose it out loud."
- "Justify your base image to a reviewer who prefers the other one."
- "Is `-ldflags=\"-s -w\"` worth it if it breaks `delve` on the shipped
  artifact? When would you not do it?"

## Grading rubric

- **A** — All static checks pass; the choice of base is defended with the
  specific trade-off it implies, not a slogan; they can explain the cgo
  failure mode as a missing *loader*; layer ordering is explained in terms of
  change frequency; `NOTES.md` shows real reasoning, and real numbers if they
  had Docker. Live checks green where available.
- **B** — Checks pass and the mechanics are right, but the explanation is
  partly recited: e.g. `CGO_ENABLED=0` "because scratch needs it" with no
  linking model behind it, or a base chosen because the lesson used it.
- **C** — Green only after heavy hinting, or the Dockerfile is a copied
  template they cannot modify: they cannot say what changes if the base
  swaps, or what would break without one of the flags. Pass only if a
  time-boxed remediation lands.
- **Fail** — Single-stage, or a toolchain image in the final stage, or root
  user, or `NOTES.md` unanswered; or they cannot explain any line they wrote.
  Remediate, do not advance.

## Remediation ladder

1. "Read the FAIL line and its `fix:` hint aloud. Which instruction does it
   name, and which stage is that instruction in?"
2. "Draw the two stages as two boxes. What crosses the line between them, and
   what is the one instruction that moves things across?"
3. For the cgo failure: "The file exists — you can see it in the COPY. So
   what *other* file is the kernel looking for when it starts an executable?"
   For caching: "Which of these two files changes more often: `go.mod`, or
   `main.go`? Now which one should be copied first, and why does the order
   matter at all?"
4. Walk the skeleton aloud — `FROM … AS build`, manifests, download, source,
   build with flags, `FROM` runtime, `COPY --from`, `USER`, `EXPOSE`,
   `ENTRYPOINT` — naming each instruction's purpose, but let them type every
   line and choose their own base.

## After passing

Preview: "You can build one image. Next lesson runs several together —
Compose gives your service a database, a network where they find each other
by name, and volumes that decide what survives a restart."
