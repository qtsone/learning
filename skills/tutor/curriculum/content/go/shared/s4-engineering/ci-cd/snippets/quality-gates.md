In Go, the two gates and one trap:

- `go vet ./...` — the standard analyzer; exits non-zero on findings. Its
  classic catch is a `Printf`-family format verb that disagrees with its
  argument — code that compiles and prints garbage.
- `gofmt -l .` — lists files that are not formatted. The trap: it *always
  exits 0*, even when it finds offenders, so as a bare step it can never fail.
  The idiom is to fail when its output is non-empty:

  ```sh
  test -z "$(gofmt -l .)"
  ```

  `test -z` exits non-zero when the string is non-empty — an offending file
  name in the output turns the step red.
