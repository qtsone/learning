package gql

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// fakeBatch is a batch function that records what it was asked for.
type fakeBatch struct {
	data  map[string]int
	calls [][]string
	err   error
}

func (f *fakeBatch) load(_ context.Context, keys []string) (map[string]int, error) {
	f.calls = append(f.calls, append([]string(nil), keys...))
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]int{}
	for _, k := range keys {
		if v, ok := f.data[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func TestLoaderBatchesQueuedKeysIntoOneCall(t *testing.T) {
	fb := &fakeBatch{data: map[string]int{"a": 1, "b": 2}}
	l := NewLoader(fb.load)

	ra := l.Load("a")
	rb := l.Load("b")
	ra2 := l.Load("a")

	if len(fb.calls) != 0 {
		t.Fatalf("batch called %d time(s) before Dispatch: Load must not fetch", len(fb.calls))
	}
	if ra2 != ra {
		t.Error("Load(\"a\") twice returned different results: one key is one fetch per request")
	}

	l.Dispatch(context.Background())

	if len(fb.calls) != 1 {
		t.Fatalf("batch called %d time(s), want exactly 1", len(fb.calls))
	}
	if want := []string{"a", "b"}; !reflect.DeepEqual(fb.calls[0], want) {
		t.Errorf("batch keys = %v, want %v (deduplicated, in the order first asked for)", fb.calls[0], want)
	}
	if v, err := ra.Get(); v != 1 || err != nil {
		t.Errorf("a = (%v, %v), want (1, nil)", v, err)
	}
	if v, err := rb.Get(); v != 2 || err != nil {
		t.Errorf("b = (%v, %v), want (2, nil)", v, err)
	}
}

func TestLoaderCachesAcrossDispatches(t *testing.T) {
	fb := &fakeBatch{data: map[string]int{"a": 1}}
	l := NewLoader(fb.load)

	l.Load("a")
	l.Dispatch(context.Background())
	r := l.Load("a")
	l.Dispatch(context.Background())

	if len(fb.calls) != 1 {
		t.Fatalf("batch called %d time(s), want 1: a key already loaded must not be queued again", len(fb.calls))
	}
	if v, err := r.Get(); v != 1 || err != nil {
		t.Errorf("a = (%v, %v), want (1, nil)", v, err)
	}
}

func TestLoaderDispatchWithNothingQueuedDoesNotFetch(t *testing.T) {
	fb := &fakeBatch{data: map[string]int{"a": 1}}
	l := NewLoader(fb.load)

	l.Dispatch(context.Background())

	if len(fb.calls) != 0 {
		t.Fatalf("batch called %d time(s) with an empty queue, want 0: the executor dispatches after every level", len(fb.calls))
	}
}

func TestLoaderReportsMissingKeysIndividually(t *testing.T) {
	fb := &fakeBatch{data: map[string]int{"a": 1}}
	l := NewLoader(fb.load)

	ok := l.Load("a")
	missing := l.Load("ghost")
	l.Dispatch(context.Background())

	if v, err := ok.Get(); v != 1 || err != nil {
		t.Errorf("a = (%v, %v), want (1, nil): one missing key must not spoil the batch", v, err)
	}
	_, err := missing.Get()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ghost error = %v, want one wrapping ErrNotFound", err)
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("ghost error = %q, want it to name the key", err)
	}
}

func TestLoaderGivesTheBatchErrorToEveryKey(t *testing.T) {
	boom := errors.New("connection reset")
	fb := &fakeBatch{err: boom}
	l := NewLoader(fb.load)

	ra := l.Load("a")
	rb := l.Load("b")
	l.Dispatch(context.Background())

	for name, r := range map[string]*Result[int]{"a": ra, "b": rb} {
		if _, err := r.Get(); !errors.Is(err, boom) {
			t.Errorf("%s error = %v, want the batch error: nothing was answered", name, err)
		}
	}
}
