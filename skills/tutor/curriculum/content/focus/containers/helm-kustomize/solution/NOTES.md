# Notes — packaging timesvc twice

Reference answers. Yours will differ in wording; what matters is that the
trade-off is argued from what the tools do.

## Q1. Which one, and why

A: For our own service across dev, staging and prod I would use Kustomize.
The environments differ by four numbers, and a patch file states those four
numbers in plain Kubernetes YAML that anybody on the team can read without
learning a templating language — what I read in `overlays/prod` is what gets
applied, and `kubectl kustomize` proves it in one command. Templating would be
a tax paid on every future edit for parameterisation nobody outside the team
needs. For software published for other organisations to install, I would ship
a Helm chart: `values.yaml` is a documented API with defaults, the chart is a
versioned artifact people can fetch from a repository by name and version, and
`helm install` / `helm upgrade` / `helm rollback` give installers a lifecycle
that does not require them to have my git repository at all. Kustomize has no
answer to "install version 1.4.2 of your thing", because it has no artifact.

## Q2. Making a configuration change actually reach the pods

A: Environment variables from `envFrom` are snapshotted when the container is
created, so editing the ConfigMap alone leaves every running pod on the old
values. Kustomize solves it by generating the ConfigMap with a hash of its
content appended to the name and rewriting every reference to that name, so a
changed literal produces a new object name, a changed pod template and
therefore a rollout — and it leaves the old ConfigMap behind, which is what
makes rolling back to the previous manifests actually work. Helm has no
generator, so the chart hashes the rendered ConfigMap into a
`checksum/config` pod annotation: the annotation changes, the pod template
changes, the Deployment rolls. Turn either off — `disableNameSuffixHash: true`,
or drop the annotation — and `helm upgrade` or `kubectl apply` will report
success, the ConfigMap will hold the new value, and the pods will keep serving
the old one until something unrelated restarts them, which is the worst kind
of bug because nothing looks broken.

## Q3. Reviewing a change before it touches a cluster

A: Render first, apply second. For the chart, `helm template timesvc chart -f
prod-values.yaml` prints the manifests without contacting a cluster, and
`helm diff upgrade` (the diff plugin) or `helm upgrade --dry-run` compares
them with the release that is installed. For the overlay, `kubectl kustomize
overlays/prod` prints the built manifests, and `kubectl diff -k overlays/prod`
shows them against the live objects. Both let me review the change as
manifests, which is the only form that tells me what the cluster will do — a
one-line values change can move a great deal of rendered YAML, and reading the
diff is how whitespace and indentation mistakes in a template get caught
before they become a failed apply. A Helm rollback is a specific thing: Helm
stores each release revision in the cluster, so `helm rollback timesvc 4`
re-applies revision 4 without me having any files at all. Kustomize has no
release history, so a rollback means checking out the previous commit and
building and applying it again — which works, and depends entirely on git
being the record of what is deployed.
