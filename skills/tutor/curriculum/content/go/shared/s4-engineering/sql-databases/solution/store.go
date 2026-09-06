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
const schema = `
CREATE TABLE IF NOT EXISTS categories (
	id   INTEGER PRIMARY KEY,
	name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS expenses (
	id          INTEGER PRIMARY KEY,
	category_id INTEGER NOT NULL REFERENCES categories(id),
	description TEXT NOT NULL,
	cents       INTEGER NOT NULL
);
`

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
	res, err := s.db.Exec(`INSERT INTO categories (name) VALUES (?)`, name)
	if err != nil {
		return 0, fmt.Errorf("add category %q: %w", name, err)
	}
	return res.LastInsertId()
}

// AddExpense inserts an expense and returns the new row's id.
func (s *Store) AddExpense(categoryID int64, description string, cents int64) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO expenses (category_id, description, cents) VALUES (?, ?, ?)`,
		categoryID, description, cents,
	)
	if err != nil {
		return 0, fmt.Errorf("add expense %q: %w", description, err)
	}
	return res.LastInsertId()
}

// ExpenseByID returns the expense with the given id, or ErrNotFound.
func (s *Store) ExpenseByID(id int64) (Expense, error) {
	var e Expense
	err := s.db.QueryRow(
		`SELECT id, category_id, description, cents FROM expenses WHERE id = ?`, id,
	).Scan(&e.ID, &e.CategoryID, &e.Description, &e.Cents)
	if errors.Is(err, sql.ErrNoRows) {
		return Expense{}, fmt.Errorf("expense %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return Expense{}, fmt.Errorf("expense %d: %w", id, err)
	}
	return e, nil
}

// UpdateCents sets the amount of an existing expense, or returns
// ErrNotFound if no expense has that id.
func (s *Store) UpdateCents(id, cents int64) error {
	res, err := s.db.Exec(`UPDATE expenses SET cents = ? WHERE id = ?`, cents, id)
	if err != nil {
		return fmt.Errorf("update expense %d: %w", id, err)
	}
	return exactlyOne(res, id)
}

// DeleteExpense removes an expense, or returns ErrNotFound if no expense
// has that id.
func (s *Store) DeleteExpense(id int64) error {
	res, err := s.db.Exec(`DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete expense %d: %w", id, err)
	}
	return exactlyOne(res, id)
}

// exactlyOne translates "zero rows affected" into ErrNotFound: UPDATE and
// DELETE succeed even when their WHERE clause matches nothing.
func exactlyOne(res sql.Result, id int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("expense %d: %w", id, ErrNotFound)
	}
	return nil
}

// TotalsByCategory reports summed cents per category that has at least one
// expense, highest total first, ties broken by category name ascending.
func (s *Store) TotalsByCategory() ([]CategoryTotal, error) {
	rows, err := s.db.Query(`
		SELECT c.name, SUM(e.cents)
		FROM expenses e
		JOIN categories c ON c.id = e.category_id
		GROUP BY c.name
		ORDER BY SUM(e.cents) DESC, c.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("totals by category: %w", err)
	}
	defer rows.Close()
	var totals []CategoryTotal
	for rows.Next() {
		var t CategoryTotal
		if err := rows.Scan(&t.Category, &t.Cents); err != nil {
			return nil, fmt.Errorf("totals by category: %w", err)
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

// AddBatch inserts all items inside a single transaction: if any insert
// fails, none of the items remain in the database.
func (s *Store) AddBatch(items []Expense) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("add batch: %w", err)
	}
	defer tx.Rollback() // no-op once Commit succeeds; undoes everything on early return

	for _, e := range items {
		if _, err := tx.Exec(
			`INSERT INTO expenses (category_id, description, cents) VALUES (?, ?, ?)`,
			e.CategoryID, e.Description, e.Cents,
		); err != nil {
			return fmt.Errorf("add batch %q: %w", e.Description, err)
		}
	}
	return tx.Commit()
}
