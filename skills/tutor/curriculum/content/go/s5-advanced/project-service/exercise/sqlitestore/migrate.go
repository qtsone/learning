package sqlitestore

import (
	"context"
	"database/sql"
	"fmt"
)

// migrations is the schema's entire history, in order: version N is
// migrations[N-1]. Entries are append-only — once a version has shipped you
// fix a mistake by appending a new migration, never by editing an old one.
// (This is the databases lesson's pattern, provided here so the capstone is
// about composing it, not retyping it.)
var migrations = []string{
	// v1 — initial schema. Timestamps are TEXT in RFC 3339 with a UTC
	// offset: SQLite has no date type, and a sortable, unambiguous string
	// beats a bare integer you have to remember the unit of.
	`
CREATE TABLE tasks (
	id         INTEGER PRIMARY KEY,
	title      TEXT NOT NULL,
	status     TEXT NOT NULL CHECK (status IN ('open', 'done')),
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`,

	// v2 — the listing endpoint filters by status, and a filter without an
	// index is a full table scan. See NOTES.md: measure it, don't assume it.
	`CREATE INDEX idx_tasks_status ON tasks(status);`,
}

// applyMigrations brings db to the newest schema version, recording progress
// in schema_migrations so applied versions are skipped, and running each
// pending migration in its own transaction together with its version row: a
// crash leaves the database at a clean version boundary, never half-migrated.
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
