# Dockerizing Go Apps

> `focus.containers.dockerizing-go` · ~2-3h · Stage: Focus: Containers I

## Objectives

By the end of this lesson you can:

- Write a multi-stage Dockerfile that builds a Go binary in one stage and
  copies only that binary into a minimal final stage.
- Explain why `CGO_ENABLED=0` is required for a Go binary to run on
  `scratch`, and what static versus dynamic linking means here.
- Choose between `scratch` and distroless and justify the trade-off —
  debuggability, CA certificates, timezone data, shell access.
- Order Dockerfile instructions so `go.mod`/`go.sum` downloads are cached
  separately from source changes.
- Compare final image sizes before and after multi-stage optimization and
  say where the savings came from.

You should install Docker to get the full experience of this lesson — the
measurements land differently when you watch them yourself. But the exercise
is graded by reading the files you write, so you can complete it without a
daemon, and `check.sh` will tell you exactly which checks it skipped.

## Why Go is unusually good at this

Containerizing a Python or Node service means shipping the interpreter, its
standard library, the package directory, and often a C compiler that some
dependency needed at install time. The application is a thin film on top of a
runtime environment you have to reproduce exactly.

Go compiles to a single executable file that contains everything it needs:
your code, every package you imported, the garbage collector, the scheduler,
the whole runtime. There is no `go` on the target machine. Given the right
build flags there is not even a C library on the target machine.

That means the ideal Go image is *one file*. Everything above that is
overhead you chose to accept.

## The naive image, and where its weight is

The obvious Dockerfile — the one in `exercise/Dockerfile.naive` — starts from
`golang:1.22-bookworm`, copies the source in, builds, and runs the result. It
works. It is also around 900 MB of image to ship a 9 MB program.

Three things are in there that never execute in production:

- **The toolchain** — compiler, linker, `go` command: a few hundred MB.
- **The standard library source** — the toolchain ships `src/` for every
  package, because it compiles from source. Your binary already contains the
  *compiled* parts it uses.
- **A Debian userland** — shell, `apt`, coreutils, package metadata. Useful
  for building, irrelevant for running, and every package in it is attack
  surface and CVE reports for the rest of the image's life.

Remember from the fundamentals lesson: an image is a stack of layers, and the
image you ship is the sum of *all* of them. Deleting the toolchain in a later
`RUN rm -rf` does not help — the earlier layer still exists underneath, and
the file is merely hidden by a whiteout entry in the union filesystem. You
cannot subtract weight from an image. You can only decline to add it.

## Multi-stage builds

The fix is to build in one image and ship a different one:

```dockerfile
FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY . .
RUN go build -o /out/timesvc .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/timesvc /usr/local/bin/timesvc
ENTRYPOINT ["/usr/local/bin/timesvc"]
```

Each `FROM` starts a new stage with a fresh filesystem. Only the **last**
stage becomes the image you ship. `COPY --from=build` reaches into an earlier
stage's filesystem and lifts one path across; nothing else travels.

Two things people get wrong here:

- Earlier stages are still built, and still cached. Multi-stage is not "build
  twice"; it is "keep one".
- Naming stages (`AS build`) is optional — you can write `--from=0` — but
  names survive reordering and read better. Name them.

The builder stage can be as fat as it likes. That is the point: the
constraint on your build environment and the constraint on your runtime
environment are now separate decisions.

## CGO, static linking, and the error that lies to you

Go can call C, and two standard-library packages have C implementations it
prefers when C is available: `net` (the system DNS resolver) and `os/user`.
The `CGO_ENABLED` variable decides. It defaults to `1` when a C toolchain is
present — and `golang:1.22-bookworm` has gcc, so on a default build it *is*
present.

With cgo on, your binary is **dynamically linked**: it does not contain the C
library, it contains a note saying "at startup, load `libc` using the program
interpreter at `/lib64/ld-linux-x86-64.so.2`". Copy that binary into an empty
image and the kernel cannot find the interpreter, so it refuses to start the
process with:

```
exec /usr/local/bin/timesvc: no such file or directory
```

The file it means is not your binary. Your binary is right there. The missing
file is the loader. This error costs people whole afternoons.

