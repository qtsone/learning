// Package store persists expense records in a SQLite database via
// database/sql. Every query must use ? placeholders — never build SQL
// by concatenating or formatting values into the statement text.
package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// ErrNotFound reports that a lookup, update, or delete matched no row.
var ErrNotFound = errors.New("store: not found")

var errNotImplemented = errors.New("not implemented")

// Expense is one spent amount. Money is integer cents — see LESSON.md for
// why it is never a float.
type Expense struct {
	ID          int64
	CategoryID  int64
	Description string
	Cents       int64
}

// CategoryTotal is one row of the per-category spending report.
type CategoryTotal struct {
	Category string
	Cents    int64
}

// schema is applied on every Open; CREATE TABLE IF NOT EXISTS makes that
// idempotent.
//
// TODO: declare the two tables from acceptance criterion 1 in LESSON.md.
const schema = ``

// Store persists categories and expenses in a SQLite database file.
type Store struct {
	db *sql.DB
}

// Open opens (creating it if needed) the SQLite database at path and
// applies the schema. The _pragma DSN option turns foreign-key enforcement
// on for every connection in the pool; a plain `PRAGMA foreign_keys = ON`
// via Exec would only reach the single connection that happened to run it.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the underlying connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// AddCategory inserts a category and returns the new row's id.
func (s *Store) AddCategory(name string) (int64, error) {
	// TODO: INSERT with a ? placeholder; return the id via LastInsertId.
	return 0, errNotImplemented
}

// AddExpense inserts an expense and returns the new row's id.
func (s *Store) AddExpense(categoryID int64, description string, cents int64) (int64, error) {
	// TODO: INSERT with placeholders for all three values.
	return 0, errNotImplemented
}

// ExpenseByID returns the expense with the given id, or ErrNotFound.
func (s *Store) ExpenseByID(id int64) (Expense, error) {
	// TODO: QueryRow + Scan; translate sql.ErrNoRows into ErrNotFound.
	return Expense{}, errNotImplemented
}

// UpdateCents sets the amount of an existing expense, or returns
// ErrNotFound if no expense has that id.
func (s *Store) UpdateCents(id, cents int64) error {
	// TODO: UPDATE, then check RowsAffected — zero rows means ErrNotFound.
	return errNotImplemented
}

// DeleteExpense removes an expense, or returns ErrNotFound if no expense
// has that id.
func (s *Store) DeleteExpense(id int64) error {
	// TODO: DELETE, then check RowsAffected — zero rows means ErrNotFound.
	return errNotImplemented
}

// TotalsByCategory reports summed cents per category that has at least one
// expense, highest total first, ties broken by category name ascending.
func (s *Store) TotalsByCategory() ([]CategoryTotal, error) {
	// TODO: one query — JOIN, GROUP BY, ORDER BY — then iterate the rows.
	return nil, errNotImplemented
}

// AddBatch inserts all items inside a single transaction: if any insert
// fails, none of the items remain in the database.
func (s *Store) AddBatch(items []Expense) error {
	// TODO: Begin, defer Rollback, insert each item through tx, Commit.
	// Each item's ID field is ignored; the database assigns ids.
	return errNotImplemented
}
