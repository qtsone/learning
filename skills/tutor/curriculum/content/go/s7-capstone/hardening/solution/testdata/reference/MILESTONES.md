# Milestones

Each milestone leaves the project runnable and green. Tick a box only when
`go test -race ./...` passes with the milestone's behaviour in place.

- [x] **M1 — walking skeleton.** `go run ./cmd/notes` prints one hard-coded
      line. Module, package layout and test command all work end to end.
- [x] **M2 — domain rules.** `internal/note` validates and normalises titles
      and tags, with table-driven tests and a named error vocabulary.
- [x] **M3 — storage behind an interface.** `internal/store` holds notes,
      preserves listing order, filters by tag, and is safe under `-race`.
- [x] **M4 — wired and documented.** `main` composes the two packages, the
      README explains how to run it, and ADR-001 records the storage choice.
- [x] **M5 — hardened.** Input arrives from stdin through a validating parser
      with a fuzz target and a committed corpus, the one outbound call is
      bounded and redacts its URL, and SECURITY.md records the findings, the
      fixes and the risks accepted.

## Cut (not in this milestone set)

- Persistence to disk — deliberately deferred; see ADR-001.
- A real CLI argument parser — the seeded list is enough to demonstrate wiring.
