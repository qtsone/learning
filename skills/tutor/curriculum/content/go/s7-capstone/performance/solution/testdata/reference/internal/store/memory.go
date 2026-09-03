// Package store keeps notes. Memory is the only implementation today; the
// Store interface is declared where it is consumed, so a file-backed or
// database-backed store can replace it without the domain noticing.
package store

import (
	"errors"
	"fmt"
	"sync"

	"tutor.local/capstone-reference/internal/note"
)

// ErrDuplicate is returned when an id is already taken.
var ErrDuplicate = errors.New("note already exists")

// Store is the behaviour the command line depends on.
type Store interface {
	Add(n note.Note) error
	Get(id string) (note.Note, error)
	List(tag string) []note.Note
}

// Memory is a concurrency-safe in-memory Store. Insertion order is preserved
// so listings are stable, which keeps the tests honest.
//
// byTag is an index maintained by Add: List used to scan every note and call
// HasTag on each, which the line-level profile blamed for 860 ms of the 940 ms
// spent inside List. See PERF.md — the invariant it costs is that every write
// path touching byID must touch byTag in the same critical section.
type Memory struct {
	mu    sync.RWMutex
	byID  map[string]note.Note
	order []string
	byTag map[string][]string
}

// NewMemory returns an empty store ready for use.
func NewMemory() *Memory {
	return &Memory{
		byID:  make(map[string]note.Note),
		byTag: make(map[string][]string),
	}
}

// Add stores a note, rejecting an id that is empty or already present.
func (m *Memory) Add(n note.Note) error {
	if n.ID == "" {
		return fmt.Errorf("add note: %w", errors.New("id is required"))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[n.ID]; ok {
		return fmt.Errorf("add note %s: %w", n.ID, ErrDuplicate)
	}
	m.byID[n.ID] = n
	m.order = append(m.order, n.ID)
	for _, tag := range n.Tags {
		m.byTag[tag] = append(m.byTag[tag], n.ID)
	}
	return nil
}

// Get returns the note with the given id, or note.ErrNotFound.
func (m *Memory) Get(id string) (note.Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.byID[id]
	if !ok {
		return note.Note{}, fmt.Errorf("get note %s: %w", id, note.ErrNotFound)
	}
	return n, nil
}

// List returns the notes in insertion order. An empty tag returns everything.
// A non-empty tag is served from the index, so the cost is proportional to the
// notes that match rather than to the notes that exist.
func (m *Memory) List(tag string) []note.Note {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := m.order
	if tag != "" {
		ids = m.byTag[note.NormaliseTag(tag)]
	}
	out := make([]note.Note, 0, len(ids))
	for _, id := range ids {
		out = append(out, m.byID[id])
	}
	return out
}

// Len reports how many notes are stored.
func (m *Memory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.byID)
}
