# Deploying Go on Kubernetes

> `focus.containers.deploying-go` · ~2.5-3.5h · Stage: Focus: Containers II

## Objectives

By the end of this lesson you can:

- Implement complete manifests — Deployment, Service, ConfigMap, probes,
  resources — for your own Go service.
- Implement graceful shutdown in Go: catch SIGTERM, drain in-flight requests
  with `http.Server.Shutdown`, and finish inside
  `terminationGracePeriodSeconds`.
- Explain the pod termination sequence (endpoint removal, SIGTERM, grace
  period, SIGKILL) and why ignoring SIGTERM drops requests.
- Implement distinct readiness and liveness endpoints in your service and
  justify what each should and should not check.
- Perform a zero-downtime rolling update and verify no requests fail,
  explaining how `maxSurge`/`maxUnavailable` and readiness gates make that
  possible.

You want a local cluster for this, and it costs a build and one command. From
inside `exercise/`:

```sh
docker build -t timesvc:1.1.0 .          # release 1.1.0, the tag the manifests pin
kind create cluster                      # ~1 minute, and kubectl now points at it
kind load docker-image timesvc:1.1.0     # kind keeps its own image store
kubectl create namespace timesvc         # unless it survived the last lesson
```

(`minikube start`, `k3d cluster create` and Docker Desktop's built-in
Kubernetes work just as well.) The exercise is graded by reading the files you
write, so you can finish it with no cluster, no kubectl and no daemon —
`check.sh` says which checks it skipped. But watching a rollout drop requests,
fixing it, and watching it stop is the lesson.

Rebuild and re-load after every change to `main.go`: the tag stays `1.1.0`, so
the cluster runs the old bits until you do, and `kubectl rollout restart
deployment/timesvc -n timesvc` then picks up the new ones.

## Your process is disposable, and that is the deal

The last two lessons described a desired state. This one is about the part of
the contract that runs in *your* code, because there is one thing Kubernetes
cannot do for you: stop your process politely.

And it will stop it — far more often than "when we deploy". A rolling update,
a node drained for a kernel patch, a scale-down, an eviction because a
neighbour is using too much memory, an autoscaler consolidating three
half-empty nodes into two. The twelve-factor name for the property you need is
**disposability**: start fast, stop cleanly, lose nothing on the way out.

The mechanism is a Unix signal. The kubelet sends **SIGTERM** to PID 1 in your
container and starts a clock; when the clock runs out it sends **SIGKILL**,
which no process can catch, block or handle. Go's default disposition for
SIGTERM is to terminate immediately, so a service that installs no handler
dies at the first byte of the request it was serving. (This is also why your
`ENTRYPOINT` is in exec form: with shell form PID 1 would be `/bin/sh`, which
does not forward signals, and your code would never see SIGTERM at all.)

## The termination sequence, exactly

Something deletes the pod — `kubectl delete pod`, a rolling update, a node
drain. What actually happens is not what most people assume:

1. The API server sets `deletionTimestamp` on the pod, which now shows as
   `Terminating`. **The grace-period clock starts now.**
2. Two chains of events start **in parallel**, and nothing synchronises them:

   **(a) The control plane.** The endpoints controller notices the pod is
   terminating, marks it not-ready in the Service's EndpointSlice, and every
   `kube-proxy` in the cluster — plus your ingress controller, plus any
   service mesh — has to observe that change and rewrite its forwarding rules.
   That propagation is eventually consistent and takes anywhere from
   milliseconds to seconds depending on cluster size.

   **(b) The kubelet on that node.** It runs the container's `preStop` hook if
   there is one, waits for it to finish, and *then* sends SIGTERM to PID 1.

3. Your process handles SIGTERM: stops advertising readiness, stops accepting
   new connections, finishes the requests it already accepted, exits.
4. If the container is still running when `terminationGracePeriodSeconds`
   expires — counted from step 1, including the preStop hook — the kubelet
   sends SIGKILL. Whatever was in flight is gone.
5. The pod object is removed from the API.

