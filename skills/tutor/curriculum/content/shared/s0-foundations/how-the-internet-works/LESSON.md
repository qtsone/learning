# How the Internet Works

> `shared.foundations.how-the-internet-works` · ~1-2h · Stage: Foundations

## Objectives

By the end of this lesson you can:

- Explain the client/server model and identify the client and server in everyday examples like loading a web page.
- Describe what DNS does and why names like example.com must be resolved to IP addresses.
- Outline an HTTP request/response at a high level: method, URL, status code, headers, body.
- Trace the lifecycle of a browser request from typing a URL to the page rendering, naming each major step.

## You are one command away from the internet

Last lesson your `git push` traveled only to a folder on your own disk — a
bare repository standing in for GitHub. But point the same command at the
real GitHub and your commits cross the planet to a computer somewhere else
entirely. This lesson explains what happens in that moment, because the same
machinery powers every web page, every app, and (later in your roadmap) every
program you write that talks over a network.

The **internet** is, at its core, a worldwide network of computers that can
send each other small chunks of data. Everything else — the web, email,
`git push` — is a *conversation style* layered on top of that ability.

## Clients and servers

Almost every internet conversation has two roles:

- A **client** asks for something.
- A **server** answers.

A server is not special hardware — it is just a *program*, running on some
computer, that waits for requests and responds to them. Your laptop could be a
server. The roles describe who speaks first, not what the machines look like.

Everyday examples:

- Loading a web page: your **browser** is the client; the program hosting the
  site is the server.
- `git push`: git on your machine is the client; GitHub runs the server.
- One program can be both: GitHub's servers act as clients themselves when
  they fetch data from other services.

Remember S0's very first lesson: a program is instructions a CPU executes. A
server is exactly that — a program whose instructions say "wait for a
request, compute an answer, send it back, repeat."

## Addresses: how computers find each other

Every computer reachable on the internet has an **IP address** — a number
that identifies it, like a postal address identifies a house. The classic form
(IPv4) looks like `104.20.23.154`: four numbers from 0-255 separated by dots.
You may also meet the newer, longer form (IPv6) with colons and hex digits —
same idea, bigger address space.

Data does not travel as one big blob. It is chopped into **packets** — small,
addressed chunks — that hop from network to network until they reach the
destination IP, where they are reassembled. You never manage this yourself,
but it explains why the internet keeps working when any single cable fails:
packets simply route around the damage.

## DNS: names for numbers

Humans are bad at remembering `104.20.23.154`, and a site's address can change
when it moves to a new machine. So the internet keeps a distributed phone
book: the **Domain Name System (DNS)**. It maps names like `example.com` to IP
addresses.

**Resolving** a name means asking DNS "what IP address does this name point
to right now?" Your machine asks a nearby DNS server (a *resolver*, usually
run by your network or ISP), which asks further up the chain if it doesn't
already know. Answers are cached — remembered for a while — so repeat lookups
are fast.

Two things to internalize:

1. DNS only translates *name → address*. It never fetches the page itself.
2. One name can resolve to several addresses (big sites run many servers),
   and the address can change without the name changing. That indirection is
   the whole point.

## HTTP: the conversation itself

Once the client knows the server's address, it needs a shared language — a
**protocol**: an agreed format for messages, so both sides know exactly what
to expect. The web's protocol is **HTTP** (HyperText Transfer Protocol). It
is a simple, mostly human-readable exchange of one **request** and one
**response**.

A request carries:

- a **method** — the verb. `GET` means "give me this"; `POST` means "here is
  data, act on it" (submitting a form, for example).
- a **URL** — what the request is about: `https://example.com/about` names
  the protocol (`https`), the server (`example.com`), and the **path** on
  that server (`/about`).
- **headers** — labeled metadata lines, like fields on an envelope:
  `User-Agent: curl/8.6.0` says who is asking.
- an optional **body** — the data itself, when there is data to send.

A response carries:

- a **status code** — a three-digit summary of how it went. The first digit
  is the category: **2xx** success (`200 OK`), **3xx** "look elsewhere" — a
  redirect to another URL, **4xx** the *client* erred (`404 Not Found`: that
  path doesn't exist), **5xx** the *server* erred (`500`: it crashed trying).
- **headers** — metadata about the answer, e.g. `Content-Type: text/html`
  ("the body is a web page").
- a **body** — the actual content: the HTML of a page, an image, anything.

Note what 404 really means: the server is alive and answered you — it is
saying, politely and machine-readably, "no such thing here."

`https` is HTTP with encryption added: anyone relaying your packets sees only
scrambled bytes. It protects the conversation in transit — it says nothing
about whether the site itself is trustworthy.

## The life of a request

Trace what happens between typing `example.com` and seeing the page. Learn
these steps by name — you will meet each one again, in depth, later in the
roadmap:

1. **URL** — the browser completes what you typed into `https://example.com/`.
2. **DNS lookup** — it resolves `example.com` to an IP address (cached
   answers make this instant most of the time).
3. **Connect** — it opens a connection to that IP on a **port** — a numbered
   "door" on the server, 443 for https — so packets from both sides flow
   reliably and encrypted.
4. **Request** — it sends `GET /` over the connection.
5. **Response** — the server program builds an answer and sends back
   `200` plus headers plus the HTML body.
6. **Render** — the browser reads the HTML and draws the page. The HTML
   usually references more resources — images, styles, scripts — and *each
   one triggers its own request* (steps 2-5 again). A single page view is
   often dozens of requests.

## Exercise

Open [`exercise/`](exercise/) and follow `explore.md`. A warm-up question
assigns the client and server roles; then you watch each lifecycle step
happen for real, using two terminal tools: `dig` (asks DNS
directly) and `curl` (speaks HTTP and shows you the raw messages). Write your
answers in your own words in `exercise/notes.md` — the tutor will review and
discuss them with you; there is no automated check for this lesson.

Acceptance criteria:

1. `notes.md` names the client and the server in two examples: loading a web
   page, and `git push`.
2. You resolved `example.com` with `dig` and your notes explain what the
   answer is and why more than one address can come back.
3. You made requests with `curl` and your notes identify, from the real
   output: the method, the status code, two response headers, and the body.
4. You can narrate the six lifecycle steps from memory, in order, and say
   what each one contributes.

## Further reading

- [MDN — How does the Internet work?](https://developer.mozilla.org/en-US/docs/Learn_web_development/Howto/Web_mechanics/How_does_the_Internet_work)
- [MDN — What is a domain name?](https://developer.mozilla.org/en-US/docs/Learn_web_development/Howto/Web_mechanics/What_is_a_domain_name)
- [MDN — An overview of HTTP](https://developer.mozilla.org/en-US/docs/Web/HTTP/Overview)
- [everything curl — the book on curl](https://everything.curl.dev/)
