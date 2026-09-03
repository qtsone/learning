# Tutor notes — Containers Capstone

## Where the learner is

The last lesson of the containers pack, and the first one whose subject is not
a tool but a *system*: the path from a merged commit to a verified rollout,
with a rollback and a document for whoever is paged. They have every piece —
image (dockerizing-go), manifests and lifecycle (deploying-go), base and
overlays (helm-kustomize), GitHub Actions from S4's CI/CD lesson — and have
never had to make them agree with each other.

The one new idea is **artifact identity**: a tag is a mutable pointer, a
digest is the image, and almost every failure in this lesson is a version of
"we do not actually know what is deployed". Everything else is composition.

The grade is static. `check.sh` reads the Dockerfile, the workflow, the
manifests and overlays, and RUNBOOK.md; whatever tooling exists on their
machine adds bonus checks (Go build, a real `docker build` and a container
that must report the revision it was built from, `kustomize build`, a
`kubectl` dry run, `actionlint`) and skips otherwise. If they have a cluster,
push them to run the four deploy commands by hand once before writing the
runbook. It is 4-6 hours; it is normal for the workflow alone to take two.

## Common misconceptions

- **"`latest` is fine, we always redeploy anyway."** The specific damage is
  `rollout undo` restoring an identical pod template, so the rollback silently
  does nothing. Make them say that sentence themselves.
- **"The SHA tag and the digest are the same thing."** The tag is an alias
  someone (or a re-run of the same workflow) can move; the digest is content.
  Ask what happens if the build is re-run after a base image update.
- **"Jobs run in order because they are written in order."** They run in
  parallel; `needs` is the only ordering primitive. A missing `needs: test` is
  the commonest real defect in this exercise.
- **"The tests ran on the source, so the image is tested."** Only if nothing
  rebuilds between test and deploy — which is exactly why prod deploys dev's
  digest rather than a fresh build of the same commit.
- **"The deploy is done when `kubectl apply` returns."** It returns when the
  API server has stored an object. Nothing has been pulled, scheduled or
  probed. `rollout status` is the step that makes the job's green mean
  something.
- **"`if: failure()` catches everything."** It catches step failures inside
  that job — which is why the `rollout status` with a `--timeout` matters: a
  rollout that hangs forever fails nothing until the job is cancelled.
- **"Repository secrets are the secure option."** They are the *manageable*
  option. `github.token` is short-lived and scoped; OIDC removes the stored
  credential entirely. Ask what rotating a long-lived registry password across
  four repositories actually costs.
- **"Base64 in a Secret is encryption."** Also: "the secret is only in an
  early layer, I deleted it later." Layers are immutable and pullable.
- **"Git is the record of what is deployed."** Only if a step commits the
  overlay edit back. Most push-based pipelines do not, and the learner has to
  pick a side and write it in the runbook.
- **"The runbook is documentation."** It is an operational tool graded on
  whether commands can be pasted at 02:40 by someone who did not write them.

## Grilling points

- "Point at the exact string that identifies what is running in prod right
  now. Where does it come from, and who else could have changed it?"
- "Your rollback fails. Walk me through what you do, in order, and how you
  know each step worked."
- "The workflow is green and prod is serving the old version. Give me three
  ways that happens." (Rollout not waited on; `:latest`/stale tag; deploy job
  pointed at the wrong overlay or namespace; image pull failure that no step
  checked.)
- "Why does prod deploy dev's digest instead of rebuilding the same commit?
  What could differ between two builds of one commit?"
- "Someone leaks a build log containing your kubeconfig. What is the blast
  radius, and what would OIDC have changed?"
- "Delete the `needs:` from `deploy-prod`. What still works, and what is now
  a lie?"
- "Your image is `sha-abc1234` and `/version` says `dev`. What happened?"
  (Build arg not passed, or `-ldflags` not wired, or a stale image.)
- Stretch: "Your service grows a database migration. Which section of the
  runbook becomes wrong, and what would you add to the pipeline?"
- Stretch: "How would this pipeline change if you had twelve services instead
  of one?" (Reusable workflows, a shared chart, or the pull-based shape — let
  them reason, do not lecture.)

## Grading rubric

- **A** — All static checks pass; the workflow reads like something a
  colleague could take over, with least-privilege permissions and the digest
  threaded from build to both deploys; the runbook is specific (real
  namespaces, real commands, a stated position on git-versus-cluster); they
  can explain artifact identity and the rollback mechanism without prompting.
  Bonus: they ran it against a real cluster or registry and can say what broke
  first.
- **B** — Checks pass but one layer is mechanical: permissions granted
  workflow-wide, the deploy naming a tag when the digest was available, or a
  runbook that is correct and generic. One round of grilling fixes it.
- **C** — Checks pass only after heavy hinting, or the rollback story is still
  "re-run the pipeline on the old commit" with no awareness that this rebuilds
  rather than restores. Pass only if a re-explanation lands in their own
  words.
- **Fail** — Checks failing; or `latest` still deployed anywhere; or they
  cannot say what `rollout status` adds over `apply`. This is the pack's
  capstone: remediate rather than advance.

## Remediation ladder

1. "Read the FAIL line and its `fix:` aloud. Which acceptance criterion is
   that, and which file does it live in?"
2. For the workflow: "Draw the four jobs as boxes and put an arrow for every
   `needs`. Now cover the arrows — which boxes could start at the same
   moment?"
3. For the digest thread: "Trace the commit from `github.sha` to a running
   pod. Name every hop." (Build arg → `-ldflags` → binary; tag/digest →
   overlay → pod template → `/version`.) Whichever hop they cannot name is
   the broken one.
4. For the runbook: "Hand it to me. I have never seen this service and it is
   02:40 — I will do exactly what it says, literally." Read it back at them.
5. Give the shape of the one block they are stuck on — the `outputs:` stanza,
   or the `if: failure()` step — never the whole workflow.

## After passing

Preview: "You can build, ship and roll back a containerized Go service, which
is the whole pack. From here the packs branch: web services goes deeper into
what the service *does*, and the systems stage into how many of them fit
together."
