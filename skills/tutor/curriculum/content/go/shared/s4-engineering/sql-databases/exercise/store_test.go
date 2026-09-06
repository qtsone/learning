package store

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "expenses.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func mustCategory(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	id, err := s.AddCategory(name)
	if err != nil {
		t.Fatalf("AddCategory(%q): %v", name, err)
	}
	return id
}

func mustExpense(t *testing.T, s *Store, categoryID int64, description string, cents int64) int64 {
	t.Helper()
	id, err := s.AddExpense(categoryID, description, cents)
	if err != nil {
		t.Fatalf("AddExpense(%q): %v", description, err)
	}
	return id
}

func expenseCount(t *testing.T, s *Store) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM expenses`).Scan(&n); err != nil {
		t.Fatalf("counting expenses: %v", err)
	}
	return n
}

// --- acceptance criterion 1: schema ---

func TestSchemaTablesExist(t *testing.T) {
	s := newStore(t)
	for _, table := range []string{"categories", "expenses"} {
		var name string
		err := s.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q does not exist — complete the schema const (%v)", table, err)
		}
	}
}

func TestSchemaPrimaryKeys(t *testing.T) {
	s := newStore(t)
	for _, table := range []string{"categories", "expenses"} {
		var pk int
		err := s.db.QueryRow(
			`SELECT pk FROM pragma_table_info(?) WHERE name = 'id'`, table,
		).Scan(&pk)
		if err != nil || pk != 1 {
			t.Errorf("table %q needs a column id INTEGER PRIMARY KEY (err %v, pk %d)", table, err, pk)
		}
	}
}

func TestSchemaForeignKeyDeclared(t *testing.T) {
	s := newStore(t)
	var toTable, fromCol string
	err := s.db.QueryRow(
		`SELECT "table", "from" FROM pragma_foreign_key_list('expenses')`,
	).Scan(&toTable, &fromCol)
	if err != nil {
		t.Fatalf("expenses declares no foreign key — category_id must REFERENCES categories(id) (%v)", err)
	}
	if toTable != "categories" || fromCol != "category_id" {
		t.Errorf("foreign key is %s -> %s, want category_id -> categories", fromCol, toTable)
	}
}

func TestSchemaForeignKeyEnforced(t *testing.T) {
	s := newStore(t)
	_, err := s.db.Exec(
		`INSERT INTO expenses (category_id, description, cents) VALUES (999, 'orphan', 100)`,
	)
	if err == nil {
		t.Fatal("an expense whose category_id matches no category was accepted — the foreign key is missing")
	}
}

func TestSchemaCategoryNameUnique(t *testing.T) {
	s := newStore(t)
	if _, err := s.db.Exec(`INSERT INTO categories (name) VALUES ('Food')`); err != nil {
		t.Fatalf("inserting a category: %v", err)
	}
	if _, err := s.db.Exec(`INSERT INTO categories (name) VALUES ('Food')`); err == nil {
		t.Fatal("the same category name was accepted twice — name must be UNIQUE")
	}
}

// --- acceptance criteria 2-3: create and read ---

func TestAddAndGetExpense(t *testing.T) {
	s := newStore(t)
	food := mustCategory(t, s, "Food")
	if food == 0 {
		t.Fatal("AddCategory returned id 0 — return the id the database assigned (LastInsertId)")
	}
	id := mustExpense(t, s, food, "lunch", 950)
	got, err := s.ExpenseByID(id)
	if err != nil {
		t.Fatalf("ExpenseByID(%d): %v", id, err)
	}
	want := Expense{ID: id, CategoryID: food, Description: "lunch", Cents: 950}
	if got != want {
		t.Errorf("ExpenseByID(%d) = %+v, want %+v", id, got, want)
	}
}

func TestExpenseByIDMissing(t *testing.T) {
	s := newStore(t)
	_, err := s.ExpenseByID(42)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ExpenseByID on a missing id: got err %v, want ErrNotFound", err)
	}
}

func TestHostileDescriptionIsJustData(t *testing.T) {
	s := newStore(t)
	books := mustCategory(t, s, "Books")
	hostile := `O'Reilly'); DROP TABLE expenses; --`
	id, err := s.AddExpense(books, hostile, 4200)
	if err != nil {
		t.Fatalf("AddExpense with quotes in the description failed (%v) — use ? placeholders, not string building", err)
	}
	got, err := s.ExpenseByID(id)
	if err != nil {
		t.Fatalf("ExpenseByID(%d): %v", id, err)
	}
	if got.Description != hostile {
		t.Errorf("description round-trip: got %q, want %q — the value must be stored verbatim", got.Description, hostile)
	}
	if n := expenseCount(t, s); n != 1 {
		t.Errorf("expenses table holds %d rows after the insert, want 1 — was the table dropped?", n)
	}
}

// --- acceptance criterion 4: update and delete ---

func TestUpdateCents(t *testing.T) {
	s := newStore(t)
	food := mustCategory(t, s, "Food")
	lunch := mustExpense(t, s, food, "lunch", 950)
	coffee := mustExpense(t, s, food, "coffee", 300)

	if err := s.UpdateCents(lunch, 1200); err != nil {
		t.Fatalf("UpdateCents(%d, 1200): %v", lunch, err)
	}
	got, err := s.ExpenseByID(lunch)
	if err != nil {
		t.Fatalf("ExpenseByID(%d): %v", lunch, err)
	}
	if got.Cents != 1200 {
		t.Errorf("after UpdateCents, cents = %d, want 1200", got.Cents)
	}
	other, err := s.ExpenseByID(coffee)
	if err != nil {
		t.Fatalf("ExpenseByID(%d): %v", coffee, err)
	}
	if other.Cents != 300 {
		t.Errorf("UpdateCents changed another row: coffee cents = %d, want 300 — check your WHERE clause", other.Cents)
	}
	if err := s.UpdateCents(9999, 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("UpdateCents on a missing id: got err %v, want ErrNotFound — check RowsAffected", err)
	}
}

func TestDeleteExpense(t *testing.T) {
	s := newStore(t)
	food := mustCategory(t, s, "Food")
	id := mustExpense(t, s, food, "lunch", 950)

	if err := s.DeleteExpense(id); err != nil {
		t.Fatalf("DeleteExpense(%d): %v", id, err)
	}
	if _, err := s.ExpenseByID(id); !errors.Is(err, ErrNotFound) {
		t.Errorf("expense %d still readable after delete (err %v)", id, err)
	}
	if err := s.DeleteExpense(id); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleting an already-deleted id: got err %v, want ErrNotFound — check RowsAffected", err)
	}
}

// --- acceptance criterion 5: the report query ---

func TestTotalsByCategory(t *testing.T) {
	s := newStore(t)
	rent := mustCategory(t, s, "Rent")
	food := mustCategory(t, s, "Food")
	books := mustCategory(t, s, "Books")
	transport := mustCategory(t, s, "Transport")
	mustCategory(t, s, "Empty") // no expenses — must not appear in the report

	mustExpense(t, s, food, "lunch", 950)
	mustExpense(t, s, food, "groceries", 450)
	mustExpense(t, s, rent, "august rent", 120000)
	mustExpense(t, s, books, "novel", 300)
	mustExpense(t, s, transport, "bus pass", 300)

	got, err := s.TotalsByCategory()
	if err != nil {
		t.Fatalf("TotalsByCategory: %v", err)
	}
	want := []CategoryTotal{
		{Category: "Rent", Cents: 120000},
		{Category: "Food", Cents: 1400},
		{Category: "Books", Cents: 300}, // ties on cents order by name
		{Category: "Transport", Cents: 300},
	}
	if !slices.Equal(got, want) {
		t.Errorf("TotalsByCategory() =\n  %+v\nwant\n  %+v", got, want)
	}
}

// --- acceptance criterion 6: the transaction ---

func TestAddBatchCommitsAll(t *testing.T) {
	s := newStore(t)
	food := mustCategory(t, s, "Food")
	batch := []Expense{
		{CategoryID: food, Description: "milk", Cents: 250},
		{CategoryID: food, Description: "bread", Cents: 180},
	}
	if err := s.AddBatch(batch); err != nil {
		t.Fatalf("AddBatch with all-valid items: %v", err)
	}
	if n := expenseCount(t, s); n != 2 {
		t.Errorf("after a valid batch of 2, expenses table holds %d rows, want 2", n)
	}
}

func TestAddBatchRollsBackOnFailure(t *testing.T) {
	s := newStore(t)
	food := mustCategory(t, s, "Food")
	mustExpense(t, s, food, "existing", 100)

	batch := []Expense{
		{CategoryID: food, Description: "valid first item", Cents: 250},
		{CategoryID: 9999, Description: "no such category", Cents: 300},
	}
	if err := s.AddBatch(batch); err == nil {
		t.Fatal("AddBatch with an invalid category_id reported success — the failed insert must surface as an error")
	}
	if n := expenseCount(t, s); n != 1 {
		t.Errorf("after a failed batch the expenses table holds %d rows, want 1 — the valid first item must be rolled back with the rest", n)
	}
	if _, err := s.AddExpense(food, "after the failed batch", 50); err != nil {
		t.Errorf("store unusable after a failed batch (%v) — did you forget to Rollback?", err)
	}
}
