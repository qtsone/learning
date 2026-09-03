package apiperf

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Schema is the whole storage layer: authors, the articles they write, and one
// index.
//
// That index is the pagination lesson in three words. `ORDER BY created_at
// DESC, id DESC` is the feed's only ordering, so an index in exactly that
// order lets the database walk rows in output order and stop at LIMIT. Change
// the ORDER BY and the index stops helping — indexes are not a general speed
// setting, they are a promise about one access pattern.
const Schema = `
CREATE TABLE IF NOT EXISTS authors (
	id   INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS articles (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	author_id  INTEGER NOT NULL REFERENCES authors(id),
	title      TEXT NOT NULL,
	body       TEXT NOT NULL,
	created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS articles_feed ON articles(created_at DESC, id DESC);
`

// Open opens the SQLite database at path, applies Schema and sizes the pool.
//
// maxOpen is the interesting argument. database/sql hands out at most that many
// connections and *queues* every goroutine that wants one beyond it, so the
// number is a concurrency limit on your database, not a speed dial: too small
// and requests wait in Go, too large and they wait inside the database while it
// thrashes. `db.Stats().WaitCount` and `WaitDuration` tell you which side you
// are on. The lesson's connection-pool section is about picking it.
func Open(ctx context.Context, path string, maxOpen int) (*sql.DB, error) {
	dsn := "file:" + path +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if maxOpen < 1 {
		maxOpen = 1
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	if _, err := db.ExecContext(ctx, Schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return db, nil
}
