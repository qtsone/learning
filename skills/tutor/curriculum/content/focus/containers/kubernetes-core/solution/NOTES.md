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

A: The API server records the deletion and the kubelet on that node stops the container; the endpoint controller drops the pod's IP from the Service's EndpointSlice, so it stops receiving traffic. The ReplicaSet controller then compares desired (3) with actual (2), creates a replacement from the pod template, the scheduler assigns it a node, and that node's kubelet starts it — so thirty seconds later there are 3 pods again, one with a new name and a new IP. Nothing coordinated that: four independent loops each closed their own gap. Deleting the Deployment is different because I removed the desired state itself: garbage collection follows the owner references down to the ReplicaSets and their pods and deletes all of them, and nothing recreates anything, because there is no longer a spec saying it should exist.

## Q2. Reading a broken pod

One pod is in `ImagePullBackOff`; another is in `CrashLoopBackOff`. For each,
say which single `kubectl` command you run first, what part of its output you
read, and one concrete cause you would expect to find. Then explain why
`kubectl logs --previous` exists and when it is the only useful command.

A: For ImagePullBackOff I run `kubectl describe pod <name>` and read the Events at the bottom, where the kubelet names the registry and the exact pull error — with kind, the usual cause is that the image only exists in my local Docker daemon and was never loaded into the cluster's node, which `kind load docker-image timesvc:1.0.0` fixes. For CrashLoopBackOff I also start with describe (to see the exit code and how many restarts), but the answer is in `kubectl logs <pod>`: the container starts and exits, so this is my process dying, typically because a required environment variable is missing and startup validation rejects it. `--previous` exists because the kubelet has already replaced the dead container: the current one may have produced nothing yet, and the output that explains the crash belongs to the instance that just died, so during a crash loop it is often the only command that shows anything.

## Q3. A Service with no endpoints

Every pod is `Running`, but `kubectl get endpointslices` shows your Service
with no addresses. Give two different causes and how you would tell them
apart. Then say why you chose `ClusterIP` here rather than `NodePort` or
`LoadBalancer`, and how you reached the service from your laptop instead.

A: Either the selector matches no pods, or it matches them but none is Ready (Running is not Ready — a pod counts as an endpoint only when it passes its readiness gate). I tell them apart with `kubectl get pods -l app=timesvc`: if that returns nothing, the labels are the problem and I compare the Service's selector with the pod template's labels; if it returns pods, I look at their READY column and describe one. A third variant of the first cause is writing the Service selector as `matchLabels`, which makes it look for a label with that literal name. I chose ClusterIP because only other pods in the cluster call this service: NodePort would open a port on every node and LoadBalancer would provision and bill real infrastructure for something with no external callers. From my laptop I used `kubectl port-forward -n timesvc svc/timesvc 8080:80`, which tunnels through the API server and needs no change to the manifest.
