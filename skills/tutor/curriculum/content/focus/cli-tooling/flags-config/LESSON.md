# Flags & Configuration

> `focus.cli.flags-config` · ~3-4h · Stage: Focus: CLI Tooling

## Objectives

By the end of this lesson you can:

- Parse command-line flags with the standard `flag` package, including a custom
  `flag.Value` implementation for a non-primitive type.
- Implement a configuration precedence chain (defaults < config file <
  environment variables < flags) and justify that ordering.
- Load settings from environment variables into a typed config struct with
  validation and clear error messages for bad values.
- Choose between flags, environment variables, and a config file for a given
  setting and justify the choice in terms of who changes it and how often.

## Configuration is an interface, not a detail

Your S3 link checker already had its own `flag.FlagSet` and a
`run(ctx, args, in, out) error` entry point — the skeleton. Real tools
take settings from several places at once, and the *rules* for combining them
are as much a part of your public interface as the flag names. The question
that decides where a setting belongs is always: **who changes it, and how
often?**

| Layer | Who sets it | How often | Good for |
|---|---|---|---|
| Defaults | you, the author | never | what is right for most people |
| Config file | the team, checked in | per project | many settings, shared, reviewable |
| Environment | the deployment system | per machine or session | secrets, per-environment endpoints, CI |
| Flags | the human at the keyboard | per invocation | this one run, right now |

Two consequences fall out. Something tweaked per run (`-verbose`, an output
path) must be a flag, because editing a file to change one run is absurd, and
something injected by a deployment system must come from the environment,
because CI cannot type. A secret should *never* be a flag: command lines show
up in `ps` output and land in shell history. Most settings deserve all three
routes at once, which is the whole problem.

## The `flag` package, past `-v`

Package-level `flag.String` writes to a hidden global `FlagSet` — never use it
in code you want to test, as you already know from S3. Build your own:

```go
fset := flag.NewFlagSet("toolkit", flag.ContinueOnError)
fset.SetOutput(stderr)            // all output goes where you say
timeout := fset.Duration("timeout", 5*time.Second, "per-request timeout")
if err := fset.Parse(args); err != nil {
	return err                // no os.Exit anywhere in sight
}
rest := fset.Args()               // positional arguments, after the flags
```

- **`flag.ContinueOnError`** makes `Parse` *return* the error. The alternative,
  `ExitOnError`, calls `os.Exit(2)` inside your library code: untestable and
  rude. (`PanicOnError` exists; ignore it.)
- **`SetOutput`** redirects usage text and parse errors to an injected writer.
  Without it, `flag` writes to `os.Stderr` behind your back and tests cannot
  see what the user saw.
- **`-h`/`-help`** makes `Parse` print usage and return the sentinel
  `flag.ErrHelp` — not a failure, so it exits 0, not 2.

`Parse` stops at the first non-flag argument, so `toolkit -v sync -f` leaves
`["sync", "-f"]` in `fset.Args()`, `-f` untouched — the hook subcommands hang
from, next lesson.

## Was it set, or is it merely zero?

Here is the trap that makes naive precedence code wrong. A flag variable holds
`false`. Did the user type `-verbose=false`, or say nothing? The value cannot
tell you, and if you guess "zero means unset" you silently ignore everyone who
explicitly asks for the default. `FlagSet` has the answer: `VisitAll` walks
every flag that exists, `Visit` walks only the flags actually present on the
command line. So register your flags with **zero values**, parse, then ask:

```go
set := map[string]bool{}
fset.Visit(func(f *flag.Flag) { set[f.Name] = true })
```

The same question returns at every layer with a different mechanism: `fset.Visit`
for flags; `os.LookupEnv`'s `(value, ok)` for the environment, where `os.Getenv`
throws `ok` away and makes an empty-but-set variable look absent; and pointer
struct fields for the config file, where `nil` means *absent from the document*
while a pointer to `false` means *present and false* — the JSON lesson's trick,
earning its keep. One idea, three mechanisms; miss any of them and your chain
has a hole.

