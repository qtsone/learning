# Conformance harness — Performance Engineering

This module grades **your** project. There is no starter code to fill in here:
the optimization you are graded on happened in your capstone repository, and
the argument for it lives in `PERF.md` at that project's root.

## Point it at your project

The harness looks for the project in this order:

1. `TUTOR_CAPSTONE_DIR` — an absolute path in the environment.
2. `capstone.path` — one path on the first line of a file next to this README.
   Relative paths resolve against this directory.
3. `projects/capstone` at your workspace root — the convention from the
   planning lesson. The harness walks up from this directory to find it, so
   the depth does not matter.

```sh
echo ../../../../projects/capstone > capstone.path   # or any path you like
ls "$(cat capstone.path)/go.mod"                     # check before trusting it
# or
export TUTOR_CAPSTONE_DIR=/absolute/path/to/your/project
```

A project directory is one that contains a `go.mod`.

## The deliverable

[`PERF-template.md`](PERF-template.md) is the shape of the write-up. Copy it
into your project as `PERF.md`, fill every section, and delete every `TODO:`
line — the harness reads it and fails on the ones you left behind.

## Run it

```sh
go test ./...          # all six checks
go test -v ./...       # with the benchmarks and tests it found
```

What it does *not* do: assert that anything is fast. Wall-clock time and
speedup factors depend on the machine, the temperature and what else is
running, so no automated grader may claim them. It checks that your benchmarks
run, that your evidence is committed, that your numbers carry units, and that
the tests you named as proof of unchanged behaviour actually pass. The claim
itself is graded by a human, in review.

Grading never touches the network (`GOPROXY=off`), and it never runs your
benchmarks for real: `-benchtime=1x` means one iteration each, so the numbers
it prints are noise on purpose.