With `CGO_ENABLED=0`, Go uses its own pure-Go DNS resolver and user lookup,
links **statically**, and produces a binary with no external dependency at
all — no loader, no libc, nothing. That is what lets a Go service run on an
image containing literally one file.

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/timesvc .
```

Inside the `golang` image `GOOS` is already `linux`; writing it is cheap
documentation and becomes load-bearing the day you cross-compile.

What do you give up? Any dependency that needs cgo stops compiling. In
practice this is a short list, and the ecosystem has moved: the SQLite driver
you used in S4, `modernc.org/sqlite`, is pure Go precisely so builds like
this one stay possible. Its popular alternative, `mattn/go-sqlite3`, wraps
the C library and would force cgo back on — which then forces a runtime base
with a matching libc.

## Choosing a runtime base

**`scratch`** — the empty image. Zero bytes, and nothing in it. Everything
the operating system usually provides is now your problem:

- No CA certificates. Every outbound HTTPS call fails with
  `x509: certificate signed by unknown authority` — the same trust store your
  S4 HTTP-client work relied on without noticing. Fix:
  `COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/`.
- No `/usr/share/zoneinfo`. `time.LoadLocation("Europe/Berlin")` returns an
  error. Fix: `import _ "time/tzdata"` in your Go code, which embeds the IANA
  database in the binary (about 450 KB) — the exercise service already does.
- No `/etc/passwd`, so `USER app` cannot be resolved and the container fails
  to start. Use a numeric uid.
- No shell, no `ls`, no `/tmp`. `docker exec -it … sh` is not a thing here.

**Distroless** (`gcr.io/distroless/static:nonroot`) — a few MB, maintained by
Google, built as "scratch plus exactly what a static binary actually needs":
CA certificates, timezone data, `/etc/passwd` with a `nonroot` user at uid
65532, and `/tmp`. Still no shell and no package manager, so the attack
surface stays near zero. There is a `:debug` tag with busybox for the day you
need to look inside. This is the default worth reaching for.

**Alpine** (`alpine:3.20`) — about 8 MB, with a shell, `apk`, and the normal
Unix tools. Debugging is comfortable. The trap: Alpine uses **musl** as its C
library, not glibc. If you ever build with cgo on a Debian-based builder and
run on Alpine, you get the same lying "no such file or directory", because
the glibc loader your binary asks for does not exist there. With
`CGO_ENABLED=0` the question never arises — which is the honest reason so
many Go Dockerfiles pair a Debian builder with an Alpine runtime and get away
with it.

Pick by what you will do at 3 a.m.: distroless if you debug from logs and
metrics, Alpine if you want a shell, scratch if you enjoy the discipline and
have handled all four gaps above.

## Running as a non-root user

Without a `USER` instruction, your process runs as uid 0 inside the
container — and container uid 0 is host uid 0 unless a user namespace is
remapping it. A container escape or a mounted host path then starts from
root. Remember the fundamentals framing: this is a *process with a namespaced
view*, not a machine.

```dockerfile
USER 65532:65532
```

Use the numeric form. It works on images with no `/etc/passwd`, it removes any
ambiguity about which uid the files must be readable by, and it is the form an
orchestrator can actually verify: Kubernetes' `runAsNonRoot` setting reads the
uid the image declares, and refuses to start a container whose `USER` is a name
it cannot resolve to a number by itself.

Non-root has one consequence to plan for: your process cannot bind ports
below 1024. Listen on 8080 and let the port mapping deal with it, which is
what the exercise service does.

## Caching the module download

From the fundamentals lesson: each instruction produces a layer, and a
layer's cache is reused only if its inputs are unchanged — and when one layer
is invalidated, every layer after it rebuilds too. `COPY . .` invalidates on
*any* file change, so this Dockerfile re-downloads every dependency each time
you fix a typo:

```dockerfile
COPY . .
RUN go mod download
RUN go build …
```

Split it. The manifests change rarely; your source changes constantly:

```dockerfile
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build …
```

Now a source edit invalidates only the last two layers. The rule
generalizes to every language: **copy dependency manifests, resolve them,
then copy source** — ordered by how often each thing changes, rarest first.

Beyond that, BuildKit offers a cache mount —
`RUN --mount=type=cache,target=/go/pkg/mod go mod download` — that persists
between builds without ending up in the image. Useful, not required here, and
worth knowing that it is a *build* cache: a CI runner that starts empty every
time gets nothing from it unless it is configured to keep it.

## Build flags that shrink the binary

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build \
	-trimpath \
	-ldflags="-s -w" \
	-o /out/timesvc .
```

