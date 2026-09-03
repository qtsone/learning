# Curriculum Outline (canonical)

The definitive lesson list for the Go path. Registry (`skills/tutor/curriculum/registry.json`)
and content directories are derived from this outline; if they disagree, this file wins and
the others get fixed. IDs are stable slugs — never rename; insert freely (ordering is
positional in the registry's stage lists).

Verify types: `gotest` (run `go test ./...` in the exercise dir), `script` (run
`./check.sh`), `discussion` (tutor gates via conversation; no automated check).

## S0 — Foundations (shared) — dir `shared/s0-foundations`

| id | title | hints | verify |
|----|-------|-------|--------|
| shared.foundations.what-is-a-program | What Is a Program? | source→machine code, compilers vs interpreters, CPU/memory mental model | discussion |
| shared.foundations.terminal-basics | The Terminal | shell, navigation, files/dirs, PATH, man pages | script |
| shared.foundations.text-editors | Editors & IDEs | VS Code setup, language servers, keyboard efficiency | discussion |
| shared.foundations.version-control-git | Version Control with Git | why VC, init/add/commit/log/diff, .gitignore | script |
| shared.foundations.git-branching-remotes | Branches & Remotes | branch/merge, GitHub, push/pull, PR flow | script |
| shared.foundations.how-the-internet-works | How the Internet Works | client/server, DNS, HTTP at a high level, request lifecycle | discussion |
| shared.foundations.reading-docs | Reading Docs & Error Messages | official docs, error-message anatomy, asking good questions | discussion |
| shared.foundations.dev-environment | Your Dev Environment | toolchain install (per-language section), env vars, project hygiene | script |

## S1 — Programming Basics in Go — dir `go/s1-basics`

| id | title | hints | verify |
|----|-------|-------|--------|
| go.basics.hello-world | Hello, Go | go run/build, main package, fmt, gofmt | gotest |
| go.basics.variables-types | Variables & Types | var, :=, basic types, zero values, const, iota | gotest |
| go.basics.control-flow | Control Flow | if/else, switch, for (all forms), break/continue | gotest |
| go.basics.functions | Functions | params, multiple returns, named returns, variadic | gotest |
| go.basics.packages-modules | Packages & Modules | go.mod, imports, exported names, package layout | gotest |
| go.basics.arrays-slices | Arrays & Slices | arrays vs slices, append, len/cap, slicing, copy | gotest |
| go.basics.maps | Maps | CRUD, comma-ok, iteration order, maps as sets | gotest |
| go.basics.structs | Structs | fields, literals, embedding intro, comparison | gotest |
| go.basics.pointers | Pointers | & and *, pass-by-value, nil, when to use pointers | gotest |
| go.basics.methods | Methods | receivers, value vs pointer receivers, method sets intro | gotest |
| go.basics.errors | Errors | error values, fmt.Errorf, %w wrapping, sentinels | gotest |
| go.basics.strings-runes | Strings, Bytes & Runes | UTF-8, runes, strings package, builders | gotest |
| go.basics.testing-basics | Testing Basics | go test, table tests, t.Run, coverage | gotest |
| go.basics.io-files | I/O & Files | os, bufio, defer, reading/writing files | gotest |
| go.basics.project-cli-tracker | Mini-Project: CLI Tracker | stage capstone: small CLI combining everything (e.g. habit/todo tracker) | gotest |

## S2 — CS Fundamentals (shared, exercises in target language) — dir `shared/s2-cs`

| id | title | hints | verify |
|----|-------|-------|--------|
| shared.cs.big-o | Big-O: Time & Space | growth classes, analyzing loops, space/time trade-offs | gotest |
| shared.cs.arrays-linked-lists | Arrays & Linked Lists | contiguous vs linked memory; implement a linked list | gotest |
| shared.cs.stacks-queues | Stacks & Queues | LIFO/FIFO, implementations, use cases | gotest |
| shared.cs.hash-tables | Hash Tables | hashing, collisions, load factor; build a basic one | gotest |
| shared.cs.recursion | Recursion | call stack, base cases, recursion vs iteration | gotest |
| shared.cs.sorting | Sorting | insertion/merge/quick, stability, stdlib sort | gotest |
| shared.cs.searching | Searching | linear vs binary search, implement binary search | gotest |
| shared.cs.trees | Trees & BSTs | binary trees, BST invariant, traversals | gotest |
| shared.cs.heaps | Heaps & Priority Queues | heap property, container/heap | gotest |
| shared.cs.graphs | Graphs | adjacency list/matrix, BFS/DFS | gotest |
| shared.cs.dynamic-programming | Dynamic Programming Intro | memoization, tabulation, classic problems | gotest |
| shared.cs.problem-patterns | Problem-Solving Patterns | two pointers, sliding window; mixed problem set capstone | gotest |

## S3 — Intermediate Go — dir `go/s3-intermediate`

| id | title | hints | verify |
|----|-------|-------|--------|
| go.intermediate.interfaces | Interfaces | implicit satisfaction, io.Reader/Writer, small interfaces | gotest |
| go.intermediate.composition | Composition & Embedding | struct/interface embedding, composition over inheritance | gotest |
| go.intermediate.type-assertions | Type Assertions & Switches | assertions, type switches, errors.Is/As | gotest |
| go.intermediate.generics | Generics | type parameters, constraints, when (not) to use | gotest |
| go.intermediate.closures | Closures & First-Class Functions | closures, options pattern, functional style limits | gotest |
| go.intermediate.stdlib-io | The io Philosophy | io/bufio/bytes/strings composition, writing adapters | gotest |
| go.intermediate.json-encoding | JSON & Encoding | struct tags, marshal/unmarshal, custom marshalers | gotest |
| go.intermediate.time | Time | time.Time/Duration, formatting, timers, monotonic clock | gotest |
| go.intermediate.goroutines | Goroutines | go keyword, WaitGroup, race detector | gotest |
| go.intermediate.channels | Channels | unbuffered/buffered, select, closing, pipelines | gotest |
| go.intermediate.sync | Sync Primitives | Mutex/RWMutex, atomic, Once; channels vs locks | gotest |
| go.intermediate.context | Context | cancellation, deadlines, propagation, values (sparingly) | gotest |
| go.intermediate.concurrency-patterns | Concurrency Patterns | worker pools, errgroup, semaphores, graceful shutdown | gotest |
| go.intermediate.tooling | Go Tooling | vet, staticcheck, golangci-lint, pprof intro, modules deep | discussion |
| go.intermediate.project-concurrent | Mini-Project: Concurrent Tool | stage capstone: e.g. parallel link checker | gotest |

## S4 — Engineering Practice (shared, exercises in target language) — dir `shared/s4-engineering`

| id | title | hints | verify |
|----|-------|-------|--------|
| shared.eng.clean-code | Clean Code | naming, function size, cohesion, comment discipline | discussion |
| shared.eng.code-organization | Code Organization | project layout, modularity, dependency direction | gotest |
| shared.eng.tdd | Test-Driven Development | red-green-refactor, doubles, designing for testability | gotest |
| shared.eng.debugging | Debugging | scientific method, delve/debuggers, bisecting | gotest |
| shared.eng.sql-databases | SQL & Databases | relational model, SQL basics, database/sql + SQLite | gotest |
| shared.eng.security | Security Fundamentals | OWASP top 10, input validation, secrets, crypto hygiene | gotest |
| shared.eng.http-clients | Consuming HTTP APIs | HTTP semantics, clients, timeouts, retries, backoff | gotest |
| shared.eng.ci-cd | CI/CD | pipelines, GitHub Actions, automating tests/lint | script |
| shared.eng.code-review | Code Review | giving/receiving reviews, PR hygiene | discussion |
| shared.eng.documentation | Documentation | READMEs, doc comments, ADRs, changelogs | discussion |

## S5 — Advanced Go — dir `go/s5-advanced`

| id | title | hints | verify |
|----|-------|-------|--------|
| go.advanced.http-servers | HTTP Servers | net/http, handlers, middleware, 1.22+ mux patterns | gotest |
| go.advanced.rest-services | Building REST Services | routing, validation, error mapping, service structure | gotest |
| go.advanced.grpc | gRPC | protobuf, services, streaming, interceptors | gotest |
| go.advanced.databases | Databases in Production | pgx, migrations, pools, transactions | gotest |
| go.advanced.runtime-scheduler | Runtime & Scheduler | GMP model, preemption, goroutine costs | discussion |
| go.advanced.garbage-collection | Garbage Collection | GC design, GOGC/GOMEMLIMIT, allocation behavior | gotest |
| go.advanced.memory-model | Memory Model & Escape Analysis | happens-before, stack vs heap, escape analysis | gotest |
| go.advanced.profiling | Profiling & Optimization | pprof, benchmarks, flame graphs, optimization loop | gotest |
| go.advanced.advanced-testing | Advanced Testing | fuzzing, integration tests, golden files | gotest |
| go.advanced.reflection-unsafe | Reflection, unsafe & codegen | reflect, unsafe, go:generate; when not to | gotest |
| go.advanced.observability | Observability | slog, metrics (Prometheus), tracing (OTel) | gotest |
| go.advanced.project-service | Mini-Project: Production Service | stage capstone: HTTP service w/ auth, DB, tests, observability | gotest |

## S6 — Systems & Design (shared) — dir `shared/s6-systems`

| id | title | hints | verify |
|----|-------|-------|--------|
| shared.systems.design-intro | System Design Intro | requirements, estimation, trade-off thinking | discussion |
| shared.systems.networking | Networking Deep Dive | TCP/UDP, TLS, HTTP/2/3, load balancing | gotest |
| shared.systems.api-design | API Design | REST/gRPC trade-offs, versioning, pagination, idempotency | discussion |
| shared.systems.data-storage | Data Storage at Scale | SQL vs NoSQL, indexes, replication, sharding | discussion |
| shared.systems.caching | Caching | layers, invalidation, Redis, CDNs | gotest |
| shared.systems.message-queues | Message Queues | async processing, delivery semantics, outbox | gotest |
| shared.systems.distributed-systems | Distributed Systems | CAP, consistency models, consensus intro, failure modes | discussion |
| shared.systems.scalability | Scalability Patterns | statelessness, rate limiting, backpressure, sharding work | gotest |
| shared.systems.reliability | Reliability & Operations | SLOs, alerting, incident response, graceful degradation | discussion |
| shared.systems.architecture | Architecture Patterns | monolith/microservices/event-driven, hexagonal | discussion |
| shared.systems.case-studies | Design Case Studies | worked examples: URL shortener, chat system | discussion |
| shared.systems.design-capstone | System Design Capstone | learner designs a system end-to-end, tutor reviews | discussion |

## S7 — Expert Capstone (Go) — dir `go/s7-capstone`

| id | title | hints | verify |
|----|-------|-------|--------|
| go.capstone.planning | Capstone: Planning | pick + spec the project (PRD, ADRs) | discussion |
| go.capstone.build-core | Capstone: Core Build | implement core; graded milestones | gotest |
| go.capstone.hardening | Capstone: Hardening | test depth, security pass, profiling | gotest |
| go.capstone.operations | Capstone: Operations | deploy, CI/CD, observability (ties in packs) | script |
| go.capstone.oss-contribution | OSS Contribution | find a real issue, land an upstream PR | discussion |
| go.capstone.performance | Performance Engineering | optimize a real bottleneck end-to-end; graduation review | gotest |

## Focus pack: containers — dir `focus/containers` (Go-authored)

Insertions: first three lessons after S4; remainder after S5.

| id | title | hints | verify |
|----|-------|-------|--------|
| focus.containers.docker-fundamentals | Docker Fundamentals | images, containers, layers, registries | script |
| focus.containers.dockerizing-go | Dockerizing Go Apps | multi-stage builds, scratch/distroless, CGO_ENABLED=0 | script |
| focus.containers.compose | Local Dev with Compose | app+db, volumes, networks | script |
| focus.containers.kubernetes-core | Kubernetes Core | pods, deployments, services, kubectl | script |
| focus.containers.kubernetes-config | Kubernetes Configuration | configmaps/secrets, probes, resources, HPA | script |
| focus.containers.deploying-go | Deploying Go on Kubernetes | manifests for your service, graceful shutdown, readiness | script |
| focus.containers.helm-kustomize | Helm & Kustomize | packaging and environment overlays | script |
| focus.containers.capstone | Containers Capstone | build→push→deploy pipeline for your service | script |

## Focus pack: web-services — dir `focus/web-services` (Go-authored)

Insertions: all after S5.

| id | title | hints | verify |
|----|-------|-------|--------|
| focus.web.auth | Authentication | sessions, JWT, OAuth2/OIDC | gotest |
| focus.web.authorization | Authorization Patterns | RBAC/ABAC, middleware enforcement | gotest |
| focus.web.realtime | WebSockets & SSE | realtime transport choices, patterns | gotest |
| focus.web.api-hardening | API Hardening | rate limiting, CORS, headers, validation deep | gotest |
| focus.web.graphql | GraphQL in Go | schema-first, resolvers, when to choose it | gotest |
| focus.web.background-jobs | Background Jobs | queues from Go, retries, idempotent consumers | gotest |
| focus.web.performance | API Performance | caching, N+1, pagination, load testing | gotest |
| focus.web.capstone | Web Services Capstone | extend your service: auth + realtime + jobs | gotest |

## Focus pack: cli-tooling — dir `focus/cli-tooling` (Go-authored)

Insertions: all after S3.

| id | title | hints | verify |
|----|-------|-------|--------|
| focus.cli.flags-config | Flags & Configuration | flag pkg, env, config files, precedence | gotest |
| focus.cli.cobra | Cobra & Subcommands | cobra command trees; viper evaluated | gotest |
| focus.cli.terminal-ux | Terminal UX | colors, prompts, progress, TUI intro | gotest |
| focus.cli.files-processes | Files, Processes & Signals | os/exec, signals, cross-platform | gotest |
| focus.cli.distribution | Distribution | goreleaser, versioning, homebrew | script |
| focus.cli.capstone | CLI Capstone | ship a polished real tool | gotest |

## Focus pack: ml — stub (Python-only, no content yet)

## Python track — registry stub only

Stages `p1-basics`, `p3-intermediate`, `p5-advanced`, `p7-capstone` mirror the Go stage
roles with representative lesson ids and no content. Shared stages (S0/S2/S4/S6) apply
unchanged with `exercises/python/` variants to be authored when the track opens.
