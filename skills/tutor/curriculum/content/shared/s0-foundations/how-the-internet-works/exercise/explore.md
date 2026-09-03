# Exploration: watch a request happen

Work through the tasks below in your terminal, from inside this folder.
After each task, write your answer to the questions in `notes.md` — in your
own words, a sentence or two each. Your tutor reviews the notes in
conversation; nothing here is auto-graded.

You need an internet connection. Both tools ship with macOS; on Linux,
`curl` is usually present and `dig` comes with the `dnsutils` (Debian/Ubuntu)
or `bind-utils` (Fedora) package.

## Task 0 — assign the roles

No terminal needed yet. Using the lesson's client/server section, answer in
`notes.md`:

- Loading a web page: which program is the client, and which is the server?
- `git push` (pointed at GitHub): which program is the client, and which is
  the server?

## Task 1 — resolve a name with dig

`dig` asks DNS a question and prints the answer.

```sh
dig +short example.com
```

`+short` prints just the answer instead of the full report. Run it again
without `+short` and find the `ANSWER SECTION`.

Questions for `notes.md`:

- What kind of thing did DNS give you back?
- You likely got more than one line. Why might one name map to several
  addresses?
- In this exchange, which program was the client and what was the server's
  job?

## Task 2 — fetch a page with curl

`curl` is a client like your browser, except it prints the raw HTTP
conversation instead of drawing a page.

```sh
curl https://example.com/
```

That prints only the response **body**. Now include the status line and
headers with `-i`:

```sh
curl -i https://example.com/
```

Questions for `notes.md`:

- What status code did you get, and what does its first digit tell you?
- Pick two response headers and say what each one is telling the client.
- Where does the body start, and what is it?

## Task 3 — see the request itself

`-v` (verbose) also shows what curl *sends*: lines starting with `>` are the
request, lines starting with `<` are the response.

```sh
curl -v https://example.com/ >/dev/null
```

(The `>/dev/null` throws the body away — remember redirection from the
terminal lesson — so only the conversation remains.)

Questions for `notes.md`:

- Find the request line among the `>` lines. What method and path were sent?
- Find one request header. What is your machine telling the server?

## Task 4 — make it fail on purpose

```sh
curl -i https://example.com/no-such-page
```

Questions for `notes.md`:

- What status code came back, and whose "fault" is it — yours or the
  server's?
- The server still sent headers and a body. What does that tell you about
  the difference between "the server is down" and "the thing I asked for
  doesn't exist"?

## Task 5 — narrate the lifecycle

Close the lesson text. In `notes.md`, write the six steps from typing
`example.com` to seeing the page, in order, one line each. Then check
yourself against the lesson and fix anything you missed.
