# Shutdown — timesvc

Answer each question on the `A:` line below it, in your own words, in at
least a couple of sentences. `check.sh` checks that you answered and that the
answers name the mechanisms by their real names; your tutor reads what you
actually wrote.

## Q1. The sequence

Something deletes one of your pods — `kubectl delete pod`, a rolling update,
a node drain. Trace what happens from that moment until the container is
gone. Name every step in order, say which two things happen *in parallel*,
and say where SIGTERM, terminationGracePeriodSeconds, preStop and SIGKILL
each appear.

A: The API server records a deletionTimestamp on the pod and the grace-period clock (terminationGracePeriodSeconds: 45) starts. From there two chains run in parallel and nothing synchronises them. Chain one is the control plane: the endpoints controller marks this pod not-ready and terminating in the Service's EndpointSlice, and then every kube-proxy (plus any ingress or mesh dataplane) has to notice and rewrite its rules — that propagation takes an unbounded few hundred milliseconds to a few seconds. Chain two is the kubelet on the node: it runs the preStop hook first (my 10s sleep), and only when the hook returns does it send SIGTERM to PID 1 in the container. My process turns that into a cancelled context, flips /readyz to 503, and calls http.Server.Shutdown, which stops accepting and waits for in-flight requests, bounded by SHUTDOWN_TIMEOUT=20s. If the container is still alive when the 45 seconds are up, the kubelet sends SIGKILL, which no code can catch or handle — whatever was in flight is simply gone. Once the container is dead the pod object is finally removed from the API.

## Q2. The ordering requirement

Which has to happen first: this pod leaving the Service's endpoints, or your
server stopping accepting new connections? Say why that order and not the
other, what breaks when they happen the wrong way round, and which two things
in *your* files enforce the order.

A: The endpoint removal has to land first, before the server stops accepting. The reason is that the two chains above are not coordinated: if the process closes its listener the instant SIGTERM arrives, kube-proxy on some other node is still forwarding new connections to an address where nothing is listening any more, and those callers get a connection refused or a reset — errors on requests that were never served, which is the worst kind, because a retry budget is what has to absorb them. The other order costs nothing: a pod that has already left the endpoint set but is still accepting simply receives no new work. Two things in my files enforce it. The preStop sleep of 10 seconds delays SIGTERM long enough for the dataplane to catch up, and the run() function sets ready to false before it calls Shutdown, so anything that polls /readyz — an ingress controller, a load balancer, a mesh — is told 503 first too.

## Q3. The budget

Write out your arithmetic: the preStop sleep, the server's own shutdown
timeout, and terminationGracePeriodSeconds — the three numbers you chose, and
why they add up the way they do. What happens if the sum exceeds the grace
period?

A: preStop 10s + SHUTDOWN_TIMEOUT 20s = 30s of work, inside a terminationGracePeriodSeconds of 45s, leaving 15s of margin for the process to log the outcome and exit. The key fact is that the preStop hook runs inside the grace period, not before it — the clock starts at the deletion, so the hook eats into the same 45 seconds the drain needs. If the numbers did not fit, the kubelet would SIGKILL the container mid-drain: connections cut, requests that were seconds from finishing lost, and no shutdown log line explaining any of it, which makes it look like a crash rather than a deploy. I picked 20s for the drain because timesvc's slowest handler is bounded well under that; a service with long-running requests would need a bigger drain and a proportionally bigger grace period.

## Q4. The runtime

What did you set GOMAXPROCS and GOMEMLIMIT to, and why those values? Say what
the Go runtime would have decided on its own inside this container, and what
that costs you.

A: GOMAXPROCS comes from the downward API on limits.cpu, so my 500m limit rounds up to 1 and the value cannot drift when someone changes the limit; GOMEMLIMIT is a literal 110MiB under a 128Mi memory limit. My module is on go 1.22, and through Go 1.24 the runtime sizes GOMAXPROCS from the CPU affinity mask — the node's cores — because the cgroup CPU quota is not visible to it. On a 64-core node that gives 64 Ps and up to a quarter of them running GC mark workers, all fighting over half a core's worth of quota per period: the cgroup throttles the whole process, and the cost shows up as tail latency and context switching, not as a crash. Go 1.25 made the default container-aware on Linux (the minimum of the affinity mask and the cgroup quota rounded up, floor 2), but only for modules whose go directive is 1.25 or later, so setting it explicitly is both correct today and readable by whoever inherits this file. GOMEMLIMIT is the same idea for memory: the collector's default target is a ratio of the live heap, so it will happily grow past a cgroup limit it cannot see and get the process OOM-killed with no shutdown at all. Aiming the GC at 110MiB means it works harder as the heap approaches the limit instead of the kernel killing the pod, and the ~18MiB gap covers what GOMEMLIMIT does not count — the binary's own mappings and OS-side bookkeeping.
