# Runbook — <project>

Copy this to your project root as `RUNBOOK.md` and replace every prompt. Write
it for the person woken at 3am — who is you, in six months, with no memory of
this code. Prose they have to interpret is worse than no runbook, because it
reads like help.

Header facts, in one glance:

- **Service:** name, where it runs, how many instances.
- **Depends on:** the things whose failure becomes your failure.
- **Blast radius when down:** who notices, what stops, what is lost.

## How to deploy

> The exact commands, in order, in a fenced code block. Include how you know it
> worked — the check you run afterwards, and what a good answer looks like.
> If a step needs a decision (which version? which environment?), say how to
> make it.

## How to roll back

> One command if you can manage it, and safe to run *before* you understand the
> incident. Say how long it takes and how you confirm the old version is back.
> If rollback is impossible (a migration, a published release), write down what
> you do instead — that is the more important paragraph.

## What to check when it is down

> An ordered list of checks, cheapest and most likely first, each with the
> command and how to read the answer. Aim for: after these, you know whether
> the problem is in the process, in its configuration, in a dependency, or in
> front of it. This is the section that gets used most and rots fastest.

## Known failure modes

> The two or three ways this system actually breaks — not a catalogue of
> everything that could. One subsection each:
>
> - **Symptom** — what you would see first, from the outside.
> - **Confirm** — the command or graph that proves it is this one.
> - **Cause** — one sentence.
> - **Action** — what to do, and whether it is urgent.
>
> If you have never seen it break, use the two failure modes your design makes
> most likely and say they are predicted rather than observed.

## Who to page

> Who owns this, how they are reached, when it is acceptable to wake them, and
> who to escalate to after how long. If the answer is "me, and nobody else",
> write that down along with what happens when you are unreachable — an honest
> single point of failure beats an implied one.

## Configuration

> Optional here if your README documents it: every variable or flag, its
> default, what it does, and which ones are secrets. The referee checks that
> everything your code reads from the environment is documented *somewhere*.
