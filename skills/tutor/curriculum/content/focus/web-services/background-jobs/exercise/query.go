package jobs

import (
	"context"
	"database/sql"
	"fmt"
)

// scanner is what both *sql.Row and *sql.Rows satisfy.
type scanner interface {
	Scan(dest ...any) error
}

// scanJob reads one row in jobColumns order.
func scanJob(row scanner) (Job, error) {
	var (
		j          Job
		runAt      int64
		leaseUntil int64
	)
	err := row.Scan(&j.ID, &j.Kind, &j.Payload, &j.State, &j.Attempts,
		&j.MaxAttempts, &runAt, &leaseUntil, &j.Worker, &j.LastError)
	if err != nil {
		return Job{}, err
	}
	j.RunAt = fromUnixNano(runAt)
	j.LeaseUntil = fromUnixNano(leaseUntil)
	return j, nil
}

// expectOneRow turns "the row I meant to update is not there" into an error
// instead of a silent no-op.
func expectOneRow(res sql.Result, op, id string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s %s: %w", op, id, err)
	}
	if n != 1 {
		return fmt.Errorf("%s %s: %w", op, id, ErrNoJob)
	}
	return nil
}

// Get reads one job by id.
func (s *Store) Get(ctx context.Context, id string) (Job, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = ?`, id)
	return scanJob(row)
}

// CountByState reports how many jobs sit in each state — the queue depth
// numbers you would export as metrics (remember S5's observability lesson:
// "dead" climbing is an alert, "ready" climbing is a capacity signal).
func (s *Store) CountByState(ctx context.Context) (map[string]int, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT state, COUNT(*) FROM jobs GROUP BY state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var state string
		var n int
		if err := rows.Scan(&state, &n); err != nil {
			return nil, err
		}
		counts[state] = n
	}
	return counts, rows.Err()
}
