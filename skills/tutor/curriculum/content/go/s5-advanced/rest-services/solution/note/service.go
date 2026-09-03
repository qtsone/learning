package note

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Limits enforced by validation, counted in runes — "héllo" is five
// characters no matter how many bytes it takes.
const (
	MaxTitleLen   = 120
	MaxContentLen = 8000
)

// Service implements the business rules: normalization, validation, and
// ordering guarantees. It reaches storage only through the Store interface.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create normalizes and validates d, then stores it.
func (s *Service) Create(d Draft) (Note, error) {
	d = normalize(d)
	if err := validate(d); err != nil {
		return Note{}, err
	}
	return s.store.Create(d)
}

// Get returns the note with the given id.
func (s *Service) Get(id int64) (Note, error) {
	return s.store.Get(id)
}

// List returns all notes sorted by id, ascending. It never returns a nil
// slice — an empty service must encode as [] on the wire, not null.
func (s *Service) List() ([]Note, error) {
	ns, err := s.store.List()
	if err != nil {
		return nil, err
	}
	if ns == nil {
		ns = []Note{}
	}
	slices.SortFunc(ns, func(a, b Note) int { return cmp.Compare(a.ID, b.ID) })
	return ns, nil
}

// Update normalizes and validates d, then fully replaces the note with the
// given id.
func (s *Service) Update(id int64, d Draft) (Note, error) {
	d = normalize(d)
	if err := validate(d); err != nil {
		return Note{}, err
	}
	return s.store.Update(id, d)
}

// Delete removes the note with the given id.
func (s *Service) Delete(id int64) error {
	return s.store.Delete(id)
}

// normalize runs before validate so that "   " fails the required check
// instead of sneaking through as a non-empty title.
func normalize(d Draft) Draft {
	return Draft{
		Title:   strings.TrimSpace(d.Title),
		Content: strings.TrimSpace(d.Content),
	}
}

func validate(d Draft) error {
	fields := ValidationError{}
	switch {
	case d.Title == "":
		fields["title"] = "required"
	case utf8.RuneCountInString(d.Title) > MaxTitleLen:
		fields["title"] = fmt.Sprintf("must be at most %d characters", MaxTitleLen)
	}
	if utf8.RuneCountInString(d.Content) > MaxContentLen {
		fields["content"] = fmt.Sprintf("must be at most %d characters", MaxContentLen)
	}
	if len(fields) == 0 {
		// Returning the empty map here would hand callers a non-nil error
		// interface holding an empty value — the typed-nil trap.
		return nil
	}
	return fields
}
