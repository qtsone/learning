# Tutor note — what `testdata/reference` is here

The other capstone lessons carry one project, `notes`, and evolve it: the
build-core reference is the same module the hardening and performance
references grow out of. This lesson's reference is a different program,
`tinysvc`, because the operational surface it has to demonstrate — readiness
that flips during a drain, `/metrics`, a request log with a route pattern in it
— is a service's surface, and a stdin-driven CLI can only mime it. It is a
**shape alternative**, not the learner's capstone at lesson four, and it is not
what the notes project would have become. Say so if a learner asks why the
listing they read last lesson has turned into a web service.

What it is *not* allowed to be is a regression. The earlier harnesses grade
properties, not projects, so this fixture must clear them too: it carries
`MILESTONES.md` with every box ticked, 60%+ statement coverage, a validated
trust boundary with a fuzz target and committed corpus, and `SECURITY.md`. Run
build-core's and hardening's harnesses against it whenever you change it —
`docs/authoring-guide.md` has the loop.
