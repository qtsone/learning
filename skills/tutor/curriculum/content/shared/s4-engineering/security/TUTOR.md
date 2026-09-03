# Tutor notes — Security Fundamentals

## Where the learner is

Deep into S4: they have clean-code habits, TDD, debugging discipline, and —
critically — just finished sql-databases, so `database/sql`, placeholders, and
SQLite are fresh. This lesson reframes tools they already hold (placeholders,
hashing from the CS stage, env vars from S0 terminal work) as *defenses*, and
asks for a mindset flip: read code as an attacker. The exercise is
fix-the-vulnerability, not build-from-scratch — expect it to feel easier
mechanically than conceptually, and grade the *explanations* hard.

## Common misconceptions

- **"I'll escape the quotes"** — the classic wrong fix for injection. Escaping
  is a blocklist; encodings and dialect corner cases slip past it. The channel
  separation of parameterized queries is the fix. If their `LookupUser` passes
  tests via manual quote-doubling, that is a **fail** on criterion 2 even with
  green tests — the criteria say parameterized query explicitly.
- **"SHA-256 is secure, so it's fine for passwords"** — conflates
  collision-resistance with guessing-resistance. SHA-256 is *correctly* fast;
  password storage needs *deliberate* slowness plus salt. Probe with: "secure
  against what attack, exactly?"
- **"I deleted the key in the next commit, so it's gone"** — git history
  preserves it forever; the fix for a leaked secret is revocation + rotation,
  not deletion.
- **"Check for `..` in the string"** — substring blocklists miss absolute
  paths and can reject legitimate names; `filepath.IsLocal` (or `os.Root`) is
  the allowlist answer. `strings.Contains(name, "..")` passing the tests is a
  B-grade smell — ask them what `filepath.IsLocal` does that their check
  doesn't (absolute paths, Windows device names, cleaner intent).
- **"Validation deep in the helpers is safer"** — scattering half-checks
  removes ownership; the boundary validates, the interior trusts. Ask where
  the boundary *is* in the vault package (exported functions receiving
  caller data).
- **Verifying bcrypt by re-hashing** — trying
  `HashPassword(pw) == storedHash` and being confused it never matches. The
  salt makes every hash unique; verification must go through
  `CompareHashAndPassword`, which reads salt+cost out of the stored hash.
- **A silent default when `VAULT_API_KEY` is unset** — a fallback credential
  is a hardcoded secret with extra steps; missing config must fail loudly.

## Grilling points

- "Walk me through what the database actually receives, in both channels,
  when `LookupUser` runs with the payload `nobody' OR '1'='1` — before and
  after your fix."
- "Why is escaping quotes not an acceptable fix here? What class of defense
  is it, in the allowlist/blocklist language?"
- "Your notes fix: why is `filepath.IsLocal` better than rejecting names
  containing `..`? Name an input that distinguishes them." (Absolute paths.)
- "SHA-256 is used to fingerprint git commits and verify downloads. Why is
  the very property that makes it good there — speed — fatal for passwords?"
- "What do the salt and the work factor each defend against? Where does
  bcrypt keep them?" (Salt: rainbow tables/equal-hash clustering; cost:
  offline GPU guessing; both live inside the hash string.)
- "The API key was in a private repo. Who has it now, and what is the *full*
  remediation?" (Every clone/fork/backup/CI log; revoke and rotate, then
  move to env.)
- "Point at the trust boundary in this package. Which arguments cross it?"
- Stretch: "Why does `VerifyPassword` doing a constant-time comparison
  matter? What could a variable-time compare leak?"

## Grading rubric

- **A** — All tests green with the *intended* fixes (placeholder query,
  `filepath.IsLocal` or equivalent allowlist, env-var-with-error, bcrypt);
  can narrate each attack the tests encode and name its OWASP category; can
  explain salt vs work factor and allowlist vs blocklist unprompted.
- **B** — Tests green but one fix is the clumsy variant (substring `..`
  check, or validation via regex that admits more than the spec) *or*
  explanations need prompting on one topic (typically salt vs cost).
  Mechanics right, mindset forming.
- **C** — Tests green only after heavy hints, or can fix but cannot say what
  the attack *was* (e.g. shrugs at what `' OR '1'='1` does). Remediate with
  the grilling points before advancing; security cargo-culted is security
  absent.
- **Fail** — Tests failing; or injection "fixed" by escaping/sanitizing while
  insisting that's equivalent; or cannot explain why the hardcoded key was a
  problem. Re-run the relevant lesson sections and retry.

## Remediation ladder

1. "Run `go test ./...` and read one failure aloud — the message describes an
   attack that *succeeded*. What did the attacker send, and what did they
   get?"
2. "For the injection failure: write out, on paper, the exact SQL string the
   starter builds when `name` is `nobody' OR '1'='1`. Where did data become
   code?" (Same move for notes: what path does `Join` produce?)
3. "You solved this exact problem in the SQL lesson — what did you put in the
   query string instead of the value itself? For passwords: the lesson's
   `In Go:` block shows both bcrypt calls; which replaces which function?"
4. Walk one fix together verbally — placeholder in the query, value as the
   second argument — then have them type it and generalize the pattern to the
   remaining files themselves.

## After passing

Preview: "Next you leave your own process behind: consuming HTTP APIs —
clients, timeouts, retries, and how to talk to the network without letting it
hang your program."
