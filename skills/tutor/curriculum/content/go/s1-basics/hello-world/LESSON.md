# Hello, Go

> `go.basics.hello-world` · ~1-2h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Explain what the Go toolchain is and what `go run`, `go build`, and `go test` each do.
- Describe the anatomy of a minimal Go program: `package main`, `import`, and `func main()`.
- Write, run, and format a Go program that prints output with the `fmt` package.
- Explain why `gofmt` exists and format your code with it.

## Your first program

Every Go program is made of **packages**. A package is a folder of Go files that
belong together. One package is special: `package main` marks a program that can
be *executed* (as opposed to a library that other code borrows from). Inside it,
the function `main` is where execution starts.

This is the smallest useful Go program:

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
```

Read it line by line:

- `package main` — "this folder builds into an executable program."
- `import "fmt"` — borrow the standard library's **fmt** package (short for
  *format*), which knows how to print text.
- `func main() { … }` — the program's entry point. When you run the program,
  Go calls this function, and when it returns, the program exits.
- `fmt.Println(…)` — call the `Println` function that lives in the `fmt`
  package: print the text, then a newline.

The dot in `fmt.Println` is how you reach anything inside an imported package.
You will see this shape constantly: `package.Thing`.

## Running it

Go is a **compiled** language (remember S0: the whole program is translated to
machine code before it runs). The toolchain gives you two ways to get from
source to running program:

```sh
go run .      # compile *and* run in one step — great while developing
go build .    # compile to a binary you can run and ship: ./hello
```

`go run` throws the compiled result away afterwards; `go build` keeps it.
Under the hood they do the same compilation, so if your code doesn't compile,
both fail with the same errors — read them top to bottom, first error first
(later errors are often side effects of the first one).

## Strings and Println

Text between double quotes is a **string**. `fmt.Println` accepts any number of
values, prints them separated by spaces, and adds a newline:

```go
fmt.Println("temperature:", 21, "°C")   // temperature: 21 °C
```

Its sibling `fmt.Printf` gives you precise control with format verbs (`%s`,
`%d`, …) — you'll meet it properly in the next lesson. For now, `Println` is
enough.

## gofmt: one true style

Go ships a formatter, `gofmt`, and the community treats its output as the only
correct style. There are no formatting debates in Go: tabs for indentation,
braces on the same line, done. Your editor almost certainly runs it on save
(that's the `gopls` integration from your dev-environment lesson). You can also
run it yourself:

```sh
gofmt -w .    # rewrite files in place
```

If your code "looks different" from every Go example you read, it isn't
formatted. Formatted code is not cosmetic: it makes every Go codebase on earth
read the same, which is a superpower when you're learning from other people's
code.

## Exercise

Open [`exercise/`](exercise/) — it's a ready Go module with three files:

- `main.go` — has two `TODO`s for you.
- `greet.go` — declares a function `Greeting` that you must complete. It takes
  a name (a string) and *returns* the text to print — it does not print
  anything itself.
- `greet_test.go` — the tests that decide when you're done. **Read it**; tests
  are specifications you can execute.

Acceptance criteria:

1. `Greeting("Gopher")` returns exactly `Hello, Gopher!`.
2. `Greeting("")` returns `Hello, world!` (an empty name falls back to
   "world").
3. `main` prints the greeting for the name `"Go"` — run `go run .` and check
   with your own eyes.
4. `go test ./...` passes, and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

Expect them to FAIL before you write any code — that's the point: make the
red tests green.

## Further reading

- [A Tour of Go — Hello, world](https://go.dev/tour/welcome/1)
- [go.dev — Get started tutorial](https://go.dev/doc/tutorial/getting-started)
- [Effective Go — Formatting](https://go.dev/doc/effective_go#formatting)
