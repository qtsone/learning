# Kubernetes Core

> `focus.containers.kubernetes-core` · ~3-4h · Stage: Focus: Containers II

## Objectives

By the end of this lesson you can:

- Explain the role of pods, deployments, and services, and why you deploy via a
  Deployment rather than bare pods.
- Implement manifests for a Deployment and a Service that expose a
  containerized app inside a cluster.
- Explain how a Service selects pods via labels, and how ClusterIP, NodePort,
  and LoadBalancer differ.
- Inspect and debug workloads with `kubectl` (`get`, `describe`, `logs`,
  `exec`, `port-forward`), and diagnose a `CrashLoopBackOff` or an
  `ImagePullBackOff`.
- Explain Kubernetes' reconciliation model: desired state stored by the API
  server, controllers converging actual state toward it, and what happens when
  a pod dies.

Install a local cluster for this one. Reading about a control loop and watching
one recreate the pod you just deleted are different experiences, and the second
is the one that sticks. With Docker from lesson one, it is one command:

```sh
brew install kind          # or: go install sigs.k8s.io/kind@latest
kind create cluster --name tutor
kubectl cluster-info --context kind-tutor
```

minikube, k3d and Docker Desktop's built-in cluster work equally well. Install
none of them and you are still fine: `check.sh` grades the manifests by reading
them, so the exercise is completable and the grade is unaffected.

## Not a deploy script

Compose was a script with a nicer syntax: `up` created things, `down` removed
them, and nothing was watching in between. A container that died at 3 a.m.
stayed dead until someone typed something.

Kubernetes is not that. You do not tell it to *do* anything. You write down
what you want the world to look like — "three copies of this image, reachable
under this name" — and hand that to the API server, which stores it. From then
on, **controllers** run a loop, forever:

```
observe actual state  ->  compare with desired state  ->  act to close the gap
```

That loop is the whole idea. `kubectl apply` does not create a pod. It records
a wish. A controller notices the gap between the wish and reality and closes
it, then keeps closing it, every time reality drifts — a node reboots, a
process panics, someone deletes a pod by hand.

Two consequences before you write any YAML. First, it is **level-triggered**:
controllers do not chase a stream of events they must not miss, they
periodically compare the world to the spec, so a missed event corrects itself
at the next comparison — which is why the whole system survives its own control
plane going down and coming back. Second, **the manifest is the truth**, so
`kubectl edit` in production is a lie you tell yourself: the next `apply` from
the repository overwrites it. Change the file, apply the file.

The pieces doing this: the **API server** (the only component that reads and
writes the cluster's datastore, etcd), the **controller manager** (where the
Deployment and ReplicaSet controllers live), the **scheduler** (picks a node
for a pod that has none), and a **kubelet** on every node (starts and watches
the containers assigned to it). All of them talk only to the API server, never
to each other. Learn that shape and the system stops feeling magic.

## Everything is an object with the same four fields

```yaml
apiVersion: apps/v1     # which API group and version this object belongs to
kind: Deployment        # which type
metadata:               # name, namespace, labels, annotations
  name: timesvc
spec:                   # what you want (you write this)
status:                 # what is (controllers write this; never you)
```

`spec` versus `status` is the reconciliation model made visible in every single
object. You write `spec`; controllers write `status`; "the cluster converged"
means status caught up with spec. And `apiVersion` is the field beginners get
wrong most: a Deployment is `apps/v1`, a Service is plain `v1` (the core group,
which has no group name), and copying one onto the other produces a confusing
"no matches for kind" error.

## Pod: the unit of scheduling, and why you never write one

A **pod** is one or more containers always placed on the same node, sharing a
network namespace and, optionally, volumes. Sharing a network namespace means
they reach each other on `localhost` and cannot both bind the same port — the
namespace model from lesson one, with two processes in one namespace instead of
one. The pod gets its own cluster IP.

Almost always, one container per pod. The exception is a **sidecar**: a helper
so tightly coupled that it must live, die and share a network with the main
container — a log shipper, a proxy. If two containers could run on different
machines, they are two pods.

A pod is also **mortal and unrepairable**. It is never moved, never healed:
if its node dies, that pod is gone, with its IP and its name, and nothing
brings it back, because nothing recorded that it should exist. That is why a
bare `kind: Pod` manifest is a toy. Write a controller-backed object and let
the controller own pods for you.