## Teaching `flag` a new type: `flag.Value`

`flag` handles strings, ints, bools and durations. For anything else you
implement `String() string` and `Set(string) error`. `Set` is called **once per
occurrence** on the command line, which is what makes repeatable flags work:

```go
type TagList []string

func (t *TagList) String() string { return strings.Join(*t, ",") }

func (t *TagList) Set(v string) error {
	if v = strings.TrimSpace(v); v == "" {
		return errors.New("tag must not be empty")
	}
	*t = append(*t, v)
	return nil
}
```

Register it with `fset.Var(&tags, "tag", "tag to attach, repeatable")` and
`-tag build -tag nightly` gives you `TagList{"build", "nightly"}`. Two gotchas:
`Set` needs a **pointer receiver** because it mutates, and `flag` calls
`String()` on a freshly made zero value while printing usage, so it must not
panic on an empty receiver. For a one-off parse with no value worth displaying,
`flag.Func` is lighter — it takes the `Set` half and nothing else:

```go
fset.Func("label", "k=v label, repeatable", func(s string) error {
	k, v, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return fmt.Errorf("want k=v, got %q", s)
	}
	labels[k] = v
	return nil
})
```

`Func` gives you no control over `String()` (it always reports `""`), so reach
for a real `Value` when the current value should appear in usage or in your own
diagnostics. (For a type that implements both `encoding.TextUnmarshaler` and
`encoding.TextMarshaler`, `fset.TextVar` wires it up with no extra code at
all — it needs the marshaler too, because that is how it renders the default
in the usage line.)

## Usage output you actually control

`flag`'s default usage is a bare list of flags: it never mentions your
environment variables, your config file, or your precedence rules, so the one
place users go to answer "how do I configure this?" answers a third of the
question. Take it over:

```go
fset.Usage = func() {
	fmt.Fprint(out, "toolkit — demo client\n\nUsage:\n  toolkit [flags]\n\nFlags:\n")
	fset.PrintDefaults()
	fmt.Fprint(out, "\nEnvironment:\n  TOOLKIT_ENDPOINT\n  …\n")
	fmt.Fprint(out, "\nPrecedence (lowest to highest):\n  defaults < config file < environment < flags\n")
}
```

Registering zero values for `Visit` has a knock-on effect here: `flag`'s
automatic `(default …)` suffix would now print `0` and `""` — a lie. Put the
real default in the usage string yourself, and name the environment variable
feeding the same setting while you are there.

## Precedence is a contract

Layer the four sources lowest to highest — **defaults < config file <
environment < flags** — and the ordering is not arbitrary: each layer sits
above one that is *broader and less immediate*.

Defaults apply to everyone, so everything overrides them. A
config file is shared and checked in, so it outranks your opinion as the author
but must not outrank the machine it runs on. The environment speaks for one
machine, one container, one CI job — and it is how secrets arrive. Flags are a
human typing right now with full context, and nothing may override the person
at the keyboard.

Invert any pair and you get a tool people fight. If the file beat the
environment, you could not point the same checked-in repo at staging without
editing a tracked file. If the environment beat flags, a stale
`TOOLKIT_TIMEOUT` in someone's shell profile would quietly swallow their
`-timeout`, and finding that costs an afternoon. So: pick the order, **write it
in `--help`**, and never break it. It is as much a compatibility promise as
your flag names.

## "Where did this value come from?"

Precedence makes a value hard to explain. The user sees a 2-second timeout, has
`-timeout 10s` in their shell history, and is filing a bug against you. Answer
it in the data structure: return the resolved `Config` alongside an
`Origins map[string]Source`, mapping each setting name to the layer that last
wrote it (`"timeout"` → `SourceEnv`). Every layer that writes a field writes
its origin in the same breath. One extra line per assignment buys a
`--show-config` that ends the argument in ten seconds — and the value alone can
never tell you, since a setting can come from the environment *and* happen to
equal the default.

