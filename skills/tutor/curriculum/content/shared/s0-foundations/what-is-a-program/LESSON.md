# What Is a Program?

> `shared.foundations.what-is-a-program` · ~1-2h · Stage: Foundations

## Objectives

By the end of this lesson you can:

- Explain how source code becomes machine code and why the CPU cannot execute
  source text directly.
- Compare compilers and interpreters, and name one language that typifies each
  approach.
- Describe the roles of the CPU and memory when a program runs, using a simple
  mental model of instructions and data.
- Trace what happens, step by step, from typing a run command to a program
  printing output.

## A program is a list of instructions

A **program** is a list of instructions that a computer follows, one at a
time, in order. That's the whole idea. A recipe is a program for a cook; sheet
music is a program for a pianist. The difference is the performer: a computer
is spectacularly fast and spectacularly literal. It never guesses what you
meant, never skips a step because it's "obvious", and never gets bored doing
the same step a billion times.

Everything you'll ever build — a website, a game, the app you're reading this
with — is, underneath, a long list of small instructions.

## The CPU: fast, not smart

The **CPU** (central processing unit) is the chip that follows instructions.
Each instruction it understands is tiny: *add these two numbers*, *copy this
value over there*, *if that value is zero, continue from a different
instruction*. Nothing clever ever happens in a single step. What makes
computers seem clever is speed — a modern CPU performs billions of these tiny
steps every second, so enough dumb steps add up to something that looks smart.

The CPU also tracks *where it is* in the list: it keeps a marker for the
current instruction, does what it says, and normally moves to the next one.
Some instructions move the marker somewhere else instead — that's a **jump**,
and it's how programs repeat things or make decisions.

## Memory: numbered boxes for instructions and data

While a program runs, everything it works with lives in **memory** (also
called RAM): picture an enormous row of numbered boxes, each holding a small
value. Two kinds of things sit in those boxes:

- **Instructions** — the program itself, loaded into memory so the CPU can
  read it.
- **Data** — the values the program is working on: numbers, pieces of text,
  whatever it needs to remember from one step to the next.

So the CPU is the *doer* and memory is the *workspace*: the CPU reads an
instruction from memory, maybe reads or writes some data boxes, and moves on.

One thing memory is **not**: your files. Files live on the **disk**, which is
slower but keeps its contents when the power goes off. Memory is fast, and
wiped clean when a program ends (or the machine restarts). A program is a file
on disk *between* runs and lives in memory *while* it runs.

## Machine code and source code

Here's the catch: the CPU only understands **machine code** — instructions
encoded as raw numbers, in a fixed format wired into the chip itself. Machine
code is unreadable for humans and unforgiving to write.

So humans write **source code** instead: instructions as structured text,
designed for people, in a **programming language**. Source code might say
`total = price + tax` where machine code would be a handful of number-encoded
steps about copying values between boxes.

But source code is just text. The CPU cannot execute text, any more than a CD
player can play sheet music — the text must be *translated* into machine code
first. Every programming language needs a translator, and there are two main
kinds.

## Compilers and interpreters

A **compiler** translates your entire program ahead of time. You hand it your
source code; it checks it, translates all of it into machine code, and saves
the result as a runnable file (an **executable**, or *binary*). Running the
program later doesn't involve the compiler at all — the CPU executes the
finished machine code directly. **Go** works this way (so do C and Rust).

An **interpreter** skips the ahead-of-time step. It is itself a program — one
that reads your source code and *performs* it, an instruction at a time, every
time you run it. Your source is never turned into a saved machine-code file;
the interpreter is the thing the CPU actually executes, and your code is its
input. **Python** works this way (so does JavaScript).

The trade-offs follow directly from the designs:

- Compiled programs run fast (pre-translated) and many mistakes — like a
  misspelled name — are caught during compilation, before the program ever
  runs. The price is the extra translate step after each change.
- Interpreted programs start instantly from source — edit, run, repeat — but
  translation work happens on every run, and some mistakes only surface when
  the interpreter reaches that line.

(Real tools blur these lines in interesting ways, but these two pure models
are the mental anchors — they'll serve you for years.)

In Go: you'll meet these commands properly soon, but for a taste — `go build`
invokes the Go compiler on your source and produces an executable; running
that file executes pure machine code, no translator in sight.

## From "run" to output, step by step

Soon you'll run programs by typing a command in the **terminal** (a
text-based window for talking to your computer — it's the whole next lesson).
Here is what happens between pressing Enter and seeing output, for a compiled
program:

1. You type the program's name as a command and press Enter.
2. The **operating system** (the always-running manager program — macOS or
   Linux for you) finds the executable file on disk.
3. It copies the file's machine code into memory, sets aside more memory for
   the program's data, and points the CPU at the first instruction.
4. The CPU does its thing: read an instruction, execute it, move to the next
   — billions of times per second, reading and writing data boxes as told.
5. Some instructions say "hand these characters to the output". The operating
   system carries them to your screen — that's your program *printing*.
6. After the final instruction, the program exits and the operating system
   reclaims its memory.

For an interpreted program, only step 3 changes: what gets loaded is the
*interpreter's* machine code, and your source file is handed to it as data.
Everything else is the same picture.

Keep this trace — you will reuse it in every language you ever learn.

## Exercise

Open [`exercise/README.md`](exercise/README.md). It contains a tiny program
written for a **paper machine** — two memory boxes, a screen, and eight
numbered instructions. There is nothing to install and nothing to type: *you*
are the machine, executing it by hand with a pencil.

Acceptance criteria:

1. You completed the trace table and can say exactly what number the program
   prints on the screen.
2. You can answer: how many times did step 4 run, and what decides which
   instruction the machine executes next?
3. You changed exactly one number in the program so it prints `8`, and traced
   it to prove it.
4. You can say whether executing the program by hand made you a compiler or
   an interpreter — and defend the answer.

This lesson has no automated check: when you're done, walk your tutor through
the trace and your answers. Wrong guesses are welcome; untried guesses are
not.

## Further reading

- [Crash Course Computer Science #7 — The Central Processing Unit](https://www.youtube.com/watch?v=FZGugFqdr60)
  — the fetch/execute cycle, animated.
- [Crash Course Computer Science #11 — The First Programming Languages](https://www.youtube.com/watch?v=RU1u-js7db8)
  — from machine code to source code and compilers.
- [The Go Playground](https://go.dev/play/) — a real Go program you can run in
  your browser, no installation. Press "Run" and map what happens onto the
  six-step trace above.
