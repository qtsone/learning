// Package memstore is an in-memory implementation of note.Store. It lets the
// service run and be tested without a database; a SQL-backed store would slot
// in behind the same interface, without note or api changing.
package memstore

import (
	"sync"

	"tutor.local/rest-services/note"
)

// Store implements note.Store with a mutex-guarded map. Handlers run on
// separate goroutines, so storage must be safe for concurrent use.
type Store struct {
	mu    sync.Mutex
	seq   int64
	notes map[int64]note.Note
}

func New() *Store {
	return &Store{notes: make(map[int64]note.Note)}
}

func (s *Store) Create(d note.Draft) (note.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	n := note.Note{ID: s.seq, Title: d.Title, Content: d.Content}
	s.notes[n.ID] = n
	return n, nil
}

func (s *Store) Get(id int64) (note.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notes[id]
	if !ok {
		return note.Note{}, note.ErrNotFound
	}
	return n, nil
}

// List returns the notes in no particular order — map iteration is
// deliberately randomized in Go. Ordering is the service's promise, not ours.
func (s *Store) List() ([]note.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]note.Note, 0, len(s.notes))
	for _, n := range s.notes {
		out = append(out, n)
	}
	return out, nil
}

func (s *Store) Update(id int64, d note.Draft) (note.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notes[id]; !ok {
		return note.Note{}, note.ErrNotFound
	}
	n := note.Note{ID: id, Title: d.Title, Content: d.Content}
	s.notes[id] = n
	return n, nil
}

func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notes[id]; !ok {
		return note.ErrNotFound
	}
	delete(s.notes, id)
	return nil
}
