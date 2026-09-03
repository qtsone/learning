# Security notes — tinysvc

Written at the end of the hardening pass, 2026-03-08, before the operations
work. It describes what this service trusts, what it does not, and what it is
knowingly not defending against. It is short because the service is small; it
is not short because the questions were skipped.

## Trust boundaries

There are two places where something we do not control becomes something we act
on. **The listening socket** is the first and the important one: anything that
can reach the port can send any bytes it likes, and the `{id}` segment of
`GET /work/{id}` is the only request data we read. **The environment** is the
second: whoever writes the deployment manifest decides the listen address, the
log level and the shutdown timeout, and a wrong value there is an outage rather
than a compromise.

Nothing crosses those boundaries without passing through
`httpapi.ValidateWorkID` or `config.Load`. Past them the values are trusted: an
id that reached a handler body has already been checked, and a `config.Config`
that exists has already been validated.

## Inputs and validation

| Surface | Limit | Enforced in |
|---|---|---|
| request method and route | only the four declared patterns; anything else is 404 or 405 | `httpapi.Routes` |
| `{id}` path segment | `[A-Za-z0-9_-]{1,64}` | `httpapi.ValidateWorkID` |
| request headers | `ReadHeaderTimeout` of 5s | `cmd/tinysvc` |
| request body | never read; `/work` is a GET | `httpapi` |
| `TINYSVC_ADDR` | must parse as `host:port` | `config.Load` |
| `TINYSVC_LOG_LEVEL` | one of debug, info, warn, error | `config.Load` |
| `TINYSVC_SHUTDOWN_TIMEOUT` | a duration, and positive | `config.Load` |

Validation happens once, at the boundary, and produces a value the rest of the
program can use without re-checking. A rejected id is answered with 400 and a
message that reports a length rather than repeating the bytes, because that
message goes back to whoever supplied them.

## Secrets handling

This service has no secrets: no credentials, no keys, no database, no outbound
calls, and nothing at rest. That is a property worth stating rather than
assuming, because it is the thing that changes first. The moment a dependency
arrives, its credential comes from the environment — not a flag, because flags
are visible in the process list, and not a file in the repository — and this
section is rewritten before that ships.

What already holds regardless: the request log carries the route *pattern*, the
status and a duration, never the request path, the query string or a header, so
a token smuggled into a URL is not copied into the log by us.

## Dependency policy

Standard library only. A third-party module gets added when writing it
ourselves would be worse, and then only if it is maintained, has a licence we
can live with, and pulls in a dependency tree we are willing to read. That
policy is cheap to hold at this size and it is the reason the vulnerability
surface is the Go toolchain itself.

`govulncheck ./...` is run by hand against the toolchain currently in use,
and the result is recorded here with its date and Go version — including
when it finds something, since the standard library gets advisories too and
the usual fix is upgrading the toolchain. It now also runs as a CI gate on every
push and proposed merge, because dependencies grow advisories while nobody is
looking, and govulncheck reports what is *reachable* — a change in our own code
can make a dormant advisory reachable without any dependency moving.

## Findings and fixes

1. **The work id was bounded but not validated (fixed).** The handler checked
   only that the id was at most 64 bytes, then echoed it into the response. A
   traversal sequence, an ANSI escape or a NUL byte went straight back to the
   caller, and would have reached a log line or a file name the moment this
   service grew either. Ids now match a strict character class before anything
   uses them.
   Regression test: TestValidateWorkIDRejectsHostileIDs
2. **A bad listen address failed late (fixed).** `TINYSVC_ADDR` was passed to
   `ListenAndServe` unchecked, so a typo produced a pod that started, reported
   healthy and never listened. It is validated at startup now, which turns the
   typo into a rollout that fails while the old version is still serving.
   Regression test: TestLoadRejectsBadValues
3. **Startup errors were indistinguishable from bugs (fixed).** Every
   configuration failure now wraps `config.ErrInvalidValue`, so a caller can
   branch on "the operator gave us something wrong" with `errors.Is` instead of
   matching message text.
   Regression test: TestLoadRejectsBadValues

## Accepted risks

- **No authentication or authorisation on `/work`.** The service is deployed
  behind the cluster's ingress and has no data to protect; anyone who can reach
  the port can already reach the pod. The trigger that reopens this is the first
  request that returns something a caller did not supply.
- **`/metrics` is served on the same port as the work route.** A second
  listener on an internal port would be tidier, and the exposure today is the
  runtime's memory statistics and two counters — no request data, no labels
  carrying user input. We accepted the exposure rather than the extra socket.
- **No rate limiting.** One misbehaving client can use the whole replica set.
  The blast radius is degraded latency for a service that stores nothing, and
  the ingress can throttle sooner than we can. Revisit the day a request does
  work that is expensive.
- **Nothing is persisted, so there is nothing to encrypt at rest.** That follows
  from the design rather than from a security decision, and the moment storage
  lands, "what is on disk and who can read it" becomes a new section here.
