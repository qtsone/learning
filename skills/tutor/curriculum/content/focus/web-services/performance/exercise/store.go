package apiperf

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
)

// ErrNotFound is returned when a row a caller named does not exist.
var ErrNotFound = errors.New("not found")

// articleColumns is the projection every article read shares, in scan order.
const articleColumns = `id, author_id, title, body, created_at`

// Store is the data layer, instrumented. Every trip to the database goes
// through query, queryRow or exec, and each of those bumps a counter.
//
// That counter is the point of this exercise. "The endpoint feels slow" is an
// opinion; "this handler makes 41 round trips to serve 40 rows" is a number a
// test can assert and a reviewer can argue with. In production the same
// instrument is a histogram of queries-per-request next to your S5 request
// metrics; here it is an int64 so the suite can be exact.
type Store struct {
	DB    *sql.DB
	Clock Clock

	queries atomic.Int64

	mu        sync.Mutex
	lastQuery string
	lastArgs  []any
}

// NewStore returns a Store over db, reading time through clock.
func NewStore(db *sql.DB, clock Clock) *Store {
	return &Store{DB: db, Clock: clock}
}

// Queries reports how many database round trips this Store has made.
func (s *Store) Queries() int64 { return s.queries.Load() }

// ResetQueries zeroes the counter. Tests call it after seeding so the number
// they assert is the work of the code under test alone.
func (s *Store) ResetQueries() { s.queries.Store(0) }

// record counts a round trip and remembers the statement, so a test can ask
// the planner what it thought of it. See ExplainLast.
func (s *Store) record(q string, args []any) {
	s.queries.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastQuery, s.lastArgs = q, args
}

func (s *Store) query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	s.record(q, args)
	return s.DB.QueryContext(ctx, q, args...)
}

func (s *Store) queryRow(ctx context.Context, q string, args ...any) *sql.Row {
	s.record(q, args)
	return s.DB.QueryRowContext(ctx, q, args...)
}

func (s *Store) exec(ctx context.Context, q string, args ...any) (sql.Result, error) {
	s.record(q, args)
	return s.DB.ExecContext(ctx, q, args...)
}

// ExplainLast asks SQLite how it intends to run the last statement this Store
// sent, and returns the plan as text.
//
// This is `EXPLAIN QUERY PLAN` — Postgres spells it `EXPLAIN (ANALYZE,
// BUFFERS)`, MySQL `EXPLAIN ANALYZE` — and it is the single most useful habit
// in this lesson: it tells you what the database will *do*, before you have
// argued about how long it took. Two words carry most of the meaning here:
//
//	SEARCH … USING INDEX …   the planner seeks into the index and stops
//	SCAN …                   the planner walks rows and discards the ones it
//	                         does not want (a SCAN "USING INDEX" still walks;
//	                         it just walks the index instead of the table)
//
// It deliberately does not go through record, so asking a question about the
// last query does not become the last query.
func (s *Store) ExplainLast(ctx context.Context) (string, error) {
	s.mu.Lock()
	q, args := s.lastQuery, append([]any(nil), s.lastArgs...)
	s.mu.Unlock()
	if q == "" {
		return "", errors.New("no statement has been run yet")
	}

	rows, err := s.DB.QueryContext(ctx, "EXPLAIN QUERY PLAN "+q, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var plan []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			return "", err
		}
		plan = append(plan, detail)
	}
	return strings.Join(plan, "\n"), rows.Err()
}

// CreateAuthor inserts an author.
func (s *Store) CreateAuthor(ctx context.Context, name string) (Author, error) {
	res, err := s.exec(ctx, `INSERT INTO authors (name) VALUES (?)`, name)
	if err != nil {
		return Author{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Author{}, err
	}
	return Author{ID: id, Name: name}, nil
}

// CreateArticle inserts an article stamped with the Store's clock.
func (s *Store) CreateArticle(ctx context.Context, authorID int64, title, body string) (Article, error) {
	now := s.Clock.Now().UTC()
	res, err := s.exec(ctx,
		`INSERT INTO articles (author_id, title, body, created_at) VALUES (?, ?, ?, ?)`,
		authorID, title, body, unixNano(now))
	if err != nil {
		return Article{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Article{}, err
	}
	return Article{ID: id, AuthorID: authorID, Title: title, Body: body, CreatedAt: now}, nil
}

// AuthorByID reads one author. One call, one round trip — which is exactly
// what makes it dangerous inside a loop.
func (s *Store) AuthorByID(ctx context.Context, id int64) (Author, error) {
	var a Author
	err := s.queryRow(ctx, `SELECT id, name FROM authors WHERE id = ?`, id).Scan(&a.ID, &a.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return Author{}, ErrNotFound
	}
	if err != nil {
		return Author{}, err
	}
	return a, nil
}

// ListArticlesOffset pages the feed the way almost everybody writes it first:
// skip `offset` rows, then take `limit`.
//
// It is here to be measured, not to be used. The benchmarks compare it against
// the keyset version you are about to write, and one test pins the correctness
// bug it has as well as the speed one. Nothing in the service should call it.
func (s *Store) ListArticlesOffset(ctx context.Context, limit, offset int) ([]Article, error) {
	rows, err := s.query(ctx,
		`SELECT `+articleColumns+` FROM articles
		  ORDER BY created_at DESC, id DESC
		  LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	return scanArticles(rows)
}

// scanArticles drains rows into articles, decoding stored timestamps.
func scanArticles(rows *sql.Rows) ([]Article, error) {
	defer rows.Close()

	out := make([]Article, 0, 16)
	for rows.Next() {
		var (
			a         Article
			createdAt int64
		)
		if err := rows.Scan(&a.ID, &a.AuthorID, &a.Title, &a.Body, &createdAt); err != nil {
			return nil, err
		}
		a.CreatedAt = fromUnixNano(createdAt)
		out = append(out, a)
	}
	return out, rows.Err()
}
