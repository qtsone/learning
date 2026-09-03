# SQL & Databases

> `shared.eng.sql-databases` · ~4-6h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Design a small relational schema with primary keys, foreign keys, and
  appropriate types, and justify its normalization.
- Write SELECT queries using WHERE, JOIN, GROUP BY, and ORDER BY to answer
  questions over a multi-table dataset.
- Implement CRUD operations against SQLite from code using parameterized
  queries via the standard database API.
- Explain why parameterized queries prevent SQL injection while string
  concatenation does not.
- Use a transaction to keep a multi-statement change atomic and demonstrate
  what rollback prevents.

## Why programs outgrow files

Your CLI tracker capstone persisted state the obvious way: serialize
everything to a JSON file, read it all back on start. Every team ships that
design once, and every team hits the same ceiling:

- **Querying.** "Total per category, biggest first" means loading every
  record and hand-writing the loop — again for each new question.
- **Concurrent access.** Two processes writing one file corrupt it. S3
  taught you what unsynchronized writers do to shared memory; a shared file
  is that, without even a mutex to reach for.
- **Partial writes.** Crash halfway through rewriting the file and you own
  neither the old state nor the new one.
- **Integrity.** Nothing stops a record from pointing at a category that
  was deleted last week.

A **relational database** is a program whose whole job is those four
problems: it stores data in tables, answers questions in a query language
(**SQL**), enforces rules about what data is allowed to exist, and
guarantees that a change either happens completely or not at all.

## The relational model

A database holds **tables**. A table has typed **columns** and holds
**rows**; every row has the same shape. Here is a two-table schema for
books:

```sql
CREATE TABLE authors (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE books (
    id        INTEGER PRIMARY KEY,
    author_id INTEGER NOT NULL REFERENCES authors(id),
    title     TEXT NOT NULL,
    pages     INTEGER NOT NULL
);
```

Two kinds of column do the structural work:

- A **primary key** (`id`) uniquely identifies a row. Every table gets one.
  In SQLite, `INTEGER PRIMARY KEY` also auto-assigns the next number on
  insert, so you never invent ids yourself.
- A **foreign key** (`author_id INTEGER … REFERENCES authors(id)`) says a
  row *points at* a row in another table. With enforcement on, the database
  refuses a book whose `author_id` matches no author, and refuses to delete
  an author who still has books — the "dangling reference" bug class,
  eliminated by declaration.

Constraints like `NOT NULL` and `UNIQUE` are the same idea smaller: rules
the database enforces so bad data cannot exist, no matter which program —
or which bug — does the writing.

### Normalization: every fact once

Why two tables instead of one wide `books` table with an `author_name`
column? Because with the wide table the fact "author #3 is named Le Guin"
is stored once *per book*. Copies drift: rename the author and you must
update every row; miss one and your data now disagrees with itself (an
*update anomaly*). Delete her last book and the author vanishes entirely (a
*deletion anomaly*). And one typo quietly creates a second, almost-identical
author.

**Normalization** is the discipline of structuring tables so each fact is
stored exactly once, with relationships expressed by foreign keys instead
of by copying. The formal ladder (1NF, 2NF, 3NF, …) exists, but the working
rule that covers most schemas is the 3NF slogan: every non-key column is a
fact about *the key, the whole key, and nothing but the key*. If a column
is really a fact about something else — like `author_name` is a fact about
the author, not the book — it belongs in that other table.

### Choosing types

SQLite keeps types minimal: `INTEGER`, `REAL`, `TEXT`, `BLOB`. One choice
deserves a callout because getting it wrong loses money: **store money as
integer cents, never as floating point**. Binary floating point cannot
represent most decimal fractions — `0.1` is already inexact, and summing
thousands of inexact values drifts (sum `0.1` a thousand times and print
the result; you will not get `100`). Integers are exact; format cents as
currency only at the display edge.

## Asking questions: SELECT

SQL is *declarative*: you state the result you want, and the database plans
how to compute it.

```sql
SELECT title, pages FROM books
WHERE pages >= 300
ORDER BY pages DESC, title ASC;
```

`WHERE` filters rows; `ORDER BY` sorts the result (results are otherwise in
*no guaranteed order*), with later columns breaking ties.

