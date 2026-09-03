package store

import (
	"context"
	"database/sql"
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

	// v2 — TODO: ALTER TABLE transfers to add the column
	// note TEXT NOT NULL DEFAULT '' (acceptance criterion 2). The table
	// already holds rows in production, so the column must be additive:
	// NOT NULL is only legal here because DEFAULT gives old rows a value.
	``,
}

// applyMigrations brings db to the newest schema version. It records
// progress in a schema_migrations table (version INTEGER PRIMARY KEY,
// applied_at TEXT NOT NULL) so that already-applied versions are skipped,
// and it applies each pending migration inside its own transaction
// together with its version row — a crash or a bad statement leaves the
// database at a clean version boundary, never half-migrated.
func applyMigrations(ctx context.Context, db *sql.DB, migrations []string) error {
	// Every verb here takes ctx — the rule from this lesson holds at
	// startup too.
	// TODO: create schema_migrations if missing (this one *is*
	// IF NOT EXISTS — it is the bookkeeping, not a migration).
	// TODO: read the current version: SELECT COALESCE(MAX(version), 0).
	// TODO: for each pending version, in one transaction per migration:
	// execute the migration SQL, insert its version row, commit; on any
	// error, roll back and return an error that names the version.
	return errNotImplemented
}
