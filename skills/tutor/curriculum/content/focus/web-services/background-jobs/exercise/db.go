package jobs

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Schema is the whole storage layer of this exercise: the business table
// (orders), the queue itself (jobs), the consumer's dedup ledger
// (processed_jobs) and the table a handler writes to (emails).
//
// Times are unix nanoseconds, not SQLite datetimes, so that a fake clock and a
// real one store the same thing and comparisons are plain integer compares.
const Schema = `
CREATE TABLE IF NOT EXISTS orders (
	id          TEXT PRIMARY KEY,
	email       TEXT NOT NULL,
	total_cents INTEGER NOT NULL
);

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

CREATE TABLE IF NOT EXISTS emails (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	order_id TEXT NOT NULL,
	address  TEXT NOT NULL,
	sent_at  INTEGER NOT NULL
);
`

// Open opens the SQLite database at path and applies Schema.
//
// The DSN options are the ones a queue needs and are easy to get wrong:
//   - journal_mode(WAL) lets readers run while a worker holds the write lock;
//   - busy_timeout makes a blocked writer wait instead of failing instantly;
//   - _txlock=immediate takes the write lock when the transaction begins.
//     A deferred transaction that starts by reading and later writes can fail
//     with SQLITE_BUSY that busy_timeout cannot retry away, because SQLite
//     will not upgrade a lock underneath another writer.
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
