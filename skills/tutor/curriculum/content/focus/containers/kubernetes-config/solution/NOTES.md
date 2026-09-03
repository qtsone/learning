# Notes — configuring timesvc

Answer each question on the `A:` line below it, in your own words, in at
least a couple of sentences. `check.sh` checks that you answered; your tutor
reads what you actually wrote.

## Q1. Environment variables versus mounted files

`LOG_LEVEL` reaches the container as an environment variable and
`features.yaml` reaches it as a file, from the same ConfigMap. Say what
happens to each one when you edit the ConfigMap and re-apply it without
touching the Deployment — and which of the two you would choose for a setting
you expect to change during an incident.

A: The environment variable does not change at all. A process's environment is fixed at exec time, so the running container keeps the old value until a pod is replaced, and re-applying the ConfigMap alone changes nothing the process can see — the Deployment is untouched, so no rollout happens. The mounted file does change: the kubelet refreshes the projected volume within about a minute and swaps the symlink atomically, and because main.go re-reads the file on every request the new content takes effect with no restart. The one exception is a subPath mount, which is copied once and never updated. For something I might want to change during an incident I would use the file, and I would not use either without thinking about the failure mode: an env var means a rollout (safe, auditable, slow), a live file means a change with no deploy record and no rollback button.

## Q2. What a Secret is and is not

You wrote `API_KEY` into a Secret. Somebody with `kubectl get secret
timesvc-secrets -o yaml` in that namespace sees base64. Explain what base64
buys you, what it does not, and name two mechanisms that actually protect the
value — one that protects it in the cluster and one that keeps it out of your
git history.

A: Base64 is an encoding, not encryption: it exists so arbitrary bytes survive a JSON/YAML field, and `base64 -d` undoes it with no key. Anyone who can `get secrets` in the namespace has the plaintext, and by default the value is also stored in etcd in the clear, so anyone with an etcd backup has it too. What actually protects it in the cluster is RBAC — nobody gets `get`/`list` on secrets by default, and it is a separate verb from reading ConfigMaps — plus encryption at rest configured on the API server so an etcd snapshot is not a credential dump. What keeps it out of git is never committing the value: SOPS or Sealed Secrets commit an encrypted form only the cluster can open, or External Secrets pulls it from a real secret manager at runtime so the repository holds a reference instead of a credential.

## Q3. The probe you must not confuse

Imagine your database is unreachable for forty seconds and all six replicas
notice. Trace what happens if the liveness probe hits `/readyz` (the
dependency-aware endpoint), and then what happens if only the readiness probe
does. Say what the user sees in each case, and what state the service is in
when the database comes back.

A: If liveness hits /readyz, all six pods start failing liveness at the same moment, because they all depend on the same database. After failureThreshold periods the kubelet kills every container and restarts it, all six together, and the restarts keep failing while the database is down — so they back off into CrashLoopBackOff. The user sees connection errors rather than a clean 503, because the pods are dying rather than reporting unready. When the database returns, every pod is cold: empty caches, empty connection pools, and six of them reconnecting at once into a database that has just come back, which is a good way to knock it over again. If only readiness hits /readyz, the pods stay up and simply leave the Service's endpoint list, so the Service has no backends and callers get a fast, honest failure instead of a hang; nothing is restarted, no state is lost. When the database recovers, the readiness probe passes within a couple of periods and the same warm processes are added back to the endpoints. Liveness is for a process that can only be fixed by a restart; readiness is for a process that is fine but should not be sent traffic right now.
