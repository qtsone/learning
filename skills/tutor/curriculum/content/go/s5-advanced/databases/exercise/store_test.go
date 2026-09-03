package store

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "ledger.db"), PoolConfig{
		MaxOpen:     4,
		MaxIdle:     4,
		MaxIdleTime: time.Minute,
		MaxLifetime: time.Hour,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func mustAccount(t *testing.T, s *Store, name string, openingCents int64) int64 {
	t.Helper()
	id, err := s.CreateAccount(context.Background(), name, openingCents)
	if err != nil {
		t.Fatalf("CreateAccount(%q): %v", name, err)
	}
	return id
}

func balance(t *testing.T, s *Store, id int64) int64 {
	t.Helper()
	a, err := s.AccountByID(context.Background(), id)
	if err != nil {
		t.Fatalf("AccountByID(%d): %v", id, err)
	}
	return a.BalanceCents
}

func transferCount(t *testing.T, s *Store) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM transfers`).Scan(&n); err != nil {
		t.Fatalf("counting transfers: %v", err)
	}
	return n
}

// --- acceptance criterion 3: Open wires pool and migrations ---

func TestOpenRunsMigrations(t *testing.T) {
	s := newStore(t)
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("schema_migrations unreadable after Open (%v) — Open must run applyMigrations", err)
	}
	if n != len(migrations) {
		t.Errorf("database is at version %d after Open, want %d", n, len(migrations))
	}
}

func TestOpenBoundsThePool(t *testing.T) {
	s := newStore(t)
	if got := s.db.Stats().MaxOpenConnections; got != 4 {
		t.Errorf("Stats().MaxOpenConnections = %d, want 4 — apply PoolConfig in Open", got)
	}
}

// --- acceptance criterion 4: accounts ---

func TestCreateAndGetAccount(t *testing.T) {
	s := newStore(t)
	id := mustAccount(t, s, "alice", 5000)
	if id == 0 {
		t.Fatal("CreateAccount returned id 0 — return the id the database assigned (LastInsertId)")
	}
	got, err := s.AccountByID(context.Background(), id)
	if err != nil {
		t.Fatalf("AccountByID(%d): %v", id, err)
	}
	want := Account{ID: id, Name: "alice", BalanceCents: 5000}
	if got != want {
		t.Errorf("AccountByID(%d) = %+v, want %+v", id, got, want)
	}
}

func TestAccountByIDMissing(t *testing.T) {
	s := newStore(t)
	_, err := s.AccountByID(context.Background(), 42)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("AccountByID on a missing id: got err %v, want ErrNotFound", err)
	}
}

// --- acceptance criterion 5: the transfer transaction ---

func TestTransferMovesMoneyAndLogs(t *testing.T) {
	s := newStore(t)
	alice := mustAccount(t, s, "alice", 5000)
	bob := mustAccount(t, s, "bob", 1000)

	if err := s.Transfer(context.Background(), alice, bob, 1500, "rent split"); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if got := balance(t, s, alice); got != 3500 {
		t.Errorf("source balance = %d, want 3500", got)
	}
	if got := balance(t, s, bob); got != 2500 {
		t.Errorf("destination balance = %d, want 2500", got)
	}
	history, err := s.History(context.Background(), alice)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("History returned %d transfers, want 1 — Transfer must insert a transfers row", len(history))
	}
	tr := history[0]
	if tr.FromID != alice || tr.ToID != bob || tr.Cents != 1500 || tr.Note != "rent split" {
		t.Errorf("logged transfer = %+v, want from %d to %d, 1500 cents, note %q", tr, alice, bob, "rent split")
	}
}

func TestTransferInsufficientFunds(t *testing.T) {
	s := newStore(t)
	alice := mustAccount(t, s, "alice", 100)
	bob := mustAccount(t, s, "bob", 0)

	err := s.Transfer(context.Background(), alice, bob, 500, "too much")
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("overdraft transfer: got err %v, want ErrInsufficientFunds", err)
	}
	if got := balance(t, s, alice); got != 100 {
		t.Errorf("source balance = %d after the refused transfer, want 100", got)
	}
	if n := transferCount(t, s); n != 0 {
		t.Errorf("transfers table holds %d rows after the refused transfer, want 0", n)
	}
}

func TestTransferRollsBackOnUnknownDestination(t *testing.T) {
	s := newStore(t)
	alice := mustAccount(t, s, "alice", 5000)

	err := s.Transfer(context.Background(), alice, 999, 1500, "to nowhere")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("transfer to an unknown account: got err %v, want ErrNotFound", err)
	}
	if got := balance(t, s, alice); got != 5000 {
		t.Errorf("source balance = %d after the failed transfer, want 5000 — the debit must roll back with the rest", got)
	}
	if n := transferCount(t, s); n != 0 {
		t.Errorf("transfers table holds %d rows after the failed transfer, want 0", n)
	}
}

func TestTransferUnknownSource(t *testing.T) {
	s := newStore(t)
	bob := mustAccount(t, s, "bob", 1000)
	err := s.Transfer(context.Background(), 999, bob, 100, "from nowhere")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("transfer from an unknown account: got err %v, want ErrNotFound", err)
	}
}

func TestTransferRejectsNonPositiveAmounts(t *testing.T) {
	s := newStore(t)
	alice := mustAccount(t, s, "alice", 5000)
	bob := mustAccount(t, s, "bob", 1000)

	for _, cents := range []int64{0, -500} {
		err := s.Transfer(context.Background(), alice, bob, cents, "bogus")
		if !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("Transfer of %d cents: got err %v, want ErrInvalidAmount", cents, err)
		}
	}
	if got := balance(t, s, alice); got != 5000 {
		t.Errorf("source balance = %d after rejected transfers, want 5000", got)
	}
}

func TestTransferCanceledContext(t *testing.T) {
	s := newStore(t)
	alice := mustAccount(t, s, "alice", 5000)
	bob := mustAccount(t, s, "bob", 1000)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled before the transfer even starts

	err := s.Transfer(ctx, alice, bob, 1500, "doomed")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Transfer with a canceled context: got err %v, want context.Canceled — pass ctx to BeginTx and wrap errors with %%w", err)
	}
	if got := balance(t, s, alice); got != 5000 {
		t.Errorf("source balance = %d after the canceled transfer, want 5000", got)
	}
	if n := transferCount(t, s); n != 0 {
		t.Errorf("transfers table holds %d rows after the canceled transfer, want 0", n)
	}
}

// A transaction that reads before it writes takes SQLite's write lock late.
// Under a deferred BEGIN the losers of that upgrade fail with SQLITE_BUSY,
// and the busy handler is never consulted — see LESSON.md.
func TestConcurrentTransfersDoNotCollide(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	alice := mustAccount(t, s, "alice", 1000)
	bob := mustAccount(t, s, "bob", 0)

	const n = 200
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Transfer(ctx, alice, bob, 1, "concurrent"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	var failed int
	var first error
	for err := range errs {
		if failed == 0 {
			first = err
		}
		failed++
	}
	if failed > 0 {
		t.Fatalf("%d of %d concurrent transfers failed, first: %v — a transaction that reads before it writes must begin as BEGIN IMMEDIATE", failed, n, first)
	}
	if got := balance(t, s, bob); got != n {
		t.Errorf("destination balance = %d after %d concurrent transfers of 1 cent, want %d", got, n, n)
	}
}

// --- acceptance criterion 6: history ---

func TestHostileNoteIsJustData(t *testing.T) {
	s := newStore(t)
	alice := mustAccount(t, s, "alice", 5000)
	bob := mustAccount(t, s, "bob", 1000)

	hostile := `x'); DROP TABLE transfers; --`
	if err := s.Transfer(context.Background(), alice, bob, 100, hostile); err != nil {
		t.Fatalf("Transfer with quotes in the note failed (%v) — use ? placeholders, not string building", err)
	}
	history, err := s.History(context.Background(), alice)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 1 || history[0].Note != hostile {
		t.Errorf("note round-trip: got %+v, want one transfer with note %q stored verbatim", history, hostile)
	}
}

