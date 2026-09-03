# Packages & Modules

> `go.basics.packages-modules` · ~2-3h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Create a module with `go mod init` and explain what go.mod records.
- Explain how identifier capitalization controls what a package exports.
- Organize code into multiple packages and import them with module-rooted paths.
- Add an external dependency and explain how `go mod tidy` keeps go.mod accurate.

## From one file to many

So far every exercise fit into a single folder: `package main`, a few files,
done. Real programs don't stay that small. Go's unit of organization is the
**package**: one folder of `.go` files that all start with the same
`package <name>` line. Everything declared in one file of a package — functions,
constants, variables — is visible to every other file in the same folder, no
import needed. That's why `main.go` in hello-world could call `Greeting` from
`greet.go` directly: same folder, same package.

Two rules to internalize:

- One folder = one package. All files in a folder must declare the same
  package name, and by convention that name matches the folder name.
- `package main` is the only package that builds into a runnable program.
  Every other package is a **library**: a toolbox other packages import.

You've been *using* library packages since your first line of Go — `fmt` is
one. Today you write your own.

## Modules: what go.mod records

A **module** is a collection of packages that ships together — usually one
module per project, one `go.mod` file at its root. Every exercise so far
already had one; now you learn what it says. You create one yourself with:

```sh
mkdir scratch && cd scratch
go mod init tutor.local/scratch
```

That writes a `go.mod` with two facts:

```
module tutor.local/scratch

go 1.22        ← yours will show your installed Go version
```

- `module tutor.local/scratch` — the **module path**, your module's full name.
  Real projects use the place the code lives (`github.com/you/project`) so the
  name is globally unique; our exercises use the made-up `tutor.local/…`
  because they're never published.
- The `go` line — the minimum Go version the code expects. `go mod init`
  copies it from the toolchain you're running, so your file will match
  whatever `go version` prints, not necessarily `1.22`.

When you add dependencies, `go.mod` grows `require` lines recording exactly
which version of each one you use — more on that below. That's the whole file:
name, Go version, dependencies. It is the anchor that tells the toolchain
"the module starts here", which is why `go run .` complains when you're in a
folder with no `go.mod` above it.

## Exported vs unexported: the capitalization rule

A package draws a hard line between its public surface and its internals, and
Go marks the line with nothing more than a capital letter:

- `CToF`, `Println`, `Greeting` — **exported**: first letter capitalized,
  usable from other packages.
- `cToF`, `println`, `helper` — **unexported**: lowercase first letter, usable
  only inside their own package.

There is no `public`/`private` keyword; the name *is* the declaration of
intent. Now you know why every standard-library function you've called —
`fmt.Println`, `fmt.Sprintf` — starts with a capital: anything lowercase would
be invisible to you. If you import a package and try to call an unexported
name, the compiler stops you with `undefined` — the name simply doesn't exist
from the outside.

Why hide anything? Because everything you export is a promise: other code will
call it, so you can't rename or change it freely anymore. Keep helpers
lowercase and your package stays easy to change. Export the few names that
form the package's purpose, nothing else.

## Importing your own packages

Inside a module, a package's import path is the **module path plus the folder
path**. Given this layout:

```
weather/                   ← go.mod: module tutor.local/weather
├── go.mod
├── main.go                ← package main
└── temperature/
    └── temperature.go     ← package temperature
```

`main.go` imports the sub-package like this:

```go
package main

import (
	"fmt"

	"tutor.local/weather/temperature"
)

func main() {
	fmt.Println(temperature.CToF(25))
}
```

Read the import path as an address: module `tutor.local/weather`, folder
`temperature/`. After importing, you use the *package name* (the last
segment): `temperature.CToF`. Same `package.Thing` shape as `fmt.Println` —
your packages and the standard library's work identically. The blank line in
the import block separating standard library from everything else is the
conventional grouping; `gofmt` keeps each group sorted.

## External dependencies and go mod tidy

The same import mechanism reaches code written by strangers. Try it in the
scratch module you made above — write this `main.go`:

```go
package main

import (
	"fmt"

	"rsc.io/quote"
)

func main() {
	fmt.Println(quote.Go())
}
```

`go run .` now fails: the module doesn't know `rsc.io/quote` yet. Fix that:

```sh
go mod tidy
go run .
```

`go mod tidy` scans your source for imports, downloads what's missing, and
rewrites `go.mod` to match reality — it adds `require` lines for what you
import and drops lines for what you no longer do. Look at `go.mod` afterwards:

```
require rsc.io/quote v1.5.2

require (
	golang.org/x/text v0.0.0-20170915032832-14c0d48ead0c // indirect
	rsc.io/sampler v1.3.0 // indirect
)
```

The first line is the package you asked for. The `// indirect` block is its
supply chain: `quote` itself imports `sampler`, which imports `text`, and Go
pins *their* versions too so your build is reproducible all the way down. You
never manage indirect lines yourself — `go mod tidy` does.

A second file appeared too: `go.sum`, a list of cryptographic checksums for
every downloaded module. It guarantees that everyone who builds your project
gets byte-for-byte the same dependency code. You never edit `go.mod` or
`go.sum` by hand — commit both, and let `go mod tidy` keep them honest.

(`go get rsc.io/quote` does the download-and-record step explicitly; `tidy`
is the habit to build because it also cleans up.)

## Exercise

Open [`exercise/`](exercise/) — a module named `tutor.local/packages-modules`
with three packages:

- `temperature/` — a library package with three `TODO` functions: `CToF`,
  `FToC`, and `Describe`.
- `report/` — a library package whose `Line` function must *import and use*
  `temperature` to format one city's reading.
- `main.go` — a `package main` that imports `report` and prints something.

Read the test files. Notice their package lines: `package temperature_test`,
not `package temperature`. They live *outside* the package they test, so they
can only reach what the package exports — the exact view any importer gets.

Acceptance criteria:

1. `temperature.CToF` and `temperature.FToC` convert correctly
   (Celsius→Fahrenheit: multiply by 9, divide by 5, add 32; reverse it for
   the other direction).
2. `temperature.Describe` classifies a Celsius value: below 0 `"freezing"`,
   0 up to (not including) 15 `"cold"`, 15 up to (not including) 25 `"mild"`,
   25 and above `"hot"`.
3. `report.Line("Berlin", 25)` returns exactly
   `Berlin: 25.0°C / 77.0°F (hot)` — Fahrenheit and the word in parentheses
   must come from the `temperature` package, formatted with `fmt.Sprintf`
   and `%.1f`.
4. `main` prints the report line for at least two cities of your choice —
   run `go run .` and check with your own eyes.
5. `go test ./...` passes from the exercise root, and the code is
   `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

`./...` means "this folder and everything below it" — one command tests all
packages in the module. Expect failures before you write any code.

One experiment before you finish: rename `CToF` to `cToF` in
`temperature.go`, run `go test ./...`, and read the errors. That compile
failure is the capitalization rule enforcing itself. Rename it back.

## Further reading

- [A Tour of Go — Exported names](https://go.dev/tour/basics/3)
- [go.dev — Create a Go module](https://go.dev/doc/tutorial/create-module)
- [go.dev blog — Using Go Modules](https://go.dev/blog/using-go-modules)
- [Effective Go — Names](https://go.dev/doc/effective_go#names)
