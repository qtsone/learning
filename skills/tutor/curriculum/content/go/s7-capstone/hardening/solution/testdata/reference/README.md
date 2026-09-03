# notes

A minimal note keeper: a domain package with the rules and the parser, an
in-memory store behind a small interface, an outbound webhook, and a command
that wires them together. It exists as the reference project for the capstone
conformance harnesses — small enough to read in five minutes, structured and
hardened the way a real project is.

## Run it

```sh
printf 'n1|read the threat model|work\nn2|fuzz the parser|work,design\n' | go run ./cmd/notes
```

Each input line is `id|title` or `id|title|tag,tag`. Lines that fail
validation are reported on stderr with their line number and skipped.

## Test it

```sh
go test -race ./...
go test -fuzz=FuzzParseLine -fuzztime=60s ./internal/note
go vet ./...
govulncheck ./...
```

## Configure it

| Variable | Meaning |
|---|---|
| `NOTES_WEBHOOK` | If set, the listing is POSTed to this URL. It is a secret: it usually carries a token, and it is never logged unredacted. |

## Layout

```
cmd/notes        composition root: wiring, input loop, process lifetime
internal/note    domain: the Note type, validation, the parser, the fuzz target
internal/store   storage: concurrency-safe in-memory Store
internal/remote  the one outbound call: bounded client, redacted errors
docs/adr         the decisions that were expensive to change
SECURITY.md      trust boundaries, inputs, secrets, dependencies, accepted risk
```

Dependencies point inwards: `store` imports `note`, `note` imports nothing of
ours, `remote` imports nothing of ours, and only `main` knows they all exist.
