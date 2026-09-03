# Pre-PR checklist

The gate. Every box ticked, or waived on the line next to it with a one-line
reason ("no CHANGELOG in this project", "docs fix, no test possible — evidence
below"). A waiver with a reason is fine; an unticked box is not.

Run this *after* you have read your own diff top to bottom as a diff, not as
files.

## Correctness

- [ ] The change does what the issue asked, and nothing else
- [ ] A test exists that **fails before** my change and **passes after**
      — command and both outputs pasted below
- [ ] Their full suite is green the way `CONTRIBUTING.md` says to run it
- [ ] `go vet ./...` clean (or their lint config, if stricter)
- [ ] Edge cases the surrounding code cares about are covered: empty input,
      error path, the platform difference I noticed while reading

```sh
# before:
# after:
```

## Scope

- [ ] One concern. A second bug I found is filed separately: `<link>` or "none"
- [ ] No renames, reformatting or refactors the issue did not require
- [ ] No new module dependency (`go.mod` and `go.sum` unchanged, or the
      maintainer agreed in the issue: `<link>`)
- [ ] No new exported API, or the addition is called out at the top of the PR
      description with the reason
- [ ] Every file in the diff would be there if someone else had fixed this

## Their conventions

- [ ] `gofmt -l .` prints nothing
- [ ] Test style matches the file I edited: table shape, helper names,
      `t.Run` naming, assertion phrasing
- [ ] Error handling matches local practice (sentinel errors and `errors.Is`
      if that is what the package does; wrapping phrased their way)
- [ ] Naming and comment style match the surrounding code, not my habits
- [ ] Docs updated where the behaviour is documented
- [ ] `CHANGELOG.md` entry added if the project keeps one

## Commits

- [ ] Commit series (or single commit) matches what they merge — I checked the
      last three merged PRs
- [ ] Subject lines are imperative, under ~70 characters, and say *why*
- [ ] `Signed-off-by` present if the project uses the DCO (`git commit -s`)
- [ ] CLA signed if required — done on `<date>`
- [ ] Branch is rebased on the current default branch and CI-clean

## The description

- [ ] Links the issue so it closes on merge
- [ ] **What**: the behaviour before and after, in two sentences
- [ ] **Why this approach**: including the alternative I rejected and why
- [ ] **Testing**: the test I added and the fails-before/passes-after claim
- [ ] **Not included**: what I left out on purpose, and where it went instead
- [ ] Nothing in it assumes the reviewer remembers the issue

## Last look

- [ ] I read the whole diff as a diff, and I would approve it
- [ ] No debug prints, commented-out code, stray files, or editor config
- [ ] Nothing personal, private or local leaked into a path, test fixture or
      commit message
- [ ] I can say in one sentence why this PR is worth a maintainer's time
