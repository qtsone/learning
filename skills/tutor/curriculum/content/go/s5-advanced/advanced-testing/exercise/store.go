package logkit

import (
	"context"
	"database/sql"

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
	// TODO: sql.Open("sqlite", path), verify the connection, apply
	// storeSchema, and clean up on failure.
	return nil, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	// TODO
	return nil
}

// Insert stores one event. An event whose level is unknown is rejected with an
// error wrapping ErrMalformed — the database is the last place to discover bad
// data.
func (s *Store) Insert(ctx context.Context, ev Event) error {
	// TODO: validate, then insert with placeholders and a context.
	return nil
}

// All returns every stored event, oldest first.
func (s *Store) All(ctx context.Context) ([]Event, error) {
	// TODO: query, scan, and check rows.Err().
	return nil, nil
}

// CountsByLevel returns the number of stored events per level. Levels with no
// events are absent from the map.
func (s *Store) CountsByLevel(ctx context.Context) (map[string]int, error) {
	// TODO: GROUP BY in SQL, not in Go.
	return nil, nil
}
