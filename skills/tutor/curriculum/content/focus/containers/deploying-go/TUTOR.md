# Tutor notes — Deploying Go on Kubernetes

## Where the learner is

Sixth lesson of the containers pack, and the one where the two halves of the
pack finally meet: Kubernetes objects on one side, the Go runtime and
`net/http` on the other. They can write a Deployment, a Service, a ConfigMap
and probes; they have written production HTTP servers with graceful shutdown
in S5 — but in S5 nothing was *taking the process away from them*, so shutdown
was hygiene rather than a contract with a platform. This lesson makes it a
contract, and adds the two runtime-versus-cgroup mismatches (`GOMAXPROCS`,
`GOMEMLIMIT`) that connect straight back to the S5 scheduler and GC lessons.

The grade is static: `check.sh` reads every YAML file in the folder, the Go
source and `SHUTDOWN.md`. With a Go toolchain present it also runs `shutdown_test.go`,
which is where the Go half is really decided; kubectl and a cluster add dry
runs. If they have a cluster, push them to do the load-loop-during-rollout
experiment by hand, twice — once with the preStop hook, once without. Seeing
`000` scroll past is worth more than any explanation.

They have not met Helm or Kustomize yet, so keep everything in one file and
resist "you'd template this" tangents.

## Common misconceptions

- **"Kubernetes removes the pod from the Service, then stops it."** The two
  happen in parallel and nothing synchronises them. This is the misconception
  the whole lesson exists to break; if it survives, nothing else lands.
- **"The preStop sleep is a hack / cargo cult."** It is the only lever that
  delays SIGTERM. Ask what else could make the kubelet wait for kube-proxy.
- **"preStop runs before the grace period starts."** It runs inside it. This
  is the arithmetic error that gets services SIGKILLed mid-drain.
- **"Failing readiness is enough."** It covers pollers (ingress, LBs, meshes),
  not the endpoint-removal race on the deletion path. Both, not either.
- **"`Shutdown` and `Close` are the same, one is just newer."** `Close`
  severs in-flight requests. The test makes this concrete.
- **"`ErrServerClosed` is an error."** Returning it from `main` makes a clean
  stop look like a crash, which then looks like a failing deploy.
- **Liveness and readiness as synonyms.** Watch for a liveness probe pointed
  at `/readyz`, and for dependency checks migrating into liveness "so we
  notice faster". Both produce restarts that cannot help.
- **"The container has 1 CPU, so Go uses 1 CPU."** The runtime reads the
  affinity mask, not the quota — through Go 1.24 unconditionally, and on
  1.25+ only when the module's `go` directive allows the new default.
- **"`GOMEMLIMIT` prevents OOMKill."** It is soft. It makes the GC work
  harder; the kernel's limit is still the one that kills.
- **"`GOMEMLIMIT: 512Mi`".** Kubernetes syntax in a Go env var. The runtime
  refuses to start, and the pod crash-loops with a one-line message.
- **"maxUnavailable: 0 means no downtime, done."** Without a working readiness
  probe it means nothing: the gate is readiness, the setting only decides
  whether the gate is consulted before capacity is removed.

## Grilling points

- "Walk me from `kubectl delete pod` to the container being gone. Where does
  SIGTERM come from, and what else is happening at that moment?"
- "Your preStop sleeps 10s, your drain takes up to 20s, grace period is 30s.
  What happens, and at what second?"
- "Delete the preStop hook. Which requests fail, whose fault are they, and
  what does the caller see — a 503, or something else?" (Connection refused /
  reset: not an HTTP status at all.)
- "Why can't the readiness flip alone fix the race? Who is listening to
  readiness, and who is not?"
- "A pod is `OOMKilled` at 3 a.m. Which of your two runtime env vars was
  wrong, and how would you tell the difference between 'leak' and 'limit set
  too low'?"
- "Your service runs on a 96-core node with `limits.cpu: 500m` and no
  `GOMAXPROCS`. Describe the latency graph." (Throttling shelves; and note the
  1.25 floor of 2 if they are on a new module.)
- "Where should a database health check live: liveness, readiness, neither?
  Defend it in terms of what the platform does with the answer."
- Stretch: "Your handler streams for two minutes. What breaks in this design,
  and which knob do you turn?" (Grace period versus bounding the handler —
  the right answer is usually bounding the handler.)
- Stretch: "Would you use the downward API for `GOMEMLIMIT` too? Why not?"

## Grading rubric

- **A** — All static checks pass and `shutdown_test.go` is green; `SHUTDOWN.md`
  reads like something a colleague could act on; they can explain the parallel
  chains unprompted, defend their three numbers, and say what each probe must
  *not* check. Bonus: they ran the rollout with a request loop and can report
  what they saw with and without the preStop hook.
- **B** — Checks pass, but one area is mechanical — usually the budget
  arithmetic (numbers chosen to satisfy the checker) or `GOMEMLIMIT` headroom
  ("I copied 440MiB"). One round of grilling fixes it.
- **C** — Checks pass only after heavy hinting, or the ordering requirement is
  still described as "readiness fails, so Kubernetes removes the pod" with no
  awareness of the race. Pass only if a re-explanation lands in their own
  words.
- **Fail** — Checks failing; or liveness still pointed at readiness; or they
  cannot say what SIGKILL does. Remediate: the capstone deploys their own
  service, and this is the lesson it stands on.

## Remediation ladder

1. "Read the FAIL line and its `fix:` aloud. Which acceptance criterion is
   that, and which file does it live in?"
2. For the Go half: "Run `go test ./...` and read the failure message, not the
   line number. What did the test do, and what did it expect?" The three tests
   are deliberately named after the three ideas.
3. For the race: "Draw two columns, control plane and kubelet, and put the
   events in time order. Now mark the moment your listener closes. Who is
   still sending traffic at that moment?" Only after the drawing, ask what
   could delay the right-hand column.
4. For the budget: "Which of your three numbers is the clock, and which two
   spend it?" Then have them write the inequality in a comment.
5. Give the shape of the one block they are stuck on — the `lifecycle.preStop`
   keys, or the `resourceFieldRef` stanza — never the whole file.

## After passing

Preview: "Three environments will want the same manifests with different
numbers, and copy-paste is not a strategy. Next: packaging and overlays —
Helm and Kustomize."
