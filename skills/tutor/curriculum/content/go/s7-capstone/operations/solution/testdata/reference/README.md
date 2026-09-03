# tinysvc

A minimal HTTP service that exists to be *operated*: it is the reference
project for the operations checks. Small enough to read in five minutes, and
carrying the whole operational surface a real service needs — a container
image, CI gates, telemetry, a lifecycle that survives a rollout, a documented
configuration surface, and a runbook.

## Run it

```sh
go run ./cmd/tinysvc
curl localhost:8080/healthz
curl localhost:8080/readyz
curl localhost:8080/metrics
curl localhost:8080/work/42
```

A work id is 1-64 characters of `A-Z a-z 0-9 _ -`; anything else is answered
with 400 and never echoed back.

## Test it

```sh
go test -race ./...
go test -fuzz=FuzzValidateWorkID -fuzztime=60s ./internal/httpapi
go vet ./...
govulncheck ./...
```

## Build the image

```sh
make image VERSION=v0.1.0
```

## Configuration

Everything is read from the environment at startup and validated there: a bad
value stops the process rather than surfacing as strange behaviour under load.
There are no secrets in this service; if there were, they would arrive the same
way and never be logged.

| Variable | Default | Meaning |
|---|---|---|
| `TINYSVC_ADDR` | `:8080` | Listen address. |
| `TINYSVC_LOG_LEVEL` | `info` | Minimum log level: `debug`, `info`, `warn`, `error`. |
| `TINYSVC_SHUTDOWN_TIMEOUT` | `15s` | How long in-flight requests get to finish after SIGTERM. Keep it below the platform's termination grace period. |

## Telemetry

- **Logs** — JSON on stdout via `log/slog`, one line per request with `method`,
  `route`, `status` and `duration_ms`. `route` is the route pattern, never the
  request path, so the field stays low-cardinality.
- **Metrics** — `expvar` counters on `GET /metrics`: `http_requests_total`,
  `http_errors_total`, plus the runtime's memory stats.
- **Health** — `GET /healthz` answers "is this process broken?" and
  `GET /readyz` answers "can it serve right now?". They are different
  questions, so they are different endpoints.

## Layout

```
cmd/tinysvc      composition root: config, logger, server, signal handling
internal/config  the configuration surface and its validation
internal/httpapi routes, request logging, health, readiness, metrics
deploy/          deployment as code: image rollout, rollback, manifest
MILESTONES.md    what was delivered, in the order it was delivered
SECURITY.md      trust boundaries, inputs, secrets, dependencies, accepted risk
RUNBOOK.md       deploy, rollback, triage, paging, and the two failure modes
```
