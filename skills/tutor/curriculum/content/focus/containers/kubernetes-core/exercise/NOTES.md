# Notes — timesvc on Kubernetes

Answer each question on the `A:` line below it, in your own words, in at
least a couple of sentences. `check.sh` checks that you answered; your tutor
reads what you actually wrote.

## Q1. Reconciliation

You run `kubectl delete pod timesvc-7d9f-abcde`. Trace what happens next,
naming which controller does which part, and say how many pods exist thirty
seconds later and why. Then answer the other half: you delete the *Deployment*
instead. What happens to its ReplicaSets and pods, and what is different about
that case?

A: TODO

## Q2. Reading a broken pod

One pod is in `ImagePullBackOff`; another is in `CrashLoopBackOff`. For each,
say which single `kubectl` command you run first, what part of its output you
read, and one concrete cause you would expect to find. Then explain why
`kubectl logs --previous` exists and when it is the only useful command.

A: TODO

## Q3. A Service with no endpoints

Every pod is `Running`, but `kubectl get endpointslices` shows your Service
with no addresses. Give two different causes and how you would tell them
apart. Then say why you chose `ClusterIP` here rather than `NodePort` or
`LoadBalancer`, and how you reached the service from your laptop instead.

A: TODO
