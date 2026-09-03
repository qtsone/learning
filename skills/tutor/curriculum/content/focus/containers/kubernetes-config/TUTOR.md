# Tutor notes — Kubernetes Configuration

## Where the learner is

Fifth lesson of the containers pack, second on Kubernetes. They can write a
Deployment and a Service and they have the reconciliation model from
Kubernetes Core. Everything here is new object surface, but almost none of it
is new *thinking*: secrets hygiene and reading configuration from the
environment came from S4's security lesson, health endpoints and structured
logs from S5's observability lesson, and "measure before you size it" from S5
profiling. Pull those threads deliberately — this
lesson lands much better as "the platform version of things you already do"
than as a new vocabulary list.

The grade is static: `check.sh` parses their manifests and needs no cluster,
no kubectl and no Docker; the two dry-run checks report `skip` when there is
nothing to talk to. If they do have a cluster, push them to apply the
manifests and watch `kubectl get pods -w` during the two-second warm-up, and
to break something on purpose (point liveness at `/readyz` and stop the
dependency, or set `memory: 4Mi` and watch `OOMKilled`). That is the lesson.

Graceful shutdown, `terminationGracePeriodSeconds` and what the Go runtime
believes about a CPU limit are the *next* lesson. If they ask, confirm it is
coming and keep them on configuration.

## Common misconceptions

- **"A Secret is encrypted."** It is base64 in the wire format, and encryption
  at rest is a cluster-operator opt-in. Make them run the `base64 -d`
  one-liner mentally. What a Secret actually buys: a separate RBAC verb, tmpfs
  on the node, and `describe secret` printing `API_KEY: 24 bytes` where
  `describe configmap` prints the value.
- **"Editing the ConfigMap updates my app."** For env vars, never — the
  Deployment did not change, so nothing rolls, and a process's environment is
  fixed at `exec`. For mounted files, yes, after the kubelet's sync — unless
  it is a `subPath` mount, which is a copy.
- **"Liveness should check everything."** The single most expensive mistake in
  this lesson. Its only remedy is a restart, so it must only fail for things a
  restart fixes. If they cannot trace the cascading-restart story, do not pass
  them.
- **"Readiness failing restarts the pod."** It removes the pod from the
  Service's endpoints. No restart, no lost state.
- **"A startup probe is just `initialDelaySeconds`."** It replaces the guess
  with a budget, and suspends liveness and readiness while it runs.
- **"Requests are what the pod gets."** Requests are what the *scheduler*
  reserves on paper; at runtime they become the cgroup CPU weight and the
  eviction baseline. What they are not is a ceiling — that is the limit. Accept
  the runtime half if they raise it; it is correct.
- **"Hitting the CPU limit kills the container."** CPU throttles (CFS quota),
  memory kills (OOMKilled, exit 137). Compressible versus incompressible.
- **"The HPA scales on node CPU."** It scales on a percentage of the pod's CPU
  *request*. No request, `<unknown>`, no scaling.
- **"More replicas fix any overload."** Not when the bottleneck is downstream;
  more replicas then multiply the pressure on the database.

## Grilling points

- "Your liveness probe hits `/readyz` and the database goes away for forty
  seconds. Walk me through minute by minute across six replicas — what does
  the user see, and what state is the service in when the database returns?"
- "Which of your settings can I change without a rollout, and which needs one?
  Show me where in the manifest that difference lives."
- "I have `get secrets` in your namespace. What can I read, and what would
  have stopped me?"
- "Your `/now` p99 is 900 ms and the pod's average CPU is 25%. Nothing in the
  logs. What is your first hypothesis, and which number would confirm it?"
  (CFS throttling against the CPU limit.)
- "You set `requests: cpu: 2` on a service that uses 50m because 'it should be
  fast'. What breaks?" (Scheduling density, cost, and an HPA target that is
  now 2% of reality.)
- "Delete `requests.cpu` from the Deployment. What does `kubectl get hpa`
  print?"
- "Why does the reference solution mount the whole directory instead of using
  `subPath`?"
- Stretch: "Your probe `timeoutSeconds` is 1 and a GC pause takes 1.2 s under
  load. What happens, and at which moment of the day?"

## Grading rubric

- **A** — All 20 static checks pass; the manifests are tidy; and unprompted
  they can explain the liveness/readiness split, why the HPA needs the CPU
  request, and what a Secret does and does not protect. Bonus if they applied
  the manifests and broke something deliberately.
- **B** — Checks pass, but one area is mechanical — usually the probes (copied
  shapes with defaults they cannot justify) or the resource numbers (round
  numbers with no reasoning). One round of grilling fixes it.
- **C** — Checks pass only after heavy hinting, or the base64 answer is
  "it's encrypted-ish", or they cannot say what happens when they edit a
  ConfigMap. Pass only if a re-explanation lands.
- **Fail** — Checks failing, or liveness and readiness are still the same idea
  to them, or they think limits are what the scheduler uses. Remediate: the
  next two lessons deploy a real service on exactly these objects.

## Remediation ladder

1. "Read the FAIL line and its `fix:` aloud. Which of the fourteen acceptance
   criteria is it about?"
2. "Where does that key belong — on the container, on the pod spec, or on the
   object? `volumeMounts` and `volumes` have almost the same name and live two
   levels apart." (Most early failures are structural, not conceptual.)
3. Aim the question at the concept, not the syntax. Probes: "what is the
   remedy when this probe fails, and is that the remedy you want here?"
   Resources: "which of these two numbers does the scheduler read, and which
   one does the kernel read?" HPA: "70% of *what*, exactly?"
4. Give the shape of the single block they are stuck on — the five fields of a
   probe, or the `valueFrom.secretKeyRef` triple — but never the whole file.
   `main.go`'s doc comment already states the endpoint contract; send them
   there before you answer.

## After passing

Preview: "You have configured a service the way the platform wants. Next:
deploying a *Go* service correctly — what SIGTERM does to your process, why
readiness has to fail before the server stops accepting, and what the Go
runtime believes about the CPU and memory limits you just wrote."
