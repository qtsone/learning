# Notes — Dockerizing timesvc

Answer each question on the `A:` line below it, in your own words, in at
least a couple of sentences. `check.sh` checks that you answered; your tutor
reads what you actually wrote.

## Q1. Sizes

What does `Dockerfile.naive` produce, and what does your `Dockerfile`
produce? Give both numbers (`docker image ls timesvc`) and say where the
difference went — what exactly is in the big one that is not in the small
one? No Docker installed? Then estimate both from the lesson and name the
three heaviest things the naive image carries.

A: The naive image is about 900 MB and mine is about 10 MB, roughly ninety times smaller. The naive one carries three things mine does not: the Go compiler and linker, the whole standard library source tree that ships with the toolchain, and a complete Debian userland (shell, apt, coreutils, package metadata). None of that is needed at runtime, because the compiler already turned the parts of the stdlib I use into the binary — about 9 MB of it. My final image is the distroless static base plus that one file.

## Q2. Base image

Which final base did you pick (scratch, distroless, alpine), and what did
you give up by picking it? Name one concrete thing you can no longer do
inside that container, and one thing you had to add to the Dockerfile
because the base does not provide it.

A: I picked gcr.io/distroless/static:nonroot, which is scratch plus the three things scratch users always end up re-adding: a CA bundle, tzdata, and an /etc/passwd entry. What I gave up is a shell — `docker exec -it … sh` fails, so I debug from logs, from `docker cp`, or by temporarily rebuilding on the :debug variant. What I had to add myself is the binary's absolute path in an exec-form ENTRYPOINT, because there is no shell to resolve PATH, plus an explicit numeric USER so the image also behaves under a runtime that enforces runAsNonRoot.

## Q3. CGO

Suppose you drop `CGO_ENABLED=0` and rebuild on the same builder image, then
copy the binary into your chosen final base. What breaks, at what moment
(build time or container start), and what would the error look like? Why?

A: The build still succeeds — golang:1.22-bookworm has gcc and glibc, so cgo links the net and os/user packages dynamically against Debian's glibc. It breaks at container start: the kernel reads the ELF interpreter path (/lib64/ld-linux-x86-64.so.2) recorded in the binary, cannot find that file in the distroless static image, and reports "exec /usr/local/bin/timesvc: no such file or directory" about a file that plainly exists. The missing file is the loader, not my binary. On alpine the same build fails the same way for a second reason: alpine ships musl, not glibc, so even the loader path is different.
