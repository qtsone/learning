# Reading Docs & Error Messages

> `shared.foundations.reading-docs` · ~1-2h · Stage: Foundations

## Objectives

By the end of this lesson you can:

- Locate the official documentation for a tool or language and tell it apart
  from unofficial tutorials and forum posts.
- Dissect an error message into its parts (error type, message, location or
  stack trace) and use them to find the failing line.
- Formulate a good question about a problem — what you tried, what you
  expected, what you observed — with a minimal example.
- Choose between reading a reference, a tutorial, or a changelog based on what
  kind of answer you need.

## Nobody memorizes this stuff

Professional programmers look things up constantly — command flags, function
names, exact spellings. What separates them from beginners is not memory; it is
knowing *where* to look and *how* to read what they find. That is a skill, and
this lesson is where you start practicing it.

The two things you will read most are **documentation** (what the tool's makers
wrote about how it works) and **error messages** (what the tool itself says
when something went wrong). Both are written to be read. Most frustration in
early programming comes from skipping them.

## Four kinds of documentation

"The docs" is not one thing. Most projects publish several kinds, each
answering a different kind of question:

- **Tutorial** — teaches by walking you through building something. Best when
  you are new to a tool and don't yet know what to ask.
- **How-to guide** — a recipe for one specific task ("how to undo a commit").
  Best when you know *what* you want and need the steps.
- **Reference** — the complete, precise facts: every command, every flag, every
  function. The `man` pages you met in the terminal lesson are references. Best
  when you need one exact detail: "what does this flag actually do?"
- **Changelog / release notes** — what changed between versions. Best when
  something worked yesterday, or a guide's instructions don't match what your
  version does.

Picking the wrong kind wastes time in a predictable way: a reference is a
terrible place to *learn* (it assumes you know what you're looking for), and a
tutorial is a terrible place to *look something up* (the fact you need is
buried in a story). Name the question first, then pick the document.

## Official vs everything else

**Official documentation** is published and maintained by the people who make
the tool. It lives on the project's own website — git's docs at
`git-scm.com`, Go's at `go.dev` — or ships with the tool itself (`man git`,
`git --help`). Because the same people change the tool and the docs, official
docs are the closest thing to ground truth.

Everything else — blog posts, YouTube videos, Stack Overflow answers, AI
chatbots — is **unofficial**. Unofficial is not bad: a good blog post often
explains an idea more gently than the reference does. But unofficial content
has two failure modes the official docs mostly don't:

- **It rots.** A tutorial written five years ago describes a five-year-old
  version. The commands may have changed; nobody is obliged to update the post.
- **It can simply be wrong**, and nothing forces a correction.

A reliable habit: *learn* from whatever explains it best, then *verify* the
facts against the official reference for your version. To find the official
docs, search for "<tool> documentation" and check the domain — or skip the
web entirely: for command-line tools, `man <tool>` and `<tool> --help` are
official and already on your machine.

## Anatomy of an error message

An error message is the program reporting, as precisely as it can, what went
wrong. Most have three parts — **who is speaking**, **what went wrong**, and
**where**. Here is one you can cause yourself:

```
$ cat recipes.txt
cat: recipes.txt: No such file or directory
```

Who: `cat`. Where: `recipes.txt`. What: `No such file or directory`. Read that
way, the message *is* the diagnosis: `cat` looked for that file in the current
directory and it isn't there. Compare git:

```
$ git status
fatal: not a git repository (or any of the parent directories): .git
```

`fatal` is the severity, and the message even tells you how git decided: it
looked for a `.git` folder here and in every parent directory.

When you start writing programs, a third part appears: the **location** in
code. Programs are functions calling functions, and when something fails deep
inside, the error carries a **stack trace** — the breadcrumb trail of calls
that led there:

```
error: cannot divide by zero
  in compute_average (stats, line 12)
  called by summarize (report, line 34)
  called by main (main, line 8)
```

Read it as a story: `main` called `summarize`, which called
`compute_average`, which died at line 12. The line to inspect first is the
deepest one that is *your* code.

**In Go:** the compiler reports errors as `file:line:column`, a precise home
address for the mistake:

```
./main.go:7:2: undefined: fmt.Printlnn
```

File `main.go`, line 7, column 2: there is no such name as `fmt.Printlnn` (a
typo for `fmt.Println`). You will read hundreds of these in the next stage.

## First error first

One mistake often triggers a cascade of follow-on errors, so tools may print a
wall of them. Only the first is trustworthy — the rest are often side effects
of it. Read top to bottom, fix the first error only, run again. Never try to
fix ten reported errors in one sitting; three of them probably don't exist.

## Asking a good question

When docs and errors haven't cracked it, you ask — a search engine, a forum, a
colleague, an AI. The quality of the answer tracks the quality of the
question. A good question has four ingredients:

1. **What you were trying to do** — the goal, not just the command.
2. **What you expected** to happen.
3. **What actually happened** — the *exact* error text, copy-pasted, never
   paraphrased from memory.
4. **What you already tried**, and what each attempt changed.

Plus, wherever possible, a **minimal example**: the smallest sequence of steps
that still shows the problem, so a stranger can reproduce it in a minute.

Here is the secret that makes this more than etiquette: assembling those four
ingredients forces you to look at the problem so carefully that, remarkably
often, you spot the answer before you finish writing. Programmers call this
the *rubber-duck effect* — explaining the problem out loud (even to a rubber
duck) solves it. Writing the question is debugging.

For searching, one extra trick: put the exact error message in quotes, and
strip out parts specific to you (your file names, your folder paths) so the
search matches other people's versions of the same error.

## Exercise

Open [`exercise/`](exercise/) — this one is a field trip, not a coding task.
`README.md` has four parts; you record your findings in `NOTES.md` and discuss
them with your tutor. No script grades this — the conversation does.

Acceptance criteria:

1. You reproduced the three errors from part 1 in your own terminal and, for
   each, wrote down who is speaking, what went wrong, and the one change that
   fixes it — then made it succeed.
2. You dissected both canned errors in `errors.txt`, including reading the
   stack trace's story and naming the exact file and line the Go error points
   to.
3. You answered the docs scavenger hunt using an official source each time,
   and can say how you knew it was official and which *kind* of document it
   was.
4. `NOTES.md` ends with one complete, well-formed question (goal, expected,
   observed, tried, minimal steps) for the scenario in part 4.

## Further reading

- [Git — official documentation](https://git-scm.com/doc)
- [Go — official documentation](https://go.dev/doc/) (you'll live here from
  the next stage on)
- [Julia Evans — How to ask good questions](https://jvns.ca/blog/good-questions/)
- [Stack Overflow — How do I ask a good question?](https://stackoverflow.com/help/how-to-ask)
