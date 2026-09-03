package logkit

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// storeSchema is applied by OpenStore. Real services use the versioned
// migrations you built in the databases lesson; one CREATE is enough here.
const storeSchema = `
CREATE TABLE IF NOT EXISTS events (
	id      INTEGER PRIMARY KEY AUTOINCREMENT,
	level   TEXT NOT NULL,
	source  TEXT NOT NULL,
	message TEXT NOT NULL
);`

// Store persists events in SQLite. It is the dependency the integration tests
// exercise for real, because "does this SQL actually run?" is a question no
// fake can answer.
type Store struct {
	db *sql.DB
}

// OpenStore opens the SQLite database at path, creating it if needed, and
// applies the schema. If the schema fails, the handle is closed before the
// error is returned.
func OpenStore(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to %s: %w", path, err)
	}
	if _, err := db.ExecContext(ctx, storeSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// Insert stores one event. An event whose level is unknown is rejected with an
// error wrapping ErrMalformed — the database is the last place to discover bad
// data.
func (s *Store) Insert(ctx context.Context, ev Event) error {
	if _, ok := LevelRank(ev.Level); !ok {
		return fmt.Errorf("%w: unknown level %q", ErrMalformed, ev.Level)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (level, source, message) VALUES (?, ?, ?)`,
		ev.Level, ev.Source, ev.Message)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

// All returns every stored event, oldest first.
func (s *Store) All(ctx context.Context) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT level, source, message FROM events ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var ev Event
		if err := rows.Scan(&ev.Level, &ev.Source, &ev.Message); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return events, nil
}

// CountsByLevel returns the number of stored events per level. Levels with no
// events are absent from the map.
func (s *Store) CountsByLevel(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT level, COUNT(*) FROM events GROUP BY level`)
	if err != nil {
		return nil, fmt.Errorf("query counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var level string
		var n int
		if err := rows.Scan(&level, &n); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		counts[level] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate counts: %w", err)
	}
	return counts, nil
}
