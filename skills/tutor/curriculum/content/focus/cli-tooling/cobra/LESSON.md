# Cobra & Subcommands

> `focus.cli.cobra` · ~3-4h · Stage: Focus: CLI Tooling

## Objectives

By the end of this lesson you can:

- Build a multi-level cobra command tree, distinguish persistent flags from
  command-local flags, and explain where each one is visible.
- Resolve every configuration source — defaults, environment, flags — into one
  settings struct, and explain what viper would buy and cost you here.
- Implement `RunE` handlers that return errors instead of calling `os.Exit`, and
  explain why that is what makes commands testable.
- Choose between the stdlib `flag` package and cobra for a given tool, and
  justify the trade-off.

## When stdlib `flag` stops being enough

In the previous lesson you built a config loader on top of `flag.FlagSet`, and in
your S3 mini-project you gave the link checker its own `FlagSet` and a
`Run(args, stdin, stdout, stderr) int` entry point. That design scales fine to
one command with a dozen flags. It stops scaling the moment you want `tool add`,
`tool list`, `tool tag add`. You can build that by hand — a
`map[string]*flag.FlagSet`, a switch on `args[0]`, your own help text — and
people do. What you write next, in this order, is a help renderer per level, a
"did you mean" hint, and shell completions. That list is cobra's feature list,
which is the honest argument for it — not that frameworks are good.

| | stdlib `flag` | cobra |
|---|---|---|
| dependencies | none | 3 modules (cobra, pflag, mousetrap) |
| flag syntax | `-flag`, `-flag=v` | POSIX `--flag`, `-f`, `-abc` bundling |
| subcommands | you write the dispatch | a tree, any depth |
| help/usage | you write it | generated, overridable |
| completions | you write it | bash/zsh/fish/powershell generated |
| surface to learn | one page | a large API |

Rule of thumb: **one command, few flags, and you care about a dependency-free
binary → `flag`. A tree of commands you expect to grow, plus help and
completions you would otherwise hand-write → cobra.** Note the flag syntax row is
a compatibility decision, not a taste one: switching parsers later changes your
tool's published interface, so decide before users exist.

## The shape of a command

A `cobra.Command` is a struct you fill in, not a class you subclass:

```go
cmd := &cobra.Command{
	Use:   "add <text>",       // first word is the command name
	Short: "Add a note",       // one line, shown in the parent's command list
	Args:  cobra.ExactArgs(1), // positional-argument contract
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
```

`Use`'s first word is the name cobra matches on the command line; the rest is
usage text for humans. `RunE` is the handler; its sibling `Run` has no return
value — avoid it, for reasons two sections down. Commands become a tree with
`parent.AddCommand(child)`, to any depth: `root → tag → add` is ordinary.

## Constructors, not globals

Cobra's own scaffolding generates a package-level `var rootCmd = &cobra.Command{…}`
plus an `init()` per file calling `rootCmd.AddCommand(…)`. It is the most copied
Go CLI pattern in existence, and it is a trap: a `cobra.Command` **stores parsed
flag values on itself**, so with one global tree the second test sees the first
test's `--format`.

Write constructors instead:

```go
func NewRootCmd(app *App) *cobra.Command { … }
func newAddCmd(app *App) *cobra.Command  { … }
```

Each test now gets a pristine tree, and handlers close over `app` — the struct
holding storage, the environment lookup, anything else from outside — instead of
reaching for globals. It is the same move as passing `stdout` into a function
rather than calling `fmt.Println`: name the dependency, and it becomes
substitutable.

## Persistent vs local flags

Every command has two flag sets:

```go
root.PersistentFlags().String("format", "text", "output format")  // whole subtree
cmd.Flags().StringSlice("tag", nil, "tag to attach")              // this command only
```

A **persistent** flag is visible on the command that declared it and every
descendant, at any depth: `--format` on root works on `notes add` and on
`notes tag list`. A **local** flag exists only on its own command: `--tag` on
`add` is invisible to `list`, and to root.

Cobra gives you three views, and they answer different questions:

| call | contains |
|---|---|
| `cmd.Flags()` | everything usable here: local + inherited persistent |
| `cmd.LocalFlags()` | declared on this command |
| `cmd.InheritedFlags()` | inherited from ancestors |

