# Authentication

> `focus.web.auth` · ~4-5h · Stage: Focus: Web Services

## Objectives

By the end of this lesson you can:

- Explain the difference between session-based and token-based authentication and when each fits a Go service.
- Implement server-side session auth with secure cookies (HttpOnly, Secure, SameSite) and session storage.
- Implement JWT issuance and validation in Go, and explain why the `alg` header and expiry claims must be verified.
- Describe the OAuth2 authorization-code flow with PKCE and where OIDC adds identity on top of OAuth2.
- Choose between sessions, JWT, and OAuth2/OIDC for a given service scenario and justify the trade-offs (revocation, statelessness, third-party login).

## Two questions, not one

**Authentication** answers *who is this request from?* **Authorization** answers
*may they do this?* This lesson stops at the first: proving identity once and
carrying it to the handler; the next lesson in this pack takes the second. Your
S5 service already has the shape this plugs into — mux, middleware, layered
handlers — and authentication is one more middleware, the one whose bugs are
worth money to somebody.

## Never store a password

You do not store passwords. You store a **verifier**: something that lets you
recognise the right password without holding it. That is a one-way function, and
the first instinct is wrong:

```go
// WRONG. Do not ship this.
sum := sha256.Sum256([]byte(password))
```

SHA-256 is a *general-purpose* hash, engineered to be fast — that is its whole
job. Commodity GPUs push billions of SHA-256 guesses per second, so a stolen
table of SHA-256 password hashes is a stolen table of passwords: every common
password falls in seconds. Worse, unsalted hashes are *identical* for identical
passwords, so one crack breaks every account that shares it, and precomputed
"rainbow" tables do the work in advance.

What you want is the opposite of fast: a **password hashing function** with a
tunable work factor and a per-password salt.

- **Argon2id** (`golang.org/x/crypto/argon2`) — memory-hard: you tune time,
  memory and parallelism, and the memory is what makes GPU and ASIC attacks
  expensive. Current first recommendation; you encode salt and parameters
  yourself.
- **bcrypt** (`golang.org/x/crypto/bcrypt`) — one cost knob, a 4 KB working set
  that already frustrates GPUs, and — the reason this exercise uses it — an
  encoded output carrying algorithm, cost and salt in one string, so storage is
  one `TEXT` column and verification is one call.
- **scrypt** is equally defensible; **PBKDF2** only where a rule names it.

SHA-256, MD5, and "SHA-256 but three times" are not on that list. This is also
the one place the exercise takes a third-party dependency: `golang.org/x/crypto`
is the Go project's own module, and the standard library ships no password KDF.

A bcrypt hash is self-describing:

```
$2a$12$DOMR5dH5BPn7V/6ekwejcOip8cOrxPy44CPVibI6JSk/yfW/BAA5e
 │  │  └── 22 characters of salt, then the 31-character hash
 │  └───── cost: 2^12 key-expansion rounds
 └──────── algorithm version
```

Three consequences to design for. **Cost is a moving target:** pick the highest
your login endpoint can afford — roughly 100-250 ms per hash on production
hardware — and raise it as machines get faster; a successful login is the only
moment you hold the plaintext, so it is the only moment you can silently re-hash
at the new cost (that is what `NeedsRehash` is for). **The bcrypt algorithm
ignores everything past 72 bytes**, and many implementations truncate silently;
Go's `x/crypto` refuses outright with `bcrypt.ErrPasswordTooLong`. You reject at
your own boundary anyway, with your own sentinel, so the behaviour is yours
rather than the library's. And **the hash still deserves care**:
keep it out of responses and logs — in the exercise `PasswordHash` carries
`json:"-"`, so no future handler can leak it by marshalling a `User`.

A **pepper** — a server-side secret mixed into the input, stored outside the
database — helps only when an attacker gets the database and not the application
secrets. A bonus, never a substitute for a real KDF.

## Comparisons that leak

Comparing a secret is its own trap. Go's `==` on strings and `bytes.Equal` both
stop at the first differing byte, so the time they take tells an attacker *how
many leading bytes they guessed right*. Against a remote service that signal is
buried in network noise — until it isn't, on a low-latency path with enough
samples. The fix costs nothing:

```go
subtle.ConstantTimeCompare(want, got) == 1   // for any secret you compare directly
hmac.Equal(wantMAC, gotMAC)                  // for MACs — same idea, right name
```

