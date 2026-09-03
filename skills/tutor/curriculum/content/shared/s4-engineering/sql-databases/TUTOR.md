# Tutor notes — SQL & Databases

## Where the learner is

Fifth lesson of S4, after clean code, code organization, TDD, and
debugging. This is their first structured SQL and their first third-party
dependency in an exercise (`modernc.org/sqlite` — the first `go test`
downloads it; a slow or blocked module download is environmental, not their
bug). Their Go is solid: `errors.Is`, `defer`, table tests, and the S3
concurrency arc, which pays off when explaining `*sql.DB` as a
concurrency-safe pool. The exercise runs against a temp-file SQLite
database — no server, no network in any test.

## Common misconceptions

- **"Parameterized queries escape the input for me."** No — the value never
  enters the SQL text at all. The statement is compiled first; values are
  bound to it as data. If they describe placeholders as "automatic
  quoting", the mechanism hasn't landed.
- **`fmt.Sprintf("%q", v)` as sanitizing.** Reject it in review even where
  tests can't catch it (numeric ids). The habit is the deliverable.
- **"`sql.Open` connects to the database."** It validates arguments and
  connects lazily; a bad path errors on first use. And `*sql.DB` is a pool,
  not "a connection" — which is exactly why `Open` sets `foreign_keys` in
  the DSN, not via `Exec`.
- **"`defer tx.Rollback()` will undo my commit."** After a successful
  `Commit` it returns `ErrTxDone` and does nothing. The pattern is safe by
  design.
- **Statements on `db` inside the transaction.** Runs outside the tx and
  won't roll back; `TestAddBatchRollsBackOnFailure`'s row count catches it.
- **Expecting `UPDATE`/`DELETE` to error on a missing row.** They succeed
  with zero rows affected; `RowsAffected` is the signal. Contrast with
  `QueryRow`, where the signal is `sql.ErrNoRows` from `Scan`.
- **`SELECT *` with `Scan`.** Couples scan order to schema column order;
  require an explicit column list.
- **`":memory:"` DSN for tests.** Each pool connection would get its own
  empty database — that's why the tests use a file in `t.TempDir()`.
- **Money as `REAL`.** Binary floats can't represent most decimal
  fractions; cents are integers. If they shrug, ask them to sum 0.10 a
  thousand times.

## Grilling points

- "Rename the category `Food` to `Groceries`: how many rows change in your
  schema? How many in a one-table design with a text `category` column on
  every expense?" (Normalization made concrete — the update anomaly.)
- "The `Empty` category doesn't appear in `TotalsByCategory`. Which part of
  your query hides it, and what would you change to show it with 0?"
  (Inner join drops it; `LEFT JOIN` + `COALESCE(SUM(…), 0)` — stretch.)
- "Trace the hostile description `O'Reilly'); DROP TABLE …` from
  `AddExpense`'s argument to the file on disk. Where exactly would
  concatenation have gone wrong?"
- "Point at the line where rollback happens when the second batch item
  fails. Now delete the `defer` in your head — list every return path that
  just broke."
- "Why does the foreign-key pragma live in the DSN string instead of an
  `Exec` after opening?" (Pool: per-connection state must apply to every
  connection.)
- "What does the transaction protect against that the `NOT NULL` and
  `REFERENCES` constraints don't?" (Multi-statement atomicity vs per-row
  validity.)

## Grading rubric

- **A** — Tests pass; every query is constant SQL with placeholders;
  `ErrNotFound` produced via `errors.Is(err, sql.ErrNoRows)` on reads and
  `RowsAffected` on writes; `AddBatch` runs every insert through `tx` with
  `defer tx.Rollback()` and a final `Commit`; explicit column lists;
  `rows.Err()` checked; gofmt-clean; can justify the two-table design and
  integer cents unprompted.
- **B** — Tests pass with minor unidiomatic spots: manual rollback calls on
  each error path instead of `defer`, `SELECT *`, `err == sql.ErrNoRows`
  instead of `errors.Is`, or a missing `rows.Err()`. Explanations solid.
- **C** — Tests pass only after heavy hinting, or explanation reveals
  placeholders are "magic quoting", or the transaction is cargo-culted
  (can't say what rollback prevents). Time-boxed remediation, then re-grill;
  else another iteration.
- **Fail** — Tests failing; any SQL built from user-supplied values even
  where tests didn't catch it; or a "compensating" AddBatch that manually
  deletes inserted rows on error instead of rolling back. Remediate, don't
  advance.

Review tip: string-built SQL with only numeric ids passes every test —
hunt for it by reading, and treat it as an automatic injection
conversation, not a silent pass.

## Remediation ladder

1. "Run the tests and read the first failure aloud — which acceptance
   criterion is it checking, and what did it get instead?"
2. Schema stuck: "Forget Go. Write the `CREATE TABLE` for `categories`
   alone, straight from criterion 1's words, and re-run only the schema
   tests." Then repeat for `expenses`.
3. Lookup stuck: "What exact error does `Scan` give you when there is no
   row? Which sentinel do the tests expect callers to see, and what wraps
   one into the other?"
4. Batch stuck: sketch the shape aloud — `Begin`, `defer Rollback`, loop of
   `tx.Exec`, `Commit` — and let them place the error returns. If the
   rollback test still counts wrong, ask: "which of your statements runs on
   `db` instead of `tx`?"

## After passing

Preview: "You just met the most famous attack on the internet first-hand.
Next lesson is Security Fundamentals — injection's whole family (OWASP),
input validation, secrets, and crypto hygiene."
