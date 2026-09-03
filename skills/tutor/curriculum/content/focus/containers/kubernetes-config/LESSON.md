# Kubernetes Configuration

> `focus.containers.kubernetes-config` · ~2.5-3.5h · Stage: Focus: Containers II

## Objectives

By the end of this lesson you can:

- Choose between a ConfigMap and a Secret for a given piece of configuration
  and justify the choice, including whether it arrives as an environment
  variable or as a mounted file.
- Implement liveness, readiness and startup probes for a service and explain
  the distinct consequence of each one failing.
- Explain the difference between resource requests and limits, and how they
  drive scheduling, CPU throttling and OOM kills.
- Implement a HorizontalPodAutoscaler and explain how it uses the CPU request
  to compute a utilization target.
- Explain why Kubernetes Secrets are only base64-encoded by default and what
  that implies for protecting them.

Last lesson your Deployment ran and your Service routed to it. That is a demo.
This lesson is the difference between a demo and something you would put a
pager on: the platform still knows nothing about how your process is
configured, whether it is healthy, or how much of a node it needs — so it
guesses, and it guesses wrong at the worst possible moment.

**Install a local cluster if you have not.** Reading these objects is not the
same as watching a kubelet restart a container in front of you. One command
with [kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker, on the Docker you
already have):

```sh
kind create cluster --name tutor      # then: kubectl get nodes
```

minikube, k3d and Docker Desktop's built-in cluster are equally fine. Either
way `check.sh` grades the manifests you write by reading them, so you can
finish this lesson on a plane.

## Configuration is not part of the image

The image you built earlier in this pack is immutable and content-addressed:
the same digest is the same bytes in staging and in production. That property
dies the moment a hostname or a log level is baked into it, so everything that
differs between environments has to arrive from outside the image.

Kubernetes gives you two objects for that, and they are the same object with a
different name on it: **ConfigMap** for non-secret key/value data, **Secret**
for the rest. Both are namespaced (a pod can only reference one in its own
namespace) and both cap at roughly 1 MiB, because they live in etcd and etcd is
not a filesystem. Both also come in two shapes at once, which is the part
people miss:

```yaml
data:
  LOG_LEVEL: "info"          # env-shaped: a name and a scalar
  features.yaml: |           # file-shaped: a name that is a filename,
    now:                     # and a body that is a whole file
      default_zone: UTC
```

Nothing in the object marks one as "an env var" and the other as "a file".
That is decided entirely by how the *pod* consumes it.

## Secrets, honestly

A Secret's `data` field is base64. Base64 is an encoding, not encryption: it
exists so arbitrary bytes survive a YAML field, and it is reversed by
`kubectl get secret timesvc-secrets -o jsonpath='{.data.API_KEY}' | base64 -d`
— no key, no password, no audit trail.

Writing `stringData:` instead lets you put plain text in your manifest and have
the API server encode it for you — which tells you exactly how much protection
that encoding is meant to provide. What you *do* get from a Secret is real, but
narrow: a separate RBAC verb (`get secrets` is granted deliberately, and plenty
of roles that read ConfigMaps cannot read Secrets), less accidental exposure
(`kubectl describe configmap` prints the data verbatim, while `kubectl describe
secret` prints only `API_KEY: 24 bytes` — it takes `kubectl get secret -o yaml`
to see even the base64), and node-side handling in a tmpfs — memory, never the
node's disk.

What you do not get, by default:

- **Encryption at rest is opt-in.** Unless the operator configured an
  `EncryptionConfiguration` (ideally KMS-backed), the value sits in etcd in the
  clear, and an etcd backup is a credential dump.
- **Anyone with `get secrets` in the namespace has the plaintext** — including
  every workload whose ServiceAccount you gave that permission to.
