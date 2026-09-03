// The service tests import no HTTP machinery at all: every rule they check
// lives below the HTTP edge, so httptest has nothing to offer here.
package note_test

import (
	"errors"
	"maps"
	"strings"
	"testing"

	"tutor.local/rest-services/memstore"
	"tutor.local/rest-services/note"
)

func newService(t *testing.T) *note.Service {
	t.Helper()
	return note.NewService(memstore.New())
}

func TestCreateStoresTrimmedDraft(t *testing.T) {
	svc := newService(t)
	n, err := svc.Create(note.Draft{Title: "  Shopping  ", Content: "\tmilk, eggs\n"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if n.ID == 0 {
		t.Error("Create returned a note with ID 0; the store assigns IDs starting at 1")
	}
	if n.Title != "Shopping" {
		t.Errorf("Title = %q, want %q (trim whitespace before storing)", n.Title, "Shopping")
	}
	if n.Content != "milk, eggs" {
		t.Errorf("Content = %q, want %q (trim whitespace before storing)", n.Content, "milk, eggs")
	}
}

func TestCreateValidation(t *testing.T) {
	cases := []struct {
		name  string
		draft note.Draft
		want  map[string]string
	}{
		{
			name:  "empty title",
			draft: note.Draft{Content: "fine"},
			want:  map[string]string{"title": "required"},
		},
		{
			name:  "whitespace-only title",
			draft: note.Draft{Title: " \t\n "},
			want:  map[string]string{"title": "required"},
		},
		{
			name:  "title too long",
			draft: note.Draft{Title: strings.Repeat("x", 121)},
			want:  map[string]string{"title": "must be at most 120 characters"},
		},
		{
			name:  "content too long",
			draft: note.Draft{Title: "ok", Content: strings.Repeat("x", 8001)},
			want:  map[string]string{"content": "must be at most 8000 characters"},
		},
		{
			name:  "multiple problems reported together",
			draft: note.Draft{Content: strings.Repeat("x", 8001)},
			want: map[string]string{
				"title":   "required",
				"content": "must be at most 8000 characters",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := newService(t).Create(c.draft)
			var ve note.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Create(%+v) error = %v, want a note.ValidationError", c.draft, err)
			}
			if !maps.Equal(ve, c.want) {
				t.Errorf("validation fields = %v, want %v", ve, c.want)
			}
		})
	}
}

func TestLimitsCountRunesNotBytes(t *testing.T) {
	// 120 two-byte runes: 240 bytes. Valid — unless someone used len().
	d := note.Draft{
		Title:   strings.Repeat("é", note.MaxTitleLen),
		Content: strings.Repeat("é", note.MaxContentLen),
	}
	if _, err := newService(t).Create(d); err != nil {
		t.Fatalf("Create with limit-length multi-byte fields returned %v; limits count runes (utf8.RuneCountInString), not bytes", err)
	}
}

func TestUpdateValidatesAndTrims(t *testing.T) {
	svc := newService(t)
	created, err := svc.Create(note.Draft{Title: "before"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var ve note.ValidationError
	if _, err := svc.Update(created.ID, note.Draft{Title: "   "}); !errors.As(err, &ve) {
		t.Fatalf("Update with blank title: error = %v, want a note.ValidationError", err)
	}

	n, err := svc.Update(created.ID, note.Draft{Title: "  after  "})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if n.ID != created.ID {
		t.Errorf("Update changed the ID: got %d, want %d", n.ID, created.ID)
	}
	if n.Title != "after" {
		t.Errorf("Title = %q, want %q (Update must trim like Create)", n.Title, "after")
	}
}

func TestMissingIDsReportErrNotFound(t *testing.T) {
	svc := newService(t)
	if _, err := svc.Get(42); !errors.Is(err, note.ErrNotFound) {
		t.Errorf("Get(42) error = %v, want note.ErrNotFound", err)
	}
	if _, err := svc.Update(42, note.Draft{Title: "ok"}); !errors.Is(err, note.ErrNotFound) {
		t.Errorf("Update(42, …) error = %v, want note.ErrNotFound", err)
	}
	if err := svc.Delete(42); !errors.Is(err, note.ErrNotFound) {
		t.Errorf("Delete(42) error = %v, want note.ErrNotFound", err)
	}
}

// unorderedStore returns notes scrambled, like a map-backed store or a SQL
// query without ORDER BY. The ordering guarantee belongs to the service.
type unorderedStore struct{}

func (unorderedStore) Create(d note.Draft) (note.Note, error) { return note.Note{}, nil }
func (unorderedStore) Get(id int64) (note.Note, error)        { return note.Note{}, note.ErrNotFound }
func (unorderedStore) List() ([]note.Note, error) {
	return []note.Note{{ID: 3, Title: "c"}, {ID: 1, Title: "a"}, {ID: 2, Title: "b"}}, nil
}
func (unorderedStore) Update(id int64, d note.Draft) (note.Note, error) {
	return note.Note{}, note.ErrNotFound
}
func (unorderedStore) Delete(id int64) error { return note.ErrNotFound }

func TestListSortsByID(t *testing.T) {
	svc := note.NewService(unorderedStore{})
	ns, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ns) != 3 {
		t.Fatalf("List returned %d notes, want 3", len(ns))
	}
	for i, want := range []int64{1, 2, 3} {
		if ns[i].ID != want {
			t.Errorf("ns[%d].ID = %d, want %d (List must sort by id ascending)", i, ns[i].ID, want)
		}
	}
}

func TestListEmptyIsNonNil(t *testing.T) {
	ns, err := newService(t).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ns == nil {
		t.Fatal("List returned a nil slice; return an empty one so it encodes as [], not null")
	}
	if len(ns) != 0 {
		t.Fatalf("List on an empty service returned %d notes", len(ns))
	}
}

// failingStore stands in for storage that has fallen over.
type failingStore struct{ err error }

func (f failingStore) Create(note.Draft) (note.Note, error)        { return note.Note{}, f.err }
func (f failingStore) Get(int64) (note.Note, error)                { return note.Note{}, f.err }
func (f failingStore) List() ([]note.Note, error)                  { return nil, f.err }
func (f failingStore) Update(int64, note.Draft) (note.Note, error) { return note.Note{}, f.err }
func (f failingStore) Delete(int64) error                          { return f.err }

func TestStoreErrorsPassThrough(t *testing.T) {
	boom := errors.New("boom")
	svc := note.NewService(failingStore{err: boom})
	if _, err := svc.Create(note.Draft{Title: "ok"}); !errors.Is(err, boom) {
		t.Errorf("Create error = %v, want the store's error to pass through", err)
	}
	if _, err := svc.List(); !errors.Is(err, boom) {
		t.Errorf("List error = %v, want the store's error to pass through", err)
	}
}
