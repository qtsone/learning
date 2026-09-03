# notes

A minimal note keeper: a domain package with the rules, an in-memory store
behind a small interface, and a command that wires the two together. It exists
as the reference project for the capstone conformance harness — small enough to
read in five minutes, structured the way a real project is.

## Run it

```sh
go run ./cmd/notes
```

## Test it

```sh
go test -race ./...
go test -cover ./...
```

## Layout

```
cmd/notes        composition root: wiring and output
internal/note    domain: the Note type, validation, error vocabulary
internal/store   storage: concurrency-safe in-memory Store
docs/adr         the decisions that were expensive to change
```

Dependencies point inwards: `store` imports `note`, `note` imports nothing of
ours, and only `main` knows both exist.
