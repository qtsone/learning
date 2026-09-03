// Package sqlitestore implements task.Store on top of SQLite. It is the only
// package in the service that knows SQL exists.
package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"tutor.local/project-service/task"
)

// timeLayout is how timestamps are stored. RFC 3339 with nanoseconds keeps
// the value sortable as text and round-trips a time.Time without loss.
const timeLayout = time.RFC3339Nano

// columns is the projection every read in this file uses, so one scanTask
// can serve them all.
const columns = `id, title, status, created_at, updated_at`

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
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (title, status, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		t.Title, string(t.Status), encodeTime(t.CreatedAt), encodeTime(t.UpdatedAt),
	)
	if err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}
	t.ID = id
	return t, nil
}

// Get returns the task with the given id, or task.ErrNotFound.
func (s *Store) Get(ctx context.Context, id int64) (task.Task, error) {
	t, err := scanTask(s.db.QueryRowContext(ctx,
		`SELECT `+columns+` FROM tasks WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		// sql.ErrNoRows is storage's vocabulary. Callers speak the domain's.
		return task.Task{}, fmt.Errorf("task %d: %w", id, task.ErrNotFound)
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("get task %d: %w", id, err)
	}
	return t, nil
}

// List returns the tasks with the given status, or all of them when status
// is "". Order is not part of the contract — the service sorts.
func (s *Store) List(ctx context.Context, status task.Status) ([]task.Task, error) {
	query := `SELECT ` + columns + ` FROM tasks`
	var args []any
	if status != "" {
		// The filter runs in the database, where the v2 index can serve it.
		query += ` WHERE status = ?`
		args = append(args, string(status))
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		tasks = append(tasks, t)
	}
	// A loop that ends is not the same as a query that finished: rows.Err
	// is where a mid-stream failure shows up.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// SetStatus writes the new status and updated_at, and returns the stored row.
func (s *Store) SetStatus(ctx context.Context, id int64, status task.Status, at time.Time) (task.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return task.Task{}, fmt.Errorf("set status of task %d: %w", id, err)
	}
	defer tx.Rollback() // no-op once Commit succeeds

	res, err := tx.ExecContext(ctx,
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), encodeTime(at), id)
	if err != nil {
		return task.Task{}, fmt.Errorf("set status of task %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return task.Task{}, fmt.Errorf("set status of task %d: %w", id, err)
	}
	if n == 0 {
		return task.Task{}, fmt.Errorf("task %d: %w", id, task.ErrNotFound)
	}
	// Reading inside the same transaction is what makes the returned row
	// the row this call wrote, even with other writers about.
	t, err := scanTask(tx.QueryRowContext(ctx,
		`SELECT `+columns+` FROM tasks WHERE id = ?`, id))
	if err != nil {
		return task.Task{}, fmt.Errorf("set status of task %d: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return task.Task{}, fmt.Errorf("set status of task %d: %w", id, err)
	}
	return t, nil
}

// Delete removes the task with the given id.
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("task %d: %w", id, task.ErrNotFound)
	}
	return nil
}

// Ping reports whether the database is reachable right now.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