Read step 2 again. Nobody tells the kubelet "the dataplane has finished
forgetting this pod, you may now send SIGTERM". The two chains simply run.

## Why a perfectly healthy pod still drops requests

Now the failure mode writes itself. Chain (b) is fast — the kubelet is local,
SIGTERM arrives in milliseconds. Chain (a) is slower and unbounded. So your
server closes its listener while a `kube-proxy` somewhere is still forwarding
brand-new connections to your pod's address, and those callers get
`connection refused` or a reset: failures on requests that were never served,
arriving in a burst exactly when you deploy, indistinguishable from a real
outage to whoever is calling you.

The fix is to make the fast chain wait for the slow one, and it has two parts.

**Fail readiness first.** The moment shutdown begins, `/readyz` answers 503.
Anything that polls readiness — an ingress controller, a cloud load balancer
with its own health checks, a mesh sidecar — takes you out of rotation on its
own schedule.

**Then delay SIGTERM with a preStop hook.** Readiness alone does not cover the
deletion path, because the kubelet does not wait for readiness to be observed
either. So you buy time bluntly:

```yaml
lifecycle:
  preStop:
    sleep:
      seconds: 10
```

The *hook* sleeps; your process keeps serving normally while the endpoint
removal propagates, and only afterwards does SIGTERM arrive. That is the whole
trick, and it is why almost every production Deployment you will read has an
apparently pointless sleep in it.

That `sleep:` handler is native to Kubernetes (added in 1.29, on by default
since 1.30) and needs nothing inside the image. The older spelling,
`exec: command: ["/bin/sh", "-c", "sleep 10"]`, fails on the distroless image
you built two lessons ago — no shell, no `/bin/sleep`. On an older cluster
that is a real constraint on your base image choice.

**The ordering rule, in one sentence:** the pod must stop being a routing
target *before* it stops accepting connections. Everything above is machinery
for that one sentence.

## Draining in Go

The Go side is small, and every line is load-bearing:

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

// ... serve in a goroutine ...

<-ctx.Done()          // SIGTERM arrived

a.ready.Store(false)  // 1. stop advertising: /readyz now answers 503
stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
defer cancel()
err := srv.Shutdown(stopCtx)   // 2. stop accepting, finish what is in flight
```

- `signal.NotifyContext` turns a signal into a cancelled context — the
  vocabulary the rest of your program already speaks. It also restores the
  default behaviour on the *second* signal, so an impatient operator pressing
  Ctrl-C twice still gets out.
- **`Shutdown` is not `Close`.** `Close` slams every connection shut, requests
  being served right now included. `Shutdown` closes the listeners and idle
  keep-alive connections, then waits for active requests. The difference is a
  burst of failed requests at every deploy.
- The shutdown context needs a deadline, bounded *below* the grace period:
  without one a single stuck handler holds the drain open until SIGKILL and
  takes everything nearly finished down with it.
- `srv.Serve` returns `http.ErrServerClosed` as soon as shutdown begins — what
  success looks like. Treat it as an error and your pod exits non-zero and is
  counted as a crash.
- `Shutdown` does not wait for **hijacked** connections (WebSockets, raw
  streams); close those yourself, e.g. from `srv.RegisterOnShutdown`.
  Background goroutines hang off the same context, and `main` waits for them.

## Budgeting the grace period

Three numbers have to fit together, and this is the arithmetic to write down
in a comment for whoever inherits the file:

```
preStop sleep  +  the server's shutdown timeout  +  margin  <=  terminationGracePeriodSeconds
      10s      +               20s               +   15s    <=              45s
