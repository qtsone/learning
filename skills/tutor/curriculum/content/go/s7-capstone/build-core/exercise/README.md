# Conformance harness — Capstone: Core Build

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
go test ./...          # from this directory
go test -run Coverage -v ./...
```

Every failure prints the real output of the underlying `go` command, so read
the tail of the message, not the harness frame around it.

Grading never touches the network (`GOPROXY=off`). If your project has
dependencies, run `go mod download` in it once before grading.