Use them whenever you compare a secret an attacker can retry: a MAC, an API key,
a password-reset token. You do *not* need them to look up a 256-bit random
session id in a map — there is nothing to walk toward when every guess costs a
round trip. (bcrypt's `CompareHashAndPassword` compares in constant time already.)

The louder leak is **user enumeration**: an endpoint that answers "no such user"
differently from "wrong password" is an account-discovery API. Same status code,
same body, and — the part people forget — the same *work*:

```go
hash := s.dummyHash          // a real bcrypt hash of a password nobody has
if found {
	hash = user.PasswordHash
}
_ = s.hasher.Verify(hash, creds.Password)   // always one verification
```

Skip the hash for unknown users and login latency becomes a yes/no oracle for
"does this account exist?". The exercise tests this by counting verifications,
not timing them: a timing assertion under `go test -race` is flaky nonsense.

## Sessions: the server remembers

The classic design, and still the right default for a browser talking to your
own service:

1. Verify the password.
2. Mint a **session id**: 32 bytes from `crypto/rand`, base64url-encoded.
   `math/rand/v2` is a fine generator and the wrong tool here — the package
   makes no unpredictability guarantee (its own docs say its output "might be
   easily predictable regardless of how it's seeded"), and its explicit sources
   (`PCG`, a seeded `ChaCha8`) are reproducible from their seed. Only
   `crypto/rand` promises what a session id needs.
3. Store `{id → user, issued, expires}` server-side, and send the id in a
   cookie. The id means nothing by itself; it is a lookup key.

The cookie attributes are the security:

| Attribute        | Why                                                                 |
|------------------|---------------------------------------------------------------------|
| `HttpOnly`       | JavaScript cannot read it, so an XSS bug cannot exfiltrate it        |
| `Secure`         | never sent over plain HTTP, so a network observer cannot lift it     |
| `SameSite=Lax`   | not attached to cross-site POSTs — a large bite out of CSRF          |
| `Max-Age`        | when the *browser* forgets it (a hint — the server decides for real) |
| `Path=/`, no `Domain` | explicit scope; a `Domain` shares the cookie with every subdomain |

In production, name the cookie `__Host-session`: browsers enforce that a
`__Host-` cookie is `Secure`, `Path=/` and domain-locked, so a compromised
sibling subdomain cannot plant one. `SameSite=Lax` is a mitigation, not a CSRF
cure — state-changing endpoints must be POST/PUT/DELETE (a `GET /logout` can be
fired by an `<img>` tag) and cross-site POSTs you *do* want still need a CSRF
token.

**Session fixation** is what makes rotation non-negotiable. If an attacker gets
a victim's browser to hold a session id *they* chose — a cookie planted from a
sibling subdomain, a URL parameter in an older design — and your login keeps
that id, the attacker's copy is now an authenticated session. The policy is one
line: **the id a request arrives with never becomes the id of an authenticated
session.** Mint a new one at login, and again whenever the session's privilege
changes (promotion to admin, a password change, a step-up re-authentication), so
an id captured under low privilege is not a high-privilege id afterwards.

Two more rules the exercise enforces. **Expiry is enforced on read,
server-side**: a cookie's `Max-Age` is advice to a client you do not control, so
the store compares `ExpiresAt` against the injected clock on every lookup
(absolute lifetime is the floor; production usually adds an idle timeout too).
And **logout deletes the server-side record**, not just the cookie — clearing
the cookie only makes the *browser* forget an id a stolen copy still uses.

The honest cost: sessions are state. An in-memory store means a restart logs
everyone out and two instances disagree about who is logged in. The fix is
boring — the same three methods over your database or a shared cache — and you
keep the property tokens cannot offer: **you can delete a session and it is
gone.**

## Tokens: the server forgets

A JWT moves the state to the client and signs it. Three base64url parts (no
padding, because the value travels in headers and URLs), joined by dots:

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9 . eyJzdWIiOiJ1MSIsImV4cCI6MTc2NH0 . 4f3s…
└── header {"alg":"HS256","typ":"JWT"}  └── claims                       └── MAC
```

The signature covers `header.payload` — the exact base64 text, not the decoded
JSON. With HS256 the same secret signs and verifies (fine inside one service);
with RS256/ES256 a private key signs and anyone with the public key verifies
(what you need when a third party issues tokens your service consumes).

Everything before the signature check is attacker-controlled input, which is why
the order of operations *is* the security:

1. Split into three parts, or reject.
2. **Pin the algorithm.** The header's `alg` is compared against what you
   accept; it never *selects* what you do. Two famous holes live here:
   - `alg: none` — a token with an empty signature that some libraries once
     accepted as "unsigned but valid";
   - **algorithm confusion** — a service that "reads alg from the header" and
     is handed `HS256` when it expected `RS256`, so it verifies an HMAC using
     the *public* key as the shared secret. The public key is public.
3. Verify the signature with `hmac.Equal`.
4. *Only now* parse the claims and check `exp` (plus `nbf`, `iss`, `aud` when you
   use them). A token with no `exp` never expires; treat it as malformed.

```go
// WRONG: the claims are trusted before anything is verified.
parts := strings.Split(token, ".")
raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
json.Unmarshal(raw, &claims)
if claims.Role == "admin" { … }   // anyone can type this token
```

**Revocation is the hard part.** A signed token is valid until it expires,
because nothing consults you when it is used: logging out, banning an account or
changing a role does not reach a token already in the wild. The standard answer
is short-lived **access tokens** (5-15 minutes) plus a long-lived **refresh
token** that is stored server-side, single-use and rotated — which means you
kept a session after all, one consulted every ten minutes instead of every
request. Denylists by `jti` and per-user token versions are the alternatives;
both re-introduce the lookup JWTs were sold as avoiding.

In a browser, a token in `localStorage` is readable by any XSS bug, so an
`HttpOnly` cookie is the better container — at which point ask why you are not
simply using a session. And although the exercise has you build HS256 by hand
(a hundred lines, after which you will never misread a JWT again), production
code uses a maintained library — `go-jose`, `golang-jwt` — because asymmetric
keys, JWKS fetching, key rotation and clock skew are where the remaining bugs
live. Tokens pay off where sessions hurt: service-to-service calls, several
services verifying identity without a shared store, identities issued elsewhere.

## Choosing

|                     | Server session         | JWT                          | OAuth2 / OIDC             |
|---------------------|------------------------|------------------------------|---------------------------|
| State lives         | on the server          | in the token                 | at the identity provider  |
| Revocation          | immediate, a delete    | only at expiry, or you rebuild state | at the provider, plus your session |
| Scale-out cost      | a shared store         | none for verification        | none for verification     |
| Browser story       | excellent (cookies)    | awkward (where to put it)    | ends in a session anyway  |
| Where it shines     | first-party web app    | service-to-service, fan-out  | "log in with…", delegated access |

The default for a first-party service is a session. Reach for tokens when
something in the second column is a real constraint, and for OIDC when you want
someone else holding the passwords.

## OAuth2 and OIDC, honestly

OAuth2 is a **delegated authorization** protocol: it gets your service a token
to act on a user's behalf at another service. It is not an authentication
protocol, which is why **OIDC** exists — a thin identity layer adding an
**`id_token`** (a JWT about *who* the user is) and a discovery document.
"Log in with…" means OIDC.

The flow you should be able to sketch is **authorization code with PKCE**:

1. Your server sends the browser to the provider with `client_id`,
   `redirect_uri`, `scope`, a random **`state`**, and a **`code_challenge`** —
   the SHA-256 of a random `code_verifier` you keep.
2. The user authenticates *there*. Your service never sees the password.
3. The provider redirects back with a short-lived **authorization code** and
   your `state`, which you compare against what you stored: that check is what
   makes the callback CSRF-resistant.
4. Your **server** exchanges the code for tokens, sending the original
   `code_verifier`. The provider hashes it and compares — proof that the party
   redeeming the code is the party that started the flow.
5. You validate the `id_token` (signature against the provider's published JWKS,
   plus `iss`, `aud`, `exp` and your `nonce`) and issue *your own* session. The
   provider's token is not your session.

PKCE exists because an authorization code can be intercepted on its way back
through a browser or a mobile OS handler; without the verifier, the code alone
is enough. It began as a mobile fix and is now recommended for every client.

Do not hand-roll this. Use `golang.org/x/oauth2` for the flow and
`github.com/coreos/go-oidc` for token validation — discovery, JWKS caching and
key rotation, `nonce` handling, clock skew and provider quirks are a moving
target a library tracks and you would not. Hand-rolled OIDC verification is one
of the most reliably broken things in web services.

## What this lesson leaves out

Deliberately, so you know they exist: multi-factor and passkeys (WebAuthn),
e-mail verification, password-reset tokens (single-use, hashed at rest,
short-lived), CSRF tokens, secret storage and rotation, a body-size limit on
the login handler (an unbounded body in front of a deliberately expensive
password hash is an amplifier — `http.MaxBytesReader`, two lessons away), and
login rate limiting
— that last one belongs to the API-hardening lesson later in this pack, as
authorization does to the next one.

## Exercise

Open [`exercise/`](exercise/) — one `auth` package, four files to complete:

```
password.go   bcrypt hashing, verification, rehash detection
session.go    session ids, the store, expiry, rotation
token.go      HS256 JWT signing and validation
service.go    Register, login, logout, promote, the middleware, cookies
```

`clock.go`, `users.go`, `respond.go`, `context.go` and the `Routes`/`handleMe`
part of `service.go` are provided — read them, especially `withAuth` (the only
way an identity reaches a request context) and the `json:"-"` on `PasswordHash`.
The `_test.go` files are the specification; `support_test.go` holds the fake
clock and the counting hasher the suite uses instead of measuring time.

Acceptance criteria:

1. `Hasher.Hash` returns a salted bcrypt hash at the configured cost (two hashes
   of one password differ) and rejects a password over `MaxPasswordBytes` with
   `ErrPasswordTooLong`.
2. `Hasher.Verify` returns `nil` on a match, `ErrBadCredentials` on a mismatch,
   and a *different* error for a corrupt stored hash. `NeedsRehash` is true when
   the stored cost is below the current one, or the hash is unreadable.
3. `NewSessionID` returns unique, unpadded-base64url ids carrying at least
   `SessionIDBytes` bytes from `crypto/rand`. `SessionStore` times sessions by
   the injected `Clock`, treats one as gone at `ExpiresAt` (dropping it from the
   map on that lookup), and `Rotate` moves a live session to a new id with fresh
   timestamps — `ErrNoSession` for an unknown or expired one. It is race-safe.
4. `SignHS256` emits `header.payload.signature`, base64url without padding,
   header `{"alg":"HS256","typ":"JWT"}`.
5. `ParseHS256` rejects, in this order: a token that is not three parts
   (`ErrMalformedToken`), any `alg` other than exactly `HS256`
   (`ErrBadAlgorithm`), a signature that does not verify (`ErrBadSignature`), an
   unparseable or `exp`-less payload (`ErrMalformedToken`), and a token at or
   past `exp` (`ErrTokenExpired`). A tampered *and* expired token reports the
   signature.
6. `Register` trims the username, rejects one under 3 characters
   (`ErrInvalidUsername`) and a password under `MinPasswordBytes`
   (`ErrPasswordTooShort`), and stores only the hash.
7. `POST /login` answers **200** with `{"data": user}` and a session cookie that
   is `HttpOnly`, `Secure` (when configured), `SameSite=Lax`, `Path=/`, with
   `MaxAge` equal to the session TTL in seconds. A non-JSON body is **400**
   `invalid JSON`; a wrong password and an unknown username are both **401**
   `invalid credentials`, byte-for-byte identical, each costing exactly one
   `Verify` call.
8. A login never keeps a session id the request arrived with, and destroys the
   previous session: after two logins the store holds one session and the first
   cookie no longer authenticates.
9. `Authenticate` answers **401** `authentication required` with no cookie, an
   unknown session, or a missing user, and clears the cookie when the session is
   dead; otherwise the handler finds the user and session in the request
   context. `POST /logout` deletes the server-side session, clears the cookie,
   and answers **204** whether or not a session existed.
10. `POST /me/promote` grants `RoleAdmin`, rotates the session id, returns the
    updated user, and leaves the pre-promotion cookie unauthenticated with one
    session in the store.
11. `go test -race ./...` is green and the code is `gofmt`-formatted.

```sh
cd exercise
go test -race ./...
```

Suggested order: `password.go`, `session.go`, then `service.go` (the HTTP tests
only pass once both are done). `token.go` stands alone — do it whenever.

## Further reading

- [OWASP — Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [OWASP — Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [pkg.go.dev — golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- [RFC 7636 — Proof Key for Code Exchange (PKCE)](https://datatracker.ietf.org/doc/html/rfc7636)
