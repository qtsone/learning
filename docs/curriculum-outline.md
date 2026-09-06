# Curriculum Outline (canonical)

The definitive lesson list for every track. Registry (`skills/tutor/curriculum/registry.json`)
and content directories are derived from this outline; if they disagree, this file wins and
the others get fixed. IDs are stable slugs — never rename; insert freely (ordering is
positional in the registry's stage lists).

Verify types: `tests` (run the workspace language's test runner in the exercise dir:
`go test` for Go, `uv run pytest` for Python; shared and pack lessons only), `gotest` and
`pytest` (that runner directly; language-stage lessons only), `script` (run `./check.sh`),
`discussion` (tutor gates via conversation; no automated check).

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
| shared.cs.big-o | Big-O: Time & Space | growth classes, analyzing loops, space/time trade-offs | tests |
| shared.cs.arrays-linked-lists | Arrays & Linked Lists | contiguous vs linked memory; implement a linked list | tests |
| shared.cs.stacks-queues | Stacks & Queues | LIFO/FIFO, implementations, use cases | tests |
| shared.cs.hash-tables | Hash Tables | hashing, collisions, load factor; build a basic one | tests |
| shared.cs.recursion | Recursion | call stack, base cases, recursion vs iteration | tests |
| shared.cs.sorting | Sorting | insertion/merge/quick, stability, stdlib sort | tests |
| shared.cs.searching | Searching | linear vs binary search, implement binary search | tests |
| shared.cs.trees | Trees & BSTs | binary trees, BST invariant, traversals | tests |
| shared.cs.heaps | Heaps & Priority Queues | heap property, container/heap | tests |
| shared.cs.graphs | Graphs | adjacency list/matrix, BFS/DFS | tests |
| shared.cs.dynamic-programming | Dynamic Programming Intro | memoization, tabulation, classic problems | tests |
| shared.cs.problem-patterns | Problem-Solving Patterns | two pointers, sliding window; mixed problem set capstone | tests |

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
| shared.eng.code-organization | Code Organization | project layout, modularity, dependency direction | tests |
| shared.eng.tdd | Test-Driven Development | red-green-refactor, doubles, designing for testability | tests |
| shared.eng.debugging | Debugging | scientific method, delve/debuggers, bisecting | tests |
| shared.eng.sql-databases | SQL & Databases | relational model, SQL basics, database/sql + SQLite | tests |
| shared.eng.security | Security Fundamentals | OWASP top 10, input validation, secrets, crypto hygiene | tests |
| shared.eng.http-clients | Consuming HTTP APIs | HTTP semantics, clients, timeouts, retries, backoff | tests |
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

## S6 — Systems & Design (shared, exercises in target language) — dir `shared/s6-systems`

| id | title | hints | verify |
|----|-------|-------|--------|
| shared.systems.design-intro | System Design Intro | requirements, estimation, trade-off thinking | discussion |
| shared.systems.networking | Networking Deep Dive | TCP/UDP, TLS, HTTP/2/3, load balancing | tests |
| shared.systems.api-design | API Design | REST/gRPC trade-offs, versioning, pagination, idempotency | discussion |
| shared.systems.data-storage | Data Storage at Scale | SQL vs NoSQL, indexes, replication, sharding | discussion |
| shared.systems.caching | Caching | layers, invalidation, Redis, CDNs | tests |
| shared.systems.message-queues | Message Queues | async processing, delivery semantics, outbox | tests |
| shared.systems.distributed-systems | Distributed Systems | CAP, consistency models, consensus intro, failure modes | discussion |
| shared.systems.scalability | Scalability Patterns | statelessness, rate limiting, backpressure, sharding work | tests |
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
| focus.web.auth | Authentication | sessions, JWT, OAuth2/OIDC | tests |
| focus.web.authorization | Authorization Patterns | RBAC/ABAC, middleware enforcement | tests |
| focus.web.realtime | WebSockets & SSE | realtime transport choices, patterns | tests |
| focus.web.api-hardening | API Hardening | rate limiting, CORS, headers, validation deep | tests |
| focus.web.graphql | GraphQL in Go | schema-first, resolvers, when to choose it | tests |
| focus.web.background-jobs | Background Jobs | queues from Go, retries, idempotent consumers | tests |
| focus.web.performance | API Performance | caching, N+1, pagination, load testing | tests |
| focus.web.capstone | Web Services Capstone | extend your service: auth + realtime + jobs | tests |

## Focus pack: cli-tooling — dir `focus/cli-tooling` (Go-authored)

Insertions: all after S3.

| id | title | hints | verify |
|----|-------|-------|--------|
| focus.cli.flags-config | Flags & Configuration | flag pkg, env, config files, precedence | tests |
| focus.cli.cobra | Cobra & Subcommands | cobra command trees; viper evaluated | tests |
| focus.cli.terminal-ux | Terminal UX | colors, prompts, progress, TUI intro | tests |
| focus.cli.files-processes | Files, Processes & Signals | os/exec, signals, cross-platform | tests |
| focus.cli.distribution | Distribution | goreleaser, versioning, homebrew | script |
| focus.cli.capstone | CLI Capstone | ship a polished real tool | tests |

## Focus pack: ml — stub (Python-only, no content yet)

## P1 — Programming Basics (Python) — dir `python/p1-basics`

The Python path is `s0, p1, s2, p3, s4, p5, s6, p7`: the shared stages above alternate with
the four Python stages below. `deps` names the lessons a lesson builds on (same stage unless
prefixed); only the capstone chain is enforced as registry `prereqs`.

| id | title | hints | verify | deps |
|----|-------|-------|--------|------|
| python.basics.hello-python | Hello, Python | interpreter vs compiler, `uv run`, REPL, `__main__` guard, `ruff format` | pytest | S0 terminal-basics, dev-environment |
| python.basics.variables-types | Variables & Types | names bind objects, int/float/str/bool/None, dynamic typing, truthiness, `is` vs `==`, `Final` for constants | pytest | hello-python |
| python.basics.control-flow | Control Flow | if/elif/else, while, `for` over iterables, range/enumerate/zip, break/continue/else, `match` as switch | pytest | variables-types |
| python.basics.functions | Functions | positional/keyword/default params, `*args`/`**kwargs`, keyword-only and positional-only, tuple returns, docstrings, LEGB scope | pytest | control-flow |
| python.basics.modules-packages | Modules & Packages | import forms, module vs package, `__init__.py`, `python -m`, `pyproject.toml`, `uv init --no-package`, `uv add`, `_private`/`__all__` | pytest | functions, hello-python |
| python.basics.strings-bytes | Strings & Bytes | `str` vs `bytes`, encode/decode, `len` counts code points, indexing and slicing, f-strings and format specs, str methods, `''.join` vs `+=` | pytest | control-flow, functions |
| python.basics.lists-tuples | Lists & Tuples | indexing and slicing (slices copy), append/extend/insert/pop, amortised append, unpacking, tuple immutability, `sort` vs `sorted`, `deque` | pytest | strings-bytes, control-flow |
| python.basics.dicts-sets | Dicts & Sets | CRUD, `.get`/`in` as comma-ok, insertion order, hashable keys, `defaultdict`/`Counter`, set algebra | pytest | lists-tuples |
| python.basics.iteration-comprehensions | Iteration & Comprehensions | iterable vs iterator, `iter`/`next`, `sorted(key=)` and stability, `itemgetter`/`cmp_to_key`, min/max/any/all/sum, list/dict/set comprehensions, generator expressions | pytest | dicts-sets, functions |
| python.basics.references-mutability | References & Mutability | every name is a reference, `id`/`is`, mutate vs rebind, aliasing through arguments, `copy` vs `deepcopy`, the mutable-default trap, `None` as the null reference | pytest | lists-tuples, dicts-sets, functions |
| python.basics.classes | Classes & Methods | `class`, `__init__`, `self`, instance vs class attributes, `@property`, `@classmethod`/`@staticmethod`, `__repr__`/`__eq__`, `Node \| None` | pytest | references-mutability, functions |
| python.basics.dunder-methods | Dunder Methods | `__len__`/`__iter__`/`__contains__`/`__getitem__`, `__eq__`+`__hash__` contract, `__lt__`/`total_ordering`, `NotImplemented` and reflected operators, `__str__` vs `__repr__` | pytest | classes, lists-tuples, iteration-comprehensions |
| python.basics.dataclasses | Dataclasses & Enums | `@dataclass` fields/defaults/`default_factory`, `frozen`/`order`, `NamedTuple`, `Enum`/`StrEnum`, when a plain class is still right | pytest | classes, dunder-methods |
| python.basics.exceptions | Exceptions | hierarchy, try/except/else/finally, custom exception classes, `raise ... from`, EAFP vs LBYL, tracebacks and `breakpoint()` | pytest | classes, functions |
| python.basics.testing-basics | Testing Basics | `test_*.py` discovery, plain `assert` rewriting, `parametrize` as table tests, `pytest.raises`, `tmp_path`/`monkeypatch`/`capsys`, `-k`, coverage | pytest | modules-packages, exceptions |
| python.basics.io-files | I/O & Files | `open` with `with`, text vs binary and encoding, `pathlib`, line iteration, `json`/`csv`, stdin/stdout/`sys.argv` | pytest | exceptions, strings-bytes, dicts-sets |
| python.basics.project-cli-tracker | Mini-Project: CLI Tracker | stage capstone: `tracker/` package + `__main__.py`, `\|`-separated file, custom exceptions raised `from` the cause, parametrised tests; run with `python -m tracker` | pytest | all of P1 |

## P3 — Intermediate Python — dir `python/p3-intermediate`

| id | title | hints | verify | deps |
|----|-------|-------|--------|------|
| python.intermediate.iterators-generators | Iterators & Generators | `__iter__`/`__next__` by hand, `yield` and generator state, `yield from`, `itertools` (islice, chain, groupby, batched, pairwise), lazy pipelines over files | pytest | P1 iteration-comprehensions, dunder-methods, io-files |
| python.intermediate.closures-decorators | Closures & Decorators | closures and `nonlocal`, late binding in loops, `functools.partial`, decorators with `wraps`, decorator factories, `functools.cache` | pytest | P1 functions, references-mutability |
| python.intermediate.context-managers | Context Managers | the `with` protocol, `__enter__`/`__exit__` and exceptions, `@contextmanager`, `ExitStack`, `suppress`/`closing` | pytest | closures-decorators, iterators-generators, P1 exceptions, io-files |
| python.intermediate.protocols | Duck Typing & Protocols | duck typing, `typing.Protocol`, `runtime_checkable`, `ABC`/`@abstractmethod`, `collections.abc`, small consumer-owned protocols | pytest | P1 classes, dunder-methods |
| python.intermediate.inheritance-composition | Inheritance & Composition | `super()` and the MRO, mixins, delegation via `__getattr__`, composition over inheritance, when subclassing hurts | pytest | protocols, P1 classes |
| python.intermediate.packaging | Packaging & Project Layout | `src/` layout, `[build-system]` uv_build, `[project]` metadata, dependency groups, console scripts, `uv lock`/`uv sync --locked`, `uv build` | script | P1 modules-packages, testing-basics |
| python.intermediate.pattern-matching | Pattern Matching & Runtime Types | `isinstance` narrowing, `match` class/mapping/sequence patterns with guards, except ordering, `ExceptionGroup`/`except*`, `__cause__` chains | pytest | inheritance-composition, P1 exceptions, control-flow |
| python.intermediate.typing | Static Typing | annotations as a contract, `mypy --strict`, gradual typing, `X \| None` narrowing, `TypeIs`, `TypedDict`/`Literal`/`Self`/`@overload`, `reveal_type`, where hints pay | script | pattern-matching, protocols, packaging, P1 dataclasses |
| python.intermediate.generics | Generics | PEP 695 `def f[T]`/`class Box[T]`, bounds and Protocol constraints, `type` aliases, variance basics, `ParamSpec`, when not to | script | typing, closures-decorators, protocols |
| python.intermediate.json-encoding | JSON & Encoding | `json.dumps`/`loads`, `default`/`object_hook`, dataclass round-trips, `None` vs absent key, JSON Lines and `raw_decode` streaming, `tomllib`, why `pickle` is unsafe | pytest | P1 dataclasses, dicts-sets, io-files, typing |
| python.intermediate.time | Time | naive vs aware `datetime`, `zoneinfo`, `timedelta` arithmetic, `fromisoformat`/`strftime`, monotonic vs wall clock, injectable clocks | pytest | P1 classes, closures-decorators |
| python.intermediate.threads-gil | Threads & the GIL | `threading.Thread`/`join`, `ThreadPoolExecutor`, I/O- vs CPU-bound, what the GIL does and does not guarantee, daemon and leaked threads | pytest | closures-decorators, P1 references-mutability |
| python.intermediate.asyncio | asyncio | coroutines vs tasks, the event loop, `gather`, never block the loop, `asyncio.Queue`/`Event`, streams (TCP echo), `to_thread`, pytest-asyncio | pytest | threads-gil, iterators-generators, context-managers |
| python.intermediate.synchronization | Locks & Synchronization | `Lock`/`RLock`/`Condition`/`Semaphore`/`Event`, `asyncio.Lock`/`asyncio.Event`, check-then-act races, `functools.cache` as Once, locks vs queues | pytest | threads-gil, asyncio, context-managers |
| python.intermediate.cancellation-timeouts | Cancellation & Timeouts | `asyncio.timeout`, `Task.cancel` and `CancelledError` discipline, `shield`, `contextvars`, cooperative stop flags for threads | pytest | asyncio, synchronization |
| python.intermediate.concurrency-patterns | Concurrency Patterns | `TaskGroup`/`ExceptionGroup`, `Semaphore` bounding, queue-based worker pools, graceful drain with a deadline, `ProcessPoolExecutor` and pickling | pytest | cancellation-timeouts, synchronization, pattern-matching |
| python.intermediate.tooling | Python Tooling | `uv lock`/`sync`/groups in depth, ruff config and rule families, mypy vs ty/pyright, pre-commit, `pip-audit`, the pip/venv baseline; labs: lintlab, typelab, auditlab, layoutlab | discussion | packaging, typing, P1 modules-packages |
| python.intermediate.project-concurrent | Mini-Project: Concurrent Tool | stage capstone: async link checker (`src/` layout, console script) with bounded concurrency, per-request timeouts, SIGINT cancellation, input-order results, gate-based tests | pytest | all of P3 |

## P5 — Advanced Python — dir `python/p5-advanced`

| id | title | hints | verify | deps |
|----|-------|-------|--------|------|
| python.advanced.http-servers | HTTP Servers & ASGI | from sockets to WSGI/ASGI, a raw ASGI app, Starlette routing, middleware order, uvicorn timeouts, lifespan and graceful shutdown | pytest | P3 asyncio, concurrency-patterns, context-managers |
| python.advanced.rest-services | Building REST Services | FastAPI router/service/repository layers, Pydantic v2 validation, one exception handler for domain errors, envelopes, `Depends`, `TestClient` | pytest | http-servers, P3 typing, json-encoding, pattern-matching |
| python.advanced.grpc | gRPC | protobuf and `grpcio-tools` codegen, unary and server-streaming with `grpc.aio`, deadlines and status codes, interceptors, REST vs gRPC | pytest | rest-services, P3 iterators-generators, asyncio |
| python.advanced.databases | Databases in Production | SQLAlchemy 2.0 Core vs ORM, psycopg 3, Alembic migrations, pool configuration, transactions and rollback, parameterised queries, N+1 | pytest | rest-services, S4 sql-databases, P3 context-managers |
| python.advanced.runtime-gil | CPython Runtime & the GIL | bytecode and the eval loop (`dis`), the GIL switch interval, event-loop starvation, thread/process/task cost model, JIT status; labs: dislab, gillab, looplab, costlab | discussion | P3 threads-gil, asyncio, concurrency-patterns |
| python.advanced.memory-management | Memory Management | reference counting plus the cyclic `gc`, `__del__` and `weakref`, `tracemalloc`, `__slots__`, object sizes, reducing allocations measurably | pytest | P3 iterators-generators, P1 classes, dataclasses |
| python.advanced.free-threading | Data Races & Free-Threading | what the GIL guarantees (and not), non-atomic compound ops, happens-before via Lock/Queue/Event, Barrier-driven race tests with `sys.setswitchinterval`, `python3.14t`, per-object locks, when to opt in | pytest | P3 synchronization, concurrency-patterns, runtime-gil |
| python.advanced.profiling | Profiling & Optimization | `cProfile`/`pstats`, sampling profilers (py-spy, 3.15 Tachyon), `timeit`/pyperf benchmarks, measure-change-remeasure, flame graphs | pytest | memory-management, P3 iterators-generators, S2 big-o |
| python.advanced.extension-boundary | The Extension Boundary | buffer protocol and `memoryview`, `struct`/`array`, zero-copy slicing, which stdlib calls release the GIL, `ctypes`, Cython/mypyc, PyO3/maturin; only the buffer-protocol part is graded | pytest | profiling, memory-management, P1 strings-bytes |
| python.advanced.advanced-testing | Advanced Testing | fixture scopes and factories, `conftest.py`, `unittest.mock` and `monkeypatch`, Hypothesis property tests, golden files with `--update`, unit vs integration markers, a real temporary database | pytest | databases, P1 testing-basics, P3 protocols |
| python.advanced.metaprogramming | Descriptors, Metaclasses & Imports | descriptors and how `@property` works, `__init_subclass__` vs metaclasses, `getattr`/`annotationlib` reflection, `importlib` plugin loading, when not to | pytest | P3 inheritance-composition, closures-decorators, typing |
| python.advanced.observability | Observability | `logging.dictConfig` and structlog, correlation ids via `contextvars`, `prometheus_client` counters/histograms, OpenTelemetry traces with context propagation, cardinality | pytest | http-servers, P3 cancellation-timeouts, S4 http-clients |
| python.advanced.project-service | Mini-Project: Production Service | stage capstone: FastAPI `taskd` with token auth, SQLAlchemy + Alembic, `TestClient` suites, logs/metrics/health endpoints, graceful shutdown | pytest | all of P5 |

## P7 — Expert Capstone (Python) — dir `python/p7-capstone`

| id | title | hints | verify | deps |
|----|-------|-------|--------|------|
| python.capstone.planning | Capstone: Planning | pick and spec the project (PRD, ADRs, milestones, risk register); the only code is `uv init` | discussion | S6 design-capstone, P5 project-service |
| python.capstone.build-core | Capstone: Core Build | implement the milestones; a typed, ruff-clean, `mypy --strict` src-layout package with a green suite at every milestone; harness shells out to uv | pytest | planning |
| python.capstone.hardening | Capstone: Hardening | Hypothesis targets at input boundaries, integration tests, deterministic concurrent paths, `pip-audit`, security pass, profile under load | pytest | build-core, P5 advanced-testing, profiling, S4 security |
| python.capstone.operations | Capstone: Operations | CI with uv/ruff/mypy/pytest, containerised deploy, logs/metrics/traces, readiness and graceful shutdown during rollout, runbook | script | hardening, S4 ci-cd, P5 observability |
| python.capstone.oss-contribution | OSS Contribution | find a real issue in a Python project, navigate an unfamiliar codebase, follow its conventions, land or meaningfully review a PR | discussion | build-core, S4 code-review, documentation |
| python.capstone.performance | Performance Engineering | a measurable goal for a real bottleneck, profile-driven diagnosis, before/after pyperf comparisons, graduation review | pytest | hardening, P5 profiling, memory-management, extension-boundary |

## Shared stages on the Python path

S0, S2, S4 and S6 apply unchanged. A shared lesson needs a Python overlay exercise wherever
its verify type runs something:

- the 22 `tests` lessons: all twelve S2 lessons; S4 code-organization, tdd, debugging,
  sql-databases, security, http-clients; S6 networking, caching, message-queues, scalability
- the two `script` graders whose checks are per language: S0 dev-environment and S4 ci-cd
- S0 reading-docs, whose worksheet (a traceback to read, a docs hunt) is language-bound

Every other shared lesson takes at most `In Python:` snippets. An overlay lives at
`content/python/<stage dir>/<lesson>/`, where the stage dir is the stage's registry `dir`
(for example `content/python/shared/s2-cs/arrays-linked-lists/`) and holds `snippets/`, `exercise/` with
its `README.md` brief, `solution/`, and optional `TUTOR.md`/`quiz.json`; the layout and the
anchor grammar are in `docs/authoring-guide.md`.
