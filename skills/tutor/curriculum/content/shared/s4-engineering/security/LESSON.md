# Security Fundamentals

> `shared.eng.security` · ~3-4h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Describe the OWASP Top 10 categories and map each to a concrete vulnerable
  code pattern.
- Implement input validation that rejects malformed and hostile input at a
  trust boundary, and explain where the boundary lies.
- Explain why secrets must never live in source code and handle one correctly
  via environment or a secrets store.
- Hash passwords with a modern KDF (e.g. bcrypt/argon2) and explain why fast
  hashes like SHA-256 are wrong for this.
- Identify and fix seeded vulnerabilities (injection, path traversal,
  hardcoded secret) in a provided program.

## Programs meet adversaries

Everything you have tested so far assumed an honest user: wrong input was a
typo to report politely. Security starts when you drop that assumption. An
attacker reads your code the way you read the standard library — hunting for
the one input that makes it do something you never intended. You must be
right everywhere; they must be right once. That asymmetry is why security is
not a feature bolted on at the end but a property of how every function that
touches outside data is written. The good news: a handful of patterns cover
most real-world breaches, and you already have the habit (tests as
specification, from the TDD lesson) that locks fixes in place.

## The OWASP Top 10: a map of how software gets broken

OWASP (the Open Worldwide Application Security Project) periodically publishes
the ten most common categories of web application risk. Editions shuffle the
names and order (this is the 2021 list), but the underlying patterns are
stable — learn the *pattern*, not the ranking. For each: the category, and the
shape of code that has it.

| # | Category | Vulnerable pattern |
|---|----------|--------------------|
| A01 | Broken Access Control | Checking *who you are* but not *what you may touch*: `/invoice?id=42` works, so the attacker tries `43`; a file API happily serves `../../etc/passwd`. |
| A02 | Cryptographic Failures | Passwords stored in plaintext or fast hashes; secrets sent over plain HTTP; homemade encryption. |
| A03 | Injection | Untrusted text concatenated into a language someone else executes: SQL built with `+`, shell commands built from user input, HTML rendered without escaping (XSS). |
| A04 | Insecure Design | The *plan* is flawed, so no bug fix helps: unlimited login attempts, password reset via guessable "security questions". |
| A05 | Security Misconfiguration | Default admin passwords, debug endpoints reachable in production, directory listings on. |
| A06 | Vulnerable & Outdated Components | A dependency with a published CVE, still in your build because nobody updates it. |
| A07 | Identification & Auth Failures | Session tokens that never expire, predictable tokens, credential stuffing nothing slows down. |
| A08 | Software & Data Integrity Failures | Trusting unsigned updates or serialized blobs from outside; CI pipelines that run untrusted code with deploy keys. |
| A09 | Logging & Monitoring Failures | Breaches invisible for months because nothing logged auth failures; or logs that *trust* input and get forged lines injected. |
| A10 | Server-Side Request Forgery | Your server fetches a user-supplied URL, so the attacker points it at `http://169.254.169.254/` and reads your cloud credentials. |

Three of these — injection, broken access control (as path traversal), and
cryptographic failures (as a hardcoded secret and bad password storage) — you
will find and fix yourself in the exercise.

## Trust boundaries: where validation lives

A **trust boundary** is any line where data crosses from territory you do not
control into code that acts on it: command-line arguments, environment
variables, file contents, database values written by someone else, anything
that arrives over a network. Inside the boundary you may trust your own
invariants; on the boundary you may trust nothing.

Two rules make validation effective:

1. **Validate at the boundary, not deep inside.** If every layer half-checks,
   no layer is responsible and gaps appear between them. Reject bad input at
   the door, then the rest of the program can rely on it being clean.
2. **Allowlist, don't blocklist.** "Reject the characters I know are
   dangerous" loses the moment an attacker knows one you don't (there is
   always one — encodings, unicode look-alikes, null bytes). "Accept only the
   shape I know is safe" fails closed: everything you didn't anticipate is
   rejected by default.

A validation rule is a *specification*: "3-32 bytes, lowercase ASCII letters,
digits, underscore, starting with a letter" is checkable, testable, and
explainable in an error message. "No weird stuff" is none of those.

In Go:

Remember from the strings lesson that indexing a string yields *bytes*. For an
ASCII-only allowlist that is exactly what you want — checking bytes means a
sneaky multi-byte character can never pass as a letter:

