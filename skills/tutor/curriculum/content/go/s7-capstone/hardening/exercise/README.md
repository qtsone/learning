# Conformance harness — Capstone: Hardening

This module grades **your** project. There is no starter code to fill in here:
the code you are graded on lives in your capstone repository.

## Point it at your project

The harness looks for the project in this order:

1. `TUTOR_CAPSTONE_DIR` — an absolute path in the environment.
2. `capstone.path` — one path on the first line of a file next to this README.
   Relative paths resolve against this directory.
3. `projects/capstone` at your workspace root — the convention from the
   planning lesson. The harness walks up from this directory to find it, so
   the depth does not matter.

If you followed the planning lesson, nothing to do. Otherwise:

```sh
echo ../../../../projects/capstone > capstone.path   # or any path you like
ls "$(cat capstone.path)/go.mod"                     # check before trusting it
# or
export TUTOR_CAPSTONE_DIR=/absolute/path/to/your/project
```

A project directory is one that contains a `go.mod`.

## Run it

```sh
go test ./...                 # all nine checks
go test -run Security -v .    # just the security document
go test -run Fuzz -v .        # just the fuzz targets and their corpora
```

Every failure prints the real output of the underlying `go` command or the
exact file and line of the finding, so read the tail of the message, not the
harness frame around it. Several checks log what they *did* find under `-v`,
which is usually faster than guessing.

## What it cannot do

Grading never touches the network (`GOPROXY=off`), so the harness cannot run
`govulncheck` for you — that tool needs the vulnerability database. You run it
by hand, read every finding, and record the result in your security document;
the harness only checks that the record exists.

If your project has dependencies, run `go mod download` in it once before
grading.