Normalization split your data across tables; **JOIN** recombines it by
matching foreign key to primary key:

```sql
SELECT b.title, a.name
FROM books b
JOIN authors a ON a.id = b.author_id;
```

Read it as: for each book row, find the author row its `author_id` points
at, and emit the combined row. A plain `JOIN` (an *inner* join) drops
authors with no books entirely; `LEFT JOIN` would keep them with NULLs on
the book side — remember that distinction, it decides whether "empty"
groups appear in reports.

**GROUP BY** collapses rows that share a value into one summary row, which
you fill using aggregate functions:

```sql
SELECT a.name, COUNT(*), SUM(b.pages)
FROM books b
JOIN authors a ON a.id = b.author_id
GROUP BY a.name
ORDER BY SUM(b.pages) DESC;
```

One row per author: their book count and total pages, biggest shelf first.
The rule that trips everyone: once you `GROUP BY`, the SELECT list may only
contain grouped columns and aggregates — asking for `b.title` there is
meaningless, because each output row stands for *many* books.

## SQLite: a database in a file

This lesson uses **SQLite** — not a server you install, but a library your
program embeds. The entire database is one ordinary file; the "server" is
function calls inside your own process. It is the most deployed database in
the world (every phone, every browser), and it is ideal for CLIs, desktop
apps, tests, and small services. Client/server engines like PostgreSQL and
MySQL earn their operational cost when many machines need to share one
database over a network — a later stage takes you there. The mental model
transfers: same SQL, same schema discipline, same transactions.

## Talking to a database from code

Every mainstream language ships roughly the same database API shape: open a
handle (really a **connection pool**), execute statements with
**placeholder parameters**, iterate result rows, and scan each row's
columns into your own variables.

In Go:

The standard interface is `database/sql`; a driver provides the engine.
This lesson's driver is `modernc.org/sqlite`, a pure-Go SQLite (no C
toolchain needed) — your first third-party dependency, declared in the
exercise's `go.mod`.

```go
import (
    "database/sql"

    _ "modernc.org/sqlite" // registers the "sqlite" driver
)

db, err := sql.Open("sqlite", "file:app.db?_pragma=foreign_keys(1)")
```

- The blank import runs the driver's `init`, which registers it under the
  name `"sqlite"` — imported for its side effect only.
- `sql.Open` validates arguments but does **not** connect; connections are
  made lazily, so a bad path surfaces on first use, not at open.
- `*sql.DB` is a pool, safe for concurrent use from many goroutines (S3).
  The pool has a consequence: connection-scoped state like SQLite's
  `PRAGMA foreign_keys` must go in the DSN so *every* pooled connection
  gets it — an `Exec("PRAGMA …")` would configure just one connection, and
  you don't control which. (SQLite ships with foreign-key enforcement off
  for backward compatibility; you must ask for it.)

Three verbs cover CRUD:

```go
res, err := db.Exec(`INSERT INTO authors (name) VALUES (?)`, name)
id, err := res.LastInsertId()   // Exec: statements without result rows

var a Author                    // QueryRow: exactly one expected row
err = db.QueryRow(`SELECT id, name FROM authors WHERE id = ?`, id).
    Scan(&a.ID, &a.Name)        // no row → err is sql.ErrNoRows

rows, err := db.Query(`SELECT id, name FROM authors`) // many rows
defer rows.Close()
for rows.Next() {
    var a Author
    if err := rows.Scan(&a.ID, &a.Name); err != nil { … }
}
err = rows.Err()                // did the loop end or break?
```

Two idioms matter more than they look. First, `sql.ErrNoRows` is how "not
there" arrives from `QueryRow` — check with `errors.Is` and translate it
into your package's own sentinel, exactly the error-wrapping discipline
from S1. Second, `UPDATE` and `DELETE` do *not* error when their WHERE
matches nothing — they succeed affecting zero rows — so "did it exist?"
comes from `res.RowsAffected()`, not from `err`.

## Parameterized queries: data must stay data

That `?` in every query above is the single most important habit in this
lesson. Consider the alternative:

```text
query = "SELECT id FROM users WHERE name = '" + name + "'"
```

