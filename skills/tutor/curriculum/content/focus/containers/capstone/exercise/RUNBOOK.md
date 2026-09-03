# Runbook — timesvc

The document someone who is not you reads at 03:00, with your pipeline
half-broken and no context. Write it for them: short, specific, and full of
commands they can paste.

Keep the six `##` headings exactly as they are — `check.sh` looks for them —
and replace every TODO with prose of your own. Real values, not placeholders:
your namespaces, your image repository, your commands.

## What this service is

TODO: what timesvc does, what it depends on, what its endpoints are, and what
"healthy" looks like from outside. Include the two probe endpoints and what
each one means, so nobody points a load balancer at the wrong one.

## How a change reaches production

TODO: the path from a merged commit to running pods, job by job. Name the
tagging strategy and say what identifies the artifact — and why that identity
has to be immutable.

## Verify a release

TODO: the commands that prove the release actually landed, and what each one
proves. "The workflow was green" is not verification.

## Roll back

TODO: the exact commands, in order, for getting production back onto the last
good revision — and the check that says it worked. Say how long it takes and
what it does *not* undo.

## From a clean state

TODO: everything needed to stand this service up in an empty cluster, from
nothing but the repository. Someone should be able to follow it after a
cluster is lost.

## Secrets and access

TODO: every credential the pipeline uses, where it lives, what it can do, and
how it is rotated. Say explicitly what must never appear in the image, the
manifests or the workflow file.
