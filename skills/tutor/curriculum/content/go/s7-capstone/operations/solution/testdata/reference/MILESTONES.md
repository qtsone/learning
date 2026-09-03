# Milestones

Each milestone leaves the project runnable and green. Tick a box only when
`go test -race ./...` passes with the milestone's behaviour in place.

- [x] **M1 — walking skeleton.** `go run ./cmd/tinysvc` listens and answers one
      route. Module, package layout and test command all work end to end.
- [x] **M2 — configuration surface.** `internal/config` reads the environment
      at startup, validates every value, and fails loudly on a bad one.
- [x] **M3 — the operable surface.** `internal/httpapi` serves the work route
      plus health, readiness and metrics, and logs one structured line per
      request with the route pattern rather than the path.
- [x] **M4 — lifecycle.** `main` catches SIGTERM, withdraws readiness, and
      drains in-flight requests inside a bounded window.
- [x] **M5 — hardened.** The request id is validated at the boundary with a
      fuzz target and a committed corpus behind it, and SECURITY.md records the
      findings, the fixes and the risks accepted.
- [x] **M6 — operable by someone else.** Multi-stage image, scripted deploy and
      rollback, CI gates that run before the merge, and a runbook with the two
      failure modes this service actually has.

## Cut (not in this milestone set)

- Authentication on the work route — see the accepted risks in SECURITY.md.
- Tracing: a single-process service gets most of the value from the request
  line already emitted; revisit when it makes an outbound call.
