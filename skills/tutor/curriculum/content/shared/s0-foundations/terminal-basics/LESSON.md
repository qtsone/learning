# The Terminal

> `shared.foundations.terminal-basics` · ~2-3h · Stage: Foundations

## Objectives

By the end of this lesson you can:

- Explain what a shell is and how it differs from the terminal window that hosts it.
- Navigate the filesystem using `pwd`, `cd`, and `ls`, and distinguish absolute from relative paths.
- Create, copy, move, and delete files and directories from the command line.
- Explain what PATH is and diagnose why a command produces "command not found".
- Look up a command's usage with man pages or `--help` instead of guessing flags.

## Terminal and shell: two programs, not one

Open a terminal now: on macOS press Cmd+Space, type "Terminal", press Enter;
on most Linux desktops press Ctrl+Alt+T. You'll see a mostly empty window with
a short line of text ending in `%` or `$` — the **prompt** — and a blinking
cursor.

That window is the **terminal**: a program whose only job is to draw text and
send your keystrokes onward. The program it sends them *to* is the **shell**:
it reads the line you typed, runs the command you asked for, prints the
result, and shows a fresh prompt. Terminal is the telephone; the shell is who
answers. On macOS the default shell is `zsh`; on most Linux systems it's
`bash`. For everything in this lesson they behave the same.

Why learn this when you have windows and a mouse? Because every tool in this
roadmap — compilers (remember the last lesson), test runners, version control —
is a command you type. The terminal is where programming actually happens.

## The filesystem is a tree

Everything on disk lives in one big tree of **directories** (the same thing
your desktop calls folders). The tree starts at the **root**, written `/`, and
every file has an address in it — a **path** — like
`/Users/ada/trip/notes/draft.txt`: start at the root, descend into `Users`,
then `ada`, then `trip`, then `notes`. Your personal corner of the tree is
your **home directory**, spelled `~` for short.

Your shell always sits at exactly one place in this tree: its **working
directory**. Three commands cover getting around:

```sh
pwd        # print working directory — "where am I?"
ls         # list what's here
cd trip    # change directory — step into trip
```

`cd` with no argument takes you home. One habit to build immediately: names
starting with a dot (like `.zshrc`) are **hidden** — `ls` skips them unless
you pass the right flag. You'll hunt that flag down in the exercise.

## Absolute vs relative paths

A path starting with `/` is **absolute**: directions from the root, valid no
matter where you stand. Anything else is **relative**: directions from your
working directory. If you are in `/Users/ada`, then `trip/notes` means
`/Users/ada/trip/notes` — and from anywhere else it means something else, or
nothing. Most "file not found" moments in your first weeks are really "I'm not
where I think I am" — `pwd` is the cure.

Three special names appear in paths constantly:

- `.` — this directory
- `..` — the parent directory (so `cd ..` steps up one level)
- `~` — your home directory

And one superpower: **Tab completion**. Type the first letters of a name and
press Tab — the shell finishes it for you; press Tab twice to see the options.
Use it for every path you type. It kills typos and it's faster.

## Anatomy of a command

```sh
ls -l trip
```

Three parts: the **command** (`ls`), zero or more **flags** that tweak its
behavior (`-l`, "long listing"; multi-letter flags look like `--help`), and
**arguments** — what to operate on (`trip`). Spaces separate the parts, which
is why a filename containing spaces must be quoted: `cd "My Stuff"`.

## Making, copying, moving, deleting

```sh
mkdir trip                 # make a directory
mkdir -p trip/photos       # -p also creates missing parents
echo "day one" > log.txt   # write text into a file
cp log.txt log-copy.txt    # copy: the original stays put
mv log.txt trip/           # move into trip/ (mv also renames)
rm log-copy.txt            # delete a file
rmdir trip                 # delete a directory — only if it's empty
```

Two of these deserve a closer look. `echo` just prints its arguments back at
you; the `>` sends that output into a file instead of the screen, creating the
file — or silently **overwriting** it if it exists. It's the quickest way to
make a file before you have an editor (that's the next lesson).

And `rm` is forever. There is no Trash, no undo. Read the command back to
yourself before pressing Enter. `rmdir` refusing to delete a non-empty
directory is not an annoyance — it's a seatbelt.

## PATH: how the shell finds commands

When you type `ls`, the shell doesn't magically know what that is. It looks
for an executable file named `ls` inside an ordered list of directories, and
runs the first match. That list lives in a variable called **PATH** — see
yours with `echo $PATH` (directories separated by `:`).

So `command not found` means exactly one thing: no file with that name exists
in any PATH directory. Your diagnosis checklist, in order: (1) typo? — the
shell is unforgiving about spelling; (2) is the program actually installed?;
(3) is it installed somewhere PATH doesn't list? `which ls` tells you which
file a command resolves to.

> In Go: when you install the Go toolchain in the dev-environment lesson at
> the end of this stage, "installing" means little more than placing a program
> named `go` into a directory on PATH. `go version` working is proof the shell
> can find it — and `command not found` there will be this exact story.

## Asking the manual

Never guess flags; look them up. Two ways, no internet required:

```sh
man ls       # the manual page for ls
ls --help    # many commands print a usage summary (BSD tools on macOS
             # instead print one when given any option they don't know)
```

`man` opens a pager: Space scrolls down, `/word` searches, `n` jumps to the
next match, `q` quits (everyone gets stuck in a pager exactly once). One
gotcha worth knowing early: macOS ships BSD versions of these tools while
Linux ships GNU versions, so flags occasionally differ from what an internet
tutorial shows. The `man` page on *your* machine is the truth.

## Exercise

Open [`exercise/`](exercise/) in your terminal (`cd` into it, `ls` around).
You'll find `check.sh` and a directory `mess/` holding files from a trip:
two photos (empty stand-in files — don't try to view them), a draft, and a
stale old draft. Your job is to reorganize it, using only the terminal.

Acceptance criteria:

1. Directories `trip/photos` and `trip/notes` exist inside `exercise/`.
2. `photo-001.jpg` and `photo-002.jpg` live in `trip/photos/` — moved, not
   copied (they must be gone from `mess/`).
3. `draft.txt` lives in `trip/notes/`, with an identical safety copy named
   `draft-backup.txt` beside it.
4. `old-draft.txt` is deleted — from everywhere.
5. `mess/` itself is gone (it's empty by now, so `rmdir` will oblige).
6. `trip/location.txt` contains the output of `pwd` run *inside* `trip` —
   one line ending in `/trip`.
7. `trip/notes/hidden.txt` contains the output of `ls`, run inside `trip`
   with the flag that shows hidden entries — use `man ls` to find it. You've
   got the right flag when the file includes the `.` and `..` entries.

Check your work from inside `exercise/`:

```sh
bash ./check.sh
```

It fails now — that's the starting gun. Fix the first `FAIL` it prints, run
it again, repeat until everything is green. Running it never changes your
files, so run it as often as you like.

## Further reading

- [Ubuntu tutorial — The Linux command line for beginners](https://ubuntu.com/tutorials/command-line-for-beginners)
- [MIT, The Missing Semester — Course overview + the shell](https://missing.csail.mit.edu/2020/course-shell/)
- [Apple — Terminal User Guide](https://support.apple.com/guide/terminal/welcome/mac)
- [GNU Coreutils manual](https://www.gnu.org/software/coreutils/manual/coreutils.html)
