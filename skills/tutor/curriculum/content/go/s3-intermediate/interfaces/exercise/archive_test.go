package report

import (
	"errors"
	"strings"
	"testing"
)

type call struct {
	name string
	data string
}

// memStore is a five-line fake made possible by Saver being one method:
// it records every Save so tests can inspect exactly what Archive did.
// *memStore satisfies Saver because its method set includes Save.
type memStore struct {
	calls  []call
	failOn string
	err    error
}

func (m *memStore) Save(name string, data []byte) error {
	if m.failOn != "" && name == m.failOn {
		return m.err
	}
	m.calls = append(m.calls, call{name: name, data: string(data)})
	return nil
}

func TestArchiveSavesEveryEvent(t *testing.T) {
	store := &memStore{}
	events := []Event{
		{Name: "build.log", Size: 2048},
		{Name: "test.out", Size: 117},
	}
	if err := Archive(store, events); err != nil {
		t.Fatalf("Archive returned error %v, want nil", err)
	}
	want := []call{
		{name: "build.log", data: "build.log 2048"},
		{name: "test.out", data: "test.out 117"},
	}
	if len(store.calls) != len(want) {
		t.Fatalf("Archive made %d Save call(s) %+v, want %d", len(store.calls), store.calls, len(want))
	}
	for i := range want {
		if store.calls[i] != want[i] {
			t.Errorf("Save call %d = %+v, want %+v", i, store.calls[i], want[i])
		}
	}
}

func TestArchiveNoEvents(t *testing.T) {
	store := &memStore{}
	if err := Archive(store, nil); err != nil {
		t.Errorf("Archive with no events returned %v, want nil", err)
	}
	if len(store.calls) != 0 {
		t.Errorf("Archive with no events made %d Save call(s), want 0", len(store.calls))
	}
}

func TestArchiveStopsAtFirstFailure(t *testing.T) {
	errBoom := errors.New("backend down")
	store := &memStore{failOn: "bad.log", err: errBoom}
	events := []Event{
		{Name: "good.log", Size: 1},
		{Name: "bad.log", Size: 2},
		{Name: "never.log", Size: 3},
	}
	err := Archive(store, events)
	if !errors.Is(err, errBoom) {
		t.Fatalf("Archive returned %v, want an error wrapping %q (use %%w)", err, errBoom)
	}
	if !strings.Contains(err.Error(), "bad.log") {
		t.Errorf("error %q does not name the failing event %q", err, "bad.log")
	}
	if len(store.calls) != 1 || store.calls[0].name != "good.log" {
		t.Errorf("after the failure only %q should be saved, got calls %+v", "good.log", store.calls)
	}
}
