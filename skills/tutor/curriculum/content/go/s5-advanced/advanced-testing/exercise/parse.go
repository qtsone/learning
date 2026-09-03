// Package logkit parses, summarizes and stores structured log lines.
package logkit

import (
	"errors"
	"fmt"
)

// Levels lists every valid event level, ordered from least to most severe.
var Levels = []string{"debug", "info", "warn", "error"}

// ErrMalformed wraps every error returned for input that cannot be decoded.
var ErrMalformed = errors.New("malformed log line")

// Event is one parsed log line.
type Event struct {
	Level   string
	Source  string
	Message string
}

// LevelRank reports the severity rank of level and whether it is a known level.
func LevelRank(level string) (int, bool) {
	for i, l := range Levels {
		if l == level {
			return i, true
		}
	}
	return 0, false
}

// Format encodes ev as one wire line: the three fields joined by '|', with
// '\', '|', newline and carriage return escaped as `\\`, `\|`, `\n` and `\r`.
// Format does not validate the level; ParseLine does.
func Format(ev Event) string {
	// TODO: encode the three fields with the escaping described above.
	return ""
}

// ParseLine decodes one wire line produced by Format. Every error it returns
// wraps ErrMalformed.
func ParseLine(line string) (Event, error) {
	// TODO: split on unescaped '|', unescape each field, validate the level.
	return Event{}, fmt.Errorf("%w: ParseLine is not implemented", ErrMalformed)
}
