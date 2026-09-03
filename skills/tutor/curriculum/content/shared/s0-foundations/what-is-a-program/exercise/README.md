# The Paper Machine

You are the CPU. This page is the memory. A pencil is all the hardware you
need.

## The machine

- Two memory boxes, **A** and **B**. Each holds one number. They start empty.
- A **screen** that shows whatever the program prints.
- A program: numbered instructions. You always execute the instruction your
  marker is on, then move the marker to the next number — *unless* the
  instruction is a jump, which moves the marker somewhere else instead.

Rules of being a CPU: one instruction at a time, exactly as written, no
guessing ahead, no skipping "obvious" steps.

## The program

```text
1. Put 2 in box A.
2. Put 5 in box B.
3. If box B holds 0, jump to step 7.
4. Double the number in box A.
5. Subtract 1 from box B.
6. Jump to step 3.
7. Print the number in box A on the screen.
8. Stop.
```

## Part 1 — Run it

Copy this table onto paper and add one row per instruction you execute, until
you reach *Stop*. The first three rows are done for you:

| Step executed | Box A | Box B | Screen |
|---------------|-------|-------|--------|
| 1             | 2     | —     |        |
| 2             | 2     | 5     |        |
| 3             | 2     | 5     |        |
| …             |       |       |        |

Then answer:

- What number ends up on the screen?
- How many times did you execute step 4?
- Nothing in the program says "repeat" — so what made you go around the loop?

## Part 2 — Change it

Change **exactly one number** in the program so that it prints `8` instead.
Trace your changed program the same way to prove it works. (If your first
guess prints the wrong number, good — that's what the trace is for.)

## Part 3 — Think about it

- Suppose step 6 were deleted, so step 5 flowed straight into step 7. What
  would the program print? (Trace it — don't guess.)
- While you executed this program by hand, were you behaving like a compiler
  or like an interpreter? One sentence, and why.

Bring your tables and answers to your tutor — this exercise is reviewed in
conversation.
