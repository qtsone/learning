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

TODO: your answer here.

## 2. What the build cache does with your edit

> You build your image, then change one line in `main.go` and build again.
> Walk the Dockerfile you wrote instruction by instruction: which layers come
> from cache, which are rebuilt, and why? Then answer: what would change if
> you had copied the whole source tree before `go mod download`?

TODO: your answer here.

## 3. Tags, digests, and `latest`

> Why is `latest` not a version? Describe a concrete way a team gets burned by
> `FROM someimage:latest`, and say what a digest gives you that a tag cannot.

TODO: your answer here.

## 4. Where a container's writes go

> Your containerized service writes a file to `/tmp/report.txt` while running.
> You then run `docker rm` on the container. What happened to the file, and
> why? What is still on disk afterwards, and what is gone?

TODO: your answer here.
