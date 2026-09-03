# Editors & IDEs

> `shared.foundations.text-editors` · ~1-2h · Stage: Foundations

## Objectives

By the end of this lesson you can:

- Set up VS Code with a language extension and verify features like
  autocomplete and go-to-definition work.
- Explain what a language server does and how the editor uses it to provide
  diagnostics and completions.
- Use keyboard-driven workflows (command palette, quick-open, multi-cursor)
  instead of mouse navigation for common edits.
- Choose between a full IDE and a lightweight editor for a given task and
  justify the choice.

## Why not a word processor?

Remember the first lesson: source code is **plain text** — nothing but
characters, saved in a file. Word processors (Pages, Word, Google Docs) secretly
store much more than the characters you type: fonts, sizes, colors, layout. A
compiler fed that extra data would choke on the very first byte. So programmers
use a **text editor**: a program that edits exactly the characters in a file
and nothing else. In the terminal lesson you made files with `echo` and `>`
precisely because you had no editor yet; this lesson gives you the one you'll
live in.

## Meet VS Code

**Visual Studio Code** (VS Code) is a free editor from Microsoft and currently
the most common choice across languages. Install it:

- **macOS** — download from [code.visualstudio.com](https://code.visualstudio.com/),
  unzip, drag `Visual Studio Code` into `Applications`.
- **Linux** — the download page has `.deb`/`.rpm` packages, or use your
  distribution's package manager or Flatpak/Snap.

VS Code thinks in **folders**, not single files: you open a project folder,
and the editor shows its whole file tree in the sidebar. Its second habit to
learn immediately: open folders *from the terminal*.

```sh
cd path/to/some/folder
code .        # "." is the current directory — remember the terminal lesson
```

If `code` prints "command not found" on macOS: open VS Code, press
`Cmd+Shift+P`, type "shell command", and run **Shell Command: Install 'code'
command in PATH**. That's the PATH mechanism from the terminal lesson — the
installer drops a `code` command where your shell can find it.

VS Code also has a built-in terminal (menu **Terminal → New Terminal**). It is
the same shell you already know, just living inside the editor window.

## The editor is simple; the language server is smart

Out of the box, an editor is a glorified typewriter — it shows characters and
lets you change them. The intelligence you see in screenshots (red squiggles
under mistakes, suggestions as you type) comes from somewhere else: a
**language server**, a separate program that actually understands one
particular language. The editor and the language server talk over a standard
protocol (the **Language Server Protocol**, LSP): the editor sends "here's the
file, the cursor is at line 12", and the server answers with things to show.

The main answers have names you'll use daily:

- **Diagnostics** — "line 5 has a mistake, here's why" → the red squiggle and
  the Problems panel.
- **Completions** (autocomplete) — "given what's before the cursor, here are
  the things that could legally come next" → the suggestion popup.
- **Go-to-definition** — "the thing under the cursor is defined over *there*"
  → press `F12` and jump straight to it.
- **Hover** — rest on a name and read its documentation in place.

Why a separate program and a protocol? Arithmetic: with N editors and M
languages, editor-specific plugins would need N×M integrations. With LSP,
each editor implements the protocol once and each language ships one server:
N+M. This is why a niche editor can offer the same Go or Python smarts as
VS Code — they all dial the same server.

> **In Go:** the language server is called `gopls` ("go please"). Once your Go
> toolchain is installed — that happens in this stage's dev-environment
> lesson — the Go extension runs `gopls` for you, and typing `fmt.` pops up
> everything the `fmt` package offers, with `F12` jumping into the standard
> library itself.

## Extensions

VS Code learns new tricks through **extensions** — add-ons you install from
the sidebar's Extensions view (`Cmd+Shift+X` / `Ctrl+Shift+X`). A *language
extension* (Go, Python, Rust…) is mostly a wrapper that installs and starts
that language's server, plus syntax coloring and a few commands.

Two formats need no extension at all, because their language servers ship
inside VS Code: **Markdown** (the `.md` format this lesson is written in) and
**JSON** (a structured data format you'll meet constantly). The exercise uses
both, so you can watch a language server work today — before you've installed
any programming language.

## Keyboard-driven editing

Reaching for the mouse breaks your train of thought. You don't need to
memorize fifty shortcuts — you need three habits (keys given as
macOS / Linux):

| Habit | Keys | What it does |
|---|---|---|
| Command palette | `Cmd+Shift+P` / `Ctrl+Shift+P` | Search-and-run *any* editor command by name |
| Quick open | `Cmd+P` / `Ctrl+P` | Jump to any file by fuzzily typing part of its name |
| Multi-cursor | `Cmd+D` / `Ctrl+D` | Select the next occurrence of the selected word — press again to add more, then type once to edit all |

The palette is the trick that replaces memorization: forget a shortcut, open
the palette and type what you want ("fold", "rename", "terminal") — the
command appears with its shortcut printed next to it. Quick open is why the
file tree stops mattering in big projects: you *name* the file you want
instead of hunting for it. You can also drop extra cursors by hand with
`Option+Click` / `Alt+Click` (on some Linux desktops the window manager steals
`Alt` — the palette command "Toggle Multi-Cursor Modifier" fixes that).

## Editor or IDE?

An **IDE** (Integrated Development Environment — e.g. GoLand, IntelliJ,
PyCharm) bundles everything for one language ecosystem: editor, deep code
analysis, debugger, project tooling, all preconfigured. A lightweight editor
like VS Code starts near-empty and grows via extensions. The trade-off:

- **IDE** — strongest single-language experience, works out of the box; costs
  memory, startup time, often money, and its depth can overwhelm a beginner.
- **Lightweight editor** — fast, free, one tool for every language you'll
  ever touch; you assemble the setup yourself, extension by extension.
- **Terminal editors** (`nano`, `vim`) — still essential for quick edits on a
  machine you reach only through a terminal, where no window can open.

There is no "best" — there is *fit for the task*. A one-line fix on a remote
server wants `nano`; a week refactoring a huge Java codebase wants an IDE;
learning multiple languages on your own laptop wants VS Code. This course
assumes VS Code, and every skill here transfers.

## Exercise

Open [`exercise/`](exercise/) in VS Code **from the terminal** (`cd` into it,
then `code .`). It's a small folder tour; your tutor reviews the results in
conversation — there is no automated check for this lesson.

Acceptance criteria:

1. `code .` opens the exercise folder from your terminal (install the shell
   command first if needed).
2. `data/inventory.json` shows red squiggles when you open it. Using only the
   squiggles and the Problems panel (**View → Problems**), fix the file until
   the panel shows zero problems. Note what each mistake was.
3. Complete the treasure hunt starting at `hunt/START.md`, moving between
   files **only** with quick open (`Cmd+P` / `Ctrl+P`) — no clicking in the
   file tree. Bring the codeword from the final file.
4. In `drill/palette.txt`, change every `colour` to `color` in *one* edit
   using multi-cursor (`Cmd+D` / `Ctrl+D`) — not one at a time, not
   find-and-replace.
5. In `notes/journal.md`: follow the existing link with `Cmd+Click` /
   `Ctrl+Click` — the jump works because the Markdown language server has
   resolved where the path points. Then add the missing link: type
   `[the drill](..` — the moment you type the `.` (or press `Ctrl+Space`),
   the path suggestions that appear are completions from that same server.
6. Run at least two commands you've never used via the command palette and
   tell your tutor what they did.

## Further reading

- [VS Code — Tips and Tricks](https://code.visualstudio.com/docs/getstarted/tips-and-tricks)
- [Language Server Protocol — overview](https://microsoft.github.io/language-server-protocol/)
- [VS Code keyboard shortcut reference (macOS PDF)](https://code.visualstudio.com/shortcuts/keyboard-shortcuts-macos.pdf) · [Linux PDF](https://code.visualstudio.com/shortcuts/keyboard-shortcuts-linux.pdf)
