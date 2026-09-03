# Databases in Production

> `go.advanced.databases` · ~3-5h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Explain why pgx is preferred over database/sql+lib/pq for Postgres work
  and when the stdlib interface is still the right choice.
- Write and apply versioned SQL migrations, and explain why schema changes
  must be forward-only in production.
- Configure a connection pool (max conns, idle time, lifetimes) and explain
  how pool exhaustion manifests under load.
- Implement a multi-statement operation inside a transaction with correct
  rollback on error and context cancellation.
- Write queries that are safe from SQL injection using parameterized
  statements, and scan results into structs without an ORM.

## From "it queries" to "it operates"

In S4 you built a real persistence layer: a normalized schema, parameterized
CRUD, a report query, a transaction. That code is correct — and it would
still hurt you in production, because production adds pressures that
correctness alone doesn't cover:

- The **schema changes** after the database already holds data — and after
  old versions of your code are already running against it.
- **Connections are scarce.** Your service is one of many clients of one
  database; an unbounded pool is a denial-of-service attack you run on
  yourself.
- Every query runs **on behalf of a request** that can be canceled, time
  out, or belong to a client that already hung up.
- The engine underneath may be **PostgreSQL, not SQLite** — which makes the
  driver a real decision instead of a given.

This lesson keeps SQLite (`modernc.org/sqlite`, same driver as S4) because
every discipline here — migrations, pool limits, context, transactions —
transfers unchanged to any engine. The one engine-specific topic, drivers,
comes first.

## Choosing a driver: database/sql, lib/pq, pgx

`database/sql` is an *interface* plus a connection pool; a driver plugs in
underneath, exactly like `modernc.org/sqlite` registers itself via its blank
import. For Postgres there are two drivers you will meet:

- **lib/pq** was the standard for years. It is now in maintenance mode: the
  maintainers recommend pgx, and no feature work is happening.
- **pgx** is the actively developed Postgres driver, and it is two things at
  once: a `database/sql` driver (`pgx/v5/stdlib`) and a **native API**
  (`pgx/v5` + `pgxpool`) that skips the stdlib interface entirely.

Why does the native API exist? Because `database/sql` is the lowest common
denominator across engines, and Postgres offers more than the interface can
express: binary protocol encoding (no text round-trip for timestamps,
numerics, byte arrays), native types like arrays and `jsonb` scanned without
ceremony, query pipelining and batches, `COPY` for bulk loads,
`LISTEN/NOTIFY`, and structured errors (`pgconn.PgError` carries the
SQLSTATE code, so "unique violation" is a field check, not string matching).
A service that is committed to Postgres typically uses `pgxpool` directly
and is better for it.

