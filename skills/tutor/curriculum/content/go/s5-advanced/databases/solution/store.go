// Package store is a production-shaped persistence layer for a tiny
// ledger: versioned migrations, a bounded connection pool, and
// context-aware, transactional operations. Every query uses ?
// placeholders — never build SQL by formatting values into the text.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

var (
	// ErrNotFound reports that an account id matched no row.
	ErrNotFound = errors.New("store: account not found")
	// ErrInsufficientFunds reports a transfer larger than the source balance.
	ErrInsufficientFunds = errors.New("store: insufficient funds")
	// ErrInvalidAmount reports a zero or negative transfer amount.
	ErrInvalidAmount = errors.New("store: amount must be positive")
)

// Account is one ledger account. Money is integer cents (S4: never float).
type Account struct {
	ID           int64
	Name         string
	BalanceCents int64
}

// Transfer is one recorded movement between two accounts.
type Transfer struct {
	ID     int64
	FromID int64
	ToID   int64
	Cents  int64
	Note   string
}

// PoolConfig bounds the connection pool. The database/sql zero values mean
// "unlimited open, keep connections forever" — production code always sets
// these; see LESSON.md for why.
type PoolConfig struct {
	MaxOpen     int
	MaxIdle     int
	MaxIdleTime time.Duration
	MaxLifetime time.Duration
}

// Store persists accounts and transfers in a SQLite database file.
type Store struct {
	db *sql.DB
}

// Open opens (creating it if needed) the SQLite database at path, applies
// cfg to the pool, and migrates the schema to the newest version. The DSN
// configures every pooled connection: foreign-key enforcement on, a busy
// timeout so a waiting connection waits instead of failing, and
// _txlock=immediate so BeginTx issues BEGIN IMMEDIATE. Without the last
// one, a transaction that reads before it writes claims the write lock
// late and loses the upgrade with SQLITE_BUSY — which SQLite reports
// without ever consulting the busy handler. See LESSON.md.
func Open(path string, cfg PoolConfig) (*Store, error) {
	db, err := sql.Open("sqlite",
		"file:"+path+"?_txlock=immediate&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(cfg.MaxOpen)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxIdleTime(cfg.MaxIdleTime)
	db.SetConnMaxLifetime(cfg.MaxLifetime)
	if err := applyMigrations(context.Background(), db, migrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}
	return &Store{db: db}, nil
}

// Close releases the underlying connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateAccount inserts an account with an opening balance and returns the
// database-assigned id.
func (s *Store) CreateAccount(ctx context.Context, name string, openingCents int64) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO accounts (name, balance_cents) VALUES (?, ?)`,
		name, openingCents,
	)
	if err != nil {
		return 0, fmt.Errorf("create account %q: %w", name, err)
	}
	return res.LastInsertId()
}

// AccountByID returns the account with the given id, or ErrNotFound.
func (s *Store) AccountByID(ctx context.Context, id int64) (Account, error) {
	var a Account
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, balance_cents FROM accounts WHERE id = ?`, id,
	).Scan(&a.ID, &a.Name, &a.BalanceCents)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, fmt.Errorf("account %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return Account{}, fmt.Errorf("account %d: %w", id, err)
	}
	return a, nil
}

// Transfer moves cents from one account to another and records the
// movement in the transfers table — all inside a single transaction, so
// either the debit, the credit, and the log row all happen, or none do.
// It returns ErrInvalidAmount for cents <= 0, ErrNotFound for an unknown
// account, and ErrInsufficientFunds when the source balance is too small.
func (s *Store) Transfer(ctx context.Context, fromID, toID, cents int64, note string) error {
	if cents <= 0 {
		return fmt.Errorf("transfer of %d cents: %w", cents, ErrInvalidAmount)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transfer: %w", err)
	}
	defer tx.Rollback() // every early return below undoes the whole transfer

	// Read-check-write inside one transaction. The balance cannot change
	// between this read and the debit because the DSN's _txlock=immediate
	// took the write lock back at BEGIN; a deferred BEGIN would only
	// discover a concurrent writer here, as SQLITE_BUSY. On an engine with
	// concurrent writers this read would need FOR UPDATE instead.
	var balance int64
	err = tx.QueryRowContext(ctx,
		`SELECT balance_cents FROM accounts WHERE id = ?`, fromID,
	).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("account %d: %w", fromID, ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("transfer: read balance: %w", err)
	}
	if balance < cents {
		return fmt.Errorf("account %d holds %d cents, transfer needs %d: %w",
			fromID, balance, cents, ErrInsufficientFunds)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE accounts SET balance_cents = balance_cents - ? WHERE id = ?`,
		cents, fromID,
	); err != nil {
		return fmt.Errorf("debit account %d: %w", fromID, err)
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE accounts SET balance_cents = balance_cents + ? WHERE id = ?`,
		cents, toID,
	)
	if err != nil {
		return fmt.Errorf("credit account %d: %w", toID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("credit account %d: %w", toID, err)
	}
	if n == 0 {
		return fmt.Errorf("account %d: %w", toID, ErrNotFound)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO transfers (from_id, to_id, cents, note) VALUES (?, ?, ?, ?)`,
		fromID, toID, cents, note,
	); err != nil {
		return fmt.Errorf("record transfer: %w", err)
	}
	return tx.Commit()
}

// History returns every transfer that involves the account (as source or
// destination), oldest first.
func (s *Store) History(ctx context.Context, accountID int64) ([]Transfer, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, from_id, to_id, cents, note
		FROM transfers
		WHERE from_id = ? OR to_id = ?
		ORDER BY id`,
		accountID, accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("history of account %d: %w", accountID, err)
	}
	defer rows.Close()

	var history []Transfer
	for rows.Next() {
		var tr Transfer
		if err := rows.Scan(&tr.ID, &tr.FromID, &tr.ToID, &tr.Cents, &tr.Note); err != nil {
			return nil, fmt.Errorf("history of account %d: %w", accountID, err)
		}
		history = append(history, tr)
	}
	return history, rows.Err()
}
