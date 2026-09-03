// Package memstore is a mutex-guarded, in-memory doc.Store. It is deliberately
// dull: this lesson's difficulty lives in the policy, not in the persistence.
package memstore

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"tutor.local/authorization/doc"
)

// Store keeps documents in a map. The mutex makes it safe for the concurrent
// requests an http.Server hands it — and keeps `go test -race` quiet.
type Store struct {
	mu   sync.Mutex
	docs map[string]doc.Document
	next int
}

// New returns a store pre-loaded with seed, so tests can start from documents
// with known ids and owners.
func New(seed ...doc.Document) *Store {
	s := &Store{docs: make(map[string]doc.Document, len(seed))}
	for _, d := range seed {
		s.docs[d.ID] = d
	}
	return s
}

func (s *Store) Create(ownerID string, d doc.Draft) (doc.Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	created := doc.Document{
		ID:      fmt.Sprintf("d%d", s.next),
		OwnerID: ownerID,
		Title:   d.Title,
		Body:    d.Body,
	}
	s.docs[created.ID] = created
	return created, nil
}

func (s *Store) Get(id string) (doc.Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.docs[id]
	if !ok {
		return doc.Document{}, doc.ErrNotFound
	}
	return d, nil
}

// List returns every stored document, id-sorted so tests are deterministic.
// Note what it does NOT do: filter by caller. Storage has no idea who is
// asking.
func (s *Store) List() ([]doc.Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]doc.Document, 0, len(s.docs))
	for _, d := range s.docs {
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b doc.Document) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func (s *Store) Update(id string, d doc.Draft) (doc.Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.docs[id]
	if !ok {
		return doc.Document{}, doc.ErrNotFound
	}
	cur.Title, cur.Body = d.Title, d.Body
	s.docs[id] = cur
	return cur, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.docs[id]; !ok {
		return doc.ErrNotFound
	}
	delete(s.docs, id)
	return nil
}

func (s *Store) Archive(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.docs[id]
	if !ok {
		return doc.ErrNotFound
	}
	cur.Archived = true
	s.docs[id] = cur
	return nil
}
