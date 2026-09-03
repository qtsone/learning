package apiperf

import (
	"context"
	"strings"
)

// AuthorsByIDs reads many authors in one round trip — the cure for the N+1,
// and the same contract as the GraphQL lesson's dataloader batch function.
//
// Missing ids are absent from the map rather than an error: the store reports
// what exists, and the caller decides whether a gap is a 404, a placeholder or
// nothing at all.
//
// One statement is always enough here because pages are bounded by MaxLimit.
// A batch that could exceed the database's bind-parameter limit (32766 in
// SQLite, 65535 in Postgres) would have to be chunked, and then "one query"
// becomes "one query per chunk" — still constant per page, not per row.
func (s *Store) AuthorsByIDs(ctx context.Context, ids []int64) (map[int64]Author, error) {
	out := make(map[int64]Author, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	seen := make(map[int64]bool, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		args = append(args, id)
	}

	query := `SELECT id, name FROM authors WHERE id IN (?` +
		strings.Repeat(", ?", len(args)-1) + `)`
	rows, err := s.query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			return nil, err
		}
		out[a.ID] = a
	}
	return out, rows.Err()
}
