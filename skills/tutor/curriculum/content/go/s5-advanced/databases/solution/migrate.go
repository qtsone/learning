package store

import (
	"context"
	"database/sql"
	"fmt"
)

// migrations is the schema's entire history, in order: version N is
// migrations[N-1]. Entries are append-only — once a version has shipped,
// you fix schema mistakes by appending a new migration, never by editing
// an old one (editing would make already-migrated databases diverge from
// freshly created ones).
var migrations = []string{
	// v1 — initial schema. Note: plain CREATE TABLE, no IF NOT EXISTS.
	// A migration runs exactly once per database; idempotence comes from
	// version bookkeeping, not from defensive SQL.
	`
CREATE TABLE accounts (
	id            INTEGER PRIMARY KEY,
	name          TEXT NOT NULL UNIQUE,
	balance_cents INTEGER NOT NULL DEFAULT 0 CHECK (balance_cents >= 0)
);

CREATE TABLE transfers (
	id      INTEGER PRIMARY KEY,
	from_id INTEGER NOT NULL REFERENCES accounts(id),
	to_id   INTEGER NOT NULL REFERENCES accounts(id),
	cents   INTEGER NOT NULL CHECK (cents > 0)
);
`,

	// v2 — additive change: NOT NULL is only legal on an existing table
	// because DEFAULT gives the rows that predate the column a value.
	`ALTER TABLE transfers ADD COLUMN note TEXT NOT NULL DEFAULT '';`,
}

// applyMigrations brings db to the newest schema version. It records
// progress in a schema_migrations table so that already-applied versions
// are skipped, and it applies each pending migration inside its own
// transaction together with its version row — a crash or a bad statement
// leaves the database at a clean version boundary, never half-migrated.
func applyMigrations(ctx context.Context, db *sql.DB, migrations []string) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	var current int
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`,
	).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	for v := current + 1; v <= len(migrations); v++ {
		if err := applyOne(ctx, db, v, migrations[v-1]); err != nil {
			return err
		}
	}
	return nil
}

// applyOne runs a single migration and its version row in one transaction:
// they become visible together or not at all.
func applyOne(ctx context.Context, db *sql.DB, version int, stmts string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migration %d: begin: %w", version, err)
	}
	defer tx.Rollback() // no-op once Commit succeeds

	if _, err := tx.ExecContext(ctx, stmts); err != nil {
		return fmt.Errorf("migration %d: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES (?)`, version,
	); err != nil {
		return fmt.Errorf("migration %d: record version: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration %d: commit: %w", version, err)
	}
	return nil
}
