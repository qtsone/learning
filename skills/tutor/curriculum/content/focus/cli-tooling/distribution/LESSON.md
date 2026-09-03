# Distribution

> `focus.cli.distribution` · ~2-3h · Stage: Focus: CLI Tooling

## Objectives

By the end of this lesson you can:

- Apply semantic versioning to a CLI tool and explain which changes force a
  major version bump.
- Embed version, commit, and build date via `-ldflags -X` and expose them
  through a `--version` flag, with a sane fallback for `go install` users.
- Cross-compile release binaries for several `GOOS`/`GOARCH` pairs, explain why
  Go makes that cheap, and make the build reproducible.
- Configure goreleaser to build that matrix from a git tag and publish a
  Homebrew formula, and weigh what each distribution channel costs.

## The last mile

A tool that works on your machine is half a tool. The other half is everything
between your working directory and someone else's `PATH`: a binary built for
*their* operating system and processor, a version they can quote in a bug
report, a way to check that what they downloaded is what you published, and an
upgrade that does not involve reading your README again. None of that is Go
programming, and all of it is why tools get abandoned — though Go's build model
makes the hard part nearly free, and one config file automates the tedium.

## A CLI has a public API, and semver is its contract

The moment someone scripts your tool, parts of it stop being yours to change:
the command tree, flag names and shorthands, the positional-argument contract,
**exit codes** (`&&`, `if` and CI branch on those), the JSON shape you emit, and
the environment variables and config keys you read. Not API — and this is
exactly why the terminal-UX lesson kept them apart — the wording and layout of
human output, colour, help text, stderr diagnostics, and every Go package inside
your module.

`MAJOR.MINOR.PATCH` then answers one question: *what must a user do when this
number changes?*

- **MAJOR** — you broke something they could rely on. Removing or renaming a
  flag or subcommand. Changing what a flag's default *does*. Renaming a JSON
  field or changing its type (`"count": 3` → `"count": "3"`). Repurposing an
  exit code. Making an optional argument required. Reordering the config
  precedence chain.
- **MINOR** — new abilities, old invocations untouched: a new subcommand, a flag
  whose default preserves today's behaviour, an *added* JSON field (which is why
  consumers must tolerate unknown fields).
- **PATCH** — a fix that makes the tool do what it already documented.

