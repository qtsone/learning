In Go:

You did this in the SQL lesson, perhaps without knowing what it was defending
against — the `?` placeholder in `database/sql`:

```go
row := db.QueryRow("SELECT id, name FROM users WHERE name = ?", name)
```

`name` can contain quotes, `OR`, or an entire hostile query — the driver hands
it to SQLite as a bound value, and it can only ever be compared as a string.