```

The mistake people make is thinking the preStop hook runs *before* the grace
period. It does not — the clock starts at deletion, so the hook spends the
same budget the drain needs. Get this wrong and your service does everything
right and still gets SIGKILLed mid-drain, with no log line explaining why,
which reads like a crash rather than a deploy.

`terminationGracePeriodSeconds` defaults to 30. Set it explicitly anyway: 30
is a fine number and a terrible decision, because nobody reading the file can
tell whether anyone checked it against the drain.

## Liveness during a drain

You wired all three probes in the previous lesson. Deployment adds two things
they must *not* do.

**Liveness must keep passing for the whole drain.** Point it at the same
endpoint as readiness and the shutdown sequence turns comic: readiness flips
to 503, the liveness probe reads that same 503, the kubelet concludes the
container is dead and restarts it in the middle of finishing real requests.
That is why your service has two endpoints that look identical and mean
different things — `/healthz` is 200 for as long as the process can serve at
all, `/readyz` reports whether it wants new work.

**Liveness must not check dependencies.** A liveness probe that pings the
database turns a five-minute database blip into a cluster-wide restart storm:
every pod fails liveness at once, every pod restarts, all of them come back
with cold pools and empty caches and hammer the database that was already
struggling. A restart cannot fix somebody else's outage. Dependency checks
belong in readiness, where the consequence is "send me no traffic for now".

## GOMAXPROCS and the CPU limit

Now the two Go-specific traps, both the same shape: the runtime tunes itself
from what it can see, and inside a container what it can see is wrong.

`GOMAXPROCS` sets how many goroutines can execute simultaneously — the number
of Ps in the scheduler from the S5 runtime lesson. Through Go 1.24, the
runtime picks it from the CPU affinity mask: the *node's* cores. Your
container's `limits.cpu: 1` is a cgroup bandwidth quota — 100ms of CPU per
100ms period — and it is invisible to that calculation. On a 64-core node that
means 64 Ps sharing one core's worth of quota:

- The scheduler runs 64 goroutines at once, blows the quota partway through
  the period, and the kernel **throttles the whole process** until the next
  period. Your p99 grows a shelf exactly one period wide.
- The GC dedicates about 25% of `GOMAXPROCS` to mark workers — 16 of them —
  competing for that same single core's budget.
- Every P carries per-P caches, so you pay memory for parallelism you cannot
  use.

Nothing crashes. It is slow in a way that profiling *inside* the process
struggles to explain, because the process cannot see the throttling.

Go 1.25 fixed the default on Linux: the runtime reads the cgroup CPU limit and
takes the minimum of the affinity mask and that limit rounded up (floor of 2),
re-checking periodically in case it changes. But it is gated on the `go`
directive in `go.mod` being 1.25 or later — the standard GODEBUG compatibility
mechanism — so a module still declaring `go 1.22`, like the exercise, gets the
old behaviour even on a brand-new toolchain. Three honest options:

1. **Set it in the manifest**, next to the limit it has to agree with:

   ```yaml
   env:
     - name: GOMAXPROCS
       valueFrom:
         resourceFieldRef:
           resource: limits.cpu     # rounds up: 1500m becomes 2
   ```

   The downward API keeps the two from drifting apart when someone raises the
   limit six months from now.
2. **Import `go.uber.org/automaxprocs`** — a blank import that reads the
   cgroup at startup, the library everyone used before 1.25 — or move the
   module to `go 1.25`+ and let the runtime do it.

The exercise asks for option 1: it works on every Go version, and it is the
one visible to whoever reads the manifest.

## GOMEMLIMIT and the memory limit

The same story with a nastier ending. The collector's default target is a
*ratio*: with `GOGC=100` the heap may grow to twice the live heap before a
collection. Nothing in that formula knows about `limits.memory`, so a Go
service under load grows past the cgroup limit and the kernel's OOM killer
ends the process — exit code 137, `OOMKilled` in `kubectl describe`, no panic,
no stack trace, no shutdown, in-flight requests simply gone.

`GOMEMLIMIT` gives the runtime the number it was missing. It is a **soft**
limit: as runtime-managed memory approaches it, the GC runs more often and
returns memory to the OS more aggressively, trading CPU for staying alive. The
cgroup limit stays hard and kernel-enforced, so the soft one sits *below* it:

```yaml
resources:
  limits:
    memory: 128Mi
env:
  - name: GOMEMLIMIT
    value: "110MiB"
