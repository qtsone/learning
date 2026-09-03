package note

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// MaxLineBytes bounds one input line. Every parser needs a size limit before
// it needs anything else: without one, the input decides how much memory the
// program uses, and the input is not ours.
const MaxLineBytes = 1024

// Errors ParseLine reports. Callers match with errors.Is; the wrapped text
// says which field was wrong, and never echoes the offending bytes.
var (
	ErrLineTooLong = errors.New("line is too long")
	ErrNotUTF8     = errors.New("line is not valid UTF-8")
	ErrControlChar = errors.New("line contains a control character")
	ErrFieldCount  = errors.New("line must be id|title or id|title|tags")
	ErrInvalidID   = errors.New("id must be 1-64 characters of A-Z a-z 0-9 _ -")
)

// idPattern is deliberately strict: ids end up in file names and log lines,
// so anything that could mean something to a filesystem or a terminal is out.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidID reports whether s is acceptable as a note id.
func ValidID(s string) bool { return idPattern.MatchString(s) }

// ParseLine turns one line of the import format into a Note:
//
//	id|title
//	id|title|tag,tag
//
// It is the program's only trust boundary for note data, so it validates
// before it builds: size, encoding, control characters, field count, then the
// per-field rules. created is supplied by the caller so parsing stays a pure
// function and tests do not depend on the clock.
func ParseLine(line string, created time.Time) (Note, error) {
	if len(line) > MaxLineBytes {
		return Note{}, fmt.Errorf("%w: %d bytes, limit %d", ErrLineTooLong, len(line), MaxLineBytes)
	}
	if !utf8.ValidString(line) {
		return Note{}, ErrNotUTF8
	}
	for _, r := range line {
		if unicode.IsControl(r) {
			return Note{}, fmt.Errorf("%w: U+%04X", ErrControlChar, r)
		}
	}

	fields := strings.Split(line, "|")
	if len(fields) < 2 || len(fields) > 3 {
		return Note{}, fmt.Errorf("%w: got %d field(s)", ErrFieldCount, len(fields))
	}

	id := strings.TrimSpace(fields[0])
	if !ValidID(id) {
		return Note{}, fmt.Errorf("%w: %d character(s)", ErrInvalidID, len([]rune(id)))
	}

	var tags []string
	if len(fields) == 3 {
		tags = strings.Split(fields[2], ",")
	}
	return New(id, fields[1], tags, created)
}

// Line renders a note back into the import format. ParseLine(n.Line()) is n,
// which is the property the fuzz target checks.
func (n Note) Line() string {
	if len(n.Tags) == 0 {
		return n.ID + "|" + n.Title
	}
	return n.ID + "|" + n.Title + "|" + strings.Join(n.Tags, ",")
}

// Equal compares two notes field by field. Time values need Equal rather than
// ==, so the whole comparison gets a method.
func (n Note) Equal(other Note) bool {
	if n.ID != other.ID || n.Title != other.Title || len(n.Tags) != len(other.Tags) {
		return false
	}
	for i := range n.Tags {
		if n.Tags[i] != other.Tags[i] {
			return false
		}
	}
	return n.Created.Equal(other.Created)
}
