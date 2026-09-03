# Exercise — pick a capstone and plan it

Nothing compiles here. The deliverable is a project directory containing a
`go.mod` and four documents that a reviewer can attack, and the verification is
a 45-90 minute design review in which they will.

## 0. Create the project first

```sh
cd <your workspace root>              # the directory holding lessons/ and projects/
mkdir -p projects/capstone
cd projects/capstone
go mod init example.com/<your-project>
git init
```

Everything from here on lives in that directory. Every later lesson in this
stage grades what it finds there — the templates in *this* folder are
scaffolding you copy from, not the thing being graded.

Where each document ends up:

```
projects/capstone/
├── go.mod
├── README.md            # write this now: one paragraph on what it is, plus how to run it
├── MILESTONES.md        # from MILESTONES.md here — checked mechanically in later lessons
└── docs/
    ├── PRD.md           # from PRD.md here
    ├── RISKS.md         # from RISKS.md here
    └── adr/
        ├── ADR-001-<slug>.md   # from ADR-000-template.md, one per decision
        ├── ADR-002-<slug>.md
        └── ADR-003-<slug>.md
```

`SELECTION.md` stays here; it is working-out, not project documentation.

## 1. Work in this order

1. **[`briefs.md`](briefs.md)** — read the four example briefs. Use them to
   calibrate size, then propose your own if you have one.
2. **[`SELECTION.md`](SELECTION.md)** — score your candidate against every
   rubric row. Score two candidates if you have two; the comparison is what
   makes the choice defensible. **Clear the choice with your tutor before you
   write the PRD** — a project that fails the rubric fails it more expensively
   after three hours of paperwork.
3. **[`PRD.md`](PRD.md)** — problem, users, requirements with IDs, non-goals.
4. **[`ADR-000-template.md`](ADR-000-template.md)** — copy it once per decision
   into `docs/adr/`, at least three. Write them *as you decide*, not afterwards.
5. **[`MILESTONES.md`](MILESTONES.md)** — the plan. M0 is a walking skeleton.
6. **[`RISKS.md`](RISKS.md)** — what could go wrong, and what you do now.

Budget roughly 3-4 hours. Steps 3-6 will send you back to change each other;
that is the process working, not you failing at it.

## 2. Rules of the game

- **No technology nouns in the problem statement.** If you cannot state the
  problem without them, you picked an implementation first — say so, and write
  the learning goal separately.
- **Every requirement gets an ID** (F1, F2, N1 …). ADRs and milestones cite
  those IDs. A requirement nothing cites is either unnecessary or a milestone
  you forgot.
- **Every ADR names its loser and its bill.** Two real alternatives, an honest
  cost for each including the one you chose, and a flip condition specific
  enough to actually observe.
- **Every milestone leaves the program runnable** and the suite green. If it
  does not, it is not a milestone, it is a checkpoint in the middle of one.
- **Numbers, not adjectives**, in non-functional requirements. "Fast" is not a
  requirement; "restores 10 GB in under 10 minutes on my laptop" is.
- **Estimate in hours and expect to be wrong.** The estimate is not a promise;
  it is what makes "I am at double the estimate" a fact you notice.

## 3. Before you say you are ready

Check yourself against the acceptance criteria in LESSON.md, then answer these
out loud:

- What is the one flow that makes this worth building? (One sentence.)
- Which milestone do you cut first if you lose a third of your hours, and what
  does the project still do without it?
- Which non-goal are you most likely to violate, and what stops you?
- Which ADR are you least sure about, and what would change your mind?

Then tell your tutor you are ready. In the review they will argue your project
is too big, take the losing side of every ADR, delete a milestone to see what
breaks, and ask what happens if you get half the hours you planned. Your job is
not to be unmoved — it is to say precisely what moves, and why.
