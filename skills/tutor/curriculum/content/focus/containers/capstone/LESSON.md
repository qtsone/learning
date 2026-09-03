# Containers Capstone

> `focus.containers.capstone` · ~4-6h · Stage: Focus: Containers II

## Objectives

By the end of this lesson you can:

- Implement an end-to-end pipeline that builds your Go service image, tags it,
  pushes it to a registry, and deploys it to a cluster.
- Choose an image tagging strategy — immutable digests and commit SHAs versus
  mutable tags like `latest` — and justify it for rollbacks and reproducibility.
- Deploy the full stack with the configuration, probes, resources and graceful
  shutdown you carried through this pack.
- Perform and verify a rollback after a bad release, using the deployment
  history the platform already keeps for you.
- Explain each pipeline stage and its failure modes, and show the pipeline is
  repeatable from a clean state.

The exercise is graded by reading the files you write, so you can finish it
with no cluster, no registry, no kubectl and no Docker daemon — `check.sh`
says which checks it skipped. If you do have a cluster, run the pipeline's
commands by hand once: a runbook written by someone who never typed them reads
exactly like one.

## What is still missing

You can build a small, non-root image; write manifests that probe correctly,
size the Go runtime to its limits and drain without dropping requests; express
three environments as a base and two overlays. Every one of those is a thing
*you* do. What you do not have is the path between them that runs without you:
something that takes a merged commit and, with nobody watching, produces an
artifact, records what it is, puts it in front of dev, then puts *the same
one* in front of prod — and that can be reversed by a person who was not
involved and does not know the code.

That path is this lesson, and almost all of its difficulty is one question:
**what, exactly, is the thing that got deployed?** Tagging, rollback,
verification and the runbook are all downstream of naming the artifact and
meaning it.

## Naming the artifact

An image reference has two halves: a repository (`ghcr.io/ada/timesvc`) and
either a tag (`:v1.4.0`) or a digest (`@sha256:…`).

A **digest** is the content hash of the image manifest — the image's identity.
Two digests are equal exactly when the images are byte-identical, and nobody,
you included, can make one point somewhere else. A **tag** is a mutable
pointer the registry maintains: `:v1.4.0` means whatever was pushed last under
that name. Tags are for humans; digests are for machines and for arguments
about what was running at 02:40.

`latest` is the pathological case, and its damage is specific. A Deployment
stores its image reference in the pod template, and `kubectl rollout undo`
works by restoring the *previous* pod template. If revisions 7 and 8 both say
`image: ghcr.io/ada/timesvc:latest`, those templates are identical: the undo
restores something byte-for-byte equal to what is running, the Deployment sees
no change, and your rollback succeeds without doing anything. Worse, the tag
has moved, so re-pulling it fetches the *bad* image — and under
`imagePullPolicy: IfNotPresent`, nodes with the old bytes cached and nodes
pulling fresh run different code under one name.

So pipelines tag with something derived from the commit — `sha-<git sha>` —
and deploy either that tag or, better, the digest the registry returned on
push. The tag is the alias you read in `kubectl get pods -o wide`; the digest
is the fact. It is also why the exercise's manifests carry a bootstrap tag
like `sha-0000000`: the value is wrong, but it is *shaped* right, and nothing
about it silently no-ops.

## Tying the artifact back to source

A tag tells you what was deployed. Two more links tell you what is *inside*
it, and they cost one line each:

**Into the binary.** The linker can write into a package-level string:

```dockerfile
ARG REVISION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath -ldflags="-s -w -X main.buildRevision=${REVISION}" -o /out/timesvc .
```

`-X importpath.name=value` sets `var buildRevision string` at link time, so
the process can report its own commit — `GET /version` in `main.go` does
exactly that, and "which revision is this?" becomes a curl.

**Onto the image.** OCI labels are metadata a registry, a scanner and
`docker inspect` can read without running anything:

```dockerfile
LABEL org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.source="https://github.com/ada/timesvc"
```

One Docker rule bites everyone here: **`ARG` does not cross `FROM`.** It is
scoped to the stage that declares it, so a Dockerfile that wants the revision
in both the build command and the final stage's label declares `ARG REVISION`
twice. Miss the second and the label renders empty, with no error.

## The pipeline, job by job

GitHub Actions from the CI/CD lesson in S4, now doing the whole job. Four
jobs, each a gate for the next:

```yaml
jobs:
  test:         # go test ./...
  build-push:   # needs: test          — build, tag, push; output the digest
  deploy-dev:   # needs: build-push    — deploy that digest, wait, verify
  deploy-prod:  # needs: deploy-dev    — same digest, gated on a human
```

