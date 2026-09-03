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

A: TODO

## Q2. The ordering requirement

Which has to happen first: this pod leaving the Service's endpoints, or your
server stopping accepting new connections? Say why that order and not the
other, what breaks when they happen the wrong way round, and which two things
in *your* files enforce the order.

A: TODO

## Q3. The budget

Write out your arithmetic: the preStop sleep, the server's own shutdown
timeout, and terminationGracePeriodSeconds — the three numbers you chose, and
why they add up the way they do. What happens if the sum exceeds the grace
period?

A: TODO

## Q4. The runtime

What did you set GOMAXPROCS and GOMEMLIMIT to, and why those values? Say what
the Go runtime would have decided on its own inside this container, and what
that costs you.

A: TODO
