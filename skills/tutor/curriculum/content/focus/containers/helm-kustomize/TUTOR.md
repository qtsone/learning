# Tutor notes — Helm & Kustomize

## Where the learner is

Seventh lesson of the containers pack. They can write a Deployment, a Service,
a ConfigMap, probes and resources by hand, and last lesson they made timesvc
survive a rollout: readiness first, `preStop` sleep, drain inside the grace
period, `GOMAXPROCS` and `GOMEMLIMIT` matched to the limits. Every one of those
decisions now has to survive being *packaged* — that is the whole point of the
exercise, and where most of the FAIL lines land.

This is the first lesson in the pack with no single right answer. Two tools,
two philosophies, and the industry has genuinely not converged; a learner who
leaves with "Helm is the real one" or "Kustomize is the simple one" has learned
a habit, not a decision. Push on the *why* far harder than the *what* — both
sets of files are mechanical once they see the shape.

The grade is static: `check.sh` reads the chart, the base, both overlays and
`NOTES.md` with its own small YAML reader, plus a template-stripper for the
chart. No cluster, kubectl, helm or kustomize is required. If `helm` or
`kustomize`/`kubectl` happens to be installed, `helm lint`, `helm template` and
a real overlay build run as a bonus and are reported `skip` when absent. If
they have helm, get them to run `helm template timesvc chart` and read the
output — the lesson claims it is the debugger and the claim should be tested.

Two grader conventions worth repeating if they get stuck on parsing rather than
on the idea: keep each `{{ … }}` on one line, and reference patches as files
(`patches: - path: f.yaml`), not inline `patch: |` blocks. Both are stated in
LESSON.md; a learner failed by the reader on legal YAML has found an exercise
bug, and you should say so rather than making them work around it.

They have not done the capstone yet, so keep CI, registries and image promotion
out of the conversation — the next lesson is where the tag they pin here starts
coming from a git SHA.

## Common misconceptions

- **"Helm and Kustomize do the same thing, one is nicer."** They do not.
  Templating generates text; patching transforms data. Helm additionally owns a
  release lifecycle *in the cluster*; Kustomize owns nothing at runtime.
- **"Kustomize is Helm without the templates."** It is Helm without the
  *artifact*: no version, no repository, no `helm rollback`. That is the real
  cost of choosing it, and the honest answer names it.
- **"`values.yaml` should expose everything, to be flexible."** A chart whose
  API is all of Kubernetes has no API. Every key is a promise to the next
  version; watch for a values file with thirty keys and no defaults thought
  through.
- **"A rendered template is YAML, so the tooling will catch my mistake."**
  Nothing type-checks a chart. `{{ .Values.replicaCont }}` renders empty and
  ships `replicas:` with no value. This is the misconception the lesson's
  "Helm pain" section exists to break.
- **"`helm install --dry-run` is the same as `helm template`."** Close, but
  `--dry-run` talks to a cluster and applies its values merge and capabilities;
  `helm template` is offline. Either is fine as the habit — *not rendering* is
  what is not fine.
- **"An overlay is where environment stuff goes, so probes can live there
  too."** An overlay changes how *much* of something an environment gets, never
  whether the service works. A probe in an overlay is a fix that will reach one
  environment.
- **"Copying the base Deployment into the overlay and editing it is a patch."**
  It is a fork with extra steps. The grader measures patch size against the base
  for exactly this reason.
- **"The generator's hashed name is ugly, so I disabled the suffix."** They have
  then given up the only mechanism that makes a config change reach running
  pods — and last lesson is what tells them why (env vars are snapshotted at
  container creation).
- **"The checksum annotation is a Helm feature."** It is a convention: a hash
  written into the pod template by hand, because Helm has no generator. Ask what
  it would take to forget it. (Nothing. That is the point.)
- **"Strategic merge replaces the containers list."** It merges by `name`, which
  is why a two-field patch does not delete the probes. If they believe otherwise
  they will over-copy defensively.
- **"Two releases of one chart in a namespace is fine."** Only if every object
  name derives from `.Release.Name`. A hard-coded ConfigMap name means the
  second install adopts and overwrites the first one's object.
- **"`helm rollback` reads my git history."** It reads release revisions stored
  as Secrets in the cluster. Kustomize's "rollback" is the git one — and only
  works if git is genuinely the record of what is deployed.

## Grilling points

