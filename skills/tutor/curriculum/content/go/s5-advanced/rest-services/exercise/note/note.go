// Package note holds the domain: the types, rules, and error vocabulary of
// the notes service. It knows nothing about HTTP or how notes are stored.
package note

import (
	"errors"
	"sort"
	"strings"
)

// Note is a stored note. IDs are assigned by the storage layer.
type Note struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// Draft is the client-supplied part of a note — what it takes to create or
// fully replace one.
type Draft struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ErrNotFound is the domain's word for "no such note". Every Store
// implementation contracts to return it (possibly wrapped) from Get, Update,
// and Delete when the id does not exist.
var ErrNotFound = errors.New("note not found")

// ValidationError maps field names to what is wrong with them. It is data,
// not prose: the HTTP layer renders it as a field-level 400 response.
type ValidationError map[string]string

func (e ValidationError) Error() string {
	fields := make([]string, 0, len(e))
	for f := range e {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	return "invalid fields: " + strings.Join(fields, ", ")
}

// Store is what the domain needs from storage — declared here, on the
// consumer side, so storage depends on the domain and never the reverse.
type Store interface {
	Create(d Draft) (Note, error)
	Get(id int64) (Note, error)
	List() ([]Note, error)
	Update(id int64, d Draft) (Note, error)
	Delete(id int64) error
}
