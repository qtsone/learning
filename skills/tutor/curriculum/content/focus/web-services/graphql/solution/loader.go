package gql

import (
	"context"
	"errors"
	"fmt"
)

// Result is one key's answer, handed out before it is known. Load returns it
// immediately; Dispatch fills it in.
type Result[V any] struct {
	value    V
	err      error
	resolved bool
}

// Get returns the loaded value. Reading a result before its loader has been
// dispatched is a bug in the executor, not a missing row, so it says so.
func (r *Result[V]) Get() (V, error) {
	if !r.resolved {
		var zero V
		return zero, errors.New("dataloader: result read before its loader was dispatched")
	}
	return r.value, r.err
}

// set records an answer. Call it exactly once per result.
func (r *Result[V]) set(v V, err error) {
	r.value, r.err, r.resolved = v, err, true
}

// Loader turns many single-key loads into one batched call.
//
// It is used by one request, from one goroutine at a time (this executor
// resolves a level with a sequential loop), so it needs no locking. A loader
// shared between requests would need both a mutex and a much longer
// conversation about cache invalidation.
type Loader[K comparable, V any] struct {
	batch   func(ctx context.Context, keys []K) (map[K]V, error)
	results map[K]*Result[V]
	queued  []K
}

// NewLoader returns a loader over a batch function. The batch function must
// accept every queued key at once and return a map from key to value; keys
// with no row are simply absent from the map.
func NewLoader[K comparable, V any](batch func(ctx context.Context, keys []K) (map[K]V, error)) *Loader[K, V] {
	return &Loader[K, V]{
		batch:   batch,
		results: make(map[K]*Result[V]),
	}
}

// Load asks for one key and returns the result handle for it. It fetches
// nothing: Load is cheap enough to call once per parent object, and the
// fetching happens later, once, in Dispatch.
//
// The results map is both the dedupe and the per-request cache. One key asked
// for twice — by two posts with the same author, or by a post and one of its
// comments — is one entry, one queue slot, and one row fetched.
func (l *Loader[K, V]) Load(key K) *Result[V] {
	if r, ok := l.results[key]; ok {
		return r
	}
	r := &Result[V]{}
	l.results[key] = r
	l.queued = append(l.queued, key)
	return r
}

// Dispatch fetches every queued key in a single call to the batch function and
// fills in the results handed out by Load.
func (l *Loader[K, V]) Dispatch(ctx context.Context) {
	if len(l.queued) == 0 {
		return
	}
	keys := l.queued
	l.queued = nil

	values, err := l.batch(ctx, keys)
	for _, k := range keys {
		r := l.results[k]
		if err != nil {
			// Nothing was answered, so the failure belongs to every key.
			var zero V
			r.set(zero, err)
			continue
		}
		v, ok := values[k]
		if !ok {
			var zero V
			r.set(zero, missingKeyError(k))
			continue
		}
		r.set(v, nil)
	}
}

// missingKeyError is the per-key error for a key the batch function did not
// answer. It wraps ErrNotFound so callers can use errors.Is.
func missingKeyError[K comparable](key K) error {
	return fmt.Errorf("key %v: %w", key, ErrNotFound)
}