Now let `name` arrive from a user as `x' OR '1'='1` — the statement becomes
`… WHERE name = 'x' OR '1'='1'`, true for every row. Or arrive as
`x'; DROP TABLE users; --` and your schema is gone. This is **SQL
injection**: user input crossing the line from *data* to *code*, and it has
been a top-of-the-charts vulnerability for twenty-five years.

The fix is not escaping quotes — escaping is per-driver, per-encoding, and
forgotten under deadline. The fix is structural: with a **parameterized
query**, the statement text containing `?` is compiled by the database
first, and the values travel separately, bound into the already-compiled
statement as pure data. A value can contain quotes, semicolons, or an
entire hostile SQL script; it can never change what the statement *does*,
because parsing already happened without it.

The rule: SQL text in your code is always a constant. If you catch yourself
reaching for string formatting to put a *value* into SQL, stop. (Table and
column names cannot be parameters — if those ever need to vary, validate
against a hard-coded allowlist.) The security lesson coming up shortly puts
injection in its wider family; here you make the immune habit.

## Transactions: all or nothing

Import three expenses from a bank statement. Insert one, insert two — and
insert three fails. Your database now holds a *partial import*: not the old
state, not the new state, and retrying will duplicate the first two. No
sequencing of individual statements fixes this; you need the database's
help.

A **transaction** brackets statements into one atomic unit: `BEGIN`, the
statements, then `COMMIT` to make them all real at once — or `ROLLBACK` to
return the database to exactly its pre-`BEGIN` state, as if nothing ran.
Transactions are the A and I of **ACID**: *atomic* (all or nothing),
*consistent* (constraints hold), *isolated* (concurrent transactions don't
see each other's halves), *durable* (committed means on disk). Atomicity is
today's star.

In Go:

```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback() // no-op once Commit succeeds

for _, item := range items {
    if _, err := tx.Exec(`INSERT …`, …); err != nil {
        return err // defer rolls back — nothing landed
    }
}
return tx.Commit()
```

The `defer tx.Rollback()` pattern is the transaction sibling of the
`defer f.Close()` you have written since S1: every early return — errors
you handled, errors you didn't foresee — rolls back, and after a successful
`Commit` the deferred rollback is a harmless no-op. One trap: inside the
transaction, every statement must go through `tx`, not `db`. A statement on
`db` runs *outside* the transaction and will not roll back.

## Exercise

Open [`exercise/`](exercise/) — a Go module for an expense store: the
persistence layer your tracker capstone deserved. `store.go` has the types,
a working `Open`, and `TODO`s; `store_test.go` is the spec (the TDD lesson:
red first, read the failures). The first `go test` run downloads the driver
module; after that everything is offline and no test touches any network.

Acceptance criteria:

1. The `schema` const declares two tables: `categories` with
   `id INTEGER PRIMARY KEY` and a `name TEXT` that is `NOT NULL` and
   `UNIQUE`; and `expenses` with `id INTEGER PRIMARY KEY`,
   `category_id INTEGER NOT NULL` declared `REFERENCES categories(id)`,
   `description TEXT NOT NULL`, and `cents INTEGER NOT NULL`.
2. `AddCategory` and `AddExpense` insert via placeholders and return the
   database-assigned id.
3. `ExpenseByID` returns the stored expense; a missing id yields an error
   satisfying `errors.Is(err, ErrNotFound)`. A description containing
   quotes and SQL is stored and returned verbatim.
4. `UpdateCents` and `DeleteExpense` touch exactly the addressed row and
   return `ErrNotFound` when the id matches nothing.
5. `TotalsByCategory` returns one row per category that has expenses —
   summed cents, highest total first, ties by name ascending; categories
   without expenses do not appear.
6. `AddBatch` inserts all items in one transaction: if any item is
   invalid, no items are inserted and the store remains usable.
7. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

Work in the order the criteria are listed: schema first (its tests are
independent of your Go code), then the inserts and lookup, then update and
delete, then the report query, then the transaction.

## Further reading

- [SQLite — SQL language reference](https://sqlite.org/lang.html)
- [go.dev — Accessing relational databases](https://go.dev/doc/database/)
- [pkg.go.dev — database/sql](https://pkg.go.dev/database/sql)
- [OWASP — SQL Injection Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
