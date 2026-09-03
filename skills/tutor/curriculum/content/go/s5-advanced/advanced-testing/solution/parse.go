// Package logkit parses, summarizes and stores structured log lines.
package logkit

import (
	"errors"
	"fmt"
	"strings"
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
	var b strings.Builder
	b.Grow(len(ev.Level) + len(ev.Source) + len(ev.Message) + 2)
	writeEscaped(&b, ev.Level)
	b.WriteByte('|')
	writeEscaped(&b, ev.Source)
	b.WriteByte('|')
	writeEscaped(&b, ev.Message)
	return b.String()
}

func writeEscaped(b *strings.Builder, field string) {
	for i := 0; i < len(field); i++ {
		switch c := field[i]; c {
		case '\\':
			b.WriteString(`\\`)
		case '|':
			b.WriteString(`\|`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteByte(c)
		}
	}
}

// ParseLine decodes one wire line produced by Format. Every error it returns
// wraps ErrMalformed.
func ParseLine(line string) (Event, error) {
	if strings.ContainsAny(line, "\n\r") {
		return Event{}, fmt.Errorf("%w: raw newline in line", ErrMalformed)
	}

	fields := make([]string, 0, 3)
	var field strings.Builder
	for i := 0; i < len(line); i++ {
		switch c := line[i]; c {
		case '|':
			fields = append(fields, field.String())
			field.Reset()
		case '\\':
			i++
			if i == len(line) {
				return Event{}, fmt.Errorf(`%w: line ends with a dangling \`, ErrMalformed)
			}
			switch line[i] {
			case '\\':
				field.WriteByte('\\')
			case '|':
				field.WriteByte('|')
			case 'n':
				field.WriteByte('\n')
			case 'r':
				field.WriteByte('\r')
			default:
				return Event{}, fmt.Errorf(`%w: unknown escape \%s`, ErrMalformed, line[i:i+1])
			}
		default:
			field.WriteByte(c)
		}
	}
	fields = append(fields, field.String())

	if len(fields) != 3 {
		return Event{}, fmt.Errorf("%w: got %d fields, want 3", ErrMalformed, len(fields))
	}
	ev := Event{Level: fields[0], Source: fields[1], Message: fields[2]}
	if _, ok := LevelRank(ev.Level); !ok {
		return Event{}, fmt.Errorf("%w: unknown level %q", ErrMalformed, ev.Level)
	}
	return ev, nil
}
