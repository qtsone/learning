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

// ReadOnly wraps any Store: reads pass through via promotion, writes must be
// rejected with ErrReadOnly.
type ReadOnly struct {
	Store
}

// TODO: override Put and Delete on ReadOnly so both return an error that
// errors.Is(err, ErrReadOnly) reports true, without changing the wrapped
// store. Do NOT write a Get method — promotion already provides it. See
// LESSON.md acceptance criteria 3-4.
