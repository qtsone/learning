# Docker Fundamentals

> `focus.containers.docker-fundamentals` · ~2-3h · Stage: Focus: Containers I

## Objectives

By the end of this lesson you can:

- Explain the difference between an image and a container, and how a
  container's writable layer relates to the image's read-only layers.
- Explain how image layers are built and cached, and predict which Dockerfile
  changes invalidate the cache.
- Run, inspect, stop, and remove containers with `docker` CLI commands,
  including port publishing and log inspection.
- Pull and push images to a registry, and explain how tags and digests
  identify images.
- Explain how namespaces and cgroups make a container different from a virtual
  machine.

You will get much more out of this lesson with
[Docker installed](https://docs.docker.com/get-started/get-docker/), so install
it if you can. It is not required to pass: the exercise grades the files you
write, daemon or no daemon.

## A container is a process, not a small computer

You will hear "a container is a lightweight VM" constantly. It is wrong, and
believing it will mislead you about performance, security, and half the errors
you hit. Kill it now.

Run `ps` on your machine: dozens of processes, all sharing one kernel. A
container is one of those processes — started with a *restricted view of the
system* and a *ceiling on what it may consume*. Two Linux kernel features do
that work:

**Namespaces — what a process can see.** A namespace is a lie the kernel tells
a process about the world, and a container typically gets one of each kind:

- *PID*: its own process tree. Its main process is PID 1 and cannot see host
  processes at all.
- *Mount*: its own filesystem root. What it calls `/` is a tree assembled from
  the image, not your `/`.
- *Network*: its own interfaces, routes, and port space. Two containers can
  both "listen on 8080" without colliding — those are two different 8080s.
- *UTS, IPC, user*: its own hostname, shared-memory objects, and user-ID
  mapping.

**Cgroups (control groups) — what a process can use.** A cgroup caps CPU time,
memory, and I/O for a group of processes. `docker run --memory=64m` is a cgroup
limit; exceed it and the kernel OOM-kills your process. There is no virtual RAM
chip anywhere.

Contrast a virtual machine: a hypervisor emulates hardware, a *second kernel*
boots on it, and a full operating system runs inside — seconds of boot time and
hundreds of megabytes of RAM before your code runs. A container adds neither.

Four consequences worth carrying:

1. **A Linux container needs a Linux kernel.** On macOS and Windows, Docker
   quietly runs a small Linux VM and puts your containers inside it. That
   hidden VM is exactly why people conflate the two ideas.
2. **The "operating system" inside an image is only userland.** A Debian image
   ships Debian's files and libraries — `/bin`, `/lib`, package manager — but
   uses *your* kernel. No init, no kernel, no boot.
3. **Isolation is thinner than a VM's.** One shared kernel is one shared attack
   surface; for hostile multi-tenant workloads people still reach for VMs.
4. **You cannot cheat the kernel.** No Windows containers on a Linux kernel;
   no kernel modules from inside a container.

## Images and containers

An **image** is a read-only template: a stack of filesystem layers plus metadata
(default command, environment variables, working directory, user, declared
ports). It is inert. A **container** is an image, plus a thin **writable layer**
on top, plus a process running inside the namespaces and cgroup above.

You know this relationship from S1: an image is the compiled binary on disk, a
container is a process you started from it. One binary, many processes, each
with its own memory. One image, many containers, each with its own writable
layer.

That writable layer is the part people get wrong. Every file your program
creates or modifies while running lands there, not in the image — a running
container never modifies its image. And `docker rm` deletes the writable layer
along with the container. Data you want to keep has to live somewhere else;
that "somewhere else" is a volume, and it is the third lesson of this pack.

## Layers and the union filesystem

Each Dockerfile instruction that changes the filesystem produces a **layer**: a
diff — files added, changed, deleted — named by the SHA-256 digest of its
content. At run time the layers are stacked into a single directory tree by a **union
filesystem** (usually OverlayFS). Reading a path finds it in the topmost layer
that has it. Writing copies the file up into the writable layer first and edits
the copy — **copy-on-write**. Deleting a file that lives in a lower layer does
not remove those bytes; it writes a *whiteout* marker above that hides them.

That last sentence has a security consequence people learn the hard way:

```dockerfile
COPY secrets.env .
RUN rm secrets.env      # the file is hidden, not gone
```

The bytes are still sitting in the earlier layer, and anyone who pulls the
image can read them. Secrets must never enter the build in the first place —
which is what `.dockerignore` is for, and why you will write one today.

Layers are content-addressed, so they are shared: if two images build on the
same base, that base is stored once on disk and downloaded once. "This image is
900 MB" often means "899 MB of it you already had".

## The build cache

For each instruction, the builder computes a cache key from the parent layer's
digest plus the instruction itself — and for `COPY`/`ADD`, plus a checksum of
the files being copied. If a layer with that key already exists, it is reused
and the instruction never runs. **The first miss invalidates every instruction
after it**, because each key includes the parent.

So instruction order is a performance decision: put what changes rarely at the
top, what changes constantly at the bottom. For a Go program the canonical
shape is:

```dockerfile
COPY go.mod go.sum ./
RUN go mod download      # cached until your dependencies change
COPY . .                 # misses on every source edit
RUN go build -o /bin/app .
```

Invert those and you re-download every dependency for a one-character typo fix,
because `COPY . .` misses and drags `go mod download` down with it. (The
exercise module has no third-party dependencies yet, so it has no `go.sum` —
copy `go.mod` alone. The shape is what matters.)

Two more cache habits: chain package-manager commands into one `RUN`
(`apt-get update && apt-get install -y …`, or a cached `update` layer feeds a
months-stale package index into `install`), and keep junk out of the build
context, since a stray edit to a notes file busts `COPY . .` all by itself.

## Tags, digests, and why `latest` is not a version

A full image reference looks like:

```
ghcr.io/ada/clockwork:1.4.0
└─ registry ┘└ repository ┘└ tag ┘
```

A **tag** is a mutable, human-friendly pointer to an image; the publisher moves
it whenever they like — that is the *point* of `golang:1.22`, which slides
forward as patch releases land.

`latest` is not special. It is simply the tag Docker assumes when you do not
name one. Nothing guarantees it points at the newest release, and it moves
without warning. `FROM node:latest` is how a pipeline green on Monday goes red
on Wednesday with no commit in between — and the Monday image cannot be
reproduced, so you cannot bisect.

A **digest** is the SHA-256 of the image content, so it is an identity rather
than a pointer: `FROM golang:1.22-bookworm@sha256:8a9f…` resolves to exactly
the same bytes forever. The price is that you now own updating it — usually
delegated to a bot that opens a pull request when the tag moves.

The working rule: never `latest`; always a concrete tag; add a digest when you
need real reproducibility (releases, base images, anything you deploy).

## Registries

A **registry** stores repositories of tagged images: Docker Hub (the default),
GitHub Container Registry (`ghcr.io`), and every cloud provider's own.

```sh
docker pull golang:1.22-bookworm          # Docker Hub implied
docker login ghcr.io                      # credentials before pushing
docker tag clockwork:1.4.0 ghcr.io/ada/clockwork:1.4.0
docker push ghcr.io/ada/clockwork:1.4.0
```

Omit the registry and Docker Hub is assumed; omit the namespace too (`golang`,
`postgres`) and you get Hub's curated `library/` images. Pulls also happen
implicitly — `docker run` on an image you lack pulls it first. Pushing is
almost always CI's job rather than your laptop's; you built that muscle in S4
and this pack's capstone will use it. One operational surprise: Docker Hub
rate-limits anonymous pulls, so a pipeline suddenly reporting "too many
requests" is usually hitting that, not a bug.

## The command vocabulary

```sh
docker build -t clockwork:1.0.0 .   # build an image; "." is the build context
docker run -d -p 8080:8080 --name clock clockwork:1.0.0
docker ps                           # running containers
docker ps -a                        # including stopped ones
docker logs -f clock                # stream stdout/stderr
docker exec -it clock sh            # a second process inside the container
docker stop clock                   # SIGTERM, then SIGKILL after ~10s
docker rm clock                     # delete the container and its writable layer
docker image ls                     # local images
docker image rm clockwork:1.0.0
docker system df                    # what all this is costing you on disk
```

Five of those deserve a note:

- **The `.` in `build` is the build context**, not "here". That directory tree
  is packed up and handed to the daemon before the build starts, and `COPY` can
  only read from it. Big context, slow builds — hence `.dockerignore`.
- **`-p 8080:8080` is `host:container`.** The container has its own network
  namespace, so its port is unreachable from your machine until you publish it.
  `EXPOSE` in a Dockerfile does *not* do this; it is documentation.
- **`docker logs` reads the container's stdout and stderr** — which is why
  containerized programs log to standard output instead of to files. The
  platform collects streams, not files.
- **`docker exec` starts an extra process inside an existing container's
  namespaces** — not a new container. It is your main debugging tool, and it
  fails on minimal images with no shell, a tradeoff the next lesson makes
  deliberately.
- **`docker stop` sends SIGTERM first**, giving your program a window to shut
  down cleanly, then SIGKILL. Your process is PID 1, so it receives that signal
  directly — *if* it is really PID 1. Which brings us to:

## CMD, ENTRYPOINT, exec form and shell form

`CMD` and `ENTRYPOINT` both declare what runs when the container starts —
roughly, `ENTRYPOINT` is the command and `CMD` its default arguments; with only
`CMD`, it is the whole command. Each accepts two syntaxes, and the difference is
not cosmetic:

```dockerfile
CMD ["/bin/clockwork"]     # exec form  — your binary is PID 1
CMD /bin/clockwork         # shell form — /bin/sh -c "/bin/clockwork" is PID 1
```

In shell form the container's PID 1 is a shell, and a plain `sh` does not
forward signals to its child. `docker stop` then hits the shell, your program
never sees SIGTERM, and ten seconds later everything is SIGKILLed mid-flight.
Always use exec form for the process you actually want to run.

The other instructions you need today: `FROM` (base image), `WORKDIR`
(absolute working directory for everything after it — set one, don't inherit
whatever the base left you in), `COPY` (context → image), `RUN` (execute at
*build* time, producing a layer), `ENV` (default environment variable),
`EXPOSE` (documentation of the port your program listens on).

One last framing: the `docker` CLI talks to a daemon that does the work, but
the image format and runtime underneath are standardized by the **OCI** (Open
Container Initiative). That is why Podman runs Docker images and why Kubernetes
runs containers with no Docker installed anywhere. Learn the concepts, not the
vendor.

## Exercise

Open [`exercise/`](exercise/). It contains `clockwork`, a tiny HTTP service that
answers every request with the current time and logs each one to stdout. Read
`main.go` but change nothing in it — the few lines that register a handler and
start a server are yours to use, not to design (that is S5's subject). Also in
the folder: a `Dockerfile` a previous owner wrote in ten minutes, and some junk
that should never reach an image.

Rewrite the `Dockerfile`, write a `.dockerignore`, and answer the four
questions in `NOTES.md`. Acceptance criteria:

1. The `Dockerfile` starts `FROM` a base image pinned to a concrete tag or
   digest — never `latest`, never an untagged name. Use Go 1.22 or newer;
   `go.mod` says `go 1.22`.
2. It sets an absolute `WORKDIR` before copying anything.
3. It copies `go.mod` and runs `go mod download` *before* copying the source,
   so an ordinary edit does not invalidate the dependency layer.
4. It copies the source, builds the binary, and declares the start command with
   `CMD` (or `ENTRYPOINT`) in **exec form**.
5. `.dockerignore` excludes at least `.git`, `local.env`, and `scratch/`.
6. `NOTES.md` answers all four questions in your own words, in prose.

Then run the referee, as often as you like:

```sh
cd exercise
bash check.sh
```

It grades your files statically — no daemon, no network, no root — so it works
everywhere. With Docker running it also builds your image as a bonus check;
without Docker those lines say `SKIP` and cost you nothing, and the summary
line tells you how many passed and how many were skipped.

If you have Docker, do the full loop by hand too — it is the point of the
lesson:

```sh
docker build -t clockwork:1.0.0 .
docker run -d -p 8080:8080 --name clock clockwork:1.0.0
curl localhost:8080          # or open it in a browser
docker logs clock            # your request, logged
docker exec -it clock sh     # look around: ls /, ps, hostname
docker stop clock && docker rm clock
docker image ls              # note the size — remember it for the next lesson
```

Then change one line in `main.go`, rebuild, and watch which steps say `CACHED`.

## Further reading

- [Docker docs — What is an image?](https://docs.docker.com/get-started/docker-concepts/the-basics/what-is-an-image/)
- [Docker docs — Dockerfile reference](https://docs.docker.com/reference/dockerfile/)
- [Docker docs — Build cache](https://docs.docker.com/build/cache/)
- [`namespaces(7)` — Linux manual](https://man7.org/linux/man-pages/man7/namespaces.7.html)
  — the kernel feature itself, with no Docker in sight.
