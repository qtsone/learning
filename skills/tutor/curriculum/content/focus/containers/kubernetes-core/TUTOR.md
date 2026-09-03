# Tutor notes — Kubernetes Core

## Where the learner is

Fourth lesson of the containers pack, and the first one after S5, so they now
arrive with production Go behind them: HTTP servers with graceful shutdown,
health and readiness endpoints, structured logs, profiling. From the first
three lessons they have images, layers, tags vs digests, multi-stage builds,
non-root users, and Compose. Do not re-teach any of it.

This is the biggest conceptual jump in the pack. Everything before it was
"describe a container"; this is "describe a *wish* and let controllers chase
it". Keep pulling them back to the control loop — most Kubernetes confusion
dissolves the moment someone genuinely believes nothing is executing their
commands.

Two things they have not met and must not be leaned on: probes, ConfigMaps,
Secrets, resources and HPA are the *next* lesson, and graceful shutdown on
Kubernetes (SIGTERM ordering, `terminationGracePeriodSeconds`, GOMAXPROCS) is
the one after. If they ask why the pods have no probes, say "next lesson" and
move on. Ingress, Helm and Kustomize are later still.

The grade is static: `check.sh` never needs a cluster, and its two live checks
report `skip` when none is reachable. If they installed kind, push them to
build the lesson-two image, `kind load` it, apply, and watch `kubectl get pods
-w` while they delete a pod. That thirty seconds is the lesson.

## Common misconceptions

- **"`kubectl apply` creates the pod."** It records desired state; a
  controller creates the pod. Test it: "if the whole control plane were down
  for ten minutes and a pod died, what would happen — and what happens when it
  comes back?"
- **"Kubernetes reacts to events."** It compares state on a loop
  (level-triggered). A missed event is harmless; a wrong `spec` is not.
- **A Deployment "contains" its pods.** Nothing contains anything. Labels and
  selectors are the only glue, which is why a selector typo is silent.
- **Service selector written as `matchLabels`.** Deployment syntax leaking into
  a Service. It parses, applies cleanly, and matches nothing.
- **Selecting the Deployment's `metadata.labels` instead of the pod template's.**
  A Service selects pods; the Deployment object is not a pod.
- **`port` == `targetPort`, or omitting `targetPort`.** Omitting it defaults to
  `port`, not to the container port — a silent 80-to-80 forward into nothing.
- **Editing live objects.** `kubectl edit`/`scale` in production is undone by
  the next `apply`. The file is the truth.
- **`:latest` will redeploy.** The pod template does not change, so no new
  ReplicaSet, so nothing rolls. Plus its default pull policy is `Always`.
- **"Running means Ready."** Endpoints only include Ready pods. They will meet
  the mechanism that decides Ready next lesson; here it is enough that Running
  and Ready are different columns.
- **"A namespace isolates."** It scopes names, quotas and RBAC. Pods across
  namespaces talk freely without a NetworkPolicy.
- **ImagePullBackOff on kind** because the image is only in the local Docker
  daemon. A kind node is a container with its own image store.

## Grilling points

- "Walk me through the six things that happen between `kubectl delete pod` and
  a replacement serving traffic. Who does each one, and who coordinates them?"
  (Nobody coordinates. That is the answer.)
- "Your Service returns connection refused, all three pods are Running. Name
  three different causes and the one command that distinguishes them."
- "You changed `replicas` from 3 to 5 and applied. Did that restart any pod?
  Why not? Now you changed the image tag — what object appears, and what
  happens to the old one?"
- "Why is `spec.selector` immutable? What would break if you could change it?"
  (Existing pods become orphans of the new selector; ownership would silently
  transfer.)
- "`maxUnavailable: 0` and `maxSurge: 0` — what happens on rollout?" (Nothing,
  forever: allowed neither to remove nor to add.)
- "You have a ClusterIP and a laptop. Two ways to reach the service, and why
  one of them is a bad reason to change the manifest."
- "Where does the cluster's desired state actually live, and which component is
  allowed to touch it?"
- Stretch: "A Service selects pods from two different Deployments. Why would
  anyone want that?" (Canary, blue/green.)
- Stretch: "`targetPort: http` instead of `8080` — what does that buy you?"

## Grading rubric

- **A** — All 16 static checks pass; `NOTES.md` traces reconciliation with the
  right actors, not just "Kubernetes recreates it". The learner explains
  unprompted why the selector/label pair is the fragile part, why `:latest`
  breaks rollouts, and what `port` vs `targetPort` mean. Bonus: they ran it on
  kind, deleted a pod, and broke the selector on purpose.
- **B** — Checks pass and the model is sound, but one area is mechanical —
  usually the strategy numbers copied without meaning, or "the Service points
  at the Deployment" language that survives the first correction.
- **C** — Checks pass only after heavy hinting, or they still describe
  `kubectl apply` as an imperative command. Pass only if a re-explanation of
  the control loop lands in their own words.
- **Fail** — Checks failing, or they cannot say why a Service might have no
  endpoints, or they believe a bare Pod is a reasonable way to run a service.
  Remediate: the next two lessons add configuration and shutdown behaviour on
  top of exactly these objects.

## Remediation ladder

1. "Read the FAIL line and its `fix:` aloud. Which of the twelve acceptance
   criteria is it about?"
2. "Which object is that field on, and at what depth?" Most early failures are
   nesting: `template.spec.containers` vs `spec.containers`, or `labels` under
   `spec` instead of under `template.metadata`. Have them read their file top
   to bottom saying the path of each key out loud.
3. For the selector: "Point at the two places the string `app: timesvc` has to
   appear for the Deployment to be legal, and the third place for the Service
   to find anything. Now point at them in your file." For ports: "Which number
   does a caller dial, and which number does the process bind? Which field is
   which?"
4. Give the shape of the one block they are stuck on — the `strategy` keys and
   what each means, say — but never the whole manifest. The manifest is the
   exercise.

## After passing

Preview: "Your pods run, but they are configured by nothing and nobody has
told Kubernetes what 'healthy' means. Next: ConfigMaps and Secrets, the three
kinds of probe, and what requests and limits actually do to your process."