`cmd.Flags()` is what you read at run time. One caveat before it bites you: the
merge of ancestors' persistent flags into `Flags()` happens during parsing, so
`Flags()` on a command that has not run yet may not show inherited flags.
`LocalFlags()` and `InheritedFlags()` force the merge, which is why the
exercise's flag-scope test uses those.

Persistent flags are accepted on either side of the subcommand:
`notes --format json add "x"` and `notes add "x" --format json` are the same
invocation. That is pflag's interspersed parsing, and users expect it.

## Args validators, and one sharp edge

`Args` is a function that vets positional arguments before `RunE` runs:
`cobra.NoArgs`, `ExactArgs(n)`, `MinimumNArgs(n)`, `RangeArgs(min, max)`,
`OnlyValidArgs` (paired with `ValidArgs`), and `MatchAll(...)` to combine them.
Set it on every command — the default is lenient.

Now the sharp edge, because it is a real bug you would otherwise ship. Cobra
checks "is this command runnable?" *before* it validates arguments. A grouping
command with no `Run`/`RunE` — the natural way to write `notes tag`, which exists
only to hold `tag add` and `tag list` — short-circuits straight to printing help
and returns **no error**. So `notes tag bogus` prints help and exits 0: a typo
that looks like success, which is exactly what a script cannot detect. The fix is
a `RunE` that prints help, plus an `Args` validator:

```go
func helpRunE(cmd *cobra.Command, _ []string) error { return cmd.Help() }

tag := &cobra.Command{Use: "tag", Short: "Work with tags", Args: cobra.NoArgs, RunE: helpRunE}
```

Now `notes tag` still prints help and succeeds, and `notes tag bogus` fails with
`unknown command "bogus" for "notes tag"`.

## RunE, not os.Exit

`os.Exit` inside a handler ends the process immediately: deferred functions do
not run, buffered writers do not flush, and no test can call that path in-process
without taking the test binary down with it. The same objection applies to
`cobra.CheckErr` and `log.Fatal` — both are `os.Exit` under a friendlier name.

So handlers return errors, and exactly one place — `main`, which you can read in
`exercise/main.go` — turns an error into a message on stderr and an exit code.
For that to be the *only* report, tell cobra to stop printing on your behalf:

```go
root := &cobra.Command{ …, SilenceErrors: true, SilenceUsage: true}
```

`SilenceErrors` stops cobra writing `Error: …` to stderr; `SilenceUsage` stops it
dumping the whole usage block after a failure, which is noise when the failure
was "note not found" rather than "you typed the command wrong". `Execute` still
returns the error either way — silencing changes who prints, not what happens.

Errors keep the properties S1 and S3 taught you: a handler that returns the
store's error unchanged (or wrapped with `%w`, never `%v`) lets a caller — or a
test — match it with `errors.Is(err, ErrNoteNotFound)`.

## Testing a command tree in-process

Inside a handler, write to `cmd.OutOrStdout()` (and `cmd.ErrOrStderr()`), never
to `os.Stdout`. Those methods return whatever was injected, walking up to the
root, and fall back to the real thing only when nothing was set. Splitting data
from diagnostics across those two streams is the next lesson's subject; the habit
that makes it possible is this one.

Put the pieces together and a CLI test needs no subprocess, no `os.Args`, no
golden files:

```go
root := NewRootCmd(app)      // fresh tree: no flag state from the last run
root.SetOut(&out)            // set on root, inherited by the whole subtree
root.SetErr(&errOut)
root.SetArgs([]string{"add", "x", "--format", "json"})
err := root.Execute()        // assert on out, errOut and err
```

One landmine: `SetArgs(nil)` means "not set", and cobra falls back to
`os.Args[1:]`, which under `go test` is full of `-test.*` flags. A variadic
helper called with no arguments hands you exactly that `nil`, which is why the
exercise's `run` helper guards against it.

## One settings struct, and where viper fits

The precedence chain from the previous lesson still applies: defaults <
environment < flags. Cobra does not resolve it for you; it gives you the piece
you need, `Changed`, which reports whether the user actually typed the flag:

