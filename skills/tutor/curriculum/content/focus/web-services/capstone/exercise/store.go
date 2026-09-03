package board

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Store is every read and write this service makes. It owns no goroutines and
// no timers: whenever it needs "now" it asks the injected Clock.
type Store struct {
	DB    *sql.DB
	Clock Clock
}

// NewStore builds a Store over an already-open database.
func NewStore(db *sql.DB, clock Clock) *Store { return &Store{DB: db, Clock: clock} }

// Scope is how much of the task table a caller may see. It is derived from the
// policy, not from a role literal, and it exists so that "you may only read
// your own tasks" can be a predicate in the query instead of a filter applied
// to rows you already loaded — a filter can forget a row; a WHERE clause
// cannot.
type Scope struct {
	// All means every row is visible.
	All bool
	// OwnerID is the only owner whose rows are visible when All is false.
	OwnerID string
}

// Cursor is a keyset position in the feed: the row after which the next page
// begins. It is not an offset — OFFSET re-counts every skipped row and shifts
// under concurrent inserts.
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// EncodeCursor renders a cursor as one opaque URL-safe token.
func EncodeCursor(c Cursor) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatInt(unixNano(c.CreatedAt), 10) + ":" + c.ID))
}

// DecodeCursor parses a token produced by EncodeCursor. Every malformed input
// is ErrBadCursor — a client error, so a handler answers 400 rather than 500.
func DecodeCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: not base64url", ErrBadCursor)
	}
	created, id, ok := strings.Cut(string(raw), ":")
	if !ok || id == "" {
		return Cursor{}, fmt.Errorf("%w: want <created>:<id>", ErrBadCursor)
	}
	n, err := strconv.ParseInt(created, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: unreadable timestamp", ErrBadCursor)
	}
	return Cursor{CreatedAt: fromUnixNano(n), ID: id}, nil
}

// CreateUser inserts an account. The caller has already hashed the password.
func (s *Store) CreateUser(ctx context.Context, u User) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, role) VALUES (?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash, u.Role)
	if err != nil {
		return fmt.Errorf("create user %s: %w", u.Username, err)
	}
	return nil
}

const userColumns = `id, username, password_hash, role`

func scanUser(row *sql.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// UserByUsername looks an account up for login.
func (s *Store) UserByUsername(ctx context.Context, username string) (User, error) {
	return scanUser(s.DB.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE username = ?`, username))
}

// UserByID looks an account up for a session.
func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	return scanUser(s.DB.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = ?`, id))
}

// SetRole changes a user's role and returns the updated account.
func (s *Store) SetRole(ctx context.Context, id string, role Role) (User, error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE users SET role = ? WHERE id = ?`, role, id)
	if err != nil {
		return User{}, fmt.Errorf("set role %s: %w", id, err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return User{}, ErrNotFound
	}
	return s.UserByID(ctx, id)
}

// InsertTask writes a task on the caller's transaction, so that it can commit
// with whatever else that transaction is doing — its outbox job, for instance.
func (s *Store) InsertTask(ctx context.Context, tx *sql.Tx, t Task) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO tasks (id, owner_id, title, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.OwnerID, t.Title, t.State, unixNano(t.CreatedAt), unixNano(t.UpdatedAt))
	if err != nil {
		return fmt.Errorf("insert task %s: %w", t.ID, err)
	}
	return nil
}

const taskColumns = `id, owner_id, title, state, created_at, updated_at`

// scanner is what *sql.Row and *sql.Rows have in common, so one scan function
// serves the single-row and the many-row query.
type scanner interface{ Scan(dest ...any) error }

func scanTask(sc scanner) (Task, error) {
	var (
		t                = Task{}
		created, updated int64
	)
	if err := sc.Scan(&t.ID, &t.OwnerID, &t.Title, &t.State, &created, &updated); err != nil {
		return Task{}, err
	}
	t.CreatedAt = fromUnixNano(created)
	t.UpdatedAt = fromUnixNano(updated)
	return t, nil
}

// TaskByID loads one task, or ErrNotFound. It applies no policy: authorization
// needs the object, so the object is loaded first and judged immediately after.
func (s *Store) TaskByID(ctx context.Context, id string) (Task, error) {
	t, err := scanTask(s.DB.QueryRowContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("task %s: %w", id, err)
	}
	return t, nil
}

// UpdateTaskState moves a task to a new state on the caller's transaction.
func (s *Store) UpdateTaskState(ctx context.Context, tx *sql.Tx, id string, state TaskState, now time.Time) error {
	res, err := tx.ExecContext(ctx,
		`UPDATE tasks SET state = ?, updated_at = ? WHERE id = ?`,
		state, unixNano(now), id)
	if err != nil {
		return fmt.Errorf("update task %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update task %s: %w", id, err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteTask removes a task. Nothing in this service is allowed to reach it —
// that is the point of the route it sits behind.
func (s *Store) DeleteTask(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task %s: %w", id, err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListTasks returns at most limit tasks, newest first, restricted to scope and
// starting strictly after cur (nil means the first page).
//
// One statement, no OFFSET, and the ordering matches the index: newest first
// with ties on created_at broken by id, which is also what makes the cursor
// comparison a total order.
func (s *Store) ListTasks(ctx context.Context, scope Scope, cur *Cursor, limit int) ([]Task, error) {
	var (
		where []string
		args  []any
	)
	if !scope.All {
		where = append(where, `owner_id = ?`)
		args = append(args, scope.OwnerID)
	}
	if cur != nil {
		where = append(where, `(created_at < ? OR (created_at = ? AND id < ?))`)
		args = append(args, unixNano(cur.CreatedAt), unixNano(cur.CreatedAt), cur.ID)
	}
	query := `SELECT ` + taskColumns + ` FROM tasks`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]Task, 0, limit)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// InsertNotification records that a task's owner was notified. This is the
// effect a background job applies, and it runs on the job's transaction.
func (s *Store) InsertNotification(ctx context.Context, tx *sql.Tx, taskID, ownerID string, at time.Time) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO notifications (task_id, owner_id, created_at) VALUES (?, ?, ?)`,
		taskID, ownerID, unixNano(at))
	if err != nil {
		return fmt.Errorf("insert notification %s: %w", taskID, err)
	}
	return nil
}

// CountNotifications is how the tests ask "did this effect happen exactly
// once?" without measuring anything.
func (s *Store) CountNotifications(ctx context.Context, taskID string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE task_id = ?`, taskID).Scan(&n)
	return n, err
}
