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

var errNotImplemented = errors.New("not implemented")

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
	// TODO: apply all four PoolConfig knobs (SetMaxOpenConns,
	// SetMaxIdleConns, SetConnMaxIdleTime, SetConnMaxLifetime).
	// TODO: run applyMigrations(context.Background(), db, migrations); on error, close db and
	// return the error wrapped.
	return &Store{db: db}, nil
}

// Close releases the underlying connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateAccount inserts an account with an opening balance and returns the
// database-assigned id.
func (s *Store) CreateAccount(ctx context.Context, name string, openingCents int64) (int64, error) {
	// TODO: ExecContext with placeholders; return the id via LastInsertId.
	return 0, errNotImplemented
}

// AccountByID returns the account with the given id, or ErrNotFound.
func (s *Store) AccountByID(ctx context.Context, id int64) (Account, error) {
	// TODO: QueryRowContext + Scan into an Account; translate
	// sql.ErrNoRows into ErrNotFound.
	return Account{}, errNotImplemented
}

// Transfer moves cents from one account to another and records the
// movement in the transfers table — all inside a single transaction, so
// either the debit, the credit, and the log row all happen, or none do.
// It returns ErrInvalidAmount for cents <= 0, ErrNotFound for an unknown
// account, and ErrInsufficientFunds when the source balance is too small.
func (s *Store) Transfer(ctx context.Context, fromID, toID, cents int64, note string) error {
	// TODO: validate cents first. Then BeginTx(ctx, nil), defer Rollback,
	// and do everything through tx: read the source balance (missing row
	// → ErrNotFound; balance < cents → ErrInsufficientFunds), debit the
	// source, credit the destination (RowsAffected 0 → ErrNotFound),
	// insert the transfers row, Commit.
	return errNotImplemented
}

// History returns every transfer that involves the account (as source or
// destination), oldest first.
func (s *Store) History(ctx context.Context, accountID int64) ([]Transfer, error) {
	// TODO: QueryContext + rows loop scanning into Transfer structs;
	// remember rows.Close and rows.Err.
	return nil, errNotImplemented
}