```

Three details that bite:

- **The suffix.** Kubernetes writes `128Mi`; the Go runtime wants `128MiB`
  (`B`, `KiB`, `MiB`, `GiB`, `TiB`). A value it cannot parse makes the runtime
  refuse to start.
- **The headroom is not optional.** `GOMEMLIMIT` covers the heap and other
  runtime-managed memory, but explicitly *excludes* the mapping of your own
  binary and anything the OS holds on your behalf. 80-90% of the container
  limit is the usual landing zone.
- **`GOGC=off` with `GOMEMLIMIT` is a real strategy and a real risk** — if the
  live heap genuinely exceeds the limit, the GC spins forever chasing a target
  it cannot reach. Leave `GOGC` alone until profiles say otherwise.

## Config and logs are part of the deployment

**Configuration comes from the environment.** Your process reads `PORT`,
`LOG_LEVEL` and `SHUTDOWN_TIMEOUT` at startup and never again. Env vars from
`envFrom` are snapshotted at container creation, so editing the ConfigMap does
not change a running pod — you roll the Deployment, which produces new pods
with the new values *and* a revision you can roll back to. (Mounted config
updates in place instead; env vars are choosing restart-to-change.)

**Logs go to stdout as structured lines.** The container's stdout *is* the
logging interface: the kubelet captures it, `kubectl logs` reads it, the
cluster's collector ships it. Your `slog` JSON handler from the observability
lesson is already right; a log file inside the container is invisible to all
of that and dies with the pod.

## Rolling updates that nobody notices

A rolling update is the termination sequence above, run once per pod, while
new pods start alongside. You already set the strategy in the core lesson:

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1          # may run one pod above the replica count
    maxUnavailable: 0    # may never run below it
minReadySeconds: 5       # ready for 5s before it counts as available
```

What is worth adding now is *why* it works. `maxUnavailable: 0` means capacity
is never removed before a replacement is ready — and "ready" is your readiness
probe, so the whole strategy is only as good as that endpoint. Each old pod
then leaves through the drain you just built, which is the other half: a
perfect strategy over a process that ignores SIGTERM still drops requests, one
burst per pod. (`maxSurge: 0` with `maxUnavailable: 0` is the deadlock —
allowed neither to add a pod nor to remove one, the rollout never starts.
`minReadySeconds` covers the pod that passes readiness and falls over two
seconds later.)

The vocabulary you already have does the rest:

```sh
kubectl apply -f . -n timesvc
kubectl rollout status deployment/timesvc -n timesvc   # blocks; non-zero on failure
kubectl rollout undo deployment/timesvc -n timesvc     # back to the previous revision
```

And to *verify* zero downtime rather than believe in it, hold a request loop
against the Service while you roll — from inside the cluster, since it is a
ClusterIP:

```sh
kubectl run probe --rm -it -n timesvc --image=curlimages/curl --restart=Never -- \
  sh -c 'while true; do curl -sS -o /dev/null -w "%{http_code}\n" http://timesvc/now; done'
```

A wall of `200`s is the result you want. A `000` or a `503` is the race in this
lesson, and it is worth deliberately breaking your manifest — remove the
preStop hook, roll again — to watch it appear.

## Exercise

Open [`exercise/`](exercise/). You get `timesvc` back with its lifecycle
removed, and your manifests roughly as they stood at the end of the Kubernetes
Configuration lesson — trimmed to ConfigMap, Deployment and Service, in the
`timesvc` namespace. They schedule, they serve, they pass their probes, and
they drop requests every time a pod moves.

- `main.go` — four `TODO`s: readiness, liveness, the drain, and hearing
  SIGTERM at all. `shutdown_test.go` is their specification; read it.
- `configmap.yaml`, `deployment.yaml`, `service.yaml` — a `TODO` at every place
  this lesson touches.
- `SHUTDOWN.md` — four questions to answer in your own words.
- `Dockerfile` — the 1.1.0 build, as you wrote it two lessons ago; ungraded
  here, and the reason the tag the manifests pin is one command away.
- `check.sh` — the referee.

Acceptance criteria (1-3 and 8-15 are the work; 4-7 are already true and must
stay true):