Three details carry most of the weight. **Jobs run in parallel unless `needs`
says otherwise.** A `build-push` without
`needs: test` publishes an image while the tests are still running — an
automated way to ship a bad build faster than you could by hand. `needs` is
the only ordering primitive; steps within a job are sequential.

**Outputs are how jobs speak.** Each job runs on its own machine with its own
filesystem, so `build-push` exposes the digest it pushed as a job output and
the deploy jobs read `${{ needs.build-push.outputs.digest }}`. Without it, a
deploy job can only say "the tag for this commit" — *probably* the same image,
and not the same statement.

**Triggers are two, not one.** `push: branches: [main]` makes a release a
consequence of merging rather than something someone remembers to do.
`workflow_dispatch` lets you start the same pipeline by hand during an
incident, from the last good commit, without inventing a revert commit first.

## Credentials, and the two ways to get them wrong

Every job gets a `GITHUB_TOKEN` minted for that run and expiring with it. It
starts read-only, and you widen it where it is needed:

```yaml
permissions:
  contents: read          # the workflow-wide default
jobs:
  build-push:
    permissions:
      contents: read
      packages: write     # only this job may publish
```

A workflow-level `packages: write` hands every step in every job — including
every third-party action you pulled in — a token that can publish to your
registry. Grant it on the job that publishes.

For GHCR that token *is* the credential: `docker/login-action` with
`password: ${{ github.token }}` and no stored secret at all. Elsewhere the
modern equivalent is **OIDC**: with `permissions: id-token: write` the runner
requests a signed token proving "this is workflow X in repo Y on branch main",
and the provider exchanges it for a credential that lives minutes. The trust
relationship is configured once on their side, and there is no long-lived key
in your repository to leak, rotate, or find in a log three years later. Stored
registry passwords still work everywhere — and are what OIDC exists to replace.

The failures worth naming out loud:

- **A secret baked into an image.** Layers are content-addressed and readable
  by anyone who can pull the image; `RUN rm /root/.netrc` in a later layer
  does not remove it from the earlier one. Build-time secrets need
  `RUN --mount=type=secret`, which never lands in a layer.
- **A secret in a manifest.** `kind: Secret` is base64, not encryption, and
  deleting a committed one does not un-commit it. Cluster secrets are created
  out of band, or by a controller reading a real secret store.
- **A secret a step prints.** Actions masks known secret values in logs, but
  cannot mask something a step decoded, re-encoded or derived.

## Promotion: one artifact, moving forward

Promotion is the discipline of deploying **the same artifact** to dev and then
to prod. Not the same commit — the same digest. Rebuilding for prod produces a
different image: different base-layer contents if upstream moved, different
build cache, different timestamps. Whatever dev proved, it did not prove that.

The mechanism is the overlay from the previous lesson: the deploy job points
an environment's overlay at this run's artifact and applies it.

```sh
cd deploy/overlays/prod
kustomize edit set image ghcr.io/ada/timesvc@${{ needs.build-push.outputs.digest }}
kubectl apply -k .
```

Everything else that differs between environments — replicas, resources,
namespace, log level — is already in the overlay, so the pipeline changes
exactly one thing per deploy. (With Helm it is `helm upgrade --install timesvc
./chart -f values-prod.yaml --set image.digest=…`; the argument changes shape,
not the idea.)

Prod gets one thing dev does not: a **GitHub environment**.

```yaml
  deploy-prod:
    needs: [build-push, deploy-dev]
    environment: production
```

An environment is where per-environment secrets live and where a required
reviewer is configured. Naming one is the difference between a pipeline that
*can* deploy to production and one that *may* — and the approval is recorded
against the run, which is half of what an audit asks for.

## A deploy you did not verify is a hope

`kubectl apply` returns as soon as the API server has stored the object. It
has not scheduled a pod, pulled an image or run a probe. A job that ends there
is green because a write succeeded. So every deploy job ends with the checks:

```sh
kubectl -n timesvc-prod rollout status deployment/timesvc --timeout=300s
kubectl -n timesvc-prod get deploy timesvc -o jsonpath='{.spec.template.spec.containers[0].image}'
kubectl -n timesvc-prod run verify --rm -i --restart=Never \
  --image=curlimages/curl:8.11.1 -- -sS http://timesvc/version
```

