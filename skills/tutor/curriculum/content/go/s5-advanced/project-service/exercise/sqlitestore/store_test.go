package sqlitestore

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"tutor.local/project-service/task"
)

var stamp = time.Date(2024, 5, 1, 9, 0, 0, 123456789, time.UTC)

// openStore returns a store on a database file of its own, inside the
// directory the testing package will delete afterwards. Every test gets a
// fresh database: no shared fixture, no ordering dependency between tests.
func openStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tasks.db")
	st, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open(%s) = %v", path, err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return st
}

func create(t *testing.T, st *Store, title string, status task.Status) task.Task {
	t.Helper()
	created, err := st.Create(context.Background(), task.Task{
		Title:     title,
		Status:    status,
		CreatedAt: stamp,
		UpdatedAt: stamp,
	})
	if err != nil {
		t.Fatalf("Create(%q) = %v", title, err)
	}
	return created
}

func TestCreateAssignsIDsAndRoundTrips(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	first := create(t, st, "write the store", task.StatusOpen)
	second := create(t, st, "write the tests", task.StatusOpen)
	if first.ID == 0 || second.ID == 0 {
		t.Fatalf("ids = %d, %d: Create must return the id the database assigned", first.ID, second.ID)
	}
	if first.ID == second.ID {
		t.Fatalf("both rows got id %d", first.ID)
	}

	got, err := st.Get(ctx, first.ID)
	if err != nil {
		t.Fatalf("Get(%d) = %v", first.ID, err)
	}
	if got.Title != "write the store" || got.Status != task.StatusOpen {
		t.Errorf("Get = %+v, want title %q status %q", got, "write the store", task.StatusOpen)
	}
	if !got.CreatedAt.Equal(stamp) || !got.UpdatedAt.Equal(stamp) {
		t.Errorf("timestamps = %v / %v, want %v: store and read them without losing precision",
			got.CreatedAt, got.UpdatedAt, stamp)
	}
}

func TestGetUnknownIDReturnsDomainError(t *testing.T) {
	st := openStore(t)

	_, err := st.Get(context.Background(), 999)
	if !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("Get(999) = %v, want task.ErrNotFound (sql.ErrNoRows must not escape this package)", err)
	}
}

func TestListFiltersByStatus(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	open1 := create(t, st, "a", task.StatusOpen)
	open2 := create(t, st, "b", task.StatusOpen)
	done1 := create(t, st, "c", task.StatusDone)

	cases := []struct {
		filter task.Status
		want   []int64
	}{
		{"", []int64{open1.ID, open2.ID, done1.ID}},
		{task.StatusOpen, []int64{open1.ID, open2.ID}},
		{task.StatusDone, []int64{done1.ID}},
	}
	for _, c := range cases {
		got, err := st.List(ctx, c.filter)
		if err != nil {
			t.Fatalf("List(%q) = %v", c.filter, err)
		}
		ids := make([]int64, 0, len(got))
		for _, tk := range got {
			ids = append(ids, tk.ID)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		if len(ids) != len(c.want) {
			t.Fatalf("List(%q) returned ids %v, want %v", c.filter, ids, c.want)
		}
		for i := range ids {
			if ids[i] != c.want[i] {
				t.Fatalf("List(%q) returned ids %v, want %v", c.filter, ids, c.want)
			}
		}
	}
}

func TestSetStatusWritesAndReturnsTheRow(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	created := create(t, st, "finish me", task.StatusOpen)
	at := stamp.Add(2 * time.Hour)

	got, err := st.SetStatus(ctx, created.ID, task.StatusDone, at)
	if err != nil {
		t.Fatalf("SetStatus = %v", err)
	}
	if got.Status != task.StatusDone {
		t.Errorf("returned status = %q, want %q", got.Status, task.StatusDone)
	}
	if !got.UpdatedAt.Equal(at) {
		t.Errorf("returned UpdatedAt = %v, want %v", got.UpdatedAt, at)
	}
	if !got.CreatedAt.Equal(stamp) {
		t.Errorf("returned CreatedAt = %v, want it untouched at %v", got.CreatedAt, stamp)
	}

	reread, err := st.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get after SetStatus = %v", err)
	}
	if reread.Status != task.StatusDone || !reread.UpdatedAt.Equal(at) {
		t.Errorf("stored row = %+v, want the same status and stamp SetStatus returned", reread)
	}
}

func TestSetStatusAndDeleteReportMissingRows(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	if _, err := st.SetStatus(ctx, 999, task.StatusDone, stamp); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("SetStatus(999) = %v, want task.ErrNotFound", err)
	}
	if err := st.Delete(ctx, 999); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Delete(999) = %v, want task.ErrNotFound", err)
	}
}

func TestDeleteRemovesTheRow(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	created := create(t, st, "temporary", task.StatusOpen)

	if err := st.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete = %v", err)
	}
	if _, err := st.Get(ctx, created.ID); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Get after Delete = %v, want task.ErrNotFound", err)
	}
	if err := st.Delete(ctx, created.ID); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("second Delete = %v, want task.ErrNotFound", err)
	}
}

func TestPingReportsAClosedDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	st, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open = %v", err)
	}
	if err := st.Ping(context.Background()); err != nil {
		t.Fatalf("Ping on an open store = %v, want nil", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close = %v", err)
	}
	if err := st.Ping(context.Background()); err == nil {
		t.Error("Ping on a closed store = nil, want an error: readiness depends on this answer being honest")
	}
}

// Reopening an existing database must be safe: migrations already applied
// are skipped, and the rows are still there.
func TestMigrationsAreIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	ctx := context.Background()

	first, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("first Open = %v", err)
	}
	created, err := first.Create(ctx, task.Task{
		Title: "survives a restart", Status: task.StatusOpen, CreatedAt: stamp, UpdatedAt: stamp,
	})
	if err != nil {
		t.Fatalf("Create = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close = %v", err)
	}

	second, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("reopening a migrated database = %v, want success", err)
	}
	defer second.Close()
	if _, err := second.Get(ctx, created.ID); err != nil {
		t.Errorf("Get after reopen = %v, want the row that was there before", err)
	}
}

// Handlers run on their own goroutines, so the store is used concurrently.
// This test exists for the race detector and for the pool: it asserts that
// every write landed, never how long any of it took.
func TestConcurrentWrites(t *testing.T) {
	st := openStore(t)
	const workers = 8

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, err := st.Create(context.Background(), task.Task{
				Title: "concurrent", Status: task.StatusOpen, CreatedAt: stamp, UpdatedAt: stamp,
			})
			if err != nil {
				errs <- err
				return
			}
			if _, err := st.SetStatus(context.Background(), created.ID, task.StatusDone, stamp); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent write failed: %v", err)
	}

	got, err := st.List(context.Background(), task.StatusDone)
	if err != nil {
		t.Fatalf("List = %v", err)
	}
	if len(got) != workers {
		t.Errorf("List(done) returned %d rows, want %d", len(got), workers)
	}
}
