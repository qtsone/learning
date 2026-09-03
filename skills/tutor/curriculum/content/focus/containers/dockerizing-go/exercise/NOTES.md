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

A: TODO

## Q2. Base image

Which final base did you pick (scratch, distroless, alpine), and what did
you give up by picking it? Name one concrete thing you can no longer do
inside that container, and one thing you had to add to the Dockerfile
because the base does not provide it.

A: TODO

## Q3. CGO

Suppose you drop `CGO_ENABLED=0` and rebuild on the same builder image, then
copy the binary into your chosen final base. What breaks, at what moment
(build time or container start), and what would the error look like? Why?

A: TODO