## ReplicaSet: keep N of them alive

A **ReplicaSet** has exactly one job: make the number of pods matching its
selector equal `replicas`. Too few, it creates pods from its template; too
many, it deletes some. That is the entire control loop, and it is the reason a
crashed pod comes back. You do not write ReplicaSets either — you write a
Deployment, which writes them.

## Deployment: ReplicaSets over time

A **Deployment** manages ReplicaSets so that changing the pod template is a
*controlled rollout* rather than a stop-the-world replace. Change the image tag
in `spec.template` and apply: the Deployment controller creates a **new**
ReplicaSet at zero replicas, scales it up while scaling the old one down a few
pods at a time within the limits you set, and keeps the old one at zero as a
revision you can return to.

```yaml
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0    # never drop below 3 ready pods
      maxSurge: 1          # allow a 4th pod while rolling
  revisionHistoryLimit: 5  # how many old ReplicaSets to keep for rollback
```

The defaults are 25% each, which means a rollout may run at 75% capacity — fine
for a big fleet, not what you want for three pods. `maxUnavailable: 0` with
`maxSurge: 1` is the conservative pair: always create the replacement first,
never take capacity away. It costs one pod of headroom and buys a rollout that
cannot dip below what you declared.

Two facts that make rollouts feel less mysterious:

- **A rollout only happens if the pod template changes.** Editing `replicas` is
  a scale, not a rollout. This is why redeploying "the same tag" does nothing —
  the template is byte-identical, so there is no new ReplicaSet, so nothing
  restarts — and why `:latest` is a trap: the tag never changes, so Kubernetes
  never notices your new build. Pin tags, or deploy by digest.
- **Rollback is just a scale of an old ReplicaSet.** That is why it is fast,
  and why it only reaches back as far as `revisionHistoryLimit` kept.

## Labels and selectors: the only glue there is

Nothing in Kubernetes points at anything by name. A ReplicaSet does not hold a
list of its pods; a Service does not hold a list of endpoints. Objects find
each other by **labels** — arbitrary key/value pairs in `metadata.labels` — and
**selectors** that match them.

```yaml
spec:
  selector:
    matchLabels:
      app: timesvc      # which pods this Deployment owns
  template:
    metadata:
      labels:
        app: timesvc    # the stamp those pods carry
```

Those two blocks must agree — the API server rejects a Deployment whose
selector does not match its own template — and `spec.selector` is **immutable**
after creation. Choose it once, keep it minimal, and put whatever changes per
release, like a version, in the template labels only.

This loose coupling is powerful and sharp. Powerful: a Service can select pods
from two Deployments at once, which is how a canary works. Sharp: a typo in a
selector is not an error anywhere. It matches zero pods, everything reports
healthy, and traffic goes nowhere. **This is the single most common beginner
bug in Kubernetes**, and the exercise checks it explicitly.

## Service: a stable name in front of mortal pods

Pod IPs change on every restart. A **Service** is the stable thing in front of
them: one virtual IP and one DNS name, allocated at creation and kept until you
delete the Service.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: timesvc
spec:
  type: ClusterIP
  selector:
    app: timesvc        # a flat map — Services have no matchLabels
  ports:
    - name: http
      port: 80          # the port the Service listens on
      targetPort: 8080  # the port on the pod it forwards to
