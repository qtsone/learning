# Security notes — notes

Written at the end of the hardening pass, 2026-03-08. It describes what this
program trusts, what it does not, and what it is knowingly not defending
against. It is short because the program is small; it is not short because the
questions were skipped.

## Trust boundaries

There are three places where something we do not control becomes something we
act on. **Stdin** is the first: an operator or a script pipes note lines in,
and every byte of that is attacker-controlled as soon as the file came from
somewhere else. **The environment** is the second: `NOTES_WEBHOOK` decides
where the listing is sent, so anyone who can set it can exfiltrate the
listing. **The webhook response** is the third and least obvious — a peer we
call is still a peer that can answer with a gigabyte.

Nothing crosses those boundaries without passing through `note.ParseLine`,
`remote.Publish`, or the response cap in `remote`. Inside those, values are
trusted: the domain type is only constructible through `note.New`, so an
invalid `Note` cannot exist.

## Inputs and validation

| Surface | Limit | Enforced in |
|---|---|---|
| stdin line length | 1024 bytes, and a bufio buffer of the same size | `cmd/notes`, `note.MaxLineBytes` |
| line encoding | valid UTF-8, no control characters | `note.ParseLine` |
| field count | exactly `id\|title` or `id\|title\|tags` | `note.ParseLine` |
| note id | `[A-Za-z0-9_-]{1,64}` | `note.ValidID` |
| title | 1-120 runes after trimming | `note.New` |
| tags | lower-cased, de-duplicated, empty dropped | `note.NormaliseTags` |
| webhook URL | must parse, scheme must be http or https | `remote.Publish` |
| webhook response | first 64 KiB read, rest discarded | `remote.Publish` |

Validation happens once, at the boundary, and produces a value the rest of the
program can use without re-checking. A rejected line is reported with its line
number and skipped; one bad line in an import does not cost the operator the
other 999.

## Secrets handling

The only secret is the webhook URL, whose query string usually carries a
token. It arrives in the `NOTES_WEBHOOK` environment variable — not a flag,
because flags are visible in the process list, and not a file in the
repository. It is never logged: every error mentioning it goes through
`remote.Redact`, which keeps the scheme and host and drops everything else —
userinfo, path, query and fragment — because a webhook token is as likely to
sit in a path segment as in the query. The URL is parsed by us rather than by
`net/http` precisely because the standard library's parse error quotes the
whole URL back at you. There are no
credentials, keys or hashes anywhere else in the program, and no persistence,
so there is nothing at rest to protect.

## Dependency policy

Standard library only. A third-party module gets added when writing it
ourselves would be worse, and then only if it is maintained, has a licence we
can live with, and pulls in a dependency tree we are willing to read. That
policy is cheap to hold at this size and it is the reason the vulnerability
surface is the Go toolchain itself.

`govulncheck ./...` is run by hand against the toolchain currently in use,
and the result is recorded here with its date and Go version — including
when it finds something, since the standard library gets advisories too and
the usual fix is upgrading the toolchain. It is run again before each release
and after every toolchain upgrade, because govulncheck reports what is
*reachable*, and a change in our code can make a dormant advisory reachable
without any dependency changing.

## Findings and fixes

1. **Note ids were not validated (fixed).** Ids were passed through unchecked,
   and the file-backed store planned in ADR-001 would have joined them onto a
   directory path — so `../../etc/passwd` would have escaped it. Ids now match
   a strict character class before a `Note` can exist.
   Regression test: TestParseLineRejectsHostileInput
2. **Unbounded input (fixed).** The scanner grew its buffer until the input
   decided how much memory the process used, and `ParseLine` had no size limit
   at all. Both are capped at `note.MaxLineBytes` now.
   Regression test: TestRunRejectsOversizedLine
3. **Webhook token in error text (fixed).** The first version wrapped the
   `net/http` parse error, which quotes the URL, so a failed publish printed
   the token to stderr. Errors are redacted now.
   Regression test: TestPublishRejectsBadURLs
4. **Control characters reached the terminal (fixed).** Titles containing
   escape sequences were printed straight to stdout, which lets an import file
   repaint the operator's terminal. Control characters are now rejected at the
   boundary.
   Regression test: TestParseLineRejectsHostileInput

## Accepted risks

- **No authentication or authorisation.** This is a single-operator command
  line tool: anyone who can run it already has the shell it runs in. If it
  ever grows a server, this section is wrong and the threat model is rewritten
  before that ships.
- **Webhook responses are not verified.** We check the status code and discard
  the body; we do not pin certificates or verify a signature. The blast radius
  is one wrong exit code, which we accepted rather than build a trust store.
- **No rate limiting on the webhook.** A pathological import could send one
  large request; there is no loop, so there is nothing to throttle yet. The
  trigger to revisit is the first scheduled or unattended run.
- **Notes are not persisted, so there is nothing to encrypt at rest.** That
  follows from ADR-001 rather than from a security decision, and the moment
  storage lands, "what is on disk and who can read it" becomes a new section
  here.
