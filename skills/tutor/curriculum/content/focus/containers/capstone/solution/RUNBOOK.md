# Runbook — timesvc

Owner: the platform team. Repository: `github.com/ada/timesvc`.
Image: `ghcr.io/ada/timesvc`. Environments: `timesvc-dev`, `timesvc-prod`.

## What this service is

`timesvc` is a stateless HTTP service that returns the current time in a
requested IANA zone. `GET /now?zone=Europe/Bucharest` is the work,
`GET /version` reports the commit the running binary was built from. It has no
database and no queue: the only thing it depends on is the cluster.

Two probe endpoints that are not interchangeable. `GET /healthz` is liveness —
200 for as long as the process can serve at all, including the whole shutdown
drain; a failure here means the kubelet restarts the container. `GET /readyz`
is readiness — 200 only while the pod wants new traffic, and 503 from the
first instant of a drain, which is how the pod removes itself from the
Service's endpoints. Point a load balancer at `/readyz`, never at `/healthz`.

Healthy from outside: `kubectl -n timesvc-prod get deploy timesvc` shows all
replicas ready, and a curl of `/now` from inside the cluster returns 200 with
a timestamp within a second of real time.

## How a change reaches production

A merge to `main` triggers `.github/workflows/release.yml`, which is the only
path to either cluster. Four jobs, each gating the next:

1. `test` — `go test ./...`. Nothing is published if this is red.
2. `build-push` — builds the multi-stage image, passing the commit as the
   `REVISION` build argument, and pushes it to `ghcr.io/ada/timesvc` under the
   tag `sha-<commit>`. The job's outputs carry the image **digest**
   (`sha256:…`), the artifact's own content hash.
3. `deploy-dev` — points `deploy/overlays/dev` at that digest with
   `kustomize edit set image`, applies it to `timesvc-dev`, and waits.
4. `deploy-prod` — the same digest, `deploy/overlays/prod`, `timesvc-prod`,
   behind the `production` GitHub environment (a required reviewer approves
   before the job starts).

The artifact's identity is the digest; the `sha-<commit>` tag is the
human-readable alias for it. Both are immutable: nothing in the pipeline ever
pushes or deploys `latest`. That matters because a Deployment stores its image
reference in the pod template, and `rollout undo` restores an old pod
template — if two revisions carry the same moving tag, the undo restores an
identical template and changes nothing at all. Immutability is what makes a
rollback a real operation rather than a re-pull of whatever is newest.

Promotion is one artifact moving forward, never a rebuild: prod runs the exact
bytes dev proved, so "it worked in dev" is a statement about the same image.

## Verify a release

The workflow being green means the commands it ran exited 0. These are the
commands that say what is actually serving:

```sh
kubectl -n timesvc-prod rollout status deployment/timesvc --timeout=300s
kubectl -n timesvc-prod get pods -l app=timesvc
kubectl -n timesvc-prod get deploy timesvc \
  -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
kubectl -n timesvc-prod run verify --rm -i --restart=Never \
  --image=curlimages/curl:8.11.1 -- -sS http://timesvc/version
```

- `rollout status` blocks until every replica is updated *and* ready, and
  exits non-zero on a stalled rollout. It is the only one of these that waits.
- `get pods` shows restart counts: a pod that is ready after three restarts
  passed the rollout and is still telling you something.
- The jsonpath prints what the Deployment asks for.
- The `/version` curl prints the revision a pod behind the Service actually
  answers with. Compare it to the commit you shipped; if they differ, traffic
  is still being served by the old ReplicaSet.

Watch the logs for one minute afterwards:
`kubectl -n timesvc-prod logs -l app=timesvc --tail=50 -f`.

## Roll back

Fastest first. The previous ReplicaSet is still in the cluster
(`revisionHistoryLimit: 10`), so:

```sh
kubectl -n timesvc-prod rollout undo deployment/timesvc
kubectl -n timesvc-prod rollout status deployment/timesvc --timeout=300s
kubectl -n timesvc-prod run verify --rm -i --restart=Never \
  --image=curlimages/curl:8.11.1 -- -sS http://timesvc/version
```

That takes about as long as a normal rollout — 30-60 seconds, one drain per
pod at `terminationGracePeriodSeconds: 45` — and the third command is not
optional: the undo is a deploy like any other and fails the same ways.

To pin the repository to the same state, so the next release does not
reintroduce the bad image, edit the overlay and commit it:

```sh
cd deploy/overlays/prod
kustomize edit set image ghcr.io/ada/timesvc@sha256:<last-good-digest>
kubectl apply -k .
```

Find `<last-good-digest>` in the `build-push` job of the last green run, or
with `kubectl -n timesvc-prod rollout history deployment/timesvc`.

What a rollback does **not** undo: anything the old code left behind outside
the process. timesvc holds no state, so today that is nothing — but the moment
it owns a schema, a rollback of the image is not a rollback of the migration,
and the runbook needs a section on that before that day.

## From a clean state

Everything needed to stand the service up in an empty cluster, given only this
repository and credentials:

```sh
kubectl create namespace timesvc-prod
kubectl -n timesvc-prod create secret docker-registry ghcr \
  --docker-server=ghcr.io --docker-username=<bot-user> --docker-password=<token-from-the-vault>
kustomize build deploy/overlays/prod          # read it before you apply it
kubectl apply -k deploy/overlays/prod
kubectl -n timesvc-prod rollout status deployment/timesvc --timeout=300s
```

The image comes from the GHCR registry; the package is private, so the pull
secret above must exist in the namespace before the first pod schedules,
otherwise every pod sits in `ImagePullBackOff`. Nothing else is required: no
volumes, no database, no cluster-scoped objects. The committed overlay names
the last deployed digest, so a clean install lands on a known-good revision
rather than on whatever `main` builds to today.

## Secrets and access

Four credentials, and the pipeline is the only thing that holds any of them:

| Credential | Lives in | Can do | Rotation |
|---|---|---|---|
| `github.token` | minted per workflow run | push to GHCR, via `packages: write` on the `build-push` job only | expires when the run ends; nothing to rotate |
| `KUBECONFIG_DEV` | `dev` environment secret | apply to namespace `timesvc-dev` as a service account bound to one Role | quarterly, and on any team change |
| `KUBECONFIG_PROD` | `production` environment secret, behind a required reviewer | apply to namespace `timesvc-prod`, same Role scope | quarterly, and immediately if a run log is ever made public |
| GHCR pull secret | in-cluster Secret per namespace, created out of band | pull the private package | quarterly |

The registry login uses no long-lived password at all: `github.token` is
short-lived and scoped to the one job that publishes. The same move exists for
cloud registries and clusters — `permissions: id-token: write` plus the
provider's auth action exchanges a signed OIDC token for a short-lived
credential, so there is no secret to leak or rotate. Move the kubeconfigs to
that mechanism when the clusters support it.

Never, under any circumstances:

- a credential baked into an image layer — a layer is content-addressed and
  world-readable to anyone who can pull the image, and deleting it in a later
  layer does not remove it;
- a credential in a manifest or an overlay in this repository — `kind: Secret`
  is base64, not encryption, and git remembers it after you delete it;
- a credential echoed by a workflow step: GitHub masks known secret values in
  logs, but it cannot mask what a step decodes or derives from one.

A value committed once is compromised until it is rotated, not until it is
deleted.
