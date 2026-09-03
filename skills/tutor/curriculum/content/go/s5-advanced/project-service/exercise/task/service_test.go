package task

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// base is the fixed instant the test clock starts from. Fixed time makes
// timestamps assertable: no test in this file ever calls time.Now.
var base = time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC)

// testClock hands out base, then base+1m, base+2m, … one step per call, so a
// test can tell "stamped at create" apart from "stamped at update" without
// waiting for real time to pass.
type testClock struct {
	mu    sync.Mutex
	calls int
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := base.Add(time.Duration(c.calls) * time.Minute)
	c.calls++
	return t
}

// fakeStore is an in-memory task.Store: enough to exercise the rules without
// a database, and safe under -race because the service may be used from many
// goroutines.
type fakeStore struct {
	mu         sync.Mutex
	seq        int64
	tasks      map[int64]Task
	writes     int      // SetStatus calls that reached storage
	lastFilter Status   // the status List was last asked for
	err        error    // when non-nil, every method fails with it
	calls      []string // method names, in order
}

func newFakeStore() *fakeStore {
	return &fakeStore{tasks: make(map[int64]Task)}
}

func (f *fakeStore) record(name string) error {
	f.calls = append(f.calls, name)
	return f.err
}

func (f *fakeStore) Create(ctx context.Context, t Task) (Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.record("Create"); err != nil {
		return Task{}, err
	}
	f.seq++
	t.ID = f.seq
	f.tasks[t.ID] = t
	return t, nil
}

func (f *fakeStore) Get(ctx context.Context, id int64) (Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.record("Get"); err != nil {
		return Task{}, err
	}
	t, ok := f.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

// List deliberately returns tasks in descending id order: ordering is the
// service's promise, not storage's.
func (f *fakeStore) List(ctx context.Context, status Status) ([]Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.record("List"); err != nil {
		return nil, err
	}
	f.lastFilter = status
	var out []Task
	for _, t := range f.tasks {
		if status == "" || t.Status == status {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (f *fakeStore) SetStatus(ctx context.Context, id int64, status Status, at time.Time) (Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.record("SetStatus"); err != nil {
		return Task{}, err
	}
	t, ok := f.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	f.writes++
	t.Status = status
	t.UpdatedAt = at
	f.tasks[id] = t
	return t, nil
}

func (f *fakeStore) Delete(ctx context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.record("Delete"); err != nil {
		return err
	}
	if _, ok := f.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(f.tasks, id)
	return nil
}

func (f *fakeStore) Ping(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.record("Ping")
}

func newService(t *testing.T) (*Service, *fakeStore) {
	t.Helper()
	store := newFakeStore()
	return NewService(store, (&testClock{}).Now), store
}

func mustCreate(t *testing.T, s *Service, title string) Task {
	t.Helper()
	task, err := s.Create(context.Background(), Draft{Title: title})
	if err != nil {
		t.Fatalf("Create(%q) = %v, want success", title, err)
	}
	return task
}

func TestCreateNormalizesAndStamps(t *testing.T) {
	svc, _ := newService(t)

	got, err := svc.Create(context.Background(), Draft{Title: "  ship the capstone\t"})
	if err != nil {
		t.Fatalf("Create = %v, want success", err)
	}
	if got.Title != "ship the capstone" {
		t.Errorf("Title = %q, want %q (trim the title before storing it)", got.Title, "ship the capstone")
	}
	if got.Status != StatusOpen {
		t.Errorf("Status = %q, want %q (a client does not get to choose the starting status)", got.Status, StatusOpen)
	}
	if got.ID == 0 {
		t.Error("ID = 0, want the id storage assigned")
	}
	if !got.CreatedAt.Equal(base) {
		t.Errorf("CreatedAt = %v, want %v (stamp from the injected clock, not time.Now)", got.CreatedAt, base)
	}
	if !got.UpdatedAt.Equal(got.CreatedAt) {
		t.Errorf("UpdatedAt = %v, want it equal to CreatedAt on a fresh task", got.UpdatedAt)
	}
	if got.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt location = %v, want UTC", got.CreatedAt.Location())
	}
}

func TestCreateValidation(t *testing.T) {
	cases := []struct {
		name  string
		title string
		want  ValidationError
	}{
		{"empty", "", ValidationError{"title": "required"}},
		{"whitespace only", "   \t\n ", ValidationError{"title": "required"}},
		{
			"too long",
			strings.Repeat("é", MaxTitleLen+1),
			ValidationError{"title": "must be at most 200 characters"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, store := newService(t)

			_, err := svc.Create(context.Background(), Draft{Title: c.title})
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Create(%q) error = %v, want a ValidationError", c.title, err)
			}
			if len(ve) != len(c.want) || ve["title"] != c.want["title"] {
				t.Errorf("fields = %v, want %v", map[string]string(ve), map[string]string(c.want))
			}
			if len(store.calls) != 0 {
				t.Errorf("storage was called (%v) for input that never passed validation", store.calls)
			}
		})
	}
}

// A title exactly at the limit is legal, and the limit counts runes: 200
// two-byte characters are 400 bytes and must still be accepted.
func TestCreateAcceptsTitleAtTheLimit(t *testing.T) {
	svc, _ := newService(t)
	title := strings.Repeat("é", MaxTitleLen)

	if _, err := svc.Create(context.Background(), Draft{Title: title}); err != nil {
		t.Fatalf("Create with a %d-rune title = %v, want success (count runes, not bytes)", MaxTitleLen, err)
	}
}

func TestListSortsByIDAndIsNeverNil(t *testing.T) {
	svc, _ := newService(t)

	empty, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List on an empty service = %v, want success", err)
	}
	if empty == nil {
		t.Fatal("List returned a nil slice; an empty listing must encode as [], not null")
	}
	if len(empty) != 0 {
		t.Fatalf("List on an empty service returned %d tasks", len(empty))
	}

	for _, title := range []string{"first", "second", "third"} {
		mustCreate(t, svc, title)
	}
	got, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List = %v, want success", err)
	}
	if len(got) != 3 {
		t.Fatalf("List returned %d tasks, want 3", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].ID >= got[i].ID {
			t.Fatalf("List ids = %d, %d at positions %d, %d: storage returns them unordered, the service sorts",
				got[i-1].ID, got[i].ID, i-1, i)
		}
	}
}

