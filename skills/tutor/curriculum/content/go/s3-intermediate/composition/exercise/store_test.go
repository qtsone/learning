package composition

import (
	"errors"
	"testing"
)

// ReadOnly must satisfy Store — the embedded interface plus your overrides
// fill the method set.
var _ Store = ReadOnly{}

func newSeededStore(t *testing.T) *MemStore {
	t.Helper()
	ms := NewMemStore()
	if err := ms.Put("host", "localhost"); err != nil {
		t.Fatalf("seeding MemStore: Put returned %v", err)
	}
	return ms
}

func TestReadOnlyGetIsPromoted(t *testing.T) {
	ro := ReadOnly{Store: newSeededStore(t)}
	got, err := ro.Get("host")
	if err != nil {
		t.Fatalf(`Get("host") returned error %v, want the wrapped store's value`, err)
	}
	if got != "localhost" {
		t.Errorf(`Get("host") = %q, want %q`, got, "localhost")
	}
}

func TestReadOnlyRejectsPut(t *testing.T) {
	ms := newSeededStore(t)
	ro := ReadOnly{Store: ms}
	err := ro.Put("host", "evil.example")
	if !errors.Is(err, ErrReadOnly) {
		t.Fatalf("Put on ReadOnly returned %v, want an error wrapping ErrReadOnly", err)
	}
	if got, _ := ms.Get("host"); got != "localhost" {
		t.Errorf("wrapped store changed after rejected Put: host = %q, want %q", got, "localhost")
	}
}

func TestReadOnlyRejectsDelete(t *testing.T) {
	ms := newSeededStore(t)
	ro := ReadOnly{Store: ms}
	err := ro.Delete("host")
	if !errors.Is(err, ErrReadOnly) {
		t.Fatalf("Delete on ReadOnly returned %v, want an error wrapping ErrReadOnly", err)
	}
	if _, err := ms.Get("host"); err != nil {
		t.Errorf("wrapped store lost the key after rejected Delete: %v", err)
	}
}