- "You have both packagings in front of you. Which do you ship for an internal
  service, and which for software other organisations install? Argue it from
  what the tool *does*, not what you enjoyed."
- "Somebody changes `LOG_LEVEL` and nothing else. Trace what reaches the running
  pods in each system, and name the mechanism." (Hash suffix; checksum
  annotation. Both work by changing the pod template.)
- "Delete the `checksum/config` annotation from your chart. What does
  `helm upgrade` report, and what is actually running afterwards?" (Success, and
  the old value. The worst kind of bug: nothing looks broken.)
- "Your prod patch raises `limits.memory` to 1Gi. What else must move, and what
  happens at 3 a.m. if it doesn't?" (`GOMEMLIMIT`; OOMKill at the old number.)
- "Why does `resources:` in the chart get `toYaml … | nindent 12` instead of six
  templated lines?" (One key is the installer's API; six lines are six promises,
  and the indentation is the part they'd get wrong.)
- "What is `version` in `Chart.yaml` for, and when does it change without
  `appVersion` changing?"
- "What does the `-` in `{{-` do, and what is the error message when you get it
  wrong?" (Eats preceding whitespace including the newline; `error converting
  YAML to JSON: did not find expected key`, pointing at a line in output they
  cannot see without rendering.)
- "Both overlays build. How do you review the change before it touches a
  cluster, and how do you see it *as a diff against what is running*?"
- "Your dev overlay sets no namespace. What happens the first time somebody
  applies dev and prod to one cluster?"
- Stretch: "When would you use both together?" (Somebody else's chart needs a
  change their `values.yaml` does not expose: `helm template` into a base, or
  kustomize as a post-renderer. Legitimate, and a smell if it is your own
  chart.)
- Stretch: "Your chart is installed in forty clusters and you want to remove a
  values key. What is the migration?" (There isn't a cheap one — deprecate,
  keep reading the old path, bump the major. This is what "values.yaml is an
  API" costs.)
- Stretch: "`kustomize` has no release history. Under what conditions is that
  actually fine, and how would you know your team does not meet them?"

## Grading rubric

- **A** — All static checks pass; both packagings preserve the probes, the
  shutdown budget and the `GOMEMLIMIT`/limit relationship; the overlays are
  genuinely small; `NOTES.md` argues both tools in both directions rather than
  restating the lesson. They can explain the config-change mechanism in each
  system unprompted, and say what `helm template` is for. Bonus: they ran
  `helm template` or `kubectl kustomize` and read the output.
- **B** — Checks pass, but one half is mechanical: values keys added to satisfy
  the grader with no view on what belongs in an API, or a prod patch that moved
  `GOMEMLIMIT` because the checker said so. One round of grilling fixes it.
- **C** — Checks pass only after heavy hinting, or the choice question in
  `NOTES.md` comes out as a preference ("Helm is standard"). Pass only if a
  re-argument in their own words lands, using the other tool's vocabulary.
- **Fail** — Checks failing; or an overlay carrying a copy of the Deployment; or
  they cannot say why a ConfigMap edit alone leaves running pods untouched —
  that one is last lesson's material and the capstone depends on it.

## Remediation ladder

1. "Read the FAIL line and its `fix:` aloud. Which acceptance criterion is that,
   and which single file does it live in?" Every message names the file.
2. For the chart: "Open `templates/service.yaml`. It is written for you and it
   is the shape every template has. What does it take from a helper, and what
   did it leave literal?" Then: "Do that to one `TODO` in `deployment.yaml`."
3. For values: "Read your `values.yaml` as if you were installing somebody
   else's chart. Which key would you need that isn't there, and which one is
   there that you'd never touch?"
4. For the overlays: "Print the base Deployment and your patch side by side.
   Cross out every line in the patch that also appears in the base with the same
   value. What is left is the patch." (For the identity lines they must keep —
   `apiVersion`, `kind`, `metadata.name`, the container's `name` — say why:
   kustomize has to find the target and the list entry.)
5. For the generator: "Change one literal in the base's `configMapGenerator`.
   What is the ConfigMap's name now, and which other object changed as a
   result?" If they have kustomize, have them run the build twice and diff.
6. Give the shape of the one block they are stuck on — the `toYaml`/`nindent`
   line, or the `configMapGenerator` stanza — never a whole file, and never
   both packagings at once.

## After passing

Preview: "You can package it two ways. Next: getting it built, tagged, pushed
and promoted from dev to prod without a human running any of those commands —
the pack capstone."
