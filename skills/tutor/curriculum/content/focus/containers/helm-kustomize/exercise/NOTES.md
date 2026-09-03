# Notes — packaging timesvc twice

Answer each question on the `A:` line below it, in your own words, in at least
a couple of sentences. `check.sh` checks that you answered and that the
vocabulary is there; your tutor reads what you actually wrote.

## Q1. Which one, and why

You have now packaged the same service both ways. Pick the tool you would use
for (a) an internal service that your team deploys to dev, staging and prod,
and (b) a piece of software you publish for other organisations to install
into clusters you will never see. Justify each choice by what the tool does —
templating versus patching, packaging and distribution, release lifecycle —
not by which one you enjoyed more.

A: TODO

## Q2. Making a configuration change actually reach the pods

`envFrom` values are read once, when the container starts. Explain what
Kustomize's generated ConfigMap name and Helm's checksum annotation each do
about that, why both end up modifying the pod template, and what breaks if you
turn either mechanism off.

A: TODO

## Q3. Reviewing a change before it touches a cluster

Somebody hands you a one-line change to `values.yaml` and a one-line change to
`overlays/prod`. Describe how you would see the manifests those two lines
produce, and how you would see them *as a diff against what is running*, using
the commands from this lesson. Then say what a rollback looks like in each
system — and why "rollback" means something more specific in Helm than in
Kustomize.

A: TODO