func TestListPassesTheFilterDown(t *testing.T) {
	svc, store := newService(t)
	open := mustCreate(t, svc, "still open")
	done := mustCreate(t, svc, "finished")
	if _, err := svc.SetStatus(context.Background(), done.ID, StatusDone); err != nil {
		t.Fatalf("SetStatus = %v, want success", err)
	}

	got, err := svc.List(context.Background(), StatusOpen)
	if err != nil {
		t.Fatalf("List(open) = %v, want success", err)
	}
	if store.lastFilter != StatusOpen {
		t.Errorf("store was asked for status %q, want %q: filter in storage, not in Go", store.lastFilter, StatusOpen)
	}
	if len(got) != 1 || got[0].ID != open.ID {
		t.Errorf("List(open) = %v, want just task %d", got, open.ID)
	}
}

func TestListRejectsUnknownStatus(t *testing.T) {
	svc, store := newService(t)

	_, err := svc.List(context.Background(), Status("archived"))
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("List(\"archived\") error = %v, want a ValidationError", err)
	}
	if ve["status"] != `must be "open" or "done"` {
		t.Errorf("fields = %v, want status: %q", map[string]string(ve), `must be "open" or "done"`)
	}
	if len(store.calls) != 0 {
		t.Errorf("storage was called (%v) for a filter that never passed validation", store.calls)
	}
}

func TestSetStatusCompletesATask(t *testing.T) {
	svc, _ := newService(t)
	created := mustCreate(t, svc, "ship it")

	got, err := svc.SetStatus(context.Background(), created.ID, StatusDone)
	if err != nil {
		t.Fatalf("SetStatus(done) = %v, want success", err)
	}
	if got.Status != StatusDone {
		t.Errorf("Status = %q, want %q", got.Status, StatusDone)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want it unchanged at %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want a later stamp than the create-time %v", got.UpdatedAt, created.UpdatedAt)
	}
}

