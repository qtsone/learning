package vault

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	for _, name := range []string{"alice", "o'connor"} {
		if _, err := db.Exec(`INSERT INTO users (name) VALUES (?)`, name); err != nil {
			t.Fatalf("insert %q: %v", name, err)
		}
	}
	return db
}

func TestLookupUserFindsExisting(t *testing.T) {
	db := newTestDB(t)
	u, err := LookupUser(db, "alice")
	if err != nil {
		t.Fatalf("LookupUser(alice) error: %v", err)
	}
	if u.Name != "alice" || u.ID != 1 {
		t.Errorf("LookupUser(alice) = %+v, want ID 1, Name alice", u)
	}
}

func TestLookupUserMissing(t *testing.T) {
	db := newTestDB(t)
	if _, err := LookupUser(db, "mallory"); !errors.Is(err, ErrNotFound) {
		t.Errorf("LookupUser(mallory) error = %v, want ErrNotFound", err)
	}
}

func TestLookupUserQuoteInName(t *testing.T) {
	db := newTestDB(t)
	u, err := LookupUser(db, "o'connor")
	if err != nil {
		t.Fatalf("LookupUser(o'connor) error: %v — a quote in a name must be data, not SQL", err)
	}
	if u.Name != "o'connor" {
		t.Errorf("LookupUser(o'connor) = %+v, want Name o'connor", u)
	}
}

func TestLookupUserResistsInjection(t *testing.T) {
	payloads := []string{
		"nobody' OR '1'='1",
		"nobody' UNION SELECT 99, 'evil' --",
	}
	for _, p := range payloads {
		t.Run(p, func(t *testing.T) {
			db := newTestDB(t)
			u, err := LookupUser(db, p)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("LookupUser(%q) = %+v, %v — want ErrNotFound: the payload ran as SQL instead of being treated as a literal name", p, u, err)
			}
		})
	}
}