```go
for i := 0; i < len(name); i++ {
	c := name[i]
	if c < 'a' || c > 'z' { // plus whatever else the allowlist admits
		return fmt.Errorf("forbidden byte %q at index %d", c, i)
	}
}
```

## Injection: never concatenate into an interpreter

Injection is the archetypal vulnerability, and one habit prevents it. It
happens whenever untrusted text is glued into a string that some engine will
*parse* — SQL, a shell, HTML, a log format. The attacker's text stops being
data and starts being syntax:

```text
query = "SELECT id FROM users WHERE name = '" + name + "'"
name  = "nobody' OR '1'='1"
→       SELECT id FROM users WHERE name = 'nobody' OR '1'='1'
```

That `OR '1'='1'` is now part of the query — it matches every row. Payloads
escalate from there: `UNION SELECT` reads other tables, `; DROP TABLE` writes.

The fix is not escaping the quotes — escaping is a blocklist, and every
database has corner cases that slip past it. The fix is to keep code and data
in separate channels: **parameterized queries**. The query text with
placeholders travels as code; the values travel separately as data the engine
will never parse. The same principle kills the other injections: pass
arguments to processes as an argument *list* (never build a shell string) and
escape output into HTML with the encoder, not by hand.

In Go:

You did this in the SQL lesson, perhaps without knowing what it was defending
against — the `?` placeholder in `database/sql`:

```go
row := db.QueryRow("SELECT id, name FROM users WHERE name = ?", name)
```

`name` can contain quotes, `OR`, or an entire hostile query — the driver hands
it to SQLite as a bound value, and it can only ever be compared as a string.

## Path traversal: the filesystem is an interpreter too

Take a program that serves files from a notes directory by joining
`dir + "/" + name` — and a caller who asks for `../secret.txt`. The path
resolver *interprets* `..` as "go up one directory", so the caller just
walked out of the sandbox to anywhere on disk the process can read. This is
injection's sibling (OWASP files it under broken access control): user data
reached an interpreter and became instructions. The defense is allowlist
thinking again: before touching the filesystem, decide whether the *name* is
one you are willing to serve — a bare filename is; anything absolute or
upward-climbing is not.

In Go:

`filepath.IsLocal` answers exactly that question — is this path relative, and
does it stay inside the directory it is joined to?

```go
if !filepath.IsLocal(name) {
	return nil, fmt.Errorf("%w: %q", ErrBadName, name)
}
data, err := os.ReadFile(filepath.Join(dir, name))
```

It rejects `../secret.txt`, `a/../../b`, absolute paths, and (on Windows)
reserved device names. Newer Go versions add `os.Root`, which makes the
operating system itself enforce the boundary for a whole directory subtree —
worth knowing once you serve files seriously.

## Secrets never belong in source code

An API key committed to a repository is not "in one file" — it is in every
clone, every fork, every backup, and **every commit forever**: deleting it in
the next commit removes nothing, because git history preserves the old blob.
Public repositories are scraped by bots that find fresh credentials in
minutes, and "private" only means the leak waits for the repo to be shared
or the CI logs to print it.

The rule: credentials — API keys, database passwords, signing keys — live
*outside* the code, read at startup. The baseline is **environment
variables**: the deployment environment sets them, the repository never sees
them, local development uses an untracked `.env`-style file. **Secret
stores** (Vault, cloud secret managers) add access control, audit, and
rotation on top — same code-side shape.

Two consequences follow. A missing secret must be a **loud error at
startup**, never a silent fallback to a baked-in default — a default
credential is a hardcoded secret with extra steps. And a secret that *has*
leaked is revoked and reissued, not merely deleted from the code: assume
every copy of history is public.

In Go:

```go
key := os.Getenv("VAULT_API_KEY")
if key == "" {
	return "", errors.New("VAULT_API_KEY is not set")
}
```

(`os.LookupEnv` additionally distinguishes "unset" from "set but empty" when
that matters.)

## Passwords: slow hashes, unique salts

Storing passwords is a special case of cryptographic hygiene with its own
trap. You never store the password itself — you store a hash and compare
hashes at login. The trap is *which* hash. SHA-256 seems obviously right: it
is the respected, irreversible hash you met in earlier lessons. It is wrong
here, for two reasons:

