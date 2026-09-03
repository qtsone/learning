# Escape analysis notes

Fill this in as you work — the tutor reviews it with you. Section 1 must be
recorded **before** you refactor anything: once the code changes, the
"before" output is gone.

## 1. Before refactoring

From `exercise/`, run:

```sh
go build -a -gcflags='-m' . 2>&1 | grep -v 'can inline\|inlining call'
```

Paste every line that mentions `format.go` or `report.go`:

```
TODO: paste the starter's escape-analysis lines here
```

For each `escapes to heap` line above, explain in one sentence *why* that
value escapes:

- TODO

## 2. After refactoring

Run the same command after your refactor. Paste the lines for `format.go`
and `report.go` again and note what changed:

```
TODO: paste the refactored escape-analysis lines here
```

What changed and why:

- TODO

## 3. Benchmark evidence

Run:

```sh
go test -bench=Summarize -benchmem
```

Paste both result lines and explain the difference in the `allocs/op` and
`B/op` columns:

```
TODO: paste benchmark output here
```

- TODO
