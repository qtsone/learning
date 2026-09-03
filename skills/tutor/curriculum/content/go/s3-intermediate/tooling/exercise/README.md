# Exercise — The tools tell you what the compiler can't

Work through the six parts in order, from inside this directory. Record every
answer in `NOTES.md` — it has a slot for each — then walk your tutor through
it. This module compiles and its test passes; the bugs in it are exactly the
kind the compiler waves through.

First, confirm the baseline:

```sh
go build ./...
go test ./profilelab
```

Both succeed. Keep that in mind for everything the tools find below.

## Part 1 — go vet

```sh
go vet ./...
```

Expect four findings in `vetlab/`. For each, record in `NOTES.md`: the vet
check that fired, what is actually wrong with the code, and why the compiler
accepted it. Then fix all four and rerun until `go vet ./...` prints nothing
and exits 0.

Two of the findings connect straight back to earlier lessons (sync and
context) — say which, and what would go wrong at runtime in each case.

## Part 2 — staticcheck

Install and run:

```sh
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

(If `staticcheck` isn't found, your `$(go env GOPATH)/bin` is not on PATH —
you set this up in the dev-environment lesson.)

Expect at least four findings in `lintlab/`, none of which vet reported. For
each: record the check code, look it up at
<https://staticcheck.dev/docs/checks/>, and note which *family* it belongs to
(SA correctness / S simplification / ST style / U unused) and what it caught.
Then fix them all and rerun until clean.

One finding is about a function the compiler happily kept forever. Why does
the compiler reject an unused local variable but tolerate an unused function?

## Part 3 — golangci-lint

Install golangci-lint (`brew install golangci-lint`, or see
<https://golangci-lint.run/> for the install script). Then write a
`.golangci.yml` in this directory yourself — version `"2"` schema, default
set to none, explicitly enabling `govet`, `staticcheck`, `errcheck`, and
`unused` — and run:

```sh
golangci-lint run
```

`WriteReport` in `lintlab/` contains two problems that *neither* vet nor
staticcheck reported. Which linter catches them, and what is the failure mode
in production if they stay? Record the exact finding lines (file:line:col,
message, linter), fix them properly (handle the errors, don't discard them),
and rerun until clean.

## Part 4 — pprof on a real hot spot

Benchmark both implementations first:

```sh
go test ./profilelab -bench . -benchmem
```

Record ns/op, B/op, and allocs/op for both. Now profile the naive one:

```sh
cd profilelab
go test -bench BenchmarkJoinNaive -cpuprofile cpu.out -memprofile mem.out
go tool pprof -top cpu.out
go tool pprof -top -sample_index=alloc_space mem.out
cd ..
```

(If pprof complains it needs the binary, pass the test binary the run left
behind: `go tool pprof -top profilelab.test cpu.out`.)

In `NOTES.md`: which functions dominate the CPU profile, and what are they
doing? What does the heap profile say about where the bytes come from?
Explain *why* `JoinNaive` behaves this way — think about what `out += …`
must do to an immutable string on every iteration — and why `JoinBuilder`
doesn't. Connect the allocs/op numbers to your S2 Big-O instincts: what's the
asymptotic cost of naive concatenation in a loop?

## Part 5 — Modules field trip

This part needs internet access. Work in a scratch directory, not in this
module:

```sh
mkdir -p /tmp/modlab && cd /tmp/modlab
go mod init example/modlab
```

The module needs code that actually *uses* a dependency — otherwise the
build has nothing to download and nothing to verify. Create `main.go`:

```go
package main

import (
	"fmt"

	"rsc.io/quote"
)

func main() {
	fmt.Println(quote.Hello())
}
```

Then fetch the dependency and prove the module runs:

```sh
go get rsc.io/quote@v1.5.2
go run .
```

Now investigate, recording answers as you go:

1. `go list -m all` — what did one `go get` actually bring in?
2. `go mod why -m golang.org/x/text` — read the chain aloud: who needs it?
3. `go list -m -versions rsc.io/sampler` — note the versions that exist.
   Your build uses v1.3.0 (check with `go list -m rsc.io/sampler`). Newer
   versions exist — why didn't Go pick the newest? Which rule is at work?
4. Run `go get rsc.io/sampler@v1.3.1`, then look at `go.mod` and `go.sum`.
   Suppose another dependency required sampler v1.3.0 while your `go.mod`
   requires v1.3.1 — which version does MVS select, and why?
5. Tamper time: open `go.sum`, find the `rsc.io/quote v1.5.2 h1:` line, and
   change one character inside the hash. Save, then run `go clean -modcache`
   and `go build ./...`. What happens, and what real-world attack does this
   behavior defeat? (To recover: undo your edit — or delete the corrupted
   line and run `go mod tidy` to re-record the real hash. Note that `go mod
   tidy` alone refuses too: a hash in `go.sum` is trusted over anything
   downloaded.)

## Part 6 — Choose the command

For each scenario, name the `go` command you'd reach for and justify it in
one line in `NOTES.md`:

1. You deleted the last file that imported a dependency; `go.mod` still
   requires it.
2. You maintain an app and a library in separate modules, and want the app to
   build against your local, uncommitted library changes — without editing
   the app's `go.mod`.
3. A teammate asks: "which version of `golang.org/x/text` does our build
   actually use?"
4. A file starts with `//go:generate stringer -type=Level` and the `Level`
   constants just changed.
5. CI fails with "missing go.sum entry" after you added an import.

Done? Make sure `go vet ./...`, `staticcheck ./...`, and
`golangci-lint run` are all clean, then bring `NOTES.md` to the discussion.
