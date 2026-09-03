# Tutor notes — Docker Fundamentals

## Where the learner is

First lesson of the containers focus pack, inserted right after S4. They have
intermediate Go (concurrency under the race detector, interfaces, generics,
JSON, io) and engineering practice (packages, TDD, debugging, SQL, HTTP
clients over `httptest`, GitHub Actions, code review). They have **not** done
S5: no HTTP server design, no gRPC, no profiling, no observability. Do not
discuss server architecture, graceful shutdown design, or Kubernetes here —
the provided `main.go` is scaffolding they read, not a design to critique.

Operationally this is their first infrastructure lesson: much of it is
vocabulary, and the grading is deliberately static so it works on a laptop
with no Docker. If they do have Docker, push them to run the manual loop at
the end of the exercise — the concepts land ten times harder after watching
`CACHED` appear on their own screen.

## Common misconceptions

- **"A container is a lightweight VM."** The one to kill. Probe for the two
  mechanisms by name and by role: namespaces = what it can *see*, cgroups =
  what it can *use*. If they say "it has its own kernel", stop and rebuild the
  model — everything downstream (start times, isolation strength, why macOS
  needs a hidden Linux VM) depends on it.
- **"The image contains an operating system, so it boots."** It contains
  userland files. No init, no kernel, no boot.
- **Image and container used interchangeably.** Push the S1 analogy: binary on
  disk vs process. Ask how many containers one image can produce.
- **"I deleted the secret in a later RUN, so it's gone."** Whiteout markers
  hide, they don't erase. Anyone who pulls the image can read the earlier
  layer.
- **"`latest` means the newest version."** It is the default tag name, nothing
  more, and it is mutable like every other tag.
- **"`EXPOSE` publishes the port."** It documents. `-p` publishes.
- **Cache order not understood as ordering.** They often "know" the cache
  exists but cannot say *why* `COPY go.mod` comes first. Make them state the
  cache key: parent layer + instruction + (for COPY) file contents, and that
  the first miss invalidates everything below.
- **Shell-form CMD treated as style.** It changes who is PID 1 and therefore
  who receives SIGTERM.
- **"Data in the container is saved because the container still exists after
  `docker stop`."** True until `docker rm`, and false as a strategy. Volumes
  come in the Compose lesson.

## Grilling points

- "You have Docker on a Mac. My image says `FROM debian`. Whose kernel runs my
  process, and where did it come from?"
- "Two containers from the same image both write `/var/log/app.log`. Do they
  see each other's writes? Why not?"
- "Walk me down your Dockerfile and tell me, for each instruction, what busts
  its cache."
- "You move `COPY . .` above `RUN go mod download`. The build still works. Why
  is it worse?"
- "Your team pins `FROM golang:1.22`. A colleague wants digests everywhere.
  Argue both sides, then tell me what you'd actually do."
- "Your container exits immediately after `docker run`. Name three different
  causes and how you'd tell them apart." (Wrong CMD path, program crashed —
  read `docker logs` — or the process legitimately finished; a container lives
  exactly as long as PID 1.)
- "Which of the commands you learned would you reach for first if a container
  is running but not responding on the published port?"

## Grading rubric

- **A** — `check.sh` fully green; the Dockerfile's ordering is explained as a
  cache-key argument, not a recipe; `NOTES.md` answers are their own words and
  correct on namespaces/cgroups and the writable layer; they can defend exec
  form via PID 1 and signals. If they have Docker, they ran the manual loop and
  can report what `CACHED` did after a one-line edit.
- **B** — Green checks, sound Dockerfile, but explanations are recipe-shaped
  ("you always put go.mod first") or the VM distinction is hand-wavy in one
  direction. Fix the specific gap verbally, then advance.
- **C** — Green only after multiple hints, or `NOTES.md` reads like it was
  produced to satisfy the word count. Re-grill on questions 1 and 2 before
  advancing; the next lesson leans hard on both.
- **Fail** — Still believes containers are VMs, or cannot say what happens to
  files on `docker rm`. Do not advance: multi-stage builds are unintelligible
  without the layer model.

## Remediation ladder

1. "Read the failing line of `check.sh` output aloud. What exactly is it asking
   for?" (The fix message names it; a surprising number of stalls end here.)
2. For the cache ordering: "Which of your files changes ten times an hour, and
   which changes once a month? Now order the instructions by that."
3. For the VM confusion: "How long does it take to boot Linux? How long does
   `docker run` take? What does that difference tell you about what did *not*
   happen?"
4. For exec vs shell form: "In shell form, what is PID 1 inside the container —
   your binary, or `/bin/sh`? Who gets the SIGTERM from `docker stop`?"
5. Only then, walk the target Dockerfile shape aloud instruction by instruction
   — but have them type it and say why each line is where it is.

## After passing

Preview: "Your image works, and it is enormous — it ships an entire Go
toolchain to production. Next lesson: multi-stage builds, why Go's static
binaries make containers unusually small, and how to choose a runtime base
without breaking TLS."
