# Exploration notes

Answer each question in your own words, in prose, under its heading. Replace
the `TODO` line — leave the quoted question where it is. A few sentences each
(the referee wants at least ~30 words per answer, which is roughly three
sentences). These answers are what your tutor grills you on.

## 1. Container vs virtual machine

> A colleague says "a container is just a lightweight VM". Correct them.
> Name the two kernel mechanisms involved and say what each one controls, and
> give one consequence of sharing the host kernel that a VM user would not
> face.

A container is an ordinary process on the host, started with a restricted view
of the machine. Namespaces control what it can see: its own process tree, its
own filesystem root, its own network interfaces and hostname. Cgroups control
what it can use: a ceiling on CPU time, memory and I/O. There is no second
kernel and no virtual hardware, which is why a container starts in
milliseconds while a VM boots. The consequence of the shared kernel is that
the isolation is thinner: a kernel bug is a shared attack surface, a Linux
image needs a Linux kernel (my Mac runs a small Linux VM to provide one), and
hitting the cgroup memory limit gets my process OOM-killed rather than
swapped inside its own guest OS.

## 2. What the build cache does with your edit

> You build your image, then change one line in `main.go` and build again.
> Walk the Dockerfile you wrote instruction by instruction: which layers come
> from cache, which are rebuilt, and why? Then answer: what would change if
> you had copied the whole source tree before `go mod download`?

`FROM golang:1.22-bookworm`, `WORKDIR /src`, `COPY go.mod ./` and
`RUN go mod download` all hit the cache: the instruction text is unchanged and
the only file copied, `go.mod`, is byte-identical, so the cache key matches.
`COPY . .` misses, because the checksum of the copied files changed with my
edit, and every instruction after a miss is re-executed — so `go build` runs
again, which is exactly what I want. If I had copied the whole tree before
`go mod download`, the copy would miss on every edit and drag the download
step down with it, re-fetching all dependencies for a one-line change.

## 3. Tags, digests, and `latest`

> Why is `latest` not a version? Describe a concrete way a team gets burned by
> `FROM someimage:latest`, and say what a digest gives you that a tag cannot.

`latest` is just the tag name Docker assumes when you do not give one; nothing
guarantees it points at the newest release, and like any tag it is a mutable
pointer the publisher can move at will. A team gets burned when CI builds
green on Monday and red on Wednesday with no commit in between, because
`latest` moved to a new major version of the base image — and the Monday image
can no longer be reproduced, so bisecting is hopeless. A digest is a SHA-256
of the image content, so `image@sha256:…` always resolves to exactly the same
bytes; it is an identity rather than a pointer, at the price of owning updates
yourself.

## 4. Where a container's writes go

> Your containerized service writes a file to `/tmp/report.txt` while running.
> You then run `docker rm` on the container. What happened to the file, and
> why? What is still on disk afterwards, and what is gone?

The file only ever existed in the container's thin writable layer, which sits
on top of the image's read-only layers in the union filesystem; the image
itself is never modified by a running container. `docker rm` deletes the
container along with that writable layer, so `/tmp/report.txt` is gone for
good. What survives is the image and its shared layers, so I can start a fresh
container from it immediately — with an empty writable layer and no report.
Data I actually want to keep has to live outside the container's own layer,
which is what volumes are for.
