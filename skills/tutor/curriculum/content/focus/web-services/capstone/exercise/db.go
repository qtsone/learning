package board

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Schema is the whole storage layer: the accounts, the domain table, the queue
// that lives in the same database as the data it serves (which is what makes
// the transactional outbox possible), the consumer's dedup ledger, and the
// table a job handler writes to.
//
// Times are unix nanoseconds rather than SQLite datetimes, so a fake clock and
// a real one store the same thing and every comparison is an integer compare.
const Schema = `
CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	username      TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role          TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
	id         TEXT PRIMARY KEY,
	owner_id   TEXT NOT NULL,
	title      TEXT NOT NULL,
	state      TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS tasks_feed ON tasks(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS tasks_owner_feed ON tasks(owner_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS jobs (
	id           TEXT PRIMARY KEY,
	kind         TEXT NOT NULL,
	payload      TEXT NOT NULL,
	state        TEXT NOT NULL,
	attempts     INTEGER NOT NULL DEFAULT 0,
	max_attempts INTEGER NOT NULL,
	run_at       INTEGER NOT NULL,
	lease_until  INTEGER NOT NULL DEFAULT 0,
	worker       TEXT NOT NULL DEFAULT '',
	last_error   TEXT NOT NULL DEFAULT '',
	created_at   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS jobs_claimable ON jobs(state, run_at);

CREATE TABLE IF NOT EXISTS processed_jobs (
	job_id       TEXT PRIMARY KEY,
	processed_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS notifications (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id    TEXT NOT NULL,
	owner_id   TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
`

// Open opens the SQLite database at path and applies Schema.
//
// The DSN options are the ones a service with a queue in it needs:
//   - journal_mode(WAL) lets readers run while a worker holds the write lock;
//   - busy_timeout makes a blocked writer wait instead of failing instantly;
//   - _txlock=immediate takes the write lock when a transaction begins, so a
//     transaction that reads first and writes later cannot fail with a
//     SQLITE_BUSY that busy_timeout is unable to retry away.
//
// The pool is small on purpose: SQLite serialises writers, so a large pool buys
// queueing inside the driver instead of throughput. Sizing a pool is a
// measurement, not a default.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	dsn := "file:" + path +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if _, err := db.ExecContext(ctx, Schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return db, nil
}
