# Exercise — Document the snippets project

You have inherited `snippets/`, a small command-line tool that stores named
text snippets in a JSON file. The code is fine — it would survive your
clean-code review — but the documentation is a disaster: a README that fails
every stranger question, doc comments that echo signatures, a storage
decision nobody wrote down, and a release about to ship with no changelog
entry. Your job is a pure documentation pass, with every decision recorded
in `NOTES.md` and defended in conversation with your tutor.

Ground rules:

- **Behavior is off-limits.** You are documenting the code as it *is*. If
  you think a behavior is wrong, note it in `NOTES.md` part 5 as a proposed
  code change — do not make it, and do not document the version you wish
  existed.
- Doc comments are code: keep `gofmt -w .` running, and `go build ./...`
  must stay clean from inside `snippets/`.
- Every claim you write must be checkable against a line of code or a
  command you actually ran.

## Part 1 — Stranger-test the README, then rewrite it

Read `snippets/README.md` as a competent stranger: skilled, zero context,
clean machine. In `NOTES.md`, list at least six questions the stranger needs
answered that this README cannot answer (what is it for? how do I…?).

Then rewrite `snippets/README.md` so it answers all of them: one-paragraph
purpose, requirements, install, a quickstart with real commands and real
output, current flags, where the data lives. Before you call it done, run
every command in it *exactly as written* from a clean directory — the
stranger will copy-paste, so you must too. One useful sentence in the old
README is true and worth keeping in some form; find it.

## Part 2 — Doc comments: from echo to contract

Every doc comment in `snippets/store/store.go` is an echo (`// Add adds a
snippet.`). Rewrite all of them — the package comment, `ErrNotFound`,
`Store`, `Open`, `Add`, `Get`, `List` — as contracts in Go's form: complete
sentences, first sentence starting with the name, stating what a caller may
rely on.

You find the contract by reading the body, not by guessing intent. For each
identifier, fill a row in the `NOTES.md` table: what the old comment failed
to say. Between them, your comments must settle at least: what `Open` does
when the file doesn't exist, what `Add` does with an existing name and with
an empty name and when data hits disk, how `Get` reports a missing name and
how a caller should test for it, what order `List` returns, and whether a
`Store` may be shared between goroutines.

Test for every sentence you write: could a test falsify it? If not, it's an
echo — cut it.

## Part 3 — Write the missing ADR

When this project started, the author weighed storing snippets in a SQLite
database (you drove `database/sql` earlier this stage), in a directory with one
file per snippet, and in a single JSON file — and chose the JSON file. The
reasoning was never written down; reconstruct it honestly from what the
project is (a single-user CLI with tiny data) and record it in
`docs/adr/0001-single-json-file-storage.md`, which contains the skeleton.

Requirements: context stated as *forces*, not conclusions; at least two
rejected options, each with the honest reason it lost *and* what it would
have bought; a one-sentence decision; consequences that include at least two
genuine costs of the choice you're recording. In `NOTES.md`, note what would
have to change about the project for this decision to deserve superseding.

## Part 4 — Curate the changelog

`snippets/HISTORY.txt` is the raw `git log --oneline v0.1.0..HEAD` output —
nine commits, captured to a file so the exercise needs no git archaeology.
Release `0.2.0` ships from here.

In the `NOTES.md` table, classify every commit: **Added**, **Changed**,
**Fixed**, breaking (and for whom — CLI users and importers of `store` are
both your audience), or **omit** — with a one-line reason each. Then write
the `0.2.0` entry in `snippets/CHANGELOG.md`, following the `0.1.0` entry's
format: breaking changes first and clearly marked with what the user must
do, every line in the user's vocabulary, nothing a user can't observe.

## Part 5 — Placement drill

For each item in the `NOTES.md` part 5 table, name its home — README, doc
comment, ADR, changelog, or none of the four — and justify it in one
sentence using the lesson's routing principles (closest to what it
describes, one home per fact, the reader's moment).

Final check from inside `snippets/`:

```sh
gofmt -l .        # no output
go build ./...    # clean
```

Then bring the four rewritten artifacts, `NOTES.md`, and your reasoning to
the discussion.