When is the stdlib interface still right? When portability is the point:
code that must run against several engines (this curriculum's SQLite work),
libraries and middleware written against `*sql.DB`, or a codebase where the
lowest common denominator is all you use anyway. The two even compose — pgx
under `database/sql` via its stdlib adapter buys the protocol quality
without giving up the portable interface. What you should *not* carry out of
this lesson: "SQLite in tests, Postgres in production" as a strategy. Tests
that never touch the production engine will not catch engine-specific SQL,
type, or locking behavior; run the real engine at least in CI. Here we use
SQLite as the real engine, not as a stand-in.

## Migrations: schema is code with a history

S4's `CREATE TABLE IF NOT EXISTS` schema constant has a ceiling you hit the
first time you must *change* a table that already holds rows. `IF NOT
EXISTS` means "skip if present" — it cannot express "add a column to this
existing table".

The production answer is **versioned migrations**: the schema is an ordered,
append-only list of SQL changes, and the database itself records which
versions it has. Version 1 creates the initial tables; version 2 alters
them; version 7 backfills a column. On startup (or in a deploy step), a
runner compares the recorded version against the list and applies what's
missing, in order, each migration in its own transaction *together with its
version row* — so a crash mid-migration leaves the database at a clean
version boundary, never half-migrated.

Two rules make this safe, and both are about time:

- **Shipped migrations are frozen.** Editing migration 3 after it has run
  anywhere creates two kinds of database — those that ran the old text and
  those that ran the new — and no record of which is which. Fix mistakes by
  appending migration 8.
- **Production moves forward only.** Migration tools offer "down"
  migrations, and they are fine on your laptop. In production, rolling back
  a schema destroys data written in the new shape (drop the column you
  added and its data goes with it), and the old code you would roll back to
  has already been replaced. The escape hatch is not `down`, it is the next
  `up`: a new migration that fixes the problem.

Forward-only has a design consequence worth internalizing now: during a
deploy, old code and new schema overlap, so migrations must be **additive
first**. Add the column with a `DEFAULT` (or nullable) so old code keeps
working; ship code that uses it; only then remove what's obsolete, in a
later migration. This expand→migrate→contract choreography is called
*parallel change* — the further-reading link is worth ten minutes.

Real projects use a migration tool — `golang-migrate`, `goose`, and `atlas`
are the common Go choices, all built on exactly this model. In the exercise
you build the runner yourself, because it is ~40 lines and afterwards no
migration tool will ever be magic to you.

## The pool is infrastructure

You have known since S4 that `*sql.DB` is a connection pool. What S4 didn't
make you do is *configure* it — and the defaults are wrong for production:
**unlimited** open connections, 2 idle, kept forever.

```go
db.SetMaxOpenConns(25)                 // hard cap, in-use + idle
db.SetMaxIdleConns(25)                 // how many to keep warm
db.SetConnMaxIdleTime(5 * time.Minute) // recycle the unused
db.SetConnMaxLifetime(time.Hour)       // recycle even the used
```

Why cap it? Every open connection costs the *server*: Postgres spawns a
backend process per connection and has a hard `max_connections`; even
SQLite holds file handles and lock state per connection. An unbounded pool
converts a traffic spike into a database incident — a hundred goroutines
each demand a connection, the database drowns, and now *every* client of
that database is down, not just you. The cap turns overload into queueing
in your process, which is the cheapest place to absorb it.

Queueing has a signature you should be able to recognize from a dashboard:
**pool exhaustion**. All connections busy, new queries wait inside
`database/sql` for a free one. Throughput plateaus, tail latency spikes,
your service's CPU sits idle, and errors surface as `context deadline
exceeded` from code that "isn't slow" — the time went to waiting for a
connection, not to the query. The evidence lives in `db.Stats()`:
`WaitCount` (how often a caller had to wait) and `WaitDuration` (total time
spent waiting) climbing is the pool telling you it is too small — or that a
query is holding connections too long. The exercise has you read `Stats()`
in a test; turning `WaitCount` and `WaitDuration` into gauges is a few
lines once you have a metrics registry, and the observability lesson later
in this stage builds one.

The two recycling knobs earn their keep in ways you only notice when they
are missing: `ConnMaxLifetime` below your infrastructure's idle-kill
timeout (load balancers and connection proxies silently drop long-idle TCP
connections) means you rotate connections before something else breaks
them, and it lets the pool drift to new database instances after a
failover.

## Every query gets a context

Every `database/sql` verb you know has a `Context` variant, and from this
lesson on the plain forms are off-limits in service code:

```go
db.ExecContext(ctx, …)
db.QueryContext(ctx, …)
db.QueryRowContext(ctx, …)
tx, err := db.BeginTx(ctx, nil)
```

The context you learned to respect in your concurrency and HTTP work
reaches its most important application here: the database is where requests
spend their time. Cancellation covers the whole journey — waiting for a
pool connection (the exhaustion case above), the query in flight, and, for
a transaction begun with `BeginTx`, the transaction itself: cancel the
context and the transaction rolls back. A canceled request stops consuming
the scarcest resource you have.

The S4 transaction shape survives intact, with `BeginTx` in place of
`Begin`:

```go
tx, err := s.db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback() // no-op once Commit succeeds

// every statement goes through tx, never db
return tx.Commit()
```

One production pattern is new: **read-check-write inside one transaction**.
A transfer must read the source balance, check it, and update two rows —
and the check is only meaningful if the balance cannot change in between.
SQLite serializes writers, so the invariant holds; it does not follow that
the sequence is free.

`BeginTx(ctx, nil)` issues a plain `BEGIN`, which SQLite treats as
*deferred*: no lock until a statement needs one, and your first statement
is a `SELECT`, which needs only a read lock. The write lock is claimed
later, at the debit — and if another connection took it meanwhile, SQLite
returns `SQLITE_BUSY` **immediately, without consulting the busy handler**
(waiting mid-upgrade while holding a read lock is how deadlocks get built).
`busy_timeout` makes a *waiting* connection wait, and the loser here never
gets to wait: SQLite defends the invariant by aborting the second
transaction, not by scheduling it.

Say it at `BEGIN` instead. `BEGIN IMMEDIATE` takes the write lock up front,
where waiting is safe and `busy_timeout` does apply — the exercise's DSN
asks for it with `_txlock=immediate`, and a one-connection pool is the
blunt alternative. A transaction that will write should say so when it
begins. Postgres hands out no such lock: writers run concurrently, so the
read needs `SELECT … FOR UPDATE`, or two transactions both read 100, both
check "100 ≥ 60", and both withdraw. Same code shape, different engine
semantics — one more reason integration tests belong on the real engine.

## Structs, NULL, and no ORM

Nothing about production changes the S4 rules: SQL text is a constant, `?`
placeholders carry every value, and `Scan` maps columns to struct fields
positionally — select the columns in the order your struct expects and the
"ORM" is five lines you can read. Two additions round out the toolkit:

- **Nullable columns** don't scan into `string` or `int64` — a NULL makes
  `Scan` return an error. Either scan into `sql.NullString` /
  `sql.NullInt64` (or a pointer) and unwrap, or — better, when you control
  the schema — declare columns `NOT NULL DEFAULT …` and keep NULL out of
  your Go types entirely. The exercise's v2 migration takes the second
  road.
- When hand-written scanning grows old, the Go answer is still not a
  runtime ORM: `sqlc` generates type-safe Go from your SQL at build time —
  you keep writing real SQL, the repetition goes away. Worth knowing; not
  needed here.

## Exercise

Open [`exercise/`](exercise/) — a Go module for a small **ledger**:
accounts, and transfers between them. `migrate.go` holds the migration
list and the runner; `store.go` holds the store. The tests are the spec —
read them first, work criterion by criterion, and expect the first run to
download nothing new (the driver is the S4 one; no test touches the
network).

Acceptance criteria:

1. `applyMigrations` creates the `schema_migrations` bookkeeping table if
   missing, applies pending migrations in order, and records each version
   in the same transaction as its migration: re-running is a no-op, and a
   failing migration leaves earlier versions applied, later ones not, and
   nothing half-done.
2. Migration v2 adds `note TEXT NOT NULL DEFAULT ''` to `transfers` via
   `ALTER TABLE` — appended, with v1 left untouched.
3. `Open` applies all four `PoolConfig` knobs (the tests observe `MaxOpen`
   through `db.Stats().MaxOpenConnections`) and runs the migrations,
   closing the handle if they fail.
4. `CreateAccount` and `AccountByID` use placeholders and context; a
   missing id yields an error satisfying `errors.Is(err, ErrNotFound)`.
5. `Transfer` runs entirely in one transaction: `ErrInvalidAmount` for
   `cents <= 0`, `ErrNotFound` for an unknown account (with the debit
   rolled back when the *destination* is unknown), `ErrInsufficientFunds`
   when the balance is too small, a `transfers` row recorded on success —
   and a canceled context yields `context.Canceled` with nothing written.
   Two hundred concurrent transfers out of one account all succeed and the
   balances add up — on "database is locked", re-read `BEGIN IMMEDIATE`.
6. `History` returns the account's transfers (either direction), oldest
   first; a note containing quotes and SQL round-trips verbatim.
7. `go test -race ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test -race ./...
```

Suggested order: the runner (criterion 1 — its tests feed it their own
migration lists, so it works before the schema does), then v2 and `Open`
(criteria 2-3), then accounts (4), then the transfer (5), then history (6).

When you're green, run one thing by hand: open two terminals, start
`go test -run TestTransfer -count=100 .` in one — and while it runs, read
`db.Stats()` in your head: which numbers would move if `MaxOpen` were 1?

## Further reading

- [go.dev — Managing connections](https://go.dev/doc/database/manage-connections)
- [pgx — PostgreSQL driver and toolkit](https://github.com/jackc/pgx)
- [SQLite — ALTER TABLE](https://www.sqlite.org/lang_altertable.html)
- [Martin Fowler — Parallel Change](https://martinfowler.com/bliki/ParallelChange.html)