Provenance also upgrades your errors. "invalid retries" is a shrug;
`retries: invalid value "abc" from environment: …` says which of four places to
go and edit. Build it into an error type carrying `Field`, `Source`, the raw
text and the wrapped cause, with an `Unwrap` method: `errors.Is` keeps working
against sentinels like `ErrRange`, and `errors.As` lets callers and tests pull
out the field and the source without matching on message text. Same S3 errors
discipline, applied to configuration.

## Validate once, at the end

Validate the *resolved* config, not each layer as it arrives: a file may
legitimately set `retries: 20` that a flag then brings back under the limit, and
rejecting a value nobody ended up using is a bug. Parse failures ("`abc` is not
a number") belong to the layer that produced them; range failures belong to the
final result, attributed through the origins map.

## Exercise

Open [`exercise/`](exercise/) — a config loader for a fictional `toolkit`
client. `config.go` holds `Config`, `Source`, `ValueError`, `Defaults` and
`Result.Validate`; `flagvalue.go` the `TagList` custom `flag.Value`; `loader.go`
the `Loader`, one function per layer — `applyFile`, `applyEnv`, `applyFlags` —
and the `Load` that orchestrates them. Two things there are already written:
`registerFlags`, because transcribing six `fset.Var` calls teaches nothing the
comment above them does not, and `main.go`, which wires the loader to the real
process and prints every setting with its origin — read it, do not edit it. The
`*_test.go` files are the specification, so read them first.

Acceptance criteria:

1. `Source.String()` returns `default`, `config file`, `environment`, `flag`,
   and `unknown` for anything else.
2. `Defaults()` returns endpoint `https://api.example.com`, timeout `5s`,
   retries `3`, verbose `false`, tags `nil`.
3. `TagList` implements `flag.Value`: `Set` trims surrounding space, rejects an
   empty tag with an error, appends otherwise; `String` joins with commas and
   is safe on an empty list. `-tag a -tag b` yields `["a","b"]`.
4. `Load` resolves settings **defaults < config file < environment < flags**,
   and `Result.Origins` records for every one of the five settings the layer
   that supplied the final value, including when it equals the default.
5. A flag counts as set only when it appears in `Args`: `-verbose=false` and
   `-retries=0` override a config file; an absent flag does not.
6. `-tag` *replaces* the tag list from lower layers rather than appending.
   `TOOLKIT_TAGS` is comma-separated, and an empty item in it is an error.
7. The config file is JSON: absent keys leave lower layers alone, unknown keys
   are an error naming the key, `timeout` is a duration string like `"1s"`. A
   missing file at `Loader.ConfigPath` is skipped; a missing file named by
   `-config` is an error satisfying `errors.Is(err, fs.ErrNotExist)`.
8. Bad values produce a `*ValueError` reachable with `errors.As`, carrying
   `Field`, `Source` and the raw text; range and scheme failures wrap
   `ErrRange`, `ErrEmpty` or `ErrScheme`. Validation order: endpoint, timeout,
   retries.
9. `Load` never calls `os.Exit`: a bad flag returns an error and writes to
   `Loader.Stderr`; `-h` returns `flag.ErrHelp` after writing usage that lists
   the flags, the `TOOLKIT_*` variable names, and the precedence line.
10. `go test ./...` passes from inside `exercise/`, `gofmt`-formatted.

The tests fail on the starter. Take the layers bottom-up — `Load` and the
defaults first, then `applyFile`, then `applyEnv`, then `applyFlags` — and a
clump of cases goes green with each one, because a layer you have not written
yet simply leaves the result alone. The failure messages name the setting and
the layer.

## Further reading

- [pkg.go.dev — flag](https://pkg.go.dev/flag) (read `Value`, `Func`, `Visit`)
- [The Twelve-Factor App — Config](https://12factor.net/config): the case for
  the environment layer, and its limits
- [go.dev blog — Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