```

How it actually works: a controller watches for pods matching the selector that
are Ready, and writes their IPs into an **EndpointSlice** object attached to
the Service. On every node, `kube-proxy` turns those endpoints into forwarding
rules. There is no proxy process in the path and nothing listening on the
cluster IP — it is a virtual address rewritten by the kernel. Hence the debug
move: `kubectl get endpointslices` tells you whether the Service found any
pods. Empty means your selector matches nothing, or no pod is Ready.

Two details worth memorizing:

- **`port` and `targetPort` are different things.** `port` is what clients dial
  on the Service; `targetPort` is the port on the pod. Writing
  `port: 80, targetPort: 8080` is common, because your Go service listens on
  8080 as a non-root user (lesson two) while callers want a boring 80.
  `targetPort` may also be a *name*: if the container declares a port named
  `http`, `targetPort: http` resolves per pod.
- **DNS**: the Service answers to `timesvc` from its own namespace and
  `timesvc.<namespace>.svc.cluster.local` from anywhere. This is the Compose
  service-name discovery you already trust, one layer up: names, never IPs.

### The three types

| Type | What it gives you | Reachable from |
|---|---|---|
| `ClusterIP` (default) | a virtual IP + DNS inside the cluster | inside only |
| `NodePort` | that, plus the same port opened on **every node** | outside, via any node IP |
| `LoadBalancer` | that, plus a cloud load balancer pointed at the nodes | the internet |

They are cumulative: a `LoadBalancer` is a `NodePort` is a `ClusterIP`. Default
to `ClusterIP` for anything internal — a service other pods call needs nothing
more. `NodePort` picks a port from 30000-32767 and is mostly a building block.
`LoadBalancer` provisions (and bills you for) real infrastructure, one per
Service, which is why production clusters usually put a single ingress in front
of many ClusterIP Services instead. To reach a ClusterIP from your laptop while
debugging, do not change the type — tunnel:

```sh
kubectl port-forward service/timesvc 8080:80
```

## Namespaces

A **namespace** is a name scope inside one cluster: two teams can both own a
Service called `api`. Names are unique per namespace, most objects live in one,
and `kubectl` defaults to `default` unless you pass `-n`. It is not a security
boundary by itself and not a network boundary — pods in different namespaces
reach each other freely unless someone writes a NetworkPolicy. What it gives
you is a scope for quotas, for access control, and for
`kubectl delete namespace`, the tidiest way to remove an experiment. Give this
exercise its own.

## kubectl, and the debugging habit

```sh
kubectl apply -f k8s/            # send the desired state (whole directory)
kubectl get pods -n timesvc      # one line per object; -o wide, -o yaml, -w
kubectl describe pod <name>      # full state + recent Events at the bottom
kubectl logs <pod> -f            # stdout/stderr; --previous for the last crash
kubectl exec -it <pod> -- sh     # a shell, if the image has one (it may not)
kubectl port-forward svc/x 8080:80
kubectl rollout status deployment/timesvc     # blocks until rolled out or stuck
kubectl rollout history deployment/timesvc
kubectl rollout undo deployment/timesvc       # back one revision
kubectl delete -f k8s/
```

The habit that separates people who debug Kubernetes from people who guess at
it: **describe first, and read the Events at the bottom.** `get` tells you a
pod is unhappy. `describe` tells you why, in plain sentences written by the
component that made the decision — the scheduler saying no node has enough
memory, the kubelet saying a pull failed, a probe reporting failures.

Two states you will meet in your first hour:

- **`ImagePullBackOff`** — the kubelet could not fetch the image and is backing
  off between retries. The Events line names the registry and the error: a
  typo, a private registry with no credentials, or — most common locally — an
  image that exists in *your Docker daemon* but not in the cluster's, because a
  kind node is a separate container with its own image store. Fix:
  `kind load docker-image timesvc:1.0.0`. Related gotcha: the default pull
  policy is `IfNotPresent` for a normal tag but `Always` for `:latest` or no
  tag, so `:latest` guarantees a pull attempt and guarantees this failure.
- **`CrashLoopBackOff`** — the container starts, exits, and the kubelet
  restarts it with exponential backoff (10s, 20s, 40s… capped at 5 minutes).
  This is not a Kubernetes problem; it is your process dying. Read
  `kubectl logs <pod> --previous`: the current container may have no logs yet,
  and the interesting output belongs to the instance that died. Classic causes:
  a missing environment variable your startup validation rejects, a port
  already bound, an entrypoint that is not what you think it is.

### What happens when a pod dies

Walk it once, because it is the whole model in one story. You run
`kubectl delete pod timesvc-7d9f-abcde`:

1. The API server marks the pod deleted; its kubelet stops the container.
2. The endpoint controller removes its IP from the Service's EndpointSlice, so
   traffic stops being sent to it.
3. The ReplicaSet controller compares — wants 3, sees 2 — and creates a new pod
   from the template, with a new name and a new IP.
4. The scheduler sees a pod with no node and assigns one.
5. That node's kubelet pulls the image if needed and starts the container.
6. Once the pod is Ready, the endpoint controller adds its IP to the
   EndpointSlice and traffic starts flowing to it.

Nobody orchestrated that sequence: four independent loops each closed their own
gap. Run `kubectl get pods -w` in one terminal and delete a pod in another. The
whole thing takes a couple of seconds, and they are the best two seconds of
this lesson.

## Exercise

Open [`exercise/`](exercise/). You are writing the manifests that put the
`timesvc` image from lesson two into a cluster. You are given:

- `namespace.yaml` — complete, do not change it.
- `deployment.yaml` and `service.yaml` — skeletons full of `TODO`s.
- `NOTES.md` — three questions to answer in your own words.
- `check.sh` — the referee.

**Formatting the grader expects.** `check.sh` parses your YAML with a small
reader it carries itself, so it needs no third-party tools. Stay in this subset
and nothing legal will be misread: spaces to indent (two per level), block
mappings and block sequences (`- item`, indented under its key or at the key's
own column), no anchors or aliases (`&`, `*`), no `|`/`>` block scalars in
graded fields. `---` between documents is fine, and every `.yaml`/`.yml` file
in the folder is read, so it does not matter which file an object lands in.
Quoting is free: `8080` and `"8080"` are the same to it.

Acceptance criteria:

1. A Deployment with `apiVersion: apps/v1`, `kind: Deployment`, name
   `timesvc`, and `metadata.namespace: timesvc`.
2. `spec.replicas` is at least 2 — one replica means every rollout, node drain
   and crash is an outage.
3. `spec.selector.matchLabels` is set, and every pair in it also appears in
   `spec.template.metadata.labels`.
4. The pod template carries the label `app: timesvc`.
5. Exactly one container, with a `name`, and an `image` whose repository ends
   in `timesvc` pinned to a real tag (use `timesvc:1.0.0`) or a digest — never
   `latest`, never untagged.
6. The container declares `containerPort: 8080`. Naming that port (`name:
   http`) is optional and useful.
7. `spec.strategy.type: RollingUpdate` with `maxUnavailable: 0` and `maxSurge`
   of at least 1.
8. A Service with `apiVersion: v1`, `kind: Service`, name `timesvc`, namespace
   `timesvc`.
9. `spec.type: ClusterIP`, written explicitly even though it is the default.
10. `spec.selector` is a flat map (a Service has no `matchLabels`), and every
    pair in it appears in the Deployment's **pod template** labels.
11. One port with `port: 80` and a `targetPort` that agrees with the
    container — either `8080` or the container port's name.
12. `NOTES.md` answers all three questions.

Run the referee from inside `exercise/`:

```sh
bash check.sh
```

Every graded check is static: it reads your files, with **no cluster, no
kubectl, no Docker daemon, no network and no root**, and the starter is
expected to fail. If a `kubectl` *and* a reachable cluster happen to be there,
two bonus live checks run against whatever `kubectl config current-context`
names, and otherwise report `skip`, never `FAIL`. The first is a server-side
dry run and creates nothing. The second really applies your Deployment and
Service, watches the rollout and removes them again, so it runs only on a
throwaway local context (`kind-*`, `k3d-*`, `minikube`, `docker-desktop`,
`rancher-desktop`, `orbstack`) or when you set `TUTOR_LIVE_APPLY=1`. It
creates the namespace if it is missing and never deletes it.

With a cluster, do this by hand too, because it is where the lesson lands.
Build the image first — in the Dockerizing Go lesson's folder (lesson two of
this pack), run `docker build -t timesvc:1.0.0 .` — then, back here:

```sh
kind load docker-image timesvc:1.0.0 --name tutor
kubectl apply -f .
kubectl get pods -n timesvc -w        # watch them appear
kubectl describe deployment timesvc -n timesvc
kubectl get endpointslices -n timesvc # your selector, proven
kubectl port-forward -n timesvc svc/timesvc 8080:80
curl localhost:8080/healthz
kubectl delete pod -n timesvc -l app=timesvc   # then watch them come back
```

Then break it on purpose: change the Service selector to `app: timesvcc` and
apply. Nothing errors, nothing looks unhealthy, and the endpoints list is
empty. Meet that failure mode here rather than in production.

## Further reading

- [Kubernetes concepts — Pods](https://kubernetes.io/docs/concepts/workloads/pods/)
  and [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [Service](https://kubernetes.io/docs/concepts/services-networking/service/) —
  types, `port` vs `targetPort`, EndpointSlices.
- [Labels and selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
- [kubectl quick reference](https://kubernetes.io/docs/reference/kubectl/quick-reference/)
- [kind — quick start](https://kind.sigs.k8s.io/docs/user/quick-start/)
