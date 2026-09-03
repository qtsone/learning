# Tutor notes — Databases in Production

## Where the learner is

Fourth lesson of S5. They wrote real `database/sql` code in S4 (normalized
schema, parameterized CRUD, one transaction, `modernc.org/sqlite`), and they
have just built HTTP, REST, and gRPC services — so they know contexts,
`errors.Is/As`, and boundary error translation. Nothing here is a new API
family; the whole lesson is about the *operational* half of persistence that
S4 deliberately skipped: schema over time, bounded connections, and every
query hanging off a cancelable request.

Two things reliably cause friction. First, writing a migration runner feels
like reinventing a wheel — hold the line that it is ~40 lines and demystifies
every tool they will ever use. Second, the driver discussion is the one part
they cannot exercise: they have no Postgres here, so it must land as
*reasoning* (grill it, don't hand-wave it). The pgx material is knowledge for
the job they will take, not code they will write today.

## Common misconceptions

- **"Migrations are just my schema constant with more steps."** The test to
  offer: "add a column to a table that already holds a million rows — write
  it with `CREATE TABLE IF NOT EXISTS`." `IF NOT EXISTS` means *skip*, and
  skip cannot alter. That is the whole argument.
- **Editing a shipped migration instead of appending.** Very common when they
  spot a typo in v1 while writing v2. Ask what happens to the database that
  already ran the old v1 text — it will never run it again, so now two shapes
  of database exist and nothing records which is which.
- **"Down migrations are the rollback plan."** They are a laptop convenience.
  In production the down migration destroys data written in the new shape,
  and the old code you would roll back to is already gone. Forward-only: the
  fix is the next `up`.
- **Confusing `IF NOT EXISTS` on `schema_migrations` with defensive
  migrations.** The bookkeeping table is created defensively *because* it is
  not a migration; the migrations themselves are plain `CREATE TABLE`, and
  the idempotence test only passes if the version rows do the work.
- **Committing the migration and its version row separately.** Two `Exec`
  calls outside a transaction pass the happy-path tests and fail
  `TestApplyMigrationsStopsAtomicallyAndResumes`. Make them say aloud what a
  crash between the two writes leaves behind.
- **"Bigger pool, faster service."** The pool exists to protect the *server*.
  Push until they say the cap converts a database-wide outage into queueing
  inside their own process, which is the cheapest place to absorb overload.
- **Reading pool exhaustion as slow queries.** The symptom is `context
  deadline exceeded` from code that is not slow, flat throughput, idle CPU.
  The evidence is `db.Stats().WaitCount` / `WaitDuration`, not the query
  plan.
- **`defer tx.Rollback()` "undoing" a successful commit.** It is a no-op
  after `Commit`; the deferred call returning `sql.ErrTxDone` is expected and
  ignored. If they wrote `if err != nil { tx.Rollback() }` on every branch
  instead, the code still passes — but ask which of the six early returns
  they would forget in the next refactor.
- **Doing the read through `s.db` and the writes through `tx`.** Compiles,
  passes some tests, and quietly breaks the atomicity the lesson is about.
  Grep their `Transfer` for `s.db.` after `BeginTx`.
- **"SQLite serializes writers, so read-check-write is free."** Half true,
  and the expensive half is the missing one: a deferred `BEGIN` takes the
  write lock at the first *write*, so two read-first transfers collide on the
  upgrade and the loser gets `SQLITE_BUSY` — reported immediately, with the
  busy handler never consulted, so `busy_timeout` does not save it. That is
  what `_txlock=immediate` in the DSN is for, and
  `TestConcurrentTransfersDoNotCollide` is what catches its removal. On
  Postgres the same code needs `SELECT … FOR UPDATE` instead. This is the
  single most transferable fact in the lesson — do not let it pass unspoken.
- **Thinking cancellation only stops the query in flight.** It also aborts
  the wait for a pool connection and rolls back a `BeginTx` transaction.
  `TestTransferCanceledContext` cancels *before* the call, so the failure
  comes out of `BeginTx` — they must wrap with `%w` or `errors.Is` fails.
- **"Placeholders are for convenience/escaping."** They are not escaping:
  the SQL text and the values travel separately, so a value can never become
  syntax. The hostile-note test is the proof, and `note` scanning back
  verbatim (quotes, semicolons, `--`) is the point.

## Grilling points

- "You are on call. The service is timing out, CPU is 4%, the database looks
  bored. Which three numbers do you look at first, and what would each tell
  you?" (`Stats()`: `InUse`, `WaitCount`, `WaitDuration`; then query
  duration.)
