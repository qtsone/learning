# Operations referee — Capstone: Operations

`check.sh` grades **your** project. There is no starter code here: the thing
being checked is the operational surface of your capstone repository.

## Point it at your project

The referee looks for the project in this order:

1. `TUTOR_CAPSTONE_DIR` — an absolute path in the environment.
2. `capstone.path` — one path on the first line of a file next to this README.
   Relative paths resolve against this directory.
3. `projects/capstone` at your workspace root — the convention from the
   planning lesson. The referee walks up from this directory to find it, so
   the depth does not matter.

Same chain as the core-build harness, so if you already made that one work,
copy the answer:

```sh
echo ../../../../projects/capstone > capstone.path   # or any path you like
ls "$(cat capstone.path)/go.mod"                     # check before trusting it
# or
export TUTOR_CAPSTONE_DIR=/absolute/path/to/your/project
```

A project directory is one that contains a `go.mod`.

## Run it

```sh
bash check.sh
```

Every check prints one line: `ok`, `FAIL` with what to fix, `skip` with why it
could not run, or `warn` from a bonus check. Only `FAIL` lines affect the exit
code. Run it as often as you like — it reads your project and changes nothing.

## What it does not need

No network, no docker daemon, no cluster, no cloud account. The checks are
static: they read your Dockerfile, your pipeline definition, your runbook, your
docs and your Go source.

Two bonus checks *can* use tools if you have them, and are skipped otherwise:
`docker build` and a `kubectl` client dry-run of your manifests. Both pull from
the network, so both stay off unless you ask:

```sh
TUTOR_OPS_ALLOW_NETWORK=1 bash check.sh
```

A bonus check that runs and complains prints `warn` and never blocks — the
tooling is too varied for a bonus check to be a gate.

## Also here

`RUNBOOK-template.md` — the headings the referee looks for, with prompts under
each. Copy it to your project root as `RUNBOOK.md` and replace every prompt
with something true about *your* system. The referee requires real commands in
the deploy and rollback sections, so the template alone does not pass.
