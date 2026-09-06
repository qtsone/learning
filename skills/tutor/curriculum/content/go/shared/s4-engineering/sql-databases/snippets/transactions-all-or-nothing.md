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