// Setting the status a task already has is a no-op, and a no-op does not
// write: the same request retried must not churn the row or its timestamp.
func TestSetStatusIsIdempotent(t *testing.T) {
	svc, store := newService(t)
	created := mustCreate(t, svc, "ship it")

	first, err := svc.SetStatus(context.Background(), created.ID, StatusDone)
	if err != nil {
		t.Fatalf("first SetStatus(done) = %v, want success", err)
	}
	second, err := svc.SetStatus(context.Background(), created.ID, StatusDone)
	if err != nil {
		t.Fatalf("repeated SetStatus(done) = %v, want success (the task is already in the target state)", err)
	}
	if !second.UpdatedAt.Equal(first.UpdatedAt) {
		t.Errorf("UpdatedAt changed on a no-op transition: %v then %v", first.UpdatedAt, second.UpdatedAt)
	}
	if store.writes != 1 {
		t.Errorf("storage saw %d writes, want 1: a transition to the current status must not reach storage", store.writes)
	}
}

func TestSetStatusRefusesToReopen(t *testing.T) {
	svc, store := newService(t)
	created := mustCreate(t, svc, "ship it")
	if _, err := svc.SetStatus(context.Background(), created.ID, StatusDone); err != nil {
		t.Fatalf("SetStatus(done) = %v, want success", err)
	}
	before := store.writes

	_, err := svc.SetStatus(context.Background(), created.ID, StatusOpen)
	if !errors.Is(err, ErrAlreadyDone) {
		t.Fatalf("SetStatus(open) on a done task = %v, want ErrAlreadyDone", err)
	}
	if store.writes != before {
		t.Error("a rejected transition still wrote to storage")
	}
}

func TestSetStatusValidatesTarget(t *testing.T) {
	svc, _ := newService(t)
	created := mustCreate(t, svc, "ship it")

	_, err := svc.SetStatus(context.Background(), created.ID, Status("archived"))
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("SetStatus(\"archived\") error = %v, want a ValidationError", err)
	}
	if ve["status"] != `must be "open" or "done"` {
		t.Errorf("fields = %v, want status: %q", map[string]string(ve), `must be "open" or "done"`)
	}
}

func TestUnknownIDsAreNotFound(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	if _, err := svc.Get(ctx, 404); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(404) = %v, want ErrNotFound", err)
	}
	if _, err := svc.SetStatus(ctx, 404, StatusDone); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetStatus(404) = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, 404); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete(404) = %v, want ErrNotFound", err)
	}
}

func TestDeleteRemovesTheTask(t *testing.T) {
	svc, _ := newService(t)
	created := mustCreate(t, svc, "ship it")
	ctx := context.Background()

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete = %v, want success", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete = %v, want ErrNotFound", err)
	}
}

// Storage failures are not the domain's to interpret: they travel out
// unchanged (wrapping is fine, swallowing is not) so the HTTP layer can map
// them to 500 and log the truth.
func TestStorageFailuresPropagate(t *testing.T) {
	boom := errors.New("disk on fire")
	ctx := context.Background()

	for _, c := range []struct {
		name string
		call func(*Service) error
	}{
		{"Create", func(s *Service) error { _, err := s.Create(ctx, Draft{Title: "x"}); return err }},
		{"Get", func(s *Service) error { _, err := s.Get(ctx, 1); return err }},
		{"List", func(s *Service) error { _, err := s.List(ctx, ""); return err }},
		{"Delete", func(s *Service) error { return s.Delete(ctx, 1) }},
		{"Ping", func(s *Service) error { return s.Ping(ctx) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			store := newFakeStore()
			store.err = boom
			svc := NewService(store, (&testClock{}).Now)

			if err := c.call(svc); !errors.Is(err, boom) {
				t.Errorf("%s with failing storage = %v, want the storage error to survive", c.name, err)
			}
		})
	}
}

// The service is shared by every request goroutine, so it must be safe to
// call concurrently. This test exists for the race detector.
func TestServiceIsSafeForConcurrentUse(t *testing.T) {
	svc, _ := newService(t)
	const workers = 8

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, err := svc.Create(context.Background(), Draft{Title: "concurrent"})
			if err != nil {
				t.Errorf("Create = %v, want success", err)
				return
			}
			if _, err := svc.SetStatus(context.Background(), created.ID, StatusDone); err != nil {
				t.Errorf("SetStatus = %v, want success", err)
			}
		}()
	}
	wg.Wait()

	got, err := svc.List(context.Background(), StatusDone)
	if err != nil {
		t.Fatalf("List = %v, want success", err)
	}
	if len(got) != workers {
		t.Errorf("List(done) returned %d tasks, want %d", len(got), workers)
	}
}