```go
s := Settings{Format: "text"}                                   // default
if v := app.env("NOTES_FORMAT"); v != "" { s.Format = v }       // environment
if f := cmd.Flags().Lookup("format"); f != nil && f.Changed {   // flag, if typed
	s.Format = f.Value.String()
}
```

Without `Changed`, a flag sitting at its default value would silently overwrite
the environment — the classic layering bug. `Lookup` returning `nil` is the other
half: it is how a resolver asks "can this command even see `--limit`?".

**Viper**, honestly. Viper is cobra's usual companion: `viper.BindPFlag` ties a
flag to a config key, `SetEnvPrefix` + `AutomaticEnv` map keys to environment
variables, it reads JSON/YAML/TOML config files, and `viper.Unmarshal(&cfg)`
fills your struct. On a tool with several config formats and deeply nested
settings, that is a lot of code you do not write.

What it costs: the package-level API (`viper.Get`, `viper.SetDefault`) is one
process-wide singleton, so tests share state and cannot run in parallel — use
`viper.New()` if you adopt it. Key resolution is case-insensitive with `.`
nesting and alias rules, which makes "where did this value come from?" — the
question the previous lesson told you to keep answerable — genuinely hard. And
binding is implicit: reading `cfg.Format` gives no hint whether a flag, a file or
an environment variable produced it.

The judgement: reach for viper when you have many sources and many keys, and
instantiate it explicitly. Below that, twenty lines of resolver you can read top
to bottom beat a dependency whose precedence rules you have to look up. The
exercise does it by hand so you know exactly what viper would be doing.

## Exercise

Open [`exercise/`](exercise/) — a Go module for `notes`, whose tree is `notes` →
`add`, `list`, `tag` → `tag add`, `tag list`. `store.go` and `main.go` are
complete; you write the command tree in `root.go` and the resolver in `app.go`.
Read `notes_test.go` first: it is the specification.

Acceptance criteria:

1. `NewRootCmd(app)` returns a fresh tree named `notes` with `SilenceErrors` and
   `SilenceUsage` set, and a **persistent** `--format` flag defaulting to
   `"text"`.
2. `add <text>` takes exactly one argument and a **local**, repeatable `--tag`
   flag. Text output: `added n1`. JSON output: the note object,
   `{"id":"n1","text":"buy milk","tags":[]}`.
3. `list` takes no positional arguments and has a **local** `--limit` int flag
   (0 means all). Text output is one line per note, `n1 buy milk`, with tags
   appended as ` #home`; an empty store prints `no notes`. JSON output is
   `{"notes":[…]}`.
4. `tag add <note-id> <tag>...` requires at least two arguments and prints
   `tagged n1`; an unknown id returns the store's error unchanged, so
   `errors.Is(err, ErrNoteNotFound)` holds. `tag list` prints one tag per line,
   or `no tags`; in JSON, `{"tags":[…]}`.
5. `notes` and `notes tag` print their help and return no error; `notes
   frobnicate` and `notes tag frobnicate` return an error. Bad argument counts
   and unknown flags return errors too — and on any failure nothing is written to
   stdout or stderr, because `main` owns error reporting.
6. `App.Resolve` fills `Settings` with precedence default < environment < flag,
   reading `NOTES_FORMAT` and `NOTES_LIMIT` through `app.env`. A format that is
   neither `text` nor `json` is an error mentioning `invalid format "yaml"`; an
   unparseable limit is an error mentioning `invalid NOTES_LIMIT "abc"`.
7. Handlers write to `cmd.OutOrStdout()`, return errors, and never call
   `os.Exit`. `go test -race ./...` passes and the code is `gofmt`-clean.

Run `go test -race ./...` from inside `exercise/`. The dependency is already
pinned in `go.mod`/`go.sum` — no network needed after the first download.

## Further reading

- [The Cobra documentation](https://cobra.dev) — user guide, completions, hooks
- [pkg.go.dev — cobra.Command](https://pkg.go.dev/github.com/spf13/cobra#Command)
  — every field you can set, in one page
- [pkg.go.dev — pflag](https://pkg.go.dev/github.com/spf13/pflag) — the flag
  parser cobra uses, including `Changed` and the flag-set views
- [Viper](https://github.com/spf13/viper#readme) — read it to judge it, not to
  adopt it by reflex