func TestHistoryScopeAndOrder(t *testing.T) {
	s := newStore(t)
	alice := mustAccount(t, s, "alice", 5000)
	bob := mustAccount(t, s, "bob", 5000)
	carol := mustAccount(t, s, "carol", 5000)
	dave := mustAccount(t, s, "dave", 5000)

	ctx := context.Background()
	steps := []struct {
		from, to, cents int64
		note            string
	}{
		{alice, bob, 100, "first"},
		{bob, carol, 200, "second"},
		{alice, carol, 300, "third"},
	}
	for _, st := range steps {
		if err := s.Transfer(ctx, st.from, st.to, st.cents, st.note); err != nil {
			t.Fatalf("Transfer(%q): %v", st.note, err)
		}
	}

	got, err := s.History(ctx, bob)
	if err != nil {
		t.Fatalf("History(bob): %v", err)
	}
	var notes []string
	for _, tr := range got {
		notes = append(notes, tr.Note)
	}
	want := []string{"first", "second"}
	if !slices.Equal(notes, want) {
		t.Errorf("History(bob) notes = %v, want %v — both directions, oldest first", notes, want)
	}

	empty, err := s.History(ctx, dave)
	if err != nil {
		t.Fatalf("History(dave): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("History for an uninvolved account returned %d transfers, want 0", len(empty))
	}
}