`rollout status` blocks until every replica is updated and available, and
exits non-zero if the rollout stalls — which the Deployment decides via
`progressDeadlineSeconds` (600 by default). Give it a `--timeout` shorter than
the job's, or a bad release ties up a runner for six hours before anyone hears
about it. Then check *what* is running, because a finished rollout of the
wrong image is also a finished rollout: the second command says what the
Deployment asks for, the third what a pod behind the Service actually answers
— which is where the revision you stamped into the binary earns its keep.

And the undo belongs in the pipeline, not in someone's memory:

```yaml
      - name: Undo
        if: failure()
        run: |
          kubectl -n timesvc-prod rollout undo deployment/timesvc
          kubectl -n timesvc-prod rollout status deployment/timesvc --timeout=300s
```

`if: failure()` runs the step only when an earlier step in the job failed —
which, thanks to `rollout status`, includes "the new pods never became ready".
The undo restores the previous ReplicaSet's pod template (kept because of
`revisionHistoryLimit`), which is the whole reason the image reference had to
be immutable. And the undo is a deploy like any other: it gets its own
`rollout status`. This does not replace a human; it keeps the blast radius
small in the ten minutes before the human arrives.

## Where the repository and the cluster disagree

The deploy job edits the overlay *on the runner* and applies it. Unless a step
commits that edit back, the file in git still names yesterday's digest, and
git is not the record of what is running — the workflow run is. Both answers
are defensible; not knowing which you picked is not. Either commit the edit
back (a bot commit after a successful deploy, with the workflow's own paths
excluded from its trigger so a deploy cannot start another deploy), or accept
git as a bootstrap value and have the runbook say plainly that the last green
run is authoritative and where to read its digest. Say which in RUNBOOK.md;
nobody can derive it from the files. (A third shape exists, where nothing
pushes and an agent inside the cluster pulls the desired state from a
repository. Those tools are worth a weekend after this pack; they change the
answer here, not the question.) One wart either way: `kustomize edit` rewrites
the file it touches and drops your comments.

## The runbook

The last artifact is a document, and it is both the most likely to be skipped
and the most likely to be needed. A runbook is written for someone who is not
you, at an hour you would not choose, with your pipeline half-broken:
**commands, not prose** — "roll back the deployment" is not an entry, the
command with the namespace filled in is; **verification always**, because at
02:40 nobody can tell success from a silent no-op; **a clean-state path**, the
section you use once when a cluster is gone; and **what it does not do** — a
rollback of an image is not a rollback of a schema change, and writing that
down before it is true is far cheaper than after.

## Exercise

Open [`exercise/`](exercise/). It is the whole service, ready to ship and not
shipping:

- `main.go` — finished, do not change it. It reports `buildRevision` on
  `GET /version`; nothing yet sets it.
- `Dockerfile` — the image from dockerizing-go minus what a pipeline needs
  from it: three `TODO`s.
- `.github/workflows/release.yml` — four jobs named, almost nothing in them;
  seven `TODO`s, each a decision rather than a transcription.
- `deploy/base/` — the manifests you earned in deploying-go, plus a
  kustomization with a `configMapGenerator`. Do not weaken them.
  `deploy/overlays/dev/` is complete, the worked example;
  `deploy/overlays/prod/` is yours to write.
- `RUNBOOK.md` — six headings, every body a `TODO`. `check.sh` — the referee.

Acceptance criteria — the image:

1. The Dockerfile is multi-stage, every base image is pinned (a tag that is
   not `latest`, or a digest), and the final stage is not a `golang` image,
   gets the binary by `COPY --from=`, runs no `go` command, and starts it with
   an exec-form `ENTRYPOINT`. The build stage sets `CGO_ENABLED=0`.
2. The final stage declares a non-root `USER` (numeric if the base is
   `scratch`).
3. A build `ARG` whose name mentions the revision (e.g. `REVISION`) reaches
   `go build` as `-ldflags "… -X main.buildRevision=$ARG"`.
4. The final stage carries `LABEL org.opencontainers.image.revision` set from
   that same `ARG` — not from a fixed string.

Acceptance criteria — the pipeline:

5. `.github/workflows/` holds a workflow that parses, triggered by `push` to
   `main` **and** by `workflow_dispatch`.
6. A job runs `go test`, and the publishing job depends on it — directly or
   transitively.
7. The publishing job logs in to the registry (`docker/login-action` or
   equivalent) and declares its own `permissions`, including `packages: write`
   (or `id-token: write` for OIDC to a cloud registry).
8. The image is tagged from `github.sha` — a literal `sha-${{ github.sha }}`
   tag, or `docker/metadata-action` with `type=sha` — and `:latest` is not the
   only tag pushed.
9. The build passes the commit into the image through the `ARG` from criterion
   3 (`build-args:` or `--build-arg`).
