# Runbook — tinysvc

Written for the person woken at 3am, who may be you in six months with no
memory of this code. Commands are meant to be pasted, not adapted.

- **Service:** tinysvc, namespace `tinysvc`, 2 replicas.
- **Depends on:** nothing. It has no database and makes no outbound calls.
- **Blast radius when down:** `/work` requests fail; nothing is lost, because
  the service stores nothing.

## How to deploy

Deploys are immutable: a version tag, never `latest`, never a rebuilt tag.

```sh
git tag v1.4.2 && git push origin v1.4.2
./deploy/deploy.sh v1.4.2          # build, push, roll out, wait for ready
kubectl -n tinysvc get pods -l app=tinysvc
```

The rollout waits on the readiness probe, so a version that cannot serve never
replaces the one that can.

## How to roll back

Roll back first, understand afterwards. This is one command and it is safe to
run when you are not yet sure what is wrong.

```sh
./deploy/rollback.sh               # kubectl rollout undo + wait
kubectl -n tinysvc get deployment tinysvc \
  -o jsonpath='{.spec.template.spec.containers[0].image}'
```

Expect the previous image tag back within ~60s. If `rollout undo` reports no
previous revision, deploy the last known-good tag explicitly:
`./deploy/deploy.sh <previous tag>`.

## What to check when it is down

In order, stopping when one of them explains it:

```sh
kubectl -n tinysvc get pods -l app=tinysvc          # 1. are the pods running?
kubectl -n tinysvc logs -l app=tinysvc --tail=100   # 2. what do the logs say?
kubectl -n tinysvc port-forward svc/tinysvc 8080:8080
curl -s localhost:8080/healthz                      # 3. process alive?
curl -s localhost:8080/readyz                       # 4. willing to serve?
curl -s localhost:8080/metrics | grep http_         # 5. traffic and errors
```

Reading the answer:

- `/healthz` fails → the process is broken; the pod should already be
  restarting. Check the last log line before the restart.
- `/healthz` fine, `/readyz` fails → it is draining or was never marked ready.
  A pod stuck not-ready for more than a minute is a deploy problem, not a load
  problem.
- Both fine, `http_errors_total` climbing → the failure is in `/work`; find the
  request logs with `status` >= 500 in the same window.
- Both fine, `http_requests_total` flat → nothing is reaching us. The problem is
  in front of the service, not in it.

## Known failure modes

### Rollout leaves every pod not-ready

**Symptom:** `deploy.sh` hangs at `rollout status`; new pods `0/1 READY`.
**Confirm:** `kubectl -n tinysvc logs <new pod>` shows `bad configuration`.
**Cause:** a rejected value in `TINYSVC_ADDR`, `TINYSVC_LOG_LEVEL` or
`TINYSVC_SHUTDOWN_TIMEOUT` — the process exits 2 at startup by design.
**Action:** fix the value in `deploy/deployment.yaml` and redeploy; the old
pods are still serving, so this is not an outage yet.

### Requests cut off during a deploy

**Symptom:** clients see connection resets exactly during rollouts.
**Confirm:** `TINYSVC_SHUTDOWN_TIMEOUT` is greater than or equal to
`terminationGracePeriodSeconds` in `deploy/deployment.yaml`.
**Cause:** the platform sends SIGKILL before the drain finishes.
**Action:** lower the shutdown timeout below the grace period (currently 15s
against 30s) and redeploy.

## Who to page

- **Primary:** the service owner — you. Contact route recorded in the team
  directory entry for `tinysvc`.
- **Escalate after 30 minutes** without a working rollback: the platform owner,
  since at that point the problem is the cluster, not the code.
- Not urgent outside working hours: this service has no overnight users. Say so
  explicitly, so nobody pages a human for a graph.
