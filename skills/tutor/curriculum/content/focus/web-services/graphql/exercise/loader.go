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

// Load asks for one key and returns the result handle for it. It must not
// fetch anything: the whole point is that Load is cheap enough to call once
// per parent object, and the fetching happens later, once, in Dispatch.
//
// Two keys, one contract:
//
//   - the same key asked for twice returns the same *Result — inside one
//     request, one key is fetched once, however many parents need it;
//   - a key that has already been resolved is not queued again.
func (l *Loader[K, V]) Load(key K) *Result[V] {
	// TODO: return the cached *Result for key if there is one; otherwise
	// create a pending one, remember it, queue the key, and return it.
	var zero V
	r := &Result[V]{}
	r.set(zero, errors.New("TODO: implement Loader.Load"))
	return r
}

// Dispatch fetches every queued key in a single call to the batch function and
// fills in the results handed out by Load. After it returns, every result
// queued before it was called is resolved.
//
// Three cases the tests care about:
//
//   - nothing queued: no call to the batch function at all (an empty query is
//     not a reason to talk to the database);
//   - the batch function fails: every queued key gets that error, because none
//     of them were answered;
//   - a key is missing from the returned map: that key alone gets an error
//     wrapping ErrNotFound, and the others are fine.
func (l *Loader[K, V]) Dispatch(ctx context.Context) {
	// TODO: take the queued keys, clear the queue, call l.batch once, and set
	// every queued key's Result. Use missingKeyError for keys the batch did
	// not return.
}

// missingKeyError is the per-key error for a key the batch function did not
// answer. It wraps ErrNotFound so callers can use errors.Is.
func missingKeyError[K comparable](key K) error {
	return fmt.Errorf("key %v: %w", key, ErrNotFound)
}