- `-w` omits the DWARF debug information; `-s` omits the symbol table.
  Together they typically remove a quarter of the binary. You lose `delve`
  and `nm` on the shipped artifact — panic stack traces still work, because
  the runtime carries its own function-name table for them.
- `-trimpath` strips your local filesystem paths out of the binary, which
  makes builds reproducible and stops `/Users/ada/projects/…` from shipping
  to production.

The same mechanism stamps a version: `-ldflags="-s -w -X main.version=1.4.2"`
sets a string variable at link time. That is how the `--version` output of
most Go tools gets filled in.

## EXPOSE, and what it does not do

`EXPOSE 8080` publishes nothing. It opens no firewall. It is metadata: a
declaration of which port the image's process listens on. What it buys you is
that `docker run -P` can map every declared port automatically, that Compose
and orchestrators can read the intent, and that a human reading the
Dockerfile learns the contract without reading your Go. Declare it, and
declare the truth.

## Size is a real cost

It is tempting to treat 900 MB as merely untidy. It is not:

- Every deploy pulls it, on every node, over a network someone pays for.
- Cold starts wait for that pull — the difference between a node joining in
  seconds and joining in minutes.
- Every package in the image is scanned, reported, and eventually needs a
  patched rebuild. A distroless static image has almost nothing to report.
- Registry storage and transfer are billed.

Going from ~900 MB to ~10 MB is not polish. It is the difference between an
image you can redeploy casually and one you schedule.

## Exercise

Open [`exercise/`](exercise/) — a finished HTTP service called `timesvc`
(routes `GET /healthz` and `GET /now?zone=Europe/Berlin`) that has never been
containerized properly. `main.go` is provided complete; do not change it.
This lesson grades your image, not your Go.

Steps:

1. Read `Dockerfile.naive`. If you have Docker, build and measure it:
   `docker build -f Dockerfile.naive -t timesvc:naive .` then
   `docker image ls timesvc`.
2. Write `Dockerfile` (a new file, next to it) as a multi-stage build.
3. If you have Docker: `docker build -t timesvc:slim .`, measure again, and
   run it: `docker run --rm -p 8080:8080 timesvc:slim`, then
   `curl localhost:8080/now?zone=Europe/Berlin`.
4. Answer the three questions in `NOTES.md`.
5. Run the referee until green: `bash ./check.sh`.

Acceptance criteria:

1. `Dockerfile` has two or more stages, and every base image is pinned to a
   real tag or digest (never `latest`).
2. The final stage does not start from the Go toolchain and runs no `go`
   commands — it receives the binary via `COPY --from=`.
3. The build sets `CGO_ENABLED=0`.
4. `go.mod`/`go.sum` are copied and downloaded *before* the source is copied.
5. The final stage runs as a non-root `USER` — numeric if the base has no
   `/etc/passwd`.
6. The final stage declares `EXPOSE 8080`.
7. The build passes `-ldflags` containing `-s` and `-w`.
8. CA certificates are handled for your chosen base: copied in on `scratch`,
   installed on a slim Debian/Ubuntu base, already present on distroless and
   Alpine.
9. The final stage starts the binary with an exec-form `ENTRYPOINT` or `CMD`.
10. `NOTES.md` answers all three questions in your own words.

`check.sh` grades all of that by reading your files: no daemon, no network,
no root. If a Docker daemon *is* reachable it adds three live checks — the
image builds, it fits the size budget for the base you chose (25 MB on
`scratch` or distroless, 40 MB on Alpine, 120 MB on a slim Debian/Ubuntu,
because the base is a floor you cannot build under), and the container answers
`GET /healthz`. If the daemon is missing, or the build cannot reach a registry
to pull its base images, those three are reported `SKIP` and cost you nothing.

## Further reading

- [Docker docs — Multi-stage builds](https://docs.docker.com/build/building/multi-stage/)
- [go.dev — cgo command documentation](https://pkg.go.dev/cmd/cgo) and
  [`go build` flags](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)
- [GoogleContainerTools/distroless](https://github.com/GoogleContainerTools/distroless)
  — what each base actually contains.
- [go.dev — `time/tzdata`](https://pkg.go.dev/time/tzdata) — embedding the
  timezone database, and when you need to.
