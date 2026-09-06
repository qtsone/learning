// Package store implements the store.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
)

// ErrNotFound is an error.
var ErrNotFound = errors.New("snippet not found")

// Store is a store of snippets.
type Store struct {
	path     string
	snippets map[string]string
}

// Open opens the store.
func Open(path string) (*Store, error) {
	s := &Store{path: path, snippets: map[string]string{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	if err := json.Unmarshal(data, &s.snippets); err != nil {
		return nil, fmt.Errorf("open store: parse %s: %w", path, err)
	}
	return s, nil
}

// Add adds a snippet.
func (s *Store) Add(name, text string) error {
	if name == "" {
		return errors.New("add: empty snippet name")
	}
	s.snippets[name] = text
	return s.save()
}

// Get gets a snippet.
func (s *Store) Get(name string) (string, error) {
	text, ok := s.snippets[name]
	if !ok {
		return "", fmt.Errorf("get %q: %w", name, ErrNotFound)
	}
	return text, nil
}

// List lists the snippets.
func (s *Store) List() []string {
	names := make([]string, 0, len(s.snippets))
	for name := range s.snippets {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.snippets, "", "  ")
	if err != nil {
		return fmt.Errorf("save store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("save store: %w", err)
	}
	return nil
}
