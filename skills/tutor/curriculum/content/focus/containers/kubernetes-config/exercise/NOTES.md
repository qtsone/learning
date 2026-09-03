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

A: TODO

## Q2. What a Secret is and is not

You wrote `API_KEY` into a Secret. Somebody with `kubectl get secret
timesvc-secrets -o yaml` in that namespace sees base64. Explain what base64
buys you, what it does not, and name two mechanisms that actually protect the
value — one that protects it in the cluster and one that keeps it out of your
git history.

A: TODO

## Q3. The probe you must not confuse

Imagine your database is unreachable for forty seconds and all six replicas
notice. Trace what happens if the liveness probe hits `/readyz` (the
dependency-aware endpoint), and then what happens if only the readiness probe
does. Say what the user sees in each case, and what state the service is in
when the database comes back.

A: TODO