- **Nothing keeps it out of git.** A Secret manifest with a real value is a
  committed credential, and git remembers forever (S4's security lesson).

The real answers are narrow RBAC, encryption at rest, and a tool that keeps the
value out of the repository entirely: SOPS or Sealed Secrets (commit an
encrypted blob only the cluster can open), or External Secrets (commit a
*reference*; an operator fetches from a real secret manager).

## Two routes in: environment variables and files

The same ConfigMap can reach the container both ways — on the container, then
the volume it refers to on the pod spec:

```yaml
env:                                    # one key, by name
  - name: LOG_LEVEL
    valueFrom:
      configMapKeyRef: { name: timesvc-config, key: LOG_LEVEL }
envFrom:                                # or: every key, at once
  - configMapRef: { name: timesvc-config }
volumeMounts:                           # keys become files
  - name: features
    mountPath: /etc/timesvc/config

# ...and one level up, on the pod spec rather than the container:
volumes:
  - name: features
    configMap:
      name: timesvc-config
      items:                            # optional: pick which keys
        - key: features.yaml
          path: features.yaml
```

The trade-off is not style. It is *when the value can change*:

| | Environment variable | Mounted file |
|---|---|---|
| Read | once, at `exec` | whenever your code reads it |
| Update after editing the object | never, until pods are replaced | the kubelet refreshes the volume, typically within a minute |
| Visible in | `kubectl describe pod`, `/proc/<pid>/environ`, child processes, crash dumps | the mount only |
| Good for | scalars your process reads at start-up | whole config files, large values, secrets |

Three consequences worth burning in:

- **Editing a ConfigMap does not restart anything.** The Deployment did not
  change, so no rollout happens and pods keep the environment they were exec'd
  with. Forcing a rollout on a config change means changing the pod template —
  usually an annotation holding a hash of the config, which you will meet again
  when tooling does it for you later in this pack.
- **A `subPath` mount never updates.** It is copied once at container start.
  Mounting a whole directory is what buys you live updates.
- **`envFrom` imports every key, including the file-shaped ones.**
  `features.yaml` is not a legal environment variable name, so the kubelet
  skips it and records an event saying so — visible in `describe`, invisible
  otherwise. Naming keys individually avoids it; so does keeping file-shaped
  keys in their own ConfigMap.

One more: a pod that references a ConfigMap or Secret which does not exist does
not start — it sits in `CreateContainerConfigError` until you create it, and
`describe` names the missing object. Describe and events first, as always.

## Probes: three questions, three different consequences

The kubelet on the node runs your probes. Each answers a different question and
each has a different remedy; mixing them up is the most expensive
configuration mistake in this lesson.

| Probe | Question | Consequence of failing |
|---|---|---|
| `readinessProbe` | Should this pod receive traffic *right now*? | removed from the Service's endpoints; **not** restarted |
| `livenessProbe` | Is this container beyond saving? | the kubelet kills the container; it restarts |
| `startupProbe` | Has it finished booting? | the container is killed; while it runs, the other two are suspended |

A probe handler is one of `httpGet` (2xx/3xx passes), `tcpSocket` (a connection
opens), `exec` (a command exits 0) or `grpc`. The tuning fields have defaults
you should know because they are decisions whether or not you make them:

| Field | Default | What it costs you |
|---|---|---|
| `initialDelaySeconds` | 0 | too small: restart loops on a slow boot |
| `periodSeconds` | 10 | how quickly anything is noticed |
| `timeoutSeconds` | **1** | a probe that is slow under load counts as failed |
| `failureThreshold` | 3 | period x threshold is your real detection time |
| `successThreshold` | 1 (liveness/startup: fixed at 1) | flapping back in too eagerly |

Liveness with the defaults means 30 seconds of a wedged process before anything
happens — and a 1-second timeout means a GC pause or a throttled CPU can fail
the probe on a perfectly healthy service. State these numbers explicitly.

### The mistake: a liveness probe that checks the database

`/readyz` checks the things the service cannot serve without. Pointing the
*liveness* probe at it looks like thoroughness. Here is what it buys you when
the database is unreachable for forty seconds:

1. Every replica fails liveness at the same moment — they share the database,
   so this is simultaneous, not staggered.
2. After `failureThreshold` periods the kubelet kills every container. At once.
3. They restart into the same broken dependency, fail again, and back off into
   `CrashLoopBackOff` — the backoff is exponential up to five minutes, so your
   recovery is now slower than the outage.
4. Warm caches and connection pools are gone, and when the database returns,
   every replica reconnects at the same instant into a database that has just
   come back up.

With readiness doing that job instead, the pods stay up, quietly leave the
Service's endpoint list, callers get a fast honest failure, and the moment the
database is back the same warm processes are added back. Nothing was restarted.

**Liveness answers exactly one question: would a restart help?** A wedged event
loop, a deadlock, a leaked-goroutine death spiral — yes. A dependency being
down — no. So liveness gets an endpoint that touches nothing outside the
process. The mirror-image mistake is no readiness probe at all: a pod is then
"ready" the instant the process starts, so a rolling update happily sends live
traffic to a container that is still building its cache.

A **startup probe** exists because otherwise a slow boot forces you to inflate
`initialDelaySeconds` on the liveness probe to cover the worst start-up you
have ever had — and for that whole window nothing supervises liveness at all.
It separates the two budgets: `failureThreshold x periodSeconds` is "how slow
may booting legitimately be", the other two probes are suspended while it runs,
and once it passes it never runs again.

## Requests and limits: two numbers with two audiences

```yaml
resources:
  requests: { cpu: 100m, memory: 64Mi }   # scheduler
  limits:   { cpu: 500m, memory: 128Mi }  # kernel
```

**Requests are for the scheduler**, which does arithmetic, not measurement: it
sums the requests already on a node and asks whether yours still fits. A node
whose pods request 100% of its CPU is full even while idle, and a pod
requesting more than any node has stays `Pending` forever with an "Insufficient
cpu" event. The request is also what the HPA divides by, so a wrong one quietly
corrupts autoscaling.

**Limits are for the kernel**, enforced by cgroups, and the two resources
behave completely differently:

- **CPU is compressible.** The limit becomes a CFS quota: with `cpu: 500m` the
  container may use 50 ms of CPU per 100 ms period, and when it runs out it is
  simply not scheduled until the next one. Nothing dies; requests just take
  100 ms longer. That is the classic invisible latency bug — p99 through the
  roof, average CPU at 30%, nothing in the logs, and
  `container_cpu_cfs_throttled_periods_total` the only place it shows.
- **Memory is incompressible.** You cannot give a process 90% of a byte. Cross
  the memory limit and the kernel's OOM killer terminates it; the container
  restarts with reason `OOMKilled` and exit code 137. A limit that is too tight
  turns a traffic spike into a restart loop.

Together the two numbers also assign a **QoS class**, which decides who dies
first when the node itself runs out of memory: **Guaranteed** (limits equal
requests, cpu and memory, every container) is evicted last, **Burstable** (a
request somewhere) is evicted by how far over its requests it is, and
**BestEffort** (nothing set) is evicted first — which is why "it worked until
someone else deployed" is a thing.

Practical guidance: keep the memory request and limit equal or close, and size
both from real measurements rather than optimism — that is what your S5
profiling work was for. CPU is the contested one: a limit is the only thing
stopping one workload from starving its neighbours, and it is also a reliable
way to add throttling latency to a service that is behaving perfectly. Plenty
of good teams set CPU requests and no CPU limit. This exercise asks for one
anyway, generous enough that normal work never reaches it, because there is a
second reason to care about a CPU limit in a *Go* service — and that is the
next lesson's subject.

## The HorizontalPodAutoscaler

An HPA is another control loop. Every 15 seconds it reads pod CPU (from
metrics-server, an add-on a fresh kind cluster does not have) and computes
`desiredReplicas = ceil(currentReplicas x currentMetric / targetMetric)`.

With `averageUtilization: 70` and a request of `100m`, `targetMetric` is 70m of
CPU per pod, so six pods averaging 105m give `ceil(6 x 105/70)` = 9 replicas.
Which is the whole reason the request matters: **utilization is a percentage of
the request, not of the node**. No CPU request, no denominator — the HPA
reports `<unknown>` and never scales.

Three things that surprise people:

- **Scaling down is deliberately slow.** A five-minute stabilization window by
  default, so a brief dip does not throw away capacity you are about to need.
  Scaling up is immediate.
- **It fights your git repository.** The HPA writes `spec.replicas`; if your
  manifest also declares `replicas: 2` and something re-applies it every few
  minutes, the two controllers take turns undoing each other. Keep the
  committed number inside the HPA's range and know that the HPA owns it after
  the first apply.
- **CPU is a proxy, not the truth.** If the bottleneck is the database, more
  replicas make it worse. HPAs can also target memory, custom or external
  metrics (queue depth is the honest signal for a worker, and KEDA is the usual
  tool); a VerticalPodAutoscaler resizes pods instead of multiplying them. Out
  of scope here — worth knowing the names.

## Exercise

Open [`exercise/`](exercise/). You are given:

- `app/main.go` — the service, complete, **do not change it**. Its doc comment
  is the contract you must match: `/healthz` (process-local), `/readyz`
  (dependency-aware, after a two-second warm-up), `/now` (needs `X-API-Key`),
  configuration from `LOG_LEVEL`, `NOW_CACHE_TTL`, `API_KEY` and `CONFIG_DIR`,
  and `$CONFIG_DIR/features.yaml` re-read on every request. `app/Dockerfile`
  is the build from Dockerizing Go, unchanged and not graded — it is there so
  you can produce the image the manifests pin.
- `namespace.yaml`, `service.yaml` and `deployment.yaml` — carried over from
  Kubernetes Core. The first two are finished; the Deployment is where most of
  the TODOs live. `configmap.yaml`, `secret.yaml` and `hpa.yaml` are skeletons.
- `NOTES.md` — three questions to answer in your own words — and `check.sh`,
  the referee.

Acceptance criteria:

1. Every object declares `apiVersion`, `kind` and `metadata.name`, and every
   object lives in the same namespace (`timesvc`, as carried over).
2. A ConfigMap holds `LOG_LEVEL` and `NOW_CACHE_TTL`.
3. The same ConfigMap holds a file-shaped key `features.yaml` with a few lines
   of YAML in a block scalar.
4. The container receives `LOG_LEVEL` and `NOW_CACHE_TTL` from that ConfigMap
   (`envFrom`, or per-key `configMapKeyRef`).
5. `features.yaml` is mounted as a file at `/etc/timesvc/config` — a volume on
   the pod and a `volumeMount` on the container.
6. A Secret carries `API_KEY`, with a value you invented (not `change-me`).
7. The container gets `API_KEY` **by reference** (`secretKeyRef`, `envFrom`
   `secretRef`, or a secret volume) — the value never appears in the Deployment.
8. A `readinessProbe` does an `httpGet` on `/readyz`, on the container's port
   (by name or number).
9. A `livenessProbe` does an `httpGet` on `/healthz` — and, explicitly, is not
   pointed at the readiness path.
10. A `startupProbe` on the same cheap endpoint, with
    `periodSeconds x failureThreshold` of at least 30 seconds.
11. The liveness and readiness probes state their own `periodSeconds` and
    `failureThreshold` rather than inheriting the defaults.
12. The container declares `requests.cpu`, `requests.memory`, `limits.cpu` and
    `limits.memory`, and every limit is at least its request.
13. An `autoscaling/v2` HPA targets the Deployment by name, with `minReplicas`
    >= 2, a larger `maxReplicas`, and a `Resource` metric on `cpu` with
    `target.type: Utilization` and a sane percentage — and the Deployment's
    `replicas` sits inside that range.
14. `NOTES.md` answers all three questions.

Then run the referee from inside `exercise/` — `bash check.sh`. It grades the
manifests by parsing them (**no cluster, no kubectl, no Docker, no network and
no root**), so it is the grade everywhere. Expect it to fail on the starter;
that is the point.

**How the referee reads your YAML.** PyYAML is not guaranteed to exist on your
machine, so `check.sh` carries a small parser covering the subset these
manifests need: block mappings, block lists (indented under their key or at the
key's own indentation, both fine), `|` block scalars, quoted and unquoted
scalars, simple inline `[a, b]` and `{k: v}`, and `---` between documents.
Indent with spaces only — never tabs — keep a space after every colon, and do
not use anchors or aliases (`&name`, `*name`). If it cannot read something it
says which file and line, rather than shrugging.

With a cluster running you also get two dry-run checks, one client side and one
against the real API server (which validates without creating anything);
without kubectl or a cluster they are `skip`, never `FAIL`.

Then do this by hand, because it is where the understanding lands. It starts
with a build: the manifests pin `timesvc:1.2.0`, not the `1.0.0` you built in
Dockerizing Go, because `app/main.go` has gained `/readyz`, `API_KEY` and
`CONFIG_DIR` since then. Different bytes, different tag — that is the rule this
whole pack rests on, and probing `/readyz` against the old image would only get
you a 404 and a pod that is never ready.

```sh
docker build -t timesvc:1.2.0 app/
kind load docker-image timesvc:1.2.0 --name tutor   # kind keeps its own image store
kubectl apply -f .
kubectl -n timesvc describe pod -l app=timesvc   # probe results and events
kubectl -n timesvc get pods -w                   # readiness flips after warm-up
kubectl -n timesvc edit configmap timesvc-config # change LOG_LEVEL, then:
kubectl -n timesvc exec deploy/timesvc -- env | grep LOG_LEVEL   # unchanged. Why?
```

## Further reading

- [ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/)
  and [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) —
  including the "Risks" section of the Secret page.
- [Configure liveness, readiness and startup probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Resource management for pods and containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
  and [QoS classes](https://kubernetes.io/docs/concepts/workloads/pods/pod-qos/)
- [Horizontal Pod Autoscaling](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
  — the algorithm section is short and worth reading in full.
