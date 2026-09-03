package main

import (
	"errors"
	"fmt"
	"slices"
)

// ErrNoteNotFound reports that no note with the requested id exists. Callers
// match it with errors.Is instead of comparing message text.
var ErrNoteNotFound = errors.New("note not found")

type Note struct {
	ID   string   `json:"id"`
	Text string   `json:"text"`
	Tags []string `json:"tags"`
}

// Store is everything the commands need from storage. Commands receive it as a
// dependency, so a test can hand them an in-memory implementation and a real
// main can hand them something else later.
type Store interface {
	Add(text string, tags []string) Note
	All() []Note
	Tag(id string, tags ...string) (Note, error)
	Tags() []string
}

// MemStore keeps notes in memory for the life of one process. Persistence is
// not this lesson's subject.
type MemStore struct {
	notes []Note
	seq   int
}

func NewMemStore() *MemStore { return &MemStore{} }

func (m *MemStore) Add(text string, tags []string) Note {
	m.seq++
	n := Note{ID: fmt.Sprintf("n%d", m.seq), Text: text, Tags: append([]string{}, tags...)}
	m.notes = append(m.notes, n)
	return n
}

func (m *MemStore) All() []Note {
	return append([]Note{}, m.notes...)
}

func (m *MemStore) Tag(id string, tags ...string) (Note, error) {
	for i := range m.notes {
		if m.notes[i].ID != id {
			continue
		}
		for _, t := range tags {
			if !slices.Contains(m.notes[i].Tags, t) {
				m.notes[i].Tags = append(m.notes[i].Tags, t)
			}
		}
		return m.notes[i], nil
	}
	return Note{}, fmt.Errorf("%w: %s", ErrNoteNotFound, id)
}

func (m *MemStore) Tags() []string {
	out := []string{}
	for _, n := range m.notes {
		for _, t := range n.Tags {
			if !slices.Contains(out, t) {
				out = append(out, t)
			}
		}
	}
	slices.Sort(out)
	return out
}
