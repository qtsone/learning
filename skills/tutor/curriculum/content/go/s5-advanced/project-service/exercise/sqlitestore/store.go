// Package sqlitestore implements task.Store on top of SQLite. It is the only
// package in the service that knows SQL exists.
package sqlitestore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"tutor.local/project-service/task"
)

// timeLayout is how timestamps are stored. RFC 3339 with nanoseconds keeps
// the value sortable as text and round-trips a time.Time without loss.
const timeLayout = time.RFC3339Nano

// Store is a task.Store backed by a SQLite database file.
type Store struct {
	db *sql.DB
}

// Open opens (creating it if needed) the database at path, bounds the pool,
// and migrates the schema to the newest version. The _pragma options
// configure every pooled connection: foreign keys on, and a busy timeout so
// a briefly locked file means "wait", not "fail".
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite",
		"file:"+path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(time.Hour)
	if err := applyMigrations(ctx, db, migrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}
	return &Store{db: db}, nil
}

// Close releases the connection pool.
func (s *Store) Close() error { return s.db.Close() }

// encodeTime renders a timestamp for storage: always UTC, always the same
// layout, so string comparison and time comparison agree.
func encodeTime(t time.Time) string { return t.UTC().Format(timeLayout) }

// decodeTime parses a stored timestamp back into a UTC time.Time.
func decodeTime(s string) (time.Time, error) {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", s, err)
	}
	return t.UTC(), nil
}

// scanTask reads one row in the column order every query in this file uses.
func scanTask(row interface{ Scan(...any) error }) (task.Task, error) {
	var (
		t                    task.Task
		createdAt, updatedAt string
	)
	if err := row.Scan(&t.ID, &t.Title, &t.Status, &createdAt, &updatedAt); err != nil {
		return task.Task{}, err
	}
	var err error
	if t.CreatedAt, err = decodeTime(createdAt); err != nil {
		return task.Task{}, err
	}
	if t.UpdatedAt, err = decodeTime(updatedAt); err != nil {
		return task.Task{}, err
	}
	return t, nil
}

// Create inserts t and returns it with the id storage assigned.
func (s *Store) Create(ctx context.Context, t task.Task) (task.Task, error) {
	// TODO: INSERT the task with ? placeholders (never string-formatted
	// values), then fill in the id from the result.
	return task.Task{}, nil
}

// Get returns the task with the given id, or task.ErrNotFound.
func (s *Store) Get(ctx context.Context, id int64) (task.Task, error) {
	// TODO: QueryRowContext + scanTask. sql.ErrNoRows is a storage detail;
	// the caller must see the domain's task.ErrNotFound instead.
	return task.Task{}, nil
}

// List returns the tasks with the given status, or all of them when status
// is "". Order is not part of the contract — the service sorts.
func (s *Store) List(ctx context.Context, status task.Status) ([]task.Task, error) {
	// TODO: one query with a filter, one without. Close the rows, and check
	// rows.Err() when the loop ends — a truncated result is not an empty one.
	return nil, nil
}

// SetStatus writes the new status and updated_at, and returns the stored row.
func (s *Store) SetStatus(ctx context.Context, id int64, status task.Status, at time.Time) (task.Task, error) {
	// TODO: the update and the read-back must agree even under concurrent
	// writers, so run them in one transaction. An update that matched no row
	// means task.ErrNotFound.
	return task.Task{}, nil
}

// Delete removes the task with the given id.
func (s *Store) Delete(ctx context.Context, id int64) error {
	// TODO: DELETE, then decide from RowsAffected whether the id existed.
	return nil
}

// Ping reports whether the database is reachable right now.
func (s *Store) Ping(ctx context.Context) error {
	// TODO: database/sql already has exactly the method for this.
	return nil
}