1. The service turns SIGTERM (and SIGINT) into a cancelled context.
2. Shutdown withdraws readiness **first** — the flip appears before the
   `Shutdown` call in your source — then calls `srv.Shutdown` with a context
   bounded by `SHUTDOWN_TIMEOUT`, and does not treat `http.ErrServerClosed`
   as a failure.
3. `/readyz` answers 503 while draining; `/healthz` answers 200 for the whole
   drain and reads no readiness state and no dependency. Keep it named
   `healthz` (method or plain function): that name is how the referee finds it.
4. The manifests declare a ConfigMap, a Deployment (`apps/v1`) and a Service
   (`v1`), and the Service's selector matches the pod template's labels.
5. The container image is pinned to a real tag or digest — never `latest`.
6. `replicas` is at least 2, with `strategy.type: RollingUpdate`,
   `maxUnavailable: 0` and `maxSurge` at least 1.
7. The readiness probe hits `/readyz`; the liveness probe hits `/healthz`; a
   startup probe budgets at least 30 seconds
   (`failureThreshold × periodSeconds`). Every probe port resolves to a
   declared `containerPort` — by name or by number.
8. `terminationGracePeriodSeconds` is set explicitly and is greater than zero.
9. A `preStop` hook sleeps at least 5 seconds (the native `sleep:` handler, or
   an `exec` that sleeps).
10. The three numbers add up: `preStop + SHUTDOWN_TIMEOUT + 5s margin <=
    terminationGracePeriodSeconds`.
11. The ConfigMap supplies `PORT`, `LOG_LEVEL` and `SHUTDOWN_TIMEOUT` (a valid
    Go duration) and the container pulls them in — `envFrom.configMapRef` or
    per-key `configMapKeyRef`.
12. `resources` declares both `requests` and `limits` for cpu and memory, with
    each limit at least its request.
13. `GOMAXPROCS` is set from the CPU limit — a literal equal to the limit
    rounded up to whole cores, or `resourceFieldRef` on `limits.cpu`.
14. `GOMEMLIMIT` is set as a literal in Go's syntax, at least 50% and strictly
    less than `limits.memory`.
15. `SHUTDOWN.md` answers all four questions, names the sequence (SIGTERM,
    `terminationGracePeriodSeconds`, SIGKILL, preStop, endpoints), and states
    the ordering requirement explicitly in one sentence.

Run the referee from inside `exercise/`:

```sh
bash check.sh
```

`check.sh` reads every `.yaml` file in the folder with its own small YAML
reader, because PyYAML is not guaranteed on your machine. Keep the manifests
inside the subset it understands — also the subset every `kubectl` example
uses: **spaces only, never a tab; two spaces per level; a space after every
colon and after every `- `; plain block mappings (flow style `[a, b]` /
`{a: b}` is fine for short values); no anchors (`&`), aliases (`*`) or merge
keys (`<<:`).** A sequence may sit at its key's indentation or one level in,
and several objects may share one file if you separate them with a line
containing only `---`. If the reader ever rejects legal YAML, that is an
exercise bug, not yours — tell your tutor.

The grade is static: your manifests, your Go source and `SHUTDOWN.md`, with no
cluster, no kubectl, no Docker and no network. A Go toolchain unlocks
`shutdown_test.go`; `kubectl` unlocks a client-side dry run, and a reachable
cluster a server-side one. Anything missing is `skip`, never `FAIL`.

## Further reading

- [Kubernetes — Pod lifecycle: termination](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination)
  and [Container lifecycle hooks](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/)
- [pkg.go.dev — `http.Server.Shutdown`](https://pkg.go.dev/net/http#Server.Shutdown)
  and [`signal.NotifyContext`](https://pkg.go.dev/os/signal#NotifyContext)
- [go.dev — A Guide to the Go Garbage Collector](https://go.dev/doc/gc-guide)
  — the `GOMEMLIMIT` sections, including the death-spiral warning.
- [Kubernetes — Deployment rolling updates](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment)
