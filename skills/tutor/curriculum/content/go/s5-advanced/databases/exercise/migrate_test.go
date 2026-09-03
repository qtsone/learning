package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// rawDB opens a bare database handle with no pool config and no
// migrations — the runner under test gets a blank slate.
func rawDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "migrate.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func appliedVersions(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("counting schema_migrations rows: %v — applyMigrations must create and fill the bookkeeping table", err)
	}
	return n
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("looking up table %q: %v", table, err)
	}
	return true
}

// --- acceptance criterion 1: the migration runner ---

func TestApplyMigrationsAppliesInOrder(t *testing.T) {
	db := rawDB(t)
	list := []string{
		`CREATE TABLE t_one (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE t_two (id INTEGER PRIMARY KEY, one_id INTEGER REFERENCES t_one(id));`,
	}
	if err := applyMigrations(context.Background(), db, list); err != nil {
		t.Fatalf("applyMigrations on a fresh database: %v", err)
	}
	for _, table := range []string{"t_one", "t_two", "schema_migrations"} {
		if !tableExists(t, db, table) {
			t.Errorf("table %q missing after applyMigrations", table)
		}
	}
	if n := appliedVersions(t, db); n != 2 {
		t.Errorf("schema_migrations holds %d version rows, want 2 — record every applied version", n)
	}
}

func TestApplyMigrationsIsIdempotent(t *testing.T) {
	db := rawDB(t)
	if err := applyMigrations(context.Background(), db, migrations); err != nil {
		t.Fatalf("first applyMigrations run: %v", err)
	}
	if err := applyMigrations(context.Background(), db, migrations); err != nil {
		t.Fatalf("second applyMigrations run: %v — applied versions must be skipped, not re-executed (plain CREATE TABLE would fail here)", err)
	}
	if n := appliedVersions(t, db); n != len(migrations) {
		t.Errorf("schema_migrations holds %d version rows after two runs, want %d", n, len(migrations))
	}
}

func TestApplyMigrationsStopsAtomicallyAndResumes(t *testing.T) {
	db := rawDB(t)
	broken := []string{
		`CREATE TABLE t_good (id INTEGER PRIMARY KEY);`,
		// The first statement is valid; the second is not. The runner's
		// per-migration transaction must roll the whole version back.
		`CREATE TABLE t_partial (id INTEGER PRIMARY KEY);
		 THIS IS NOT SQL;`,
	}
	if err := applyMigrations(context.Background(), db, broken); err == nil {
		t.Fatal("applyMigrations succeeded on a broken migration — the error must surface")
	}
	if !tableExists(t, db, "t_good") {
		t.Error("version 1 (t_good) missing — versions before the failure must stay applied")
	}
	if tableExists(t, db, "t_partial") {
		t.Error("t_partial exists after the failed version 2 — run each migration in a transaction and roll it back on error")
	}
	if n := appliedVersions(t, db); n != 1 {
		t.Errorf("schema_migrations holds %d version rows after the failure, want 1 — the version row must commit or roll back with its migration", n)
	}

	fixed := []string{
		broken[0],
		`CREATE TABLE t_partial (id INTEGER PRIMARY KEY);`,
	}
	if err := applyMigrations(context.Background(), db, fixed); err != nil {
		t.Fatalf("re-running with the migration fixed: %v — the runner must resume from the recorded version", err)
	}
	if !tableExists(t, db, "t_partial") {
		t.Error("t_partial still missing after the fixed run")
	}
	if n := appliedVersions(t, db); n != 2 {
		t.Errorf("schema_migrations holds %d version rows after the fixed run, want 2", n)
	}
}

// --- acceptance criterion 2: the v2 migration ---

func TestMigrationTwoAddsNoteColumn(t *testing.T) {
	s := newStore(t)
	var notNull int
	err := s.db.QueryRow(
		`SELECT "notnull" FROM pragma_table_info('transfers') WHERE name = 'note'`,
	).Scan(&notNull)
	if err != nil {
		t.Fatalf("transfers has no note column (%v) — write migration v2: ALTER TABLE transfers ADD COLUMN note TEXT NOT NULL DEFAULT ''", err)
	}
	if notNull != 1 {
		t.Errorf("transfers.note is nullable — declare it NOT NULL DEFAULT ''")
	}
}
