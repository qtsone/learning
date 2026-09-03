# Testing notes

Evidence, not vibes. Your tutor will ask for every section.

## 1. Fuzzing

Command you ran (include `-fuzztime`):

```
(paste the command)
```

How many inputs did the engine execute, and how many *interesting* ones did
it keep (the `new interesting:` counter)? Why does that number climb quickly
at first and then almost stop?

> your answer:

Did it find a failure? If so, paste the crasher and say which corpus file
under `testdata/fuzz/FuzzParseLine/` now holds it. If not, say what property
you asserted and why you believe it is strong enough to catch a real bug.

> your answer:

## 2. Coverage

```sh
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1
go tool cover -html=cover.out          # opens the annotated source
```

Total statement coverage:

> your answer:

Name one line the report shows as uncovered that you decided *not* to test,
and one that made you write a test. Then answer: why is "95% coverage" a
weaker claim than it sounds?

> your answer:

## 3. Golden file

Run `go test -run TestRunGoldenReport -update .` after a deliberate change to
`Report` (say, widen the count column), then look at the diff of
`testdata/golden/service-report.txt` before restoring it.

What would have happened if you had run `-update` without reading that diff?

> your answer:

## 4. Doubles

For each dependency in this exercise, name the double you used and why:

| Dependency | Real / fake / stub | Why |
|------------|--------------------|-----|
| time (retry backoff) | | |
| SQLite store | | |
| stdin, stdout, stderr | | |

Which of these would you *also* run against the real thing in CI, and what
would that catch that the double cannot?

> your answer:
