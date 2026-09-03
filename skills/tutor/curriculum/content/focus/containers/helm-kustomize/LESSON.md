# Helm & Kustomize

> `focus.containers.helm-kustomize` · ~2.5-3.5h · Stage: Focus: Containers II

## Objectives

By the end of this lesson you can:

- Explain the problem both Helm and Kustomize solve: keeping per-environment
  variants of the same manifests without copy-paste drift.
- Implement a Helm chart for your service, with `values.yaml` parameterizing
  the image tag, the replica count and the resources.
- Implement a Kustomize base plus dev and prod overlays that patch replicas
  and configuration per environment.
- Choose between the two for a given scenario and defend it — templating
  versus patching, packaging and distribution, release lifecycle.
- Render manifests without applying them (`helm template`, `kubectl
  kustomize`) and diff them before a deploy.

The exercise is graded by reading the files you write, so you can finish it with
no cluster, no kubectl, no helm and no kustomize. Both tools are single
binaries, though, and having them turns "I think this renders" into one command.

## Three environments, one truth

Your manifests describe one deployment of timesvc. Reality wants three: dev runs
one pod with debug logging on somebody's laptop cluster, staging runs two, prod
runs four with four times the memory. The obvious move is `cp -r prod dev` and
edit the numbers, and it works for about a month. Then a probe fix lands in
prod's copy and not in dev's, and dev's copy is the one somebody uses as the
template for the next service. Nothing tells you: three directories of legal
YAML, silently disagreeing about what timesvc *is*.

The failure has a name — drift — and both tools in this lesson exist to prevent
it by making one description authoritative and the differences explicit. They
disagree completely about how.

**Helm treats manifests as text to generate.** You write Go templates with holes
in them, `values.yaml` fills the holes, and the result is a versioned,
installable artifact — a *chart* — someone can fetch without your repository.

**Kustomize treats manifests as data to transform.** You write plain Kubernetes
YAML once, as a *base*, and each environment is an *overlay*: a small patch
listing only what differs. What you read is what applies.

Neither is a superset of the other, and the industry has not converged: learning
both takes an afternoon and saves you arguing about the one you happen to know.

## Helm: a chart is an artifact

A chart is a directory with a fixed shape, and Helm finds everything in it by
name — not by configuration:

```
chart/
├── Chart.yaml        # name, version, appVersion, description
├── values.yaml       # the defaults, and the chart's public API
└── templates/        # deployment.yaml, service.yaml, configmap.yaml …
    └── _helpers.tpl  # named template definitions; renders nothing itself
```

`Chart.yaml` carries two versions and they are not the same number: `version`
is the chart's own, bumped when the templates change; `appVersion` is the
software it installs, so a fix to your labels ships as chart 0.1.1 with
`appVersion` untouched. `apiVersion: v2` marks a Helm 3 chart (v1 was Helm 2's).

`values.yaml` deserves the most thought, because it is a **public API**: every
key is a promise somebody's `values-prod.yaml` will set, and removing it in your
next version breaks their install. So put in it exactly what an installer must
be able to change — image, replicas, resources, your service's configuration —
and leave the rest literal. A chart whose values expose all of Kubernetes has no
API at all; it has a fork.

Inside `templates/`, files are Go templates rendered with a context that has
`.Values` (the merged values), `.Chart` (Chart.yaml), and `.Release` (the name
and namespace of *this* installation):

```yaml
spec:
  replicas: {{ .Values.replicaCount }}
  template:
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
```

That last line is the idiom worth learning properly. `toYaml` turns a value from
`values.yaml` back into YAML text; `nindent 12` prepends a newline and indents
every line by twelve spaces so it lands in the right column; and the `{{-` eats
the preceding whitespace, including the line break, so the block sits where the
key is rather than one line below it. One line in the template, and an installer
replaces the entire resources block with the shape they already know.

`_helpers.tpl` holds named templates — the chart's private functions.
`{{ include "timesvc.fullname" . }}` gives the same computed name in the
Deployment, the Service and the ConfigMap. This is not tidiness: a Deployment's
`spec.selector` is **immutable**, so names or selector labels that disagree
between two templates may cost you an uninstall rather than an upgrade.

## template, install, upgrade, rollback

```sh
helm template timesvc ./chart                 # render to stdout; touches nothing
helm install  timesvc ./chart -n timesvc      # render and apply, as release "timesvc"
helm upgrade  timesvc ./chart -f prod.yaml    # re-render with new values and apply
helm rollback timesvc 3                       # put revision 3 back
helm list -n timesvc ; helm history timesvc   # what is installed, and what came before
```

The thing that makes Helm more than a template engine is the **release**: an
installation, under a name, whose every revision Helm stores *in the cluster*
(as a Secret in the release's namespace). `helm upgrade` diffs the rendered
manifests against the last stored revision and patches; `helm rollback 3`
re-applies revision 3 — from the cluster's own record, with no files and no git
history on your side.

It also means Helm holds state you have to respect. Two releases of the same
chart in one namespace are fine *if* every object name is derived from
`.Release.Name`, and a disaster if any name is hard-coded — the second install
adopts and overwrites the first one's objects.

## The Helm pain, honestly

Go templates do not know they are producing YAML. They produce **text**, and
YAML is whitespace-significant, so an `{{- if }}` that eats one newline too many
splices two keys onto one line — and the error, `error converting YAML to JSON:
did not find expected key`, names a line in the *rendered* output you cannot
see. Nothing type-checks values either: `{{ .Values.replicaCont }}` renders
empty and ships a Deployment with `replicas:` and no value, though a misspelled
nested path at least fails loudly with `nil pointer evaluating interface
{}.tag`. And your editor's YAML support is worthless inside a template, because
the file is not YAML.

The discipline that makes this manageable is one command: **`helm template` is
the debugger.** Render before you install, every time, and read the manifests
rather than the templates. `helm install --dry-run --debug` does the same with
the values Helm merged, and `helm upgrade --dry-run` (or the `helm-diff`
plugin's `helm diff upgrade`) diffs against the installed release. The render,
not the template, is the only honest description of what will happen.

## One Helm detail that catches everyone

You learned last lesson that `envFrom` values are snapshotted when the container
starts, so `helm upgrade` with a new `logLevel` patches the ConfigMap and —
because the pod template is byte-identical — changes nothing that is running:
success, no rollout, old value. The standard fix is to hash the rendered config
into the pod template:

```yaml
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
```

The annotation changes when the ConfigMap's content changes, the pod template
changes, the Deployment rolls. Remember this shape — Kustomize solves the same
problem in a completely different way.

## Kustomize: bases and overlays

Kustomize ships inside `kubectl` (`kubectl kustomize`, `kubectl apply -k`) and
also as a standalone binary, and it has no templating language at all. A
directory becomes a *kustomization* by containing `kustomization.yaml`:

```
kustomize/
├── base/       # kustomization.yaml + plain, valid, appliable Kubernetes YAML
└── overlays/
    ├── dev/    # kustomization.yaml + resources-patch.yaml
    └── prod/   # kustomization.yaml + resources-patch.yaml
```

The base holds everything true about the service everywhere, and knows nothing
about environments. Each overlay names the base and lists its differences:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: timesvc-prod          # applied to every object it builds
resources:
  - ../../base                   # the directory, so its kustomization comes too
replicas:
  - name: timesvc                # the typed field for the commonest patch
    count: 4
patches:
  - path: resources-patch.yaml   # a strategic-merge patch file
```

Read top to bottom, that file *is* the review: four lines of difference, no
hidden behaviour, and `kubectl kustomize overlays/prod` proves it.

A patch is a partial manifest — enough identity for kustomize to find the
target, plus the fields that differ:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: timesvc
spec:
  template:
    spec:
      containers:
        - name: timesvc          # the merge key: this identifies the container
          resources:
            limits:
              memory: 1Gi
```

The merge is **strategic**: maps merge key by key, and lists of objects merge by
a known key rather than being replaced — `name` for containers and env entries,
so the patch above changes one limit and leaves the probes, the lifecycle hook
and every other env var alone. Plain JSON merge would replace the whole list,
which is why `containers:` in a strategic patch deletes nothing. When strategic
merge cannot express what you want — removing a list element, say — there is a
JSON 6902 form with explicit `op: remove` paths: precise, unreadable, rare.

The rule that keeps overlays honest: **an overlay may change how much of
something an environment gets — never whether the service works.** Probes,
drain settings, the security context: those live in the base, so a fix reaches
every environment at once. The moment an overlay contains a whole copy of the
Deployment, you have a fork with extra steps.

## Generators, and why a name grows a hash

The other half of Kustomize is generators:

```yaml
configMapGenerator:
  - name: timesvc-config
    literals:
      - PORT=8080
      - LOG_LEVEL=info
      - SHUTDOWN_TIMEOUT=20s
```

This builds a ConfigMap named `timesvc-config-8k2b6c7fmd` — the name plus a hash
of the content — and rewrites every reference to `timesvc-config` in the built
manifests to the hashed name. Change one literal and the hash changes, so the
ConfigMap gets a new name, so the Deployment that references it gets a new pod
template, so the pods roll. That is Helm's checksum problem solved one layer
lower — the config is *part of* the object graph rather than something the pod
template happens to mention — and it buys what the annotation cannot: the old
ConfigMap survives under its old name, so rolling back the manifests rolls back
the configuration with them.

`generatorOptions.disableNameSuffixHash: true` turns it off. People reach for it
because the hashed name is ugly in `kubectl get cm`, and they give up the only
mechanism that makes a config change reach running pods. `secretGenerator` works
the same way — and note it puts the secret's value in a file in your repository,
which is a different problem, and not one Kustomize solves.

An overlay extends a base's generator by repeating the name with `behavior:
merge`: dev overrides `LOG_LEVEL=debug` and keeps the base's PORT and
SHUTDOWN_TIMEOUT. (`replace` silently drops whatever the base adds next month.)

## Choosing, and the honest answer

The question is not "which is better", it is "who changes this, and how do they
get it".

**Helm fits software other people install.** A chart is an artifact: pushed to
a registry, fetched by name and version, installed into clusters you will never
see, parameterized by people who have never read your templates. The whole
lifecycle — install, upgrade, rollback, uninstall — is one tool's job, and
`values.yaml` is a documented API with defaults. This is why nearly everything
you install from a third party arrives as a chart.

**Kustomize fits your own environments.** Three environments differing by six
values do not need a templating language; they need six values written down
somewhere obvious, in plain YAML that your editor, your reviewers and
`kubectl kustomize` all already understand.

The honest cost of each: Helm gives you release state in the cluster and a real
rollback, and charges you that templating pain forever. Kustomize gives you
readability and charges you the release lifecycle — a Kustomize "rollback" is
`git revert` plus apply, which is fine when git is genuinely the record of what
is deployed, and nothing at all when it is not.

They also compose. `helm template` output is just manifests, so a chart you do
not control can become the base of a kustomization, and Helm can call kustomize
as a post-renderer. It is a legitimate move for exactly one situation:
somebody else's chart needs a change their `values.yaml` does not expose.

## Exercise

Open [`exercise/`](exercise/). You get timesvc's manifests from the last lesson
and two half-built packagings of them:

- `chart/` — `Chart.yaml` and `values.yaml` to fill in, `templates/deployment.yaml`
  hard-coded and marked with `TODO`s, `templates/configmap.yaml` empty.
  `_helpers.tpl` and `templates/service.yaml` are written for you — read them
  first; they are the shape everything else should have.
- `kustomize/base/` — the manifests, and a `kustomization.yaml` to write.
- `kustomize/overlays/dev|prod/` — a `kustomization.yaml` and a patch file each.
- `NOTES.md` — three questions to answer in your own words.
- `check.sh` — the referee.

Acceptance criteria — Helm:

1. `chart/Chart.yaml` is `apiVersion: v2` with a name, a one-line description,
   a semantic `version` and a quoted `appVersion`.
2. `values.yaml` declares `replicaCount`; `image.repository`, `image.tag` and
   `image.pullPolicy`; `config.port`, `config.logLevel` and
   `config.shutdownTimeout`; and `resources` with `requests` and `limits` for
   cpu and memory. `image.tag` is a real tag, never `latest`.
3. Every template still reads as YAML once its template actions are removed.
4. Every `.Values.…` path the templates read exists in `values.yaml`.
5. The Deployment template takes `replicas`, the image (repository *and* tag)
   and the whole `resources` block from values — none of the three hard-coded.
6. The rendered Deployment still probes `/readyz` for readiness and `/healthz`
   for liveness.
7. It still sets `terminationGracePeriodSeconds` and a `preStop` sleep, and
   the budget holds: `preStop + config.shutdownTimeout + 5s <=` the grace
   period.
8. `GOMEMLIMIT` renders in Go's syntax (`110MiB`), at least 50% of and below
   `resources.limits.memory`.
9. A ConfigMap template renders `PORT`, `LOG_LEVEL` and `SHUTDOWN_TIMEOUT`
   from `.Values.config`, and the container pulls it in under a name built
   from the same helper the ConfigMap's own name uses.
10. The pod template carries an annotation hashing the rendered ConfigMap with
    `sha256sum`.
11. At least two files in `templates/` `include` a helper defined in
    `_helpers.tpl`.

Acceptance criteria — Kustomize:

12. `base/kustomization.yaml` is `apiVersion: kustomize.config.k8s.io/v1beta1`,
    `kind: Kustomization`, lists **every** manifest in the directory under
    `resources`, and builds at least a Deployment and a Service.
13. The base generates `timesvc-config` with the three keys through
    `configMapGenerator`, keeps the name-suffix hash, and the Deployment
    references the generator's name.
14. `overlays/dev` and `overlays/prod` each list the base **directory** under
    `resources`, not the base's individual files.
15. Nothing in an overlay re-declares the probes or the lifecycle hook, and no
    patch carries more than about 40% of the base Deployment's fields.
16. The two overlays cannot collide: each sets a `namespace` (or a
    `namePrefix`/`nameSuffix`), and they differ.
17. The two overlays render different `replicas` **and** different resource
    limits.
18. Both rendered overlays keep the probes, `terminationGracePeriodSeconds`,
    the `preStop` sleep, a pinned image tag, and a `GOMEMLIMIT` that is still
    below the memory limit that overlay renders.

Acceptance criteria — write-up:

19. `NOTES.md` answers all three questions on the `A:` lines (80+ characters
    each, no `TODO` left).
20. The answers use both tools' vocabulary: templating, patching/overlays,
    values, rendering, the release lifecycle, and the mechanism that makes a
    config change reach running pods.

Run the referee from inside `exercise/`:

```sh
bash check.sh
```

`check.sh` reads your files with its own small YAML reader, because PyYAML is
not guaranteed on your machine. Keep everything inside the subset it
understands — also the subset every `kubectl` example uses: **spaces only,
never a tab; two spaces per level; a space after every colon and after every
`- `; no anchors (`&`), aliases (`*`) or merge keys (`<<:`)**. Flow style
(`{cpu: 100m}`, `[a, b]`) is fine for short values, a sequence may sit at its
key's indentation or one level in, and `---` separates documents in one file.

Two more expectations, because these files are not plain YAML:

- **In the chart**, keep each `{{ … }}` on a single line. The grader removes
  template actions and reads what is left, so a line that is *only* a value
  action is taken as the value of the key above it (exactly what
  `resources:` followed by a `toYaml … | nindent` line means), and control
  lines (`if`, `with`, `range`, `end`, comments) disappear with their line.
  Single `.Values.a.b` lookups are then resolved against `values.yaml`.
- **In the overlays**, reference patches as files — `patches: - path: f.yaml`,
  or the older `patchesStrategicMerge`. The grader does not read inline
  `patch: |` blocks or JSON 6902 operations; if you want one, keep it in
  addition to a file-based patch.

If the reader ever rejects legal YAML, that is an exercise bug, not yours — tell
your tutor. The grade is static: no Docker, no network, no cluster. If `helm` is
on your PATH you also get `helm lint` and a real `helm template`; `kustomize` or
`kubectl` adds a real build of both overlays. Anything missing is `skip`, never
`FAIL`.

## Further reading

- [Helm — Charts](https://helm.sh/docs/topics/charts/) and
  [Chart template guide](https://helm.sh/docs/chart_template_guide/) — the
  values, helpers and whitespace-control sections especially.
- [Helm — Chart best practices: values](https://helm.sh/docs/chart_best_practices/values/)
- [Kustomize — Reference: kustomization.yaml fields](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/)
- [Kubernetes — Declarative management with Kustomize](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)
- [Kubernetes — Update strategy and strategic merge patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/)
