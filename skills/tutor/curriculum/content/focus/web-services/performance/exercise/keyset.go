package apiperf

import (
	"context"
	"errors"
	"time"
)

// ErrBadCursor is what a client's malformed, truncated or invented cursor
// becomes. The handler turns it into 400: a cursor the server cannot read is
// the caller's problem, not a 500.
var ErrBadCursor = errors.New("invalid cursor")

// Cursor names the last row a client saw. Both fields are needed: created_at
// orders the feed, and id breaks ties between rows created in the same
// nanosecond. Drop the id and a page boundary that lands inside a tie either
// repeats rows or skips them.
type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

// EncodeCursor renders a cursor as an opaque, URL-safe string.
//
// The exact encoding the tests expect: the decimal Unix-nanosecond timestamp,
// a colon, the decimal id — for example "1700000000000000000:42" — then
// base64.RawURLEncoding over those bytes.
//
// Opaque is a contract, not secrecy. Base64 keeps clients from parsing the
// shape and depending on it, so you can change the ordering later. It is not a
// signature: anyone can decode and edit one, which is why DecodeCursor
// validates and why the query below still passes the values as parameters.
func EncodeCursor(c Cursor) string {
	// TODO: encoding/base64 with RawURLEncoding, over "<unixnano>:<id>".
	return ""
}

// DecodeCursor parses what EncodeCursor produced.
//
// Every failure — an empty string, non-base64 input, the wrong number of
// fields, a field that is not an integer — must return an error that
// errors.Is matches to ErrBadCursor. Round-tripping a Cursor through
// EncodeCursor and DecodeCursor must give back the same values.
func DecodeCursor(s string) (Cursor, error) {
	// TODO: decode, split on ":", parse both integers, rebuild the Cursor
	// with fromUnixNano.
	return Cursor{}, ErrBadCursor
}

// ListArticles returns at most limit articles, newest first, starting strictly
// after the cursor. A nil cursor means the first page.
//
// This must be **one** query and it must not use OFFSET. The ordering is
// `created_at DESC, id DESC` — the same order as the articles_feed index — and
// "strictly after the cursor" in that ordering can be written two ways:
//
//	WHERE (created_at, id) < (?, ?)                                   -- (a)
//	WHERE created_at < ? OR (created_at = ? AND id < ?)               -- (b)
//
// They are logically identical. They are not equally fast, and you are not
// expected to take that on faith: `TestListArticlesSeeksInsteadOfScanning`
// asks SQLite's own planner, and only (a) — the row-value comparison — comes
// back as a SEARCH into articles_feed. Form (b) reads as a SCAN, because a
// planner facing an OR of two conditions usually gives up on the index. The
// lesson shows both plans; the point is the habit of asking rather than the
// specific answer.
//
// Write it with two branches (nil cursor, non-nil cursor) rather than one
// clever SQL string. Pass every value as a parameter: a cursor is client input.
func (s *Store) ListArticles(ctx context.Context, limit int, after *Cursor) ([]Article, error) {
	// TODO: build the two queries, run one through s.query, and hand the rows
	// to scanArticles.
	return nil, nil
}
