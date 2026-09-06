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
