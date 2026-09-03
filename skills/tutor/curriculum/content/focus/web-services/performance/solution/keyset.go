package apiperf

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrBadCursor is what a client's malformed, truncated or invented cursor
// becomes. The handler turns it into 400: a cursor the server cannot read is
// the caller's problem, not a 500.
var ErrBadCursor = errors.New("invalid cursor")

// Cursor names the last row a client saw. Both fields are needed: created_at
// orders the feed, and id breaks ties between rows created in the same
// nanosecond.
type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

// EncodeCursor renders a cursor as an opaque, URL-safe string.
func EncodeCursor(c Cursor) string {
	raw := strconv.FormatInt(unixNano(c.CreatedAt), 10) + ":" + strconv.FormatInt(c.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses what EncodeCursor produced. Every failure is
// ErrBadCursor: the caller only ever needs to know "this is a 400", and the
// detail belongs in the wrapped message, not in a second error type.
func DecodeCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: not base64", ErrBadCursor)
	}
	createdAt, id, ok := strings.Cut(string(raw), ":")
	if !ok || strings.Contains(id, ":") {
		return Cursor{}, fmt.Errorf("%w: want <timestamp>:<id>", ErrBadCursor)
	}
	nanos, err := strconv.ParseInt(createdAt, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: timestamp is not an integer", ErrBadCursor)
	}
	rowID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: id is not an integer", ErrBadCursor)
	}
	return Cursor{CreatedAt: fromUnixNano(nanos), ID: rowID}, nil
}

// ListArticles returns at most limit articles, newest first, starting strictly
// after the cursor. A nil cursor means the first page.
//
// One statement, no OFFSET, and every value bound as a parameter — the cursor
// arrived from a client, and "it decodes to an int64" is not a reason to put it
// in the SQL string.
//
// The WHERE clause is a *row-value* comparison: SQLite compares the pair
// (created_at, id) against the pair from the cursor, left to right, which is
// exactly the order articles_feed is stored in. The planner recognises that
// and seeks straight to the row. Writing the same condition as
// `created_at < ? OR (created_at = ? AND id < ?)` is logically identical and
// plans as a full index scan — check it yourself with ExplainLast.
//
// The two branches are worth more than a clever single query: the first page
// and the nth page really are different questions, and the reader can see
// which index serves each.
func (s *Store) ListArticles(ctx context.Context, limit int, after *Cursor) ([]Article, error) {
	const order = ` ORDER BY created_at DESC, id DESC LIMIT ?`

	if after == nil {
		rows, err := s.query(ctx, `SELECT `+articleColumns+` FROM articles`+order, limit)
		if err != nil {
			return nil, err
		}
		return scanArticles(rows)
	}

	rows, err := s.query(ctx,
		`SELECT `+articleColumns+` FROM articles
		  WHERE (created_at, id) < (?, ?)`+order,
		unixNano(after.CreatedAt), after.ID, limit)
	if err != nil {
		return nil, err
	}
	return scanArticles(rows)
}
