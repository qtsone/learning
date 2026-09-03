// Package note holds the domain type and its rules. It knows nothing about
// storage, transport, or the command line, so its tests need nothing either.
package note

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// MaxTitleRunes bounds a title. Runes, not bytes: a title is text a person
// reads, and byte length would treat "é" as two characters.
const MaxTitleRunes = 120

// Errors callers are expected to match with errors.Is.
var (
	ErrNotFound     = errors.New("note not found")
	ErrTitleEmpty   = errors.New("title is required")
	ErrTitleTooLong = errors.New("title is too long")
)

// Note is a single captured thought. Values are normalised at construction,
// so every Note in the system satisfies the rules below.
type Note struct {
	ID      string
	Title   string
	Tags    []string
	Created time.Time
}

// New validates and normalises a note: the id must satisfy ValidID, the title
// is trimmed and bounded, the tags are lower-cased, de-duplicated and sorted,
// and the timestamp is UTC.
func New(id, title string, tags []string, created time.Time) (Note, error) {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return Note{}, ErrInvalidID
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Note{}, ErrTitleEmpty
	}
	if n := len([]rune(title)); n > MaxTitleRunes {
		return Note{}, fmt.Errorf("%w: %d runes, limit %d", ErrTitleTooLong, n, MaxTitleRunes)
	}
	return Note{
		ID:      id,
		Title:   title,
		Tags:    NormaliseTags(tags),
		Created: created.UTC(),
	}, nil
}

// NormaliseTags lower-cases, trims, de-duplicates and sorts tags. It always
// returns a non-nil slice so callers never branch on nil.
func NormaliseTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

// HasTag reports whether the note carries tag, comparing normalised forms.
func (n Note) HasTag(tag string) bool {
	tag = strings.ToLower(strings.TrimSpace(tag))
	for _, t := range n.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// String renders one line per note for the command line.
func (n Note) String() string {
	if len(n.Tags) == 0 {
		return fmt.Sprintf("%s  %s", n.ID, n.Title)
	}
	return fmt.Sprintf("%s  %s  [%s]", n.ID, n.Title, strings.Join(n.Tags, " "))
}