10. One job deploys `deploy/overlays/dev`, another `deploy/overlays/prod`, and
    each points its overlay at *this run's* artifact: a line containing
    `kustomize edit set image` (or `kubectl set image`, or `helm upgrade`)
    naming `github.sha` or a `needs.<job>.outputs.…` value, never `:latest`.
11. The dev job depends on the build, the prod job depends on the dev job, and
    the prod job names an `environment:`.
12. Both deploy jobs wait — `kubectl rollout status` (or `helm upgrade
    --wait`) — and both have an undo: a step with `if: failure()` running
    `kubectl rollout undo` or `helm rollback` (`helm upgrade --atomic` counts).
13. Nothing in the Dockerfile, the workflow, the manifests or RUNBOOK.md
    assigns a credential a literal value, and `deploy/` commits no
    `kind: Secret` object. `${{ secrets.NAME }}` and `${{ github.token }}` are
    what a credential looks like here.

Acceptance criteria — the manifests:

14. The base Deployment keeps its contract: readiness on `/readyz`, liveness
    on `/healthz` (and not the same path), a `preStop` sleep of at least 5s,
    an explicit `terminationGracePeriodSeconds` that still satisfies
    `preStop + SHUTDOWN_TIMEOUT + 5s <= grace period`, and
    `maxUnavailable: 0`.
15. The base still declares `requests` and `limits` for cpu and memory,
    `GOMAXPROCS` agreeing with the CPU limit (literal or `resourceFieldRef` on
    `limits.cpu`), and a `GOMEMLIMIT` in Go's syntax that is at least 50% of
    and strictly below `limits.memory`.
16. Both overlays list the base **directory** under `resources` and no
    individual manifest file.
17. The two overlays set different namespaces and different replica counts,
    and prod runs at least 2.
18. Each overlay's `images:` entry pins the service image to a tag nothing can
    move — a digest, a `sha-`/commit-shaped tag, or a semantic version — never
    `latest`.
19. Each overlay patches its own `resources`, and any patch that changes
    `limits.memory` also sets `GOMEMLIMIT` to at least 50% of and below that
    new limit.

Acceptance criteria — the runbook. All six `##` headings survive, and each
section holds real prose (no `TODO`, at least 150 characters), and:

20. "How a change reaches production" names what identifies a release — the
    commit SHA tag, the digest, or immutability itself.
21. "Verify a release" contains `rollout status` (or `helm status`) *and*
    something that checks what is running: `/version`, the revision,
    `/readyz`, or `get pods`.
22. "Roll back" contains a pasteable command (`rollout undo`, `helm rollback`,
    `kustomize edit set image`, `kubectl set image`) *and* verifies the result.
23. "From a clean state" mentions the namespace, the apply (`kubectl apply`,
    `kustomize build` or `helm install`) and how the image is obtained.
24. "Secrets and access" names the credentials, and says both what must never
    happen to them and that they are rotatable.

Run the referee from inside `exercise/`:

```sh
bash check.sh
```

`check.sh` reads your files with its own small YAML reader, because PyYAML is
not guaranteed on your machine. Keep the workflow and the manifests inside the
subset it understands — also the subset every example you have read uses:
**spaces only, never a tab (not even inside a `run:` block); two spaces per
level; a space after every colon and after every `- `; no anchors (`&`),
aliases (`*`) or merge keys (`<<:`)**. Flow style (`[main]`, `{cpu: 100m}`) is
fine for short values, a sequence may sit at its key's indentation or one
level in, `---` separates documents, and `run: |` blocks are read as text.
Reference overlay patches as files (`patches: - path: f.yaml`); the reader
does not follow inline `patch: |` blocks. If it ever rejects legal YAML, that
is an exercise bug, not yours — tell your tutor.

The grade is static: no cluster, no registry, no network, no root. Tooling
that happens to be installed adds bonus checks — a Go build, a real `docker
build` plus a container that must report the revision it was built from, a
`kustomize build` of both overlays, a `kubectl` dry run, `actionlint` — each
`skip` when its tool or daemon is missing, never `FAIL`.

## Further reading

- [GitHub Actions — Workflow syntax](https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions): `needs`, `outputs`, `permissions`, `environment`
- [GitHub Actions — About security hardening with OpenID Connect](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- [GitHub Packages — Working with the Container registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [OCI — Annotations (`org.opencontainers.image.*`)](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
- [Kubernetes — Rolling back a Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-back-a-deployment)
- [Google SRE Workbook — On-call and playbooks](https://sre.google/workbook/on-call/)
