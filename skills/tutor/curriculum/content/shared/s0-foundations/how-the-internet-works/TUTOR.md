# Tutor notes — How the Internet Works

## Where the learner is

Five lessons into S0: comfortable enough in the terminal to run commands and
redirect output, and they have practiced the mechanics of `git push` —
against a local bare folder standing in for GitHub, not over the network.
Bridge from that: the same command pointed at the real GitHub is exactly the
networked case this lesson explains. Don't presume they have a GitHub
account or have pushed over the internet; the exercise's `dig` and `curl`
runs are their first real network interactions from the terminal. No
programming knowledge yet, no Go toolchain (that's the
dev-environment lesson, two ahead). This is a pure mental-model lesson;
verify is discussion — the `exercise/notes.md` answers are your material,
there is no automated check.

## Common misconceptions

- **"The internet and the web are the same thing"** — the web (pages in a
  browser) is one use of the internet; email and `git push` over SSH are
  others that never touch a browser. Careful with the git example: over an
  `https://` remote git *does* speak HTTP — the honest framing is "many
  programs besides browsers hold conversations over the internet, some even
  reusing HTTP", not "git isn't HTTP". Don't go deep; just separate the
  layers: network vs. conversation on top.
- **"A server is a special kind of computer"** — it's a program in a role.
  Reinforce with S0 lesson one: instructions on a CPU, looping on
  wait-answer-repeat.
- **"DNS fetches the page"** — DNS only maps name → IP. If Task 1 notes say
  DNS "found the website," have them point at which step of the lifecycle
  actually transferred HTML.
- **"One URL = one request"** — a page view is dozens of requests; each
  image/style/script repeats the cycle. Surfaces in Task 5 narrations that
  end at "the page arrives."
- **"404 means the site is broken/down"** — the server answered; 4xx blames
  the client's ask, 5xx blames the server. Task 4 exists to break this one.
- **"HTTPS means the site is safe/honest"** — it encrypts transit only;
  a scam site can have perfect HTTPS.

## Grilling points

Beyond quiz.json, push on transfer to novel cases:

- "You run `git push` at a coffee shop. Walk me through it with this
  lesson's vocabulary — who's the client, what gets resolved, roughly what
  travels?"
- "Your friend's site works by IP address but not by name. Which part of the
  system is broken, and which steps of the lifecycle still succeed?"
- "Why did `dig` return two addresses for example.com? What does that let
  the site's operators do?"
- "In your `curl -v` output, why do both sides send headers before any
  content? What problem does labeled metadata solve?"
- "Why are status codes numbers instead of sentences?" (Machines branch on
  them; the categories make 'unknown' codes still roughly interpretable.)

## Grading rubric

- **A** — Notes complete and in their own words; narrates all six lifecycle
  steps unprompted and in order; correctly assigns client/server roles in a
  novel example (coffee-shop `git push`); reads real curl output fluently
  (method, status, headers, body); cleanly separates DNS from HTTP and 4xx
  from 5xx.
- **B** — Lifecycle solid with at most one prompted step; roles right in
  familiar examples but hesitant on novel ones; minor confusion on one
  concept (e.g. header vs. body) that corrects with a nudge.
- **C** — Can recite pieces but the chain doesn't connect (e.g. thinks DNS
  returns the page, or can't say what happens after resolution). Pass only
  after a time-boxed re-walk of the lifecycle lands; otherwise iterate.
- **Fail** — Notes empty or copied; cannot identify client vs. server in
  the web-page example, or believes typing a URL "just gets the page" with
  no steps in between. Redo the exploration together, don't advance.

## Remediation ladder

1. "Open your Task 5 list next to the lesson's six steps. Which step is
   missing or out of order, and what would break without it?"
2. "Before your computer can send anything to example.com, what one fact
   must it learn first? Which tool from the exercise learns it?"
3. "Run `curl -v https://example.com/ >/dev/null` again. Read me the `>`
   lines, then the `<` lines. Which side is the client talking?"
4. Walk the lifecycle aloud yourself with a postal analogy — look up the
   address (DNS), open the mailbox route (connect), send the letter
   (request), get the reply (response), read it (render) — then have them
   replay it back in networking terms.

## After passing

Preview: "Next you learn to read documentation and error messages — the
skill that turns every official manual, including the HTTP docs you just
skimmed, into something you can use instead of fear."
