# Exercise: distributed-systems reasoning on a real expansion

You are the engineer writing the distributed-systems section of
Cartwheel's two-region design. Read [`brief.md`](brief.md) first — every
number and business demand in it exists to be cited — then work the four
worksheets **in order**:

1. [`01-consistency-menu.md`](01-consistency-menu.md) — choose the
   weakest sufficient consistency model, feature by feature.
2. [`02-partition-drill.md`](02-partition-drill.md) — decide, in
   advance, what every feature does while the ocean link is down; then
   face the PACELC else-case.
3. [`03-failure-modes.md`](03-failure-modes.md) — audit the design
   against the failure-mode catalogue and the fallacies.
4. [`04-consensus-walkthrough.md`](04-consensus-walkthrough.md) — trace
   a Raft-style cluster through elections and a partition, then judge
   where Cartwheel actually needs one.

Ground rules:

- Fill the worksheets in place — replace every `…` with your answer.
  Keep the tables; add rows freely.
- **Cite the brief.** A model choice justified "because requirement 3
  says never oversell" is an argument; "to be safe" is not. Refer to
  features and business asks by their IDs.
- **Name the anomaly.** Every consistency choice must say what a user
  would actually see if you weakened it one step. If you cannot describe
  the anomaly, you have not earned the strength.
- Numbers over adjectives, as always: the brief's RTTs are there so your
  latency claims can show arithmetic, design-intro style.
- Write in your own words. The review is a conversation, and copied
  lesson prose collapses under the first follow-up question.

Budget roughly 2 hours across the four sheets. When you are done, tell
your tutor: verification is a **design review conversation**. Expect the
partition to get longer, the drop to get bigger, and every "linearizable"
on worksheet 1 to be challenged with "why not one step weaker?".
