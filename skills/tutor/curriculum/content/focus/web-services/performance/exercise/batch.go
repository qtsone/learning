package apiperf

import "context"

// AuthorsByIDs reads many authors in one round trip.
//
// This is the cure for the N+1 you are about to measure, and it is the same
// shape as the dataloader batch function from the GraphQL lesson — same name,
// same contract, one layer lower. The rules the tests hold you to:
//
//   - an empty or nil id list makes **no** query at all and returns an empty,
//     usable map;
//   - duplicate ids collapse: [1, 1, 2] is one query with two parameters;
//   - ids with no row are simply absent from the map, not an error. The caller
//     decides what a missing author means; the store does not guess.
//
// Build the `IN (?, ?, …)` list from the deduplicated ids and pass them as
// parameters. Never format ids into the SQL string — S4's injection lesson
// applies to ints you "know" are safe, because the day one of them arrives
// from a request you will not remember this line.
//
// One production caveat worth knowing now: every database caps the number of
// bind parameters in a statement (SQLite's default is 32766, Postgres 65535).
// A batch that can exceed that must be chunked. This exercise's pages are
// bounded by MaxLimit, so one statement is always enough.
func (s *Store) AuthorsByIDs(ctx context.Context, ids []int64) (map[int64]Author, error) {
	// TODO: dedupe, build placeholders, one s.query, scan into the map.
	return nil, nil
}
