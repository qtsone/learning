# Worksheet 4 — Consensus walkthrough

A Raft-style cluster of five nodes — **A B C D E** — coordinates
Cartwheel's control-plane facts. Currently: **A is leader in term 3**;
all logs are identical. Answer each question with the *because*: quorum
arithmetic and the majority-intersection argument are what is graded,
not the bare outcome. (Run the Secret Lives of Data visualization first
if any answer feels like guessing.)

## Trace 1 — leader crash

A accepts a new entry from a client, appends it to its own log, and
**crashes** before replicating it to anyone.

1. Is that entry committed? What happens to it, and what does the
   client that sent it know?

> …

2. B and C time out and both become candidates in term 4. Can they both
   win? Walk the votes that decide it.

> …

## Trace 2 — partition, then healing

The network splits: **{A, B}** on one side (A still believes it is the
term-3 leader), **{C, D, E}** on the other.

3. A client sends a write to A. Trace what A does with it and how far
   it gets. Can A commit it? Why exactly not?

> …

4. Can the other side elect a leader? Which term number, and what does
   the majority rule guarantee about this election given A still thinks
   it leads?

> …

5. The partition heals. A, still acting like a leader, contacts D. What
   tells A it is stale, and what happens to the entries A appended
   during the partition?

> …

## Quorum arithmetic

6. State how many failures this 5-node cluster tolerates while still
   committing writes. Then explain why upgrading to 6 nodes buys zero
   additional failure tolerance — show the majority sizes.

> …

7. During the trace-2 partition, the {C, D, E} side kept working and
   {A, B} stalled. Which letter of CAP did this system choose, and what
   would it have risked by letting *both* sides keep accepting writes?

> …

## Judgment call — consensus at Cartwheel

8. Name one Cartwheel fact that genuinely belongs in this consensus
   cluster and one that emphatically does not, and defend both with the
   cost model (a quorum round trip per committed write; the brief's
   traffic numbers are your ammunition).

> Belongs: …
>
> Does not: …
