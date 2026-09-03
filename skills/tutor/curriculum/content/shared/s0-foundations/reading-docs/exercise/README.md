# Exercise — Errors on purpose, docs on purpose

Work through the four parts in order. Write what you find in `NOTES.md`
(open it in your editor — it has a slot for every answer). When you're done,
walk your tutor through it.

You only need your terminal, git, and a web browser. Nothing here can damage
anything: every error is caused in a scratch folder you create yourself.

## Part 1 — Break things on purpose

Make a scratch folder somewhere that is **not** inside a git repository, and
work there:

```sh
mkdir /tmp/docs-practice
cd /tmp/docs-practice
```

Now cause each error below for real. For each one, record in `NOTES.md`:
*who* is speaking (which program), *what* it says went wrong, and the one
change that makes the command succeed — then make it succeed.

1. `cat recipes.txt` — in your empty scratch folder.
2. `git status` — same folder (it is not a repository… yet).
3. `git stats` — a misspelled subcommand. Read the reply carefully: this is
   an example of a *helpful* error. What does git do beyond saying "no"?

## Part 2 — Dissect two canned errors

Open `errors.txt` in this folder. It contains two errors you can't produce
yourself yet — one runtime error with a stack trace, and one Go compiler
error. You are only *reading*; no tools needed. Answer the questions in
`NOTES.md`:

- For the stack trace: which function was actually running when the program
  died? Which function asked it to run? Where would you look first, and why?
- For the Go error: which file, which line, which column? What kind of
  mistake do you suspect, and what makes you say so?

## Part 3 — Docs scavenger hunt

Answer each using an **official** source, and note in `NOTES.md` where you
found it, how you knew the source was official, and which *kind* of document
it was (tutorial, how-to, reference, changelog):

1. What does `git log --oneline` do? (Try `man git-log` first — search
   inside a man page by typing `/oneline` then Enter, and `n` for the next
   match. The same page also lives at git-scm.com.)
2. Find the home page of Go's official documentation. What is the URL, and
   what on the page (or in its address) tells you it's official? You'll need
   this site in the next lesson.
3. Classify each of these as tutorial / how-to / reference / changelog, and
   official / unofficial: a `man` page; a YouTube video called "Git tutorial
   for beginners"; a Stack Overflow answer; the "release notes" page on a
   project's own website.

## Part 4 — Write the question

A scenario, from tools you already know. You did this:

```sh
mkdir /tmp/notes && cd /tmp/notes
git init
echo "buy milk" > todo.txt
git commit -m "first commit"
```

…and instead of a commit, git printed a message ending in:

```
nothing added to commit but untracked files present (use "git add" to track)
```

Reproduce it if you like. Then, in `NOTES.md`, write the question you would
post about it — title plus body — with all four ingredients: what you were
trying to do, what you expected, what actually happened (exact text), what
you tried. Include the minimal steps above so a stranger could reproduce it.

Finally, answer honestly: did writing the question show you the fix before
you "posted" it? That feeling has a name — bring it to the discussion.