1. **It is fast — by design.** General-purpose hashes are built to process
   gigabytes per second. An attacker who steals your database runs the same
   hash *offline* against billions of candidate passwords per second on a
   GPU. Speed, a virtue everywhere else, is fatal here.
2. **Equal inputs give equal outputs.** Every user with password `hunter2`
   shares one hash, so cracking it once cracks them all — and precomputed
   tables of common-password hashes ("rainbow tables") crack them instantly.

Password storage therefore uses a **key derivation function (KDF)** — bcrypt,
scrypt, or argon2 — engineered to be *deliberately slow* by a tunable **work
factor** (raise it as hardware gets faster), and to mix in a random **salt**
per password so equal passwords produce different hashes and precomputation
buys nothing. The salt is not secret; it is stored inside the hash string
alongside the cost, so verification needs no extra bookkeeping.

In Go:

`golang.org/x/crypto/bcrypt` does all of it — random salt, embedded cost,
constant-time comparison:

```go
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// store string(hash); later:
ok := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
```

Note the API shape: there is no "hash the password again and compare strings"
step. You *ask the library to verify*, because it must extract the salt and
cost from the stored hash — and because it compares in constant time, so an
attacker cannot learn anything from how fast the comparison fails.

## Crypto hygiene in one paragraph

The broader rules, briefly, since they explain the lesson's choices: **never
design your own cryptography** — use the boring, audited library primitive
for the job (the exercise's SHA-256-for-passwords is exactly the kind of
plausible-looking misuse this rule prevents). Randomness that guards anything
(tokens, keys, salts) must come from the *cryptographic* random source — in
Go `crypto/rand`, never `math/rand`, whose output is predictable. And when
comparing secrets, use a constant-time comparison (`crypto/subtle`, or the
one your KDF library builds in) so timing does not leak how close a guess
was. If you notice all three are "use the well-worn tool, resist cleverness"
— that is the entire spirit of applied cryptography.

## Exercise

Open [`exercise/`](exercise/) — a small `vault` package that manages
users, notes, and credentials. It works, in the honest-user sense. It is also
riddled with seeded vulnerabilities: **read each file as an attacker before
you touch anything** — every `TODO` marks a crime scene, and the tests are
the exploits.

Your work, file by file:

- `validate.go` — implement `ValidateUsername`: an allowlist at the trust
  boundary.
- `users.go` — `LookupUser` builds SQL by concatenation. Exploit it in your
  head first (what does `nobody' OR '1'='1` do?), then fix it properly.
- `notes.go` — `ReadNote` joins caller-supplied names into paths. Make
  traversal impossible.
- `secret.go` — a live-looking API key is hardcoded. Move it to the
  environment; delete the constant.
- `password.go` — passwords are SHA-256 hashed. Replace with bcrypt
  (`golang.org/x/crypto/bcrypt` is already in `go.mod`).

Acceptance criteria:

1. `ValidateUsername` accepts exactly: 3-32 bytes, first byte a lowercase
   ASCII letter, every byte a lowercase letter, digit, or underscore. Empty,
   short, long, uppercase, spaces, non-ASCII, control bytes, and
   metacharacter smuggling are all rejected with an error.
2. `LookupUser` finds real users — including one named `o'connor` — and
   returns `ErrNotFound` for injection payloads (`' OR '1'='1`,
   `' UNION SELECT …`): they must be treated as literal, absent names. Fix
   it with a parameterized query, not by escaping quotes.
3. `ReadNote` still serves `todo.txt` but returns `ErrBadName` for
   `../secret.txt`, `sub/../../secret.txt`, and absolute paths.
4. `APIKey` returns the value of the `VAULT_API_KEY` environment variable,
   and an error when it is empty — no hardcoded key, no fallback.
5. `HashPassword`/`VerifyPassword` round-trip correctly, reject wrong
   passwords, produce *different* hashes for the same password (salt), do
   not produce plain SHA-256, and `VerifyPassword` returns false (not a
   panic) for a garbage hash.
6. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — each failure message describes the successful
attack. The first `go test` may download the dependencies; after that
everything runs offline, and no test ever touches the network. Recommended
order: `validate.go` (independent warm-up), then `users.go`, `notes.go`,
`secret.go`, `password.go`.

## Further reading

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [pkg.go.dev — golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
