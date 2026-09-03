package composition

import (
	"errors"
	"fmt"
)

// ErrReadOnly is returned by write operations on a read-only store.
var ErrReadOnly = errors.New("store is read-only")

// ErrNotFound is returned by Get when the key does not exist.
var ErrNotFound = errors.New("key not found")

// Store is a minimal key-value store, defined consumer-side as in the
// interfaces lesson.
type Store interface {
	Get(key string) (string, error)
	Put(key, value string) error
	Delete(key string) error
}

// MemStore is an in-memory Store — complete as given, do not modify.
type MemStore struct {
	data map[string]string
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{data: map[string]string{}}
}

// Get returns the value for key, or an error wrapping ErrNotFound.
func (m *MemStore) Get(key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("get %q: %w", key, ErrNotFound)
	}
	return v, nil
}

// Put stores value under key.
func (m *MemStore) Put(key, value string) error {
	m.data[key] = value
	return nil
}

// Delete removes key if present.
func (m *MemStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

// ReadOnly wraps any Store: reads pass through via promotion, writes are
// rejected with ErrReadOnly. Get is promoted from the embedded Store.
type ReadOnly struct {
	Store
}

// Put rejects the write, leaving the wrapped store untouched.
func (r ReadOnly) Put(key, value string) error {
	return fmt.Errorf("put %q: %w", key, ErrReadOnly)
}

// Delete rejects the write, leaving the wrapped store untouched.
func (r ReadOnly) Delete(key string) error {
	return fmt.Errorf("delete %q: %w", key, ErrReadOnly)
}