Two honest complications. `0.x` means "no promises yet": by convention a
`0.MINOR` bump may break anything, and you reach `1.0.0` when you are willing to
keep promises, not when the tool feels finished. And users depend on behaviour
you never documented — someone is parsing your table with `awk` right now. You
cannot prevent that; you can publish a machine-readable format so they have a
better option, and say plainly in the release notes what changed. When you must
remove something, deprecate first: keep the old flag working for one major
cycle, warn **on stderr** (never stdout — that is the user's data), and delete
it at the next major bump.

Tag releases with a leading `v` (`v1.4.2`): that is the Go module convention and
what `go install tool@v1.4.2` speaks. goreleaser's `.Version` is the tag without
the `v`, which is why the binary here prints `digest 1.4.2`.

## Stamping identity into the binary

A version constant in the source is a version you will forget to bump. Let the
*build* supply it instead — the Go linker can overwrite a package-level string
variable:

```sh
go build -ldflags "-X tutor.local/digest/internal/buildinfo.version=1.4.2" .
```

The rules for `-X importpath.name=value`:

- The symbol is the **full import path** plus the variable name. In
  `package main` that is `main.version`; in a subpackage it is the module path
  plus the directory — `tutor.local/digest/internal/buildinfo.version`.
- The target must be a package-level `var` of type `string` initialised to a
  constant. A `const` is inlined at compile time and cannot be patched, and
  neither can a var whose initialiser is a function call.
- **A wrong symbol path is silently ignored.** The build succeeds, the linker
  says nothing, and your release ships `dev` to everyone. This is the classic
  version-stamping bug; the only defence is to *run* `--version` in the pipeline
  and assert on its output, which is what this lesson's checker does to you.
- One `-X` per value; quote values containing spaces. Two more linker flags
  travel with release builds: `-s` drops the symbol table and `-w` drops DWARF
  debug information, cutting megabytes at the cost of debugger support — worth
  it for a published artifact, pointless locally.

What to print:

```text
digest 1.4.2 (commit 9f8e7d6c, built 2024-05-01T10:00:00Z, go1.22.3 linux/amd64)
```

Version says which release; commit says exactly which source, even for an
unreleased build; the date settles "did they get the rebuilt artifact?"; the Go
version and platform pair come free from `runtime` and answer half the follow-up
questions on a bug report. Print it on **stdout** and exit 0 — a version query
is a successful run, and someone will parse it.

## What the binary already knows

Not every user gets your linker flags. `go install example.com/tool@v1.4.2`
compiles from source on their machine with no release pipeline anywhere. Go
fills that gap itself:

```go
bi, ok := debug.ReadBuildInfo()   // runtime/debug
bi.Main.Version                   // "v1.4.2", or "(devel)" for a local build
bi.Settings                       // vcs.revision, vcs.time, vcs.modified, …
```

Since Go 1.18 the toolchain stamps VCS information into any binary built inside
a repository (`vcs.modified` is `true` when the tree was dirty) and records the
module version when the source came through the module proxy; `-buildvcs=false`
turns the VCS half off. So the precedence for your version line is: linker
flags, then build info, then placeholder — and note the trap the exercise
checks, that `(devel)` and `""` are not real versions, so a plain local build
must still say `dev`. You can read all of this back out of *any* Go binary with
`go version -m ./digest`.

## Cross-compilation is two environment variables

```sh
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o dist/digest-linux-arm64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/digest.exe .
```

That is the whole thing; `go tool dist list` prints every supported pair. Why it
is this easy is unusual enough to understand: the Go compiler and linker are Go
programs that emit code for every target, and the runtime and standard library
are *compiled from source for the target as part of your build*. No system
linker to find, no libc headers, no cross toolchain — compare a C project, where
"build for the other OS" starts with acquiring a compiler.

The condition is **pure Go**. cgo compiles C, and C needs a target C compiler, so
the moment cgo is in play cross-compilation becomes the ordinary painful thing.
`CGO_ENABLED=0` is both the switch and a constraint on which dependencies you
accept. It changes two standard-library behaviours: `net` uses the pure-Go DNS
resolver rather than libc's (ignoring exotic `nsswitch.conf` setups), and
`os/user` reads `/etc/passwd` directly. For a CLI both are non-issues, and the
payoff is a **static binary** — no shared-library dependencies, so one Linux
artifact runs on Debian, on Alpine's musl, and in a `FROM scratch` container.

Details you owe your users: Windows binaries need the `.exe` suffix; macOS needs
both `amd64` and `arm64` (two archives, or one universal binary — goreleaser's
`universal_binaries`); and the `filepath` discipline from the previous lesson is
what keeps the result honest off Unix.

## Reproducible builds and checksums

Build the same commit twice and you should get the same bytes; that is what lets
anyone check a published binary against the published source. Three things
usually break it:

- **Absolute paths.** Without `-trimpath`, your working directory is baked into
  the binary and into panic traces. `-trimpath` replaces it with the module path.
- **A wall-clock timestamp.** `-X …date=$(date -u +%FT%TZ)` guarantees a
  different binary every run. Stamp the *commit* date: it is a property of the
  source, not of when you happened to build.
- **A different toolchain.** Pin the Go version in your release workflow; a
  minor bump changes code generation.

Archives have their own version of the problem — file modification times inside
a `tar.gz` change its checksum even when the binary is identical — which is what
goreleaser's `mod_timestamp` fixes. Then publish `checksums.txt`, one SHA-256
per artifact, so users can run `shasum -a 256 -c checksums.txt`. Be honest about
what that buys: it detects a truncated download or a tampered mirror, but does
*not* prove you built it — provenance needs a signature over the checksum file
(cosign, minisign or GPG; goreleaser has a `signs` section), and a signature is
only as good as the key.

## goreleaser: the release as a config file

Everything above is scriptable, and the script you would write is the one
everybody writes. goreleaser is that script, configured rather than coded. Given
a git tag it builds the `GOOS`/`GOARCH` matrix with your flags, packages each
binary into an archive alongside the README and LICENSE, writes `checksums.txt`,
generates release notes from the commit subjects since the previous tag, creates
the GitHub (or GitLab, Gitea) release and uploads everything, then publishes
downstream packages: a Homebrew formula, `deb`/`rpm` via nfpm, Scoop manifests,
container images.

It takes the version from the git tag and refuses to run on a dirty tree or
without one — the release is defined by the repository, not by your shell
history. The real invocation lives in CI, triggered by a tag push:

```sh
git tag -a v1.4.2 -m "digest v1.4.2" && git push origin v1.4.2
goreleaser release --clean          # in CI, with a token in the environment
```

Locally, `goreleaser check` validates the config and
`goreleaser release --snapshot --clean` produces the whole `dist/` — binaries,
archives, `checksums.txt`, an `artifacts.json` manifest — without publishing
anything.

The heart of the config is the build, and the interesting keys are the ones this
lesson has already argued for:

```yaml
version: 2                    # goreleaser's config format, not your tool's version
builds:
  - id: digest
    main: .
    binary: digest
    env: [CGO_ENABLED=0]      # static binaries, and cross-compilation stays free
    flags: [-trimpath]        # reproducibility
    ldflags:
      - -s -w
      - -X tutor.local/digest/internal/buildinfo.version={{ .Version }}
    mod_timestamp: "{{ .CommitTimestamp }}"
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]    # six binaries from one tag
```

`{{ .Version }}`, `{{ .FullCommit }}` and `{{ .CommitDate }}` are template values
filled from the tag and the commit — prefer `.CommitDate` over `.Date`, which is
the build clock, for the reason the previous section gave. Around this block sit
`archives` (naming, and what else goes in the tarball), `checksum`, and
`changelog`, whose `filters.exclude` list drops noise commits. That last one is
why teams write commit subjects in a consistent style: your release notes *are*
your `git log`, so a log full of "wip" produces notes full of "wip".

## Homebrew taps

A **tap** is a git repository named `homebrew-<something>` under your account.
`brew tap you/tap` clones `github.com/you/homebrew-tap`, and
`brew install you/tap/digest` installs `digest.rb` from it. The naming rule is
not cosmetic: brew derives the repository name from the tap name, so a repo
called `tap` or `brew-tools` will not be found.

A formula is a Ruby class, and for a prebuilt Go binary it is a *downloader*,
not a builder — it picks the archive for the user's platform, pins its SHA-256,
and copies the binary into `bin`:

```ruby
class Digest < Formula
  version "1.4.2"
  on_macos do
    on_arm do
      url "https://github.com/you/digest/releases/download/v1.4.2/digest_1.4.2_darwin_arm64.tar.gz"
      sha256 "…"
    end
  end
  def install
    bin.install "digest"
  end
end
```

Writing that by hand for six artifacts every release is the tedium a `brews:`
block removes: goreleaser generates the formula with URLs and checksums filled
in and commits it to your tap after a successful release. Two practical notes.
The token that pushes to the tap is a *different* credential from the one that
creates the release, because the release token only covers the tool's own
repository. And recent goreleaser versions also offer `homebrew_casks`, since
Homebrew now prefers casks for prebuilt binaries — check the deprecation notes
for the version you install. Resist the pull of homebrew-core
(`brew install digest`, no tap): it means maintainer review, notability rules,
and someone else's cadence. Your own tap costs one repository and is yours.

## Channels, and what each one costs

| channel | the user types | what it costs you |
|---|---|---|
| `go install mod@latest` | one command, needs Go | nothing — but no ldflags, so `ReadBuildInfo` is your only version source, and non-Go users are excluded |
| release archive | download, extract, `chmod +x`, move onto `PATH` | free with goreleaser; worst UX, and where checksums matter most |
| install script (`curl … \| sh`) | one line | you host and maintain the script; no upgrade or uninstall unless you write them; ask users to read before piping |
| Homebrew tap | `brew install you/tap/digest` | one config block plus a tap repo; macOS and Linux only |
| Scoop / winget | `scoop install`, `winget install` | manifests, and winget adds a review queue |
| deb / rpm (nfpm) | `apt install` from your repo | packaging metadata plus hosting a package repository |
| container image | `docker run …` | image builds; natural for CI tools, awkward for a tool that reads local files |

A sane start for a new tool: release archives with checksums, `go install` for
the Go crowd, and a Homebrew tap. Add channels when users ask, because every
channel is one more thing that can break on release day.

## Exercise

Open [`exercise/`](exercise/) — `digest`, a tool that prints the SHA-256 of the
files it is given. Its logic is finished; its *packaging* is not. `main.go`
already wires a `--version` flag to `buildinfo.String()`. You write the two
pieces that ship it:

- `internal/buildinfo/buildinfo.go` — the stamped variables and the version line.
- `.goreleaser.yaml` — the release configuration; this file does not exist yet.

`check.sh` is the referee: it builds your tool the way a release pipeline would,
runs `--version`, cross-compiles it, and reads your config. It needs no network,
no git and no goreleaser — if goreleaser or a YAML parser is missing, those
checks are reported SKIPPED rather than failed.

Acceptance criteria:

1. `internal/buildinfo` declares package-level string vars `version`, `commit`
   and `date`, defaulting to `dev`, `none` and `unknown`.
2. `String()` returns exactly the shape
   `digest <version> (commit <commit>, built <date>, <go version> <goos>/<goarch>)`,
   and `digest --version` prints it on stdout, writes nothing to stderr, and
   exits 0.
3. When `version` is still `dev`, `String()` consults
   `runtime/debug.ReadBuildInfo` for a real module version and for
   `vcs.revision` / `vcs.time` — but `(devel)` and `""` are not real versions, so
   an ordinary local build still reports `dev`.
4. Built with the documented flags below, `--version` reports `1.4.2`,
   `9f8e7d6c` and `2024-05-01T10:00:00Z` — no placeholders left.
5. Two builds with identical inputs produce byte-identical binaries.
6. `CGO_ENABLED=0` builds succeed for `linux/amd64`, `darwin/arm64` and
   `windows/amd64`.
7. `.goreleaser.yaml` exists, parses as YAML, and declares: `version: 2`; a build
   with `CGO_ENABLED=0`, `-trimpath`, and three `-X` ldflags using goreleaser
   template values; at least two `goos` and two `goarch` entries; `archives`,
   `checksum` and `changelog` sections; and a Homebrew block whose repository
   name starts with `homebrew-`.
8. `digest <file>` still prints the right hash, `gofmt` reports nothing, and
   `go vet ./...` passes.

The documented release build — the one the checker runs, and the one your
goreleaser config expresses in template form:

```sh
PKG=tutor.local/digest/internal/buildinfo
go build -trimpath \
  -ldflags "-s -w -X $PKG.version=1.4.2 -X $PKG.commit=9f8e7d6c -X $PKG.date=2024-05-01T10:00:00Z" \
  -o digest .
```

Run the referee from inside `exercise/` until it is green:

```sh
bash ./check.sh
```

It fails on the starter. If you have goreleaser installed, finish with
`goreleaser release --snapshot --clean` and look inside `dist/` — that directory
is what your users would download.

## Further reading

- [Semantic Versioning 2.0.0](https://semver.org/) — the spec, in one page.
- [pkg.go.dev — cmd/link](https://pkg.go.dev/cmd/link) — the authoritative
  description of `-X`, `-s` and `-w`.
- [pkg.go.dev — runtime/debug.ReadBuildInfo](https://pkg.go.dev/runtime/debug#ReadBuildInfo)
  — what a Go binary knows about its own build.
- [GoReleaser documentation](https://goreleaser.com/) — the customization
  reference; read `builds`, `archives` and the Homebrew page beside your config.
- [Homebrew — How to Create and Maintain a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
  — tap naming rules and formula anatomy from the source.