- "Your team is starting a new Postgres-only service. Argue for `pgxpool`,
  then argue against it. What would make you choose `database/sql` anyway?"
  (For: binary protocol, native arrays/`jsonb`, batching, `COPY`,
  `LISTEN/NOTIFY`, `pgconn.PgError` SQLSTATE instead of string matching.
  Against: portability, `*sql.DB`-shaped libraries, lowest-common-denominator
  usage. Bonus if they reach for `pgx/v5/stdlib` as the middle road.)
- "Ship a rename of `transfers.note` to `transfers.memo` with zero downtime.
  Write the migration order and say what is running against what at each
  step." (Expand → migrate → contract; three deploys, not one.)
- "Delete the `defer tx.Rollback()` line. Which test breaks and what happens
  to the database file in production?" (Connection returned to the pool with
  an open transaction; SQLite holds the write lock — a real incident.)
- "Two goroutines call `Transfer(alice, bob, …)` on the same source at the
  same time. Walk me through it here, then on Postgres." (Here `BEGIN
  IMMEDIATE` claims the write lock at `BEGIN`, so the second one waits out
  its `busy_timeout` and then proceeds. Take `_txlock=immediate` off the DSN
  and it instead fails with `SQLITE_BUSY` when it tries to upgrade at the
  debit — no busy-handler retry. On Postgres both can read 100 and both
  withdraw without `FOR UPDATE`.)
- "Why does `MaxLifetime` exist when `MaxIdleTime` already recycles idle
  connections?" (Load balancers and proxies kill long-lived TCP; failover
  moves the instance; you want to break them on your schedule.)
- "The `note` column is `NOT NULL DEFAULT ''`. What would have changed in
  your Go code if it were nullable?" (`sql.NullString` or a pointer plus
  unwrapping everywhere — the design cost of NULL leaks into every scan.)
- "Where would you put `db.Stats()` in a service?" (Seeds the observability
  lesson — take the answer, don't teach it yet.)

## Grading rubric

- **A** — All tests green under `-race`; each migration and its version row
  commit in one transaction; `Transfer` does *everything* through `tx`,
  validates before `BeginTx`, wraps sentinels with `%w`, and uses
  `RowsAffected` (not a second query) to detect the unknown destination;
  `History` closes rows and returns `rows.Err()`; can argue pgx-vs-stdlib
  both directions and state the SQLite-vs-Postgres locking difference
  unprompted.
- **B** — Tests green with rough edges: version row written outside the
  migration's transaction but still passing by luck of ordering, sentinels
  returned bare (`return ErrNotFound` — `errors.Is` passes, the id is lost),
  `rows.Err()` skipped, or `Open` not closing the handle when migrations
  fail. Understands pooling and migrations but recites the pgx material
  rather than reasoning about it.
- **C** — Passes only after the remediation ladder, or cannot explain why
  re-running `applyMigrations` does not re-execute v1, or thinks placeholders
  are an escaping mechanism. Time-box remediation and re-grill the
  transaction and injection points before passing.
- **Fail** — Tests failing; or SQL built with `fmt.Sprintf`/concatenated
  values anywhere; or `Transfer` mixing `s.db` and `tx`; or unable to say
  what a crash mid-migration leaves behind. Remediate — this material sits
  directly under the stage capstone.

## Remediation ladder

1. "Run `go test -race ./... 2>&1 | head -30` and read the first failure
   aloud. The messages name the criterion — which one is it, and which file
   owns it?" (Suggested order is in LESSON.md: runner → v2 → `Open` →
   accounts → transfer → history.)
2. Runner stuck: "Forget SQL. You have a list of N things and a number saying
   how many are done. Write the loop in Go on paper." Then: "Now, what must
   be true if the process is killed exactly between running migration 2 and
   recording that it ran?" — that question produces the transaction.
3. Transfer stuck: "List every write the transfer performs and every reason
   it can fail *after* the first write. How many of those paths currently
   leave money missing?" Then hand them the shape only —
   `tx, err := s.db.BeginTx(ctx, nil)` / `defer tx.Rollback()` /
   `return tx.Commit()` — and let them fill the middle.
4. Still stuck on a specific criterion: name the exact tool without writing
   the line. Unknown destination → "`Exec` returns a `sql.Result`; what does
   it tell you about rows that matched?" Canceled context → "which call is
   the first to touch `ctx`, and how does its error reach the caller
   unwrapped by `errors.Is`?" Missing account → "which sentinel error does
   `QueryRow(...).Scan` return, and what do you translate it to at this
   boundary?" (Same boundary translation as their gRPC/REST error mapping.)
   Only after all four, walk the solution shape verbally — never paste it.

## After passing

Preview: "Next you go underneath your own code: the runtime and scheduler —
what a goroutine actually costs and how the runtime decides who runs. It is a
discussion lesson, so bring questions rather than a passing test."
